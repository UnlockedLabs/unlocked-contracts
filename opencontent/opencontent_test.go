package opencontent_test

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

	"github.com/UnlockedLabs/unlocked-contracts/opencontent"
)

var (
	start      = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end        = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	facilityID = uint(3)
	window     = opencontent.Window{Start: &start, End: &end, FacilityID: &facilityID}
	visible    = true
)

// The answers fakeOC gives.
var (
	usage = []opencontent.ContentUsage{{
		Title: "Gutenberg", Url: "/api/proxy/libraries/7/", ThumbnailUrl: "/api/photos/gutenberg.png",
		VisibilityStatus: &visible, OpenContentProviderId: 1, ContentId: 7, ContentType: "library",
		ProviderName: "Kiwix", IsFeatured: true, TotalHours: 1.5, TotalMinutes: 90,
	}}
	engagement = opencontent.ActivityEngagement{
		TotalActiveDaysMonthly: 4, TotalHoursActiveMonthly: 0.5, TotalHoursActiveWeekly: 2,
		TotalMinutesActiveWeekly: 120, TotalHoursEngaged: 1, TotalMinutesEngaged: 60,
	}
	content      = map[uint]opencontent.FacilityContent{3: {Activities: 10, Favorites: 2, VisibilityStatuses: 5}}
	interactions = opencontent.Interactions{Total: 40, Unique: 9, Change: -25}
	repeat       = opencontent.RepeatEngagement{Once: 3, TwoToFour: 4, FivePlus: 2}
	categories   = []opencontent.CategoryViews{{Category: "Science", Views: 12}}
	visits       = []opencontent.ContentVisits{{Title: "Gutenberg", Visits: 8, Change: 100}}
)

// fakeOC records the last question it was asked and gives the answers above,
// or err when it's set.
type fakeOC struct {
	mu     sync.Mutex
	method string
	args   []any
	err    error
	// usage is what TopLibraries and RecentVideos answer.
	usage []opencontent.ContentUsage
}

func (f *fakeOC) record(method string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.method, f.args = method, args
}

func (f *fakeOC) asked() (string, []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.method, f.args
}

func (f *fakeOC) EndActivity(_ context.Context, activityID int64) {
	f.record("EndActivity", activityID)
}

func (f *fakeOC) TopLibraries(_ context.Context, userID, facilityID uint) ([]opencontent.ContentUsage, error) {
	f.record("TopLibraries", userID, facilityID)
	return f.usage, f.err
}

func (f *fakeOC) RecentVideos(_ context.Context, userID uint) ([]opencontent.ContentUsage, error) {
	f.record("RecentVideos", userID)
	return f.usage, f.err
}

func (f *fakeOC) ResourcesAccessed(_ context.Context, userID uint) (int64, error) {
	f.record("ResourcesAccessed", userID)
	return 17, f.err
}

func (f *fakeOC) ActivityEngagement(_ context.Context, userID uint) (opencontent.ActivityEngagement, error) {
	f.record("ActivityEngagement", userID)
	return engagement, f.err
}

func (f *fakeOC) FacilityContent(_ context.Context, facilityIDs []uint) (map[uint]opencontent.FacilityContent, error) {
	f.record("FacilityContent", facilityIDs)
	return content, f.err
}

func (f *fakeOC) Interactions(_ context.Context, w opencontent.Window) (opencontent.Interactions, error) {
	f.record("Interactions", w)
	return interactions, f.err
}

func (f *fakeOC) AvgSessionMinutes(_ context.Context, w opencontent.Window) (float64, error) {
	f.record("AvgSessionMinutes", w)
	return 12.5, f.err
}

func (f *fakeOC) RepeatEngagement(_ context.Context, w opencontent.Window) (opencontent.RepeatEngagement, error) {
	f.record("RepeatEngagement", w)
	return repeat, f.err
}

func (f *fakeOC) LibraryViewsByCategory(_ context.Context, w opencontent.Window) ([]opencontent.CategoryViews, error) {
	f.record("LibraryViewsByCategory", w)
	return categories, f.err
}

func (f *fakeOC) TopLibraryVisits(_ context.Context, w opencontent.Window, limit int) ([]opencontent.ContentVisits, error) {
	f.record("TopLibraryVisits", w, limit)
	return visits, f.err
}

func (f *fakeOC) TopVideoVisits(_ context.Context, w opencontent.Window, limit int) ([]opencontent.ContentVisits, error) {
	f.record("TopVideoVisits", w, limit)
	return visits, f.err
}

func (f *fakeOC) ResetDemoActivity(_ context.Context, before time.Time) error {
	f.record("ResetDemoActivity", before)
	return f.err
}

func (f *fakeOC) SeedDemoActivity(_ context.Context, facilityID uint, userIDs []uint) error {
	f.record("SeedDemoActivity", facilityID, userIDs)
	return f.err
}

// testToken is the site's secret in these tests.
const testToken = "s3cret"

// newPair serves fake with NewHandler and returns a Client that asks it.
func newPair(t *testing.T, fake *fakeOC) *opencontent.Client {
	t.Helper()
	srv := httptest.NewServer(opencontent.NewHandler(fake, testToken))
	t.Cleanup(srv.Close)
	return opencontent.NewClient(srv.URL+"/", testToken, nil)
}

// newRequest is a question to the handler, with the token.
func newRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	return req
}

// question is one of core's questions: how to ask it, what Open Content
// should be asked, and the answer core should get.
type question struct {
	name       string
	ask        func(context.Context, *opencontent.Client) (any, error)
	wantMethod string
	wantArgs   []any
	want       any
}

var questions = []question{
	{
		name: "TopLibraries",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.TopLibraries(ctx, 12, 3)
		},
		wantMethod: "TopLibraries", wantArgs: []any{uint(12), uint(3)}, want: usage,
	},
	{
		name: "RecentVideos",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.RecentVideos(ctx, 12)
		},
		wantMethod: "RecentVideos", wantArgs: []any{uint(12)}, want: usage,
	},
	{
		name: "ResourcesAccessed",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.ResourcesAccessed(ctx, 12)
		},
		wantMethod: "ResourcesAccessed", wantArgs: []any{uint(12)}, want: int64(17),
	},
	{
		name: "ActivityEngagement",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.ActivityEngagement(ctx, 12)
		},
		wantMethod: "ActivityEngagement", wantArgs: []any{uint(12)}, want: engagement,
	},
	{
		name: "FacilityContent",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.FacilityContent(ctx, []uint{3, 4})
		},
		wantMethod: "FacilityContent", wantArgs: []any{[]uint{3, 4}}, want: content,
	},
	{
		name: "Interactions",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.Interactions(ctx, window)
		},
		wantMethod: "Interactions", wantArgs: []any{window}, want: interactions,
	},
	{
		name: "Interactions over everything",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.Interactions(ctx, opencontent.Window{})
		},
		wantMethod: "Interactions", wantArgs: []any{opencontent.Window{}}, want: interactions,
	},
	{
		name: "AvgSessionMinutes",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.AvgSessionMinutes(ctx, window)
		},
		wantMethod: "AvgSessionMinutes", wantArgs: []any{window}, want: 12.5,
	},
	{
		name: "RepeatEngagement",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.RepeatEngagement(ctx, window)
		},
		wantMethod: "RepeatEngagement", wantArgs: []any{window}, want: repeat,
	},
	{
		name: "LibraryViewsByCategory",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.LibraryViewsByCategory(ctx, window)
		},
		wantMethod: "LibraryViewsByCategory", wantArgs: []any{window}, want: categories,
	},
	{
		name: "TopLibraryVisits",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.TopLibraryVisits(ctx, window, 5)
		},
		wantMethod: "TopLibraryVisits", wantArgs: []any{window, 5}, want: visits,
	},
	{
		name: "TopVideoVisits",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return c.TopVideoVisits(ctx, window, 5)
		},
		wantMethod: "TopVideoVisits", wantArgs: []any{window, 5}, want: visits,
	},
	{
		name: "ResetDemoActivity",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return nil, c.ResetDemoActivity(ctx, start)
		},
		wantMethod: "ResetDemoActivity", wantArgs: []any{start}, want: nil,
	},
	{
		name: "SeedDemoActivity",
		ask: func(ctx context.Context, c *opencontent.Client) (any, error) {
			return nil, c.SeedDemoActivity(ctx, 3, []uint{12, 13})
		},
		wantMethod: "SeedDemoActivity", wantArgs: []any{uint(3), []uint{12, 13}}, want: nil,
	},
}

// TestRoundTrip asks every question through Client and Handler and checks
// that Open Content gets the same arguments and core the same answer as a
// direct call.
func TestRoundTrip(t *testing.T) {
	for _, q := range questions {
		t.Run(q.name, func(t *testing.T) {
			fake := &fakeOC{usage: usage}
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
				t.Errorf("Open Content was asked %s%v, want %s%v", method, args, q.wantMethod, q.wantArgs)
			}
		})
	}
}

func TestEndActivity(t *testing.T) {
	fake := &fakeOC{}
	newPair(t, fake).EndActivity(context.Background(), 99)
	method, args := fake.asked()
	if method != "EndActivity" || !reflect.DeepEqual(args, []any{int64(99)}) {
		t.Errorf("Open Content was asked %s%v, want EndActivity[99]", method, args)
	}
}

// TestEmptyAnswers checks that no rows stay no rows, and nil stays nil, so
// core's JSON ([] or null) is what it was before the move.
func TestEmptyAnswers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		usage []opencontent.ContentUsage
	}{
		{"empty", []opencontent.ContentUsage{}},
		{"nil", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newPair(t, &fakeOC{usage: tc.usage}).RecentVideos(context.Background(), 12)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if !reflect.DeepEqual(got, tc.usage) {
				t.Errorf("answer = %#v, want %#v", got, tc.usage)
			}
		})
	}
}

// TestOpenContentFails checks that every question returns Open Content's
// error, and that EndActivity logs it.
func TestOpenContentFails(t *testing.T) {
	fake := &fakeOC{err: errors.New("database is down")}
	c := newPair(t, fake)
	for _, q := range questions {
		t.Run(q.name, func(t *testing.T) {
			_, err := q.ask(context.Background(), c)
			if err == nil || !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "database is down") {
				t.Errorf("err = %v, want the 500 and Open Content's message", err)
			}
		})
	}
}

func TestEndActivityLogsFailure(t *testing.T) {
	var logs bytes.Buffer
	defer slog.SetDefault(slog.Default())
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	opencontent.NewClient(srv.URL, testToken, nil).EndActivity(context.Background(), 99)

	if line := logs.String(); !strings.Contains(line, "activity_id=99") || !strings.Contains(line, "502") {
		t.Errorf("log = %q, want the activity id and the 502", line)
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
		{"200 with the wrong type", http.StatusOK, `{"message":"","data":"seventeen"}`, "read answer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			_, err := opencontent.NewClient(srv.URL, testToken, nil).ResourcesAccessed(context.Background(), 12)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestOpenContentUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	if _, err := opencontent.NewClient(srv.URL, testToken, nil).ResourcesAccessed(context.Background(), 12); err == nil {
		t.Error("err = nil, want an error when Open Content can't be reached")
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
	_, err := opencontent.NewClient(srv.URL, testToken, nil).ResourcesAccessed(ctx, 12)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

// TestWire pins the paths and the JSON on the wire, so a core and an Open
// Content built with different versions of this package still agree. Change
// one only with a plan for sites that run both versions.
func TestWire(t *testing.T) {
	for _, tc := range []struct {
		path       string
		body       string
		wantMethod string
		wantArgs   []any
		wantAnswer string
	}{
		{
			"end-activity", `{"activity_id":99}`,
			"EndActivity", []any{int64(99)}, `{"message":"","data":null}`,
		},
		{
			"top-libraries", `{"user_id":12,"facility_id":3}`,
			"TopLibraries", []any{uint(12), uint(3)},
			`{"message":"","data":[{"title":"Gutenberg","url":"/api/proxy/libraries/7/","thumbnail_url":"/api/photos/gutenberg.png",` +
				`"visibility_status":true,"open_content_provider_id":1,"content_id":7,"content_type":"library","provider_name":"Kiwix",` +
				`"is_featured":true,"total_hours":1.5,"total_minutes":90}]}`,
		},
		{"recent-videos", `{"user_id":12}`, "RecentVideos", []any{uint(12)}, ""},
		{"resources-accessed", `{"user_id":12}`, "ResourcesAccessed", []any{uint(12)}, `{"message":"","data":17}`},
		{
			"activity-engagement", `{"user_id":12}`,
			"ActivityEngagement", []any{uint(12)},
			`{"message":"","data":{"total_active_days_monthly":4,"total_hours_active_monthly":0.5,"total_hours_active_weekly":2,` +
				`"total_minutes_active_weekly":120,"total_hours_engaged":1,"total_minutes_engaged":60}}`,
		},
		{
			"facility-content", `{"facility_ids":[3,4]}`,
			"FacilityContent", []any{[]uint{3, 4}},
			`{"message":"","data":{"3":{"activities":10,"favorites":2,"visibility_statuses":5}}}`,
		},
		{
			"interactions", `{"window":{"start":"2026-09-01T00:00:00Z","end":"2026-10-01T00:00:00Z","facility_id":3}}`,
			"Interactions", []any{window}, `{"message":"","data":{"total":40,"unique":9,"change":-25}}`,
		},
		{"interactions", `{"window":{}}`, "Interactions", []any{opencontent.Window{}}, ""},
		{"avg-session-minutes", `{"window":{}}`, "AvgSessionMinutes", []any{opencontent.Window{}}, `{"message":"","data":12.5}`},
		{
			"repeat-engagement", `{"window":{}}`,
			"RepeatEngagement", []any{opencontent.Window{}}, `{"message":"","data":{"once":3,"two_to_four":4,"five_plus":2}}`,
		},
		{
			"library-views-by-category", `{"window":{}}`,
			"LibraryViewsByCategory", []any{opencontent.Window{}}, `{"message":"","data":[{"category":"Science","views":12}]}`,
		},
		{
			"top-library-visits", `{"window":{},"limit":5}`,
			"TopLibraryVisits", []any{opencontent.Window{}, 5}, `{"message":"","data":[{"title":"Gutenberg","visits":8,"change":100}]}`,
		},
		{"top-video-visits", `{"window":{},"limit":5}`, "TopVideoVisits", []any{opencontent.Window{}, 5}, ""},
		{"reset-demo-activity", `{"before":"2026-09-01T00:00:00Z"}`, "ResetDemoActivity", []any{start}, `{"message":"","data":null}`},
		{"seed-demo-activity", `{"facility_id":3,"user_ids":[12,13]}`, "SeedDemoActivity", []any{uint(3), []uint{12, 13}}, ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			fake := &fakeOC{usage: usage}
			rec := httptest.NewRecorder()
			req := newRequest(http.MethodPost, "/internal/v1/open-content/"+tc.path, tc.body)
			opencontent.NewHandler(fake, testToken).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
			}
			method, args := fake.asked()
			if method != tc.wantMethod || !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("Open Content was asked %s%v, want %s%v", method, args, tc.wantMethod, tc.wantArgs)
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
		{"arguments that aren't JSON", http.MethodPost, "/internal/v1/open-content/top-libraries", "user 12", http.StatusBadRequest},
		{"arguments of the wrong type", http.MethodPost, "/internal/v1/open-content/top-libraries", `{"user_id":"12"}`, http.StatusBadRequest},
		{"too much", http.MethodPost, "/internal/v1/open-content/seed-demo-activity", `{"user_ids":[` + strings.Repeat("1,", 1<<20) + `1]}`, http.StatusBadRequest},
		{"a GET", http.MethodGet, "/internal/v1/open-content/top-libraries", "", http.StatusMethodNotAllowed},
		{"an unknown question", http.MethodPost, "/internal/v1/open-content/top-books", "{}", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeOC{}
			rec := httptest.NewRecorder()
			opencontent.NewHandler(fake, testToken).ServeHTTP(rec, newRequest(tc.method, tc.path, tc.body))
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if method, _ := fake.asked(); method != "" {
				t.Errorf("Open Content was asked %s, want nothing", method)
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

// Only core, with the site's secret, gets answers: a missing or wrong token
// gets a 401 and Open Content isn't asked, and a handler with no token
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
			fake := &fakeOC{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/internal/v1/open-content/reset-demo-activity", strings.NewReader(`{"before":"2026-09-01T00:00:00Z"}`))
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			opencontent.NewHandler(fake, tc.handlerToken).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tc.wantStatus, rec.Body)
			}
			method, _ := fake.asked()
			if tc.wantStatus == http.StatusOK && method != "ResetDemoActivity" {
				t.Errorf("Open Content was asked %q, want ResetDemoActivity", method)
			}
			if tc.wantStatus != http.StatusOK {
				if method != "" {
					t.Errorf("Open Content was asked %s, want nothing", method)
				}
				if got := strings.TrimSpace(rec.Body.String()); got != `{"message":"missing or wrong token","data":null}` {
					t.Errorf("body = %s, want core's JSON error", got)
				}
			}
		})
	}
}

// A Client with the wrong token gets Open Content's 401 as an error.
func TestClientWrongToken(t *testing.T) {
	srv := httptest.NewServer(opencontent.NewHandler(&fakeOC{}, testToken))
	t.Cleanup(srv.Close)
	_, err := opencontent.NewClient(srv.URL, "guess", nil).ResourcesAccessed(context.Background(), 12)
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "missing or wrong token") {
		t.Errorf("err = %v, want the 401 with its message", err)
	}
}

// TestNotification pins the subject and the JSON, which core and Open Content
// built with different versions of this package must agree on.
func TestNotification(t *testing.T) {
	if opencontent.NotifySubject != "opencontent.notify" {
		t.Errorf("NotifySubject = %q, want opencontent.notify", opencontent.NotifySubject)
	}
	for _, tc := range []struct {
		n    opencontent.Notification
		want string
	}{
		{opencontent.Notification{UserID: 12, Event: "visits", ActivityID: 99}, `{"user_id":12,"event":"visits","activity_id":99}`},
		{opencontent.Notification{UserID: 12, Event: "bookmarks", Text: "Saved"}, `{"user_id":12,"event":"bookmarks","text":"Saved"}`},
	} {
		got, err := json.Marshal(tc.n)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tc.want {
			t.Errorf("JSON = %s, want %s", got, tc.want)
		}
		var back opencontent.Notification
		if err := json.Unmarshal(got, &back); err != nil || back != tc.n {
			t.Errorf("read back %+v (err %v), want %+v", back, err, tc.n)
		}
	}
}
