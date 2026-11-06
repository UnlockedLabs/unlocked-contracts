package programs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/coreapi"
	"github.com/UnlockedLabs/unlocked-contracts/programs"
)

var (
	start      = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end        = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	facilityID = uint(3)
)

// The answers fakePrograms gives.
var (
	conflicts = []programs.ResidentTransferProgramConflicts{{ProgramName: "GED", ClassName: "GED Morning"}}
	history   = []programs.ResidentProgramClassInfo{{
		ProgramName: "GED", ClassName: "GED Morning", Status: "Active", EnrollmentStatus: "Enrolled",
		StartDate: "2026-09-01", EndDate: "2026-12-01", AttendancePercentage: "90.00", CohortID: 4, ProgramID: 2,
		EnrollmentID: 9, PresentAttendance: 9, AbsentAttendance: 1, CreditTypes: "Completion", UpdatedAt: "2026-09-02",
		Schedule: "Mon 9:00", IsCanvas: true, Source: "canvas",
	}}
	facilities = map[uint]programs.FacilityPrograms{3: {ActivePrograms: 2, ActiveClasses: 5, Classes: 6, Programs: 3, Rooms: 8}}
	created    = map[uint]time.Time{7: start}
	overview   = programs.ProgramEngagementOverview{
		ActiveResidents: 10, PreviouslyEngagedResidents: 4, NeverEngagedResidents: 6, TotalResidents: 20,
		TopPrograms: []programs.ProgramCompletionRank{{ProgramName: "GED", Enrolled: 8, Completed: 4, CompletionRate: 50}},
	}
	second = []programs.SecondProgramEnrollmentRow{{FacilityName: "BCF", ProgramType: "Educational", CompletedFirst: 4, EnrolledSecond: 1, Rate: 25}}
	matrix = []programs.ProgramCompletionMatrixCell{{
		ProgramName: "GED", FacilityID: 3, FacilityName: "BCF", Enrolled: 8, Completed: 4, CompletionRate: 50,
		Insufficient: true, DeltaFromFacilityAverage: -5.5,
	}}
	load = programs.ProgramLoadDistribution{
		Statewide:  []programs.ProgramLoadBucket{{Bucket: "4+", Count: 2}},
		ByFacility: []programs.ProgramLoadFacilityRow{{FacilityID: 3, FacilityName: "BCF", Zero: 1, One: 2, Two: 3, Three: 4, FourPlus: 5, Total: 15}},
	}
	byType  = []programs.ProgramTypeEnrollment{{ProgramType: "Educational", Enrolled: 8, Completed: 4, Rate: 50}}
	metrics = programs.ClassDashboardMetrics{ActiveClasses: 5, ScheduledClasses: 2, TotalEnrollments: 40, TotalSeats: 60, AttendanceConcerns: 3}
	health  = []programs.FacilityHealthSummary{{
		FacilityID: 3, FacilityName: "BCF", Programs: 3, ActiveClasses: 5, Enrollment: 40, MissingAttendance: 2, AttendanceConcerns: 1,
	}}
	providerUsers = []coreapi.ProviderUser{{
		Username: "jdoe", NameFirst: "Jane", NameLast: "Doe", Email: "jdoe@x", ExternalUserID: "42", ExternalUsername: "jane.doe",
	}}
)

// liveUsers is what LiveUsers returns, as one value to compare.
type liveUsers struct {
	Users []coreapi.ProviderUser
	OK    bool
}

// fakePrograms records the last question it was asked and gives the answers
// above, or err when it's set.
type fakePrograms struct {
	mu     sync.Mutex
	method string
	args   []any
	err    error
	// userIDs is what EnrolledUserIDs and EligibleUserIDs answer.
	userIDs []uint
	// listsUsers is LiveUsers' ok: whether the provider lists its users.
	listsUsers bool
}

var _ programs.Programs = (*fakePrograms)(nil)

func (f *fakePrograms) record(method string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.method, f.args = method, args
}

func (f *fakePrograms) asked() (string, []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.method, f.args
}

func (f *fakePrograms) EnrolledUserIDs(_ context.Context, classID uint) ([]uint, error) {
	f.record("EnrolledUserIDs", classID)
	return f.userIDs, f.err
}

func (f *fakePrograms) EligibleUserIDs(_ context.Context, classID uint) ([]uint, error) {
	f.record("EligibleUserIDs", classID)
	return f.userIDs, f.err
}

func (f *fakePrograms) TransferConflicts(_ context.Context, userID, toFacilityID uint) ([]programs.ResidentTransferProgramConflicts, error) {
	f.record("TransferConflicts", userID, toFacilityID)
	return conflicts, f.err
}

func (f *fakePrograms) EndEnrollmentsForTransfer(_ context.Context, userID, fromFacilityID uint) error {
	f.record("EndEnrollmentsForTransfer", userID, fromFacilityID)
	return f.err
}

func (f *fakePrograms) WithdrawEnrollments(_ context.Context, userID uint) error {
	f.record("WithdrawEnrollments", userID)
	return f.err
}

func (f *fakePrograms) ResidentPrograms(_ context.Context, userID uint) ([]programs.ResidentProgramClassInfo, error) {
	f.record("ResidentPrograms", userID)
	return history, f.err
}

func (f *fakePrograms) FacilityPrograms(_ context.Context, facilityIDs []uint) (map[uint]programs.FacilityPrograms, error) {
	f.record("FacilityPrograms", facilityIDs)
	return facilities, f.err
}

func (f *fakePrograms) ClassHistoryCreatedAt(_ context.Context, ids []uint) (map[uint]time.Time, error) {
	f.record("ClassHistoryCreatedAt", ids)
	return created, f.err
}

func (f *fakePrograms) EngagementOverview(_ context.Context, start, end *time.Time, facilityID *uint) (programs.ProgramEngagementOverview, error) {
	f.record("EngagementOverview", start, end, facilityID)
	return overview, f.err
}

func (f *fakePrograms) SecondProgramEnrollment(_ context.Context, facilityID *uint) ([]programs.SecondProgramEnrollmentRow, error) {
	f.record("SecondProgramEnrollment", facilityID)
	return second, f.err
}

func (f *fakePrograms) CompletionMatrix(_ context.Context, facilityID *uint) ([]programs.ProgramCompletionMatrixCell, error) {
	f.record("CompletionMatrix", facilityID)
	return matrix, f.err
}

func (f *fakePrograms) LoadDistribution(_ context.Context, facilityID *uint) (programs.ProgramLoadDistribution, error) {
	f.record("LoadDistribution", facilityID)
	return load, f.err
}

func (f *fakePrograms) EnrollmentByProgramType(_ context.Context, facilityID *uint) ([]programs.ProgramTypeEnrollment, error) {
	f.record("EnrollmentByProgramType", facilityID)
	return byType, f.err
}

func (f *fakePrograms) ClassMetrics(_ context.Context, facilityID *uint) (programs.ClassDashboardMetrics, error) {
	f.record("ClassMetrics", facilityID)
	return metrics, f.err
}

func (f *fakePrograms) FacilityHealth(_ context.Context, facilityID *uint, days int, tz string) ([]programs.FacilityHealthSummary, error) {
	f.record("FacilityHealth", facilityID, days, tz)
	return health, f.err
}

func (f *fakePrograms) LiveUsers(_ context.Context, providerID uint) ([]coreapi.ProviderUser, bool, error) {
	f.record("LiveUsers", providerID)
	if !f.listsUsers {
		return nil, false, f.err
	}
	return providerUsers, true, f.err
}

func (f *fakePrograms) MappingsChanged(providerID, userID uint) {
	f.record("MappingsChanged", providerID, userID)
}

// testToken is the shared service token in these tests.
const testToken = "s3cret"

// newPair serves fake with NewHandler and returns a Client that asks it.
func newPair(t *testing.T, fake *fakePrograms) *programs.Client {
	t.Helper()
	srv := httptest.NewServer(programs.NewHandler(fake, testToken))
	t.Cleanup(srv.Close)
	return programs.NewClient(srv.URL+"/", testToken, nil)
}

// newRequest is a question to the handler, with the token.
func newRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	return req
}

// question is one of core's questions: how to ask it, what Programs should be
// asked, and the answer core should get.
type question struct {
	name       string
	ask        func(context.Context, *programs.Client) (any, error)
	wantMethod string
	wantArgs   []any
	want       any
}

var noFacility *uint

var questions = []question{
	{
		name: "EnrolledUserIDs",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.EnrolledUserIDs(ctx, 4)
		},
		wantMethod: "EnrolledUserIDs", wantArgs: []any{uint(4)}, want: []uint{12, 13},
	},
	{
		name: "EligibleUserIDs",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.EligibleUserIDs(ctx, 4)
		},
		wantMethod: "EligibleUserIDs", wantArgs: []any{uint(4)}, want: []uint{12, 13},
	},
	{
		name: "TransferConflicts",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.TransferConflicts(ctx, 12, 3)
		},
		wantMethod: "TransferConflicts", wantArgs: []any{uint(12), uint(3)}, want: conflicts,
	},
	{
		name: "EndEnrollmentsForTransfer",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return nil, c.EndEnrollmentsForTransfer(ctx, 12, 3)
		},
		wantMethod: "EndEnrollmentsForTransfer", wantArgs: []any{uint(12), uint(3)}, want: nil,
	},
	{
		name: "WithdrawEnrollments",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return nil, c.WithdrawEnrollments(ctx, 12)
		},
		wantMethod: "WithdrawEnrollments", wantArgs: []any{uint(12)}, want: nil,
	},
	{
		name: "ResidentPrograms",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.ResidentPrograms(ctx, 12)
		},
		wantMethod: "ResidentPrograms", wantArgs: []any{uint(12)}, want: history,
	},
	{
		name: "FacilityPrograms",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.FacilityPrograms(ctx, []uint{3, 4})
		},
		wantMethod: "FacilityPrograms", wantArgs: []any{[]uint{3, 4}}, want: facilities,
	},
	{
		name: "ClassHistoryCreatedAt",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.ClassHistoryCreatedAt(ctx, []uint{7, 8})
		},
		wantMethod: "ClassHistoryCreatedAt", wantArgs: []any{[]uint{7, 8}}, want: created,
	},
	{
		name: "EngagementOverview",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.EngagementOverview(ctx, &start, &end, &facilityID)
		},
		wantMethod: "EngagementOverview", wantArgs: []any{&start, &end, &facilityID}, want: overview,
	},
	{
		name: "EngagementOverview over everything",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.EngagementOverview(ctx, nil, nil, nil)
		},
		wantMethod: "EngagementOverview", wantArgs: []any{(*time.Time)(nil), (*time.Time)(nil), noFacility}, want: overview,
	},
	{
		name: "SecondProgramEnrollment",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.SecondProgramEnrollment(ctx, &facilityID)
		},
		wantMethod: "SecondProgramEnrollment", wantArgs: []any{&facilityID}, want: second,
	},
	{
		name: "CompletionMatrix",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.CompletionMatrix(ctx, &facilityID)
		},
		wantMethod: "CompletionMatrix", wantArgs: []any{&facilityID}, want: matrix,
	},
	{
		name: "LoadDistribution",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.LoadDistribution(ctx, &facilityID)
		},
		wantMethod: "LoadDistribution", wantArgs: []any{&facilityID}, want: load,
	},
	{
		name: "EnrollmentByProgramType",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.EnrollmentByProgramType(ctx, &facilityID)
		},
		wantMethod: "EnrollmentByProgramType", wantArgs: []any{&facilityID}, want: byType,
	},
	{
		name: "ClassMetrics",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.ClassMetrics(ctx, &facilityID)
		},
		wantMethod: "ClassMetrics", wantArgs: []any{&facilityID}, want: metrics,
	},
	{
		name: "ClassMetrics at every facility",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.ClassMetrics(ctx, nil)
		},
		wantMethod: "ClassMetrics", wantArgs: []any{noFacility}, want: metrics,
	},
	{
		name: "FacilityHealth",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			return c.FacilityHealth(ctx, &facilityID, 7, "America/Chicago")
		},
		wantMethod: "FacilityHealth", wantArgs: []any{&facilityID, 7, "America/Chicago"}, want: health,
	},
	{
		name: "LiveUsers",
		ask: func(ctx context.Context, c *programs.Client) (any, error) {
			users, ok, err := c.LiveUsers(ctx, 2)
			return liveUsers{users, ok}, err
		},
		wantMethod: "LiveUsers", wantArgs: []any{uint(2)}, want: liveUsers{providerUsers, true},
	},
}

// TestRoundTrip asks every question through Client and Handler and checks
// that Programs gets the same arguments and core the same answer as a direct
// call.
func TestRoundTrip(t *testing.T) {
	for _, q := range questions {
		t.Run(q.name, func(t *testing.T) {
			fake := &fakePrograms{userIDs: []uint{12, 13}, listsUsers: true}
			c := newPair(t, fake)
			got, err := q.ask(context.Background(), c)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if !reflect.DeepEqual(got, q.want) {
				t.Errorf("answer = %#v, want %#v", got, q.want)
			}
			method, args := fake.asked()
			if method != q.wantMethod || !reflect.DeepEqual(args, q.wantArgs) {
				t.Errorf("Programs was asked %s%v, want %s%v", method, args, q.wantMethod, q.wantArgs)
			}
		})
	}
}

// A provider that can't list its users answers ok false and no users, not an
// error.
func TestLiveUsersNotListed(t *testing.T) {
	users, ok, err := newPair(t, &fakePrograms{}).LiveUsers(context.Background(), 2)
	if err != nil || ok || users != nil {
		t.Errorf("LiveUsers = %v, %v, %v; want nil, false, nil", users, ok, err)
	}
}

func TestMappingsChanged(t *testing.T) {
	fake := &fakePrograms{}
	newPair(t, fake).MappingsChanged(2, 7)
	method, args := fake.asked()
	if method != "MappingsChanged" || !reflect.DeepEqual(args, []any{uint(2), uint(7)}) {
		t.Errorf("Programs was asked %s%v, want MappingsChanged[2 7]", method, args)
	}
}

func TestMappingsChangedLogsFailure(t *testing.T) {
	var logs bytes.Buffer
	defer slog.SetDefault(slog.Default())
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	programs.NewClient(srv.URL, testToken, nil).MappingsChanged(2, 7)

	if line := logs.String(); !strings.Contains(line, "provider_id=2") || !strings.Contains(line, "user_id=7") || !strings.Contains(line, "502") {
		t.Errorf("log = %q, want the provider, the user and the 502", line)
	}
}

// TestEmptyAnswers checks that no rows stay no rows, and nil stays nil, so
// core's JSON ([] or null) is what it was before the move.
func TestEmptyAnswers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		userIDs []uint
	}{
		{"empty", []uint{}},
		{"nil", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newPair(t, &fakePrograms{userIDs: tc.userIDs}).EnrolledUserIDs(context.Background(), 4)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if !reflect.DeepEqual(got, tc.userIDs) {
				t.Errorf("answer = %#v, want %#v", got, tc.userIDs)
			}
		})
	}
}

// TestProgramsFails checks that every question returns Programs' error.
func TestProgramsFails(t *testing.T) {
	fake := &fakePrograms{err: errors.New("database is down")}
	c := newPair(t, fake)
	for _, q := range questions {
		t.Run(q.name, func(t *testing.T) {
			_, err := q.ask(context.Background(), c)
			if err == nil || !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "database is down") {
				t.Errorf("err = %v, want the 500 and Programs' message", err)
			}
		})
	}
}

// TestBadAnswers checks the answers Client turns into errors.
func TestBadAnswers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"error without JSON", http.StatusBadGateway, "<html>bad gateway</html>", "502 Bad Gateway"},
		{"error without a message", http.StatusNotFound, `{"message":"","data":null}`, "404 Not Found"},
		{"error with a message", http.StatusBadRequest, `{"message":"read arguments: EOF","data":null}`, "read arguments: EOF"},
		{"200 that isn't JSON", http.StatusOK, "not json", "read answer"},
		{"200 with the wrong type", http.StatusOK, `{"message":"","data":"twelve"}`, "read answer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			_, err := programs.NewClient(srv.URL, testToken, nil).EnrolledUserIDs(context.Background(), 4)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestProgramsUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	if _, err := programs.NewClient(srv.URL, testToken, nil).EnrolledUserIDs(context.Background(), 4); err == nil {
		t.Error("err = nil, want an error when Programs can't be reached")
	}
}

// TestCallerCancels checks that a question stops when the caller's context
// ends, such as core's request being canceled. The server gives up after 5
// seconds, so a Client that ignored the context fails instead of hanging.
func TestCallerCancels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// The server notices the client hang up only once the body is read.
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := programs.NewClient(srv.URL, testToken, nil).EnrolledUserIDs(ctx, 4)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

// TestWire pins the paths and the JSON on the wire, so a core and a Programs
// built with different versions of this package still agree. Change one only
// with a plan for sites that run both versions.
func TestWire(t *testing.T) {
	for _, tc := range []struct {
		path       string
		body       string
		wantMethod string
		wantArgs   []any
		wantAnswer string
	}{
		{"enrolled-user-ids", `{"class_id":4}`, "EnrolledUserIDs", []any{uint(4)}, `{"message":"","data":[12,13]}`},
		{"eligible-user-ids", `{"class_id":4}`, "EligibleUserIDs", []any{uint(4)}, ""},
		{
			"transfer-conflicts", `{"user_id":12,"facility_id":3}`,
			"TransferConflicts", []any{uint(12), uint(3)},
			`{"message":"","data":[{"program_name":"GED","class_name":"GED Morning"}]}`,
		},
		{
			"end-enrollments-for-transfer", `{"user_id":12,"facility_id":3}`,
			"EndEnrollmentsForTransfer", []any{uint(12), uint(3)}, `{"message":"","data":null}`,
		},
		{"withdraw-enrollments", `{"user_id":12}`, "WithdrawEnrollments", []any{uint(12)}, `{"message":"","data":null}`},
		{
			"resident-programs", `{"user_id":12}`,
			"ResidentPrograms", []any{uint(12)},
			`{"message":"","data":[{"program_name":"GED","class_name":"GED Morning","status":"Active","enrollment_status":"Enrolled",` +
				`"start_date":"2026-09-01","end_date":"2026-12-01","attendance_percentage":"90.00","cohort_id":4,"program_id":2,` +
				`"enrollment_id":9,"present_attendance":9,"absent_attendance":1,"credit_types":"Completion","updated_at":"2026-09-02",` +
				`"change_reason":"","schedule":"Mon 9:00","is_canvas":true,"source":"canvas"}]}`,
		},
		{
			"facility-programs", `{"facility_ids":[3,4]}`,
			"FacilityPrograms", []any{[]uint{3, 4}},
			`{"message":"","data":{"3":{"active_programs":2,"active_classes":5,"classes":6,"programs":3,"rooms":8}}}`,
		},
		{
			"class-history-created-at", `{"ids":[7,8]}`,
			"ClassHistoryCreatedAt", []any{[]uint{7, 8}}, `{"message":"","data":{"7":"2026-09-01T00:00:00Z"}}`,
		},
		{
			"engagement-overview", `{"start":"2026-09-01T00:00:00Z","end":"2026-10-01T00:00:00Z","facility_id":3}`,
			"EngagementOverview", []any{&start, &end, &facilityID},
			`{"message":"","data":{"active_residents":10,"previously_engaged_residents":4,"never_engaged_residents":6,` +
				`"total_residents":20,"top_programs":[{"program_name":"GED","enrolled":8,"completed":4,"completion_rate":50}]}}`,
		},
		{"engagement-overview", `{}`, "EngagementOverview", []any{(*time.Time)(nil), (*time.Time)(nil), noFacility}, ""},
		{
			"second-program-enrollment", `{"facility_id":3}`,
			"SecondProgramEnrollment", []any{&facilityID},
			`{"message":"","data":[{"facility_name":"BCF","program_type":"Educational","completed_first":4,"enrolled_second":1,"rate":25}]}`,
		},
		{
			"completion-matrix", `{}`,
			"CompletionMatrix", []any{noFacility},
			`{"message":"","data":[{"program_name":"GED","facility_id":3,"facility_name":"BCF","enrolled":8,"completed":4,` +
				`"completion_rate":50,"insufficient":true,"delta_from_facility_average":-5.5}]}`,
		},
		{
			"load-distribution", `{}`,
			"LoadDistribution", []any{noFacility},
			`{"message":"","data":{"statewide":[{"bucket":"4+","count":2}],"by_facility":[{"facility_id":3,"facility_name":"BCF",` +
				`"zero":1,"one":2,"two":3,"three":4,"four_plus":5,"total":15}]}}`,
		},
		{
			"enrollment-by-program-type", `{}`,
			"EnrollmentByProgramType", []any{noFacility},
			`{"message":"","data":[{"program_type":"Educational","enrolled":8,"completed":4,"rate":50}]}`,
		},
		{
			"class-metrics", `{}`,
			"ClassMetrics", []any{noFacility},
			`{"message":"","data":{"active_classes":5,"scheduled_classes":2,"total_enrollments":40,"total_seats":60,"attendance_concerns":3}}`,
		},
		{
			"facility-health", `{"facility_id":3,"days":7,"tz":"America/Chicago"}`,
			"FacilityHealth", []any{&facilityID, 7, "America/Chicago"},
			`{"message":"","data":[{"facility_id":3,"facility_name":"BCF","programs":3,"active_classes":5,"enrollment":40,` +
				`"missing_attendance":2,"attendance_concerns":1}]}`,
		},
		{
			"live-users", `{"provider_id":2}`,
			"LiveUsers", []any{uint(2)},
			`{"message":"","data":{"users":[{"username":"jdoe","name_first":"Jane","name_last":"Doe","email":"jdoe@x",` +
				`"external_user_id":"42","external_username":"jane.doe"}],"ok":true}}`,
		},
		{"mappings-changed", `{"provider_id":2,"user_id":7}`, "MappingsChanged", []any{uint(2), uint(7)}, `{"message":"","data":null}`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			fake := &fakePrograms{userIDs: []uint{12, 13}, listsUsers: true}
			rec := httptest.NewRecorder()
			req := newRequest(http.MethodPost, "/internal/v1/programs/"+tc.path, tc.body)
			programs.NewHandler(fake, testToken).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
			}
			method, args := fake.asked()
			if method != tc.wantMethod || !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("Programs was asked %s%v, want %s%v", method, args, tc.wantMethod, tc.wantArgs)
			}
			if tc.wantAnswer != "" {
				if got := strings.TrimSpace(rec.Body.String()); got != tc.wantAnswer {
					t.Errorf("answer =\n%s\nwant\n%s", got, tc.wantAnswer)
				}
			}
		})
	}
}

func TestHandlerRejects(t *testing.T) {
	for _, tc := range []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"arguments that aren't JSON", http.MethodPost, "/internal/v1/programs/enrolled-user-ids", "class 4", http.StatusBadRequest},
		{"arguments of the wrong type", http.MethodPost, "/internal/v1/programs/enrolled-user-ids", `{"class_id":"4"}`, http.StatusBadRequest},
		{"too much", http.MethodPost, "/internal/v1/programs/facility-programs", `{"facility_ids":[` + strings.Repeat("1,", 1<<20) + `1]}`, http.StatusBadRequest},
		{"a GET", http.MethodGet, "/internal/v1/programs/enrolled-user-ids", "", http.StatusMethodNotAllowed},
		{"an unknown question", http.MethodPost, "/internal/v1/programs/enrolled-users", "{}", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakePrograms{}
			rec := httptest.NewRecorder()
			programs.NewHandler(fake, testToken).ServeHTTP(rec, newRequest(tc.method, tc.path, tc.body))
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if method, _ := fake.asked(); method != "" {
				t.Errorf("Programs was asked %s, want nothing", method)
			}
			if tc.wantStatus == http.StatusBadRequest {
				var body struct {
					Message string `json:"message"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || !strings.HasPrefix(body.Message, "read arguments") {
					t.Errorf("body = %s, want core's JSON error", rec.Body)
				}
			}
		})
	}
}

// Only a caller with the shared service token gets answers: a missing or
// wrong token gets a 401 and Programs isn't asked, and a handler with no token
// refuses everyone.
func TestHandlerToken(t *testing.T) {
	for _, tc := range []struct {
		name          string
		handlerToken  string
		authorization string
		wantStatus    int
	}{
		{"right token", testToken, "Bearer " + testToken, http.StatusOK},
		{"no token", testToken, "", http.StatusUnauthorized},
		{"wrong token", testToken, "Bearer guess", http.StatusUnauthorized},
		{"token without Bearer", testToken, testToken, http.StatusUnauthorized},
		{"handler without a token", "", "Bearer ", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakePrograms{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/internal/v1/programs/withdraw-enrollments", strings.NewReader(`{"user_id":12}`))
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			programs.NewHandler(fake, tc.handlerToken).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tc.wantStatus, rec.Body)
			}
			method, _ := fake.asked()
			if tc.wantStatus == http.StatusOK && method != "WithdrawEnrollments" {
				t.Errorf("Programs was asked %q, want WithdrawEnrollments", method)
			}
			if tc.wantStatus != http.StatusOK {
				if method != "" {
					t.Errorf("Programs was asked %s, want nothing", method)
				}
				if got := strings.TrimSpace(rec.Body.String()); got != `{"message":"missing or wrong token","data":null}` {
					t.Errorf("body = %s, want core's JSON error", got)
				}
			}
		})
	}
}

// A Client with the wrong token gets Programs' 401 as an error.
func TestClientWrongToken(t *testing.T) {
	srv := httptest.NewServer(programs.NewHandler(&fakePrograms{}, testToken))
	t.Cleanup(srv.Close)
	_, err := programs.NewClient(srv.URL, "guess", nil).EnrolledUserIDs(context.Background(), 4)
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "missing or wrong token") {
		t.Errorf("err = %v, want the 401 with its message", err)
	}
}
