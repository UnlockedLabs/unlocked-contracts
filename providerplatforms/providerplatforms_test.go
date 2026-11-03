package providerplatforms_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/coreapi"
	"github.com/UnlockedLabs/unlocked-contracts/providerplatforms"
)

const token = "s3cret"

var (
	day      = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	facility = uint(3)
	user     = coreapi.User{ID: 7, Username: "jdoe", NameFirst: "Jane", NameLast: "Doe", Email: "j@x.org", FacilityID: 3}
	ext      = []string{"e1", "e2"}
	provider = providerplatforms.Provider{
		ID: 2, Type: providerplatforms.Kolibri, Name: "Kolibri", BaseURL: "http://k", AccountID: "a", AccessKey: "key",
	}
	mappings = []providerplatforms.Mapping{{ProviderID: 2, ExternalUserID: "e1"}}
	mapped   = []providerplatforms.MappedUser{{ExternalUserID: "e1", UserID: 7, NameFirst: "Jane", NameLast: "Doe", DocID: "d1"}}
	stats    = providerplatforms.LearningStats{
		TotalCoursesOffered: 5, TotalStudentsEnrolled: 6, TotalHourlyActivity: 7,
		Insights: []providerplatforms.LearningInsight{{
			CourseName: "Math", TotalStudentsCompleted: 2, TotalStudentsEnrolled: 4, CompletionRate: 0.5, ActivityHours: 1.5,
		}},
	}
	perFacility = map[uint]int64{3: 4, 9: 1}
)

type fake struct {
	mu     sync.Mutex
	method string
	args   []any
	err    error
}

func (f *fake) record(method string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.method, f.args = method, args
}

func (f *fake) asked() (string, []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.method, f.args
}

func (f *fake) Provider(_ context.Context, id uint) (providerplatforms.Provider, error) {
	f.record("Provider", id)
	return provider, f.err
}

func (f *fake) ActiveProviders(context.Context) ([]providerplatforms.Provider, error) {
	f.record("ActiveProviders")
	return []providerplatforms.Provider{provider}, f.err
}

func (f *fake) UserMappings(_ context.Context, userID uint) ([]providerplatforms.Mapping, error) {
	f.record("UserMappings", userID)
	return mappings, f.err
}

func (f *fake) MappedUserIDs(_ context.Context, providerID uint) ([]uint, error) {
	f.record("MappedUserIDs", providerID)
	return []uint{7, 8}, f.err
}

func (f *fake) DeleteUserMappings(_ context.Context, userID uint) error {
	f.record("DeleteUserMappings", userID)
	return f.err
}

func (f *fake) MappedUsers(_ context.Context, providerID uint, ids []string, facilityID uint) ([]providerplatforms.MappedUser, error) {
	f.record("MappedUsers", providerID, ids, facilityID)
	return mapped, f.err
}

func (f *fake) CountMapped(_ context.Context, providerID uint, ids []string) (int64, error) {
	f.record("CountMapped", providerID, ids)
	return 11, f.err
}

func (f *fake) CountMappedAtFacility(_ context.Context, providerID uint, ids []string, facilityID uint) (int64, error) {
	f.record("CountMappedAtFacility", providerID, ids, facilityID)
	return 12, f.err
}

func (f *fake) CountMappedPerFacility(_ context.Context, providerID uint, ids []string) (map[uint]int64, error) {
	f.record("CountMappedPerFacility", providerID, ids)
	return perFacility, f.err
}

func (f *fake) AddUserToProviders(_ context.Context, u coreapi.User, providerIDs []uint) error {
	f.record("AddUserToProviders", u, providerIDs)
	return f.err
}

func (f *fake) AddUserToKolibri(_ context.Context, u coreapi.User) error {
	f.record("AddUserToKolibri", u)
	return f.err
}

func (f *fake) SetupKolibri(context.Context) error {
	f.record("SetupKolibri")
	return f.err
}

func (f *fake) LearningStats(_ context.Context, facilityID *uint) (providerplatforms.LearningStats, error) {
	f.record("LearningStats", facilityID)
	return stats, f.err
}

func (f *fake) LastDemoSeed(context.Context) (time.Time, bool, error) {
	f.record("LastDemoSeed")
	return day, true, f.err
}

func (f *fake) ResetDemoLearning(_ context.Context, through time.Time) error {
	f.record("ResetDemoLearning", through)
	return f.err
}

func (f *fake) SeedDemoLearning(_ context.Context, userIDs []uint) error {
	f.record("SeedDemoLearning", userIDs)
	return f.err
}

func start(t *testing.T) (*providerplatforms.Client, *fake, string) {
	t.Helper()
	f := &fake{}
	srv := httptest.NewServer(providerplatforms.NewHandler(f, token))
	t.Cleanup(srv.Close)
	return providerplatforms.NewClient(srv.URL, token, nil), f, srv.URL
}

type call struct {
	name   string
	do     func(context.Context, providerplatforms.ProviderPlatforms) (any, error)
	want   any
	args   []any
	path   string
	body   string
	answer string
}

func calls() []call {
	none := func(err error) (any, error) { return nil, err }
	return []call{
		{"Provider", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.Provider(ctx, 2)
		},
			provider, []any{uint(2)}, "provider", `{"id":2}`,
			`{"ID":2,"Type":"kolibri","Name":"Kolibri","BaseURL":"http://k","AccountID":"a","AccessKey":"key"}`},
		{"ActiveProviders", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.ActiveProviders(ctx)
		},
			[]providerplatforms.Provider{provider}, nil, "active-providers", `{}`,
			`[{"ID":2,"Type":"kolibri","Name":"Kolibri","BaseURL":"http://k","AccountID":"a","AccessKey":"key"}]`},
		{"UserMappings", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.UserMappings(ctx, 7)
		},
			mappings, []any{uint(7)}, "user-mappings", `{"user_id":7}`,
			`[{"ProviderID":2,"ExternalUserID":"e1"}]`},
		{"MappedUserIDs", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.MappedUserIDs(ctx, 2)
		},
			[]uint{7, 8}, []any{uint(2)}, "mapped-user-ids", `{"provider_id":2}`, `[7,8]`},
		{"DeleteUserMappings", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return none(p.DeleteUserMappings(ctx, 7))
		}, nil, []any{uint(7)}, "delete-user-mappings", `{"user_id":7}`, `null`},
		{"MappedUsers", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.MappedUsers(ctx, 2, ext, 3)
		}, mapped, []any{uint(2), ext, uint(3)}, "mapped-users",
			`{"provider_id":2,"external_user_ids":["e1","e2"],"facility_id":3}`,
			`[{"ExternalUserID":"e1","UserID":7,"NameFirst":"Jane","NameLast":"Doe","DocID":"d1"}]`},
		{"CountMapped", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.CountMapped(ctx, 2, ext)
		}, int64(11), []any{uint(2), ext}, "count-mapped",
			`{"provider_id":2,"external_user_ids":["e1","e2"]}`, `11`},
		{"CountMappedAtFacility", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.CountMappedAtFacility(ctx, 2, ext, 3)
		}, int64(12), []any{uint(2), ext, uint(3)}, "count-mapped-at-facility",
			`{"provider_id":2,"external_user_ids":["e1","e2"],"facility_id":3}`, `12`},
		{"CountMappedPerFacility", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.CountMappedPerFacility(ctx, 2, ext)
		}, perFacility, []any{uint(2), ext}, "count-mapped-per-facility",
			`{"provider_id":2,"external_user_ids":["e1","e2"]}`, `{"3":4,"9":1}`},
		{"AddUserToProviders", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return none(p.AddUserToProviders(ctx, user, []uint{1, 2}))
		}, nil, []any{user, []uint{1, 2}}, "add-user-to-providers",
			`{"user":{"id":7,"username":"jdoe","name_first":"Jane","name_last":"Doe","email":"j@x.org","facility_id":3},"provider_ids":[1,2]}`, `null`},
		{"AddUserToKolibri", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return none(p.AddUserToKolibri(ctx, user))
		}, nil, []any{user}, "add-user-to-kolibri",
			`{"user":{"id":7,"username":"jdoe","name_first":"Jane","name_last":"Doe","email":"j@x.org","facility_id":3}}`, `null`},
		{"SetupKolibri", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return none(p.SetupKolibri(ctx))
		}, nil, nil, "setup-kolibri", `{}`, `null`},
		{"LearningStats", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return p.LearningStats(ctx, &facility)
		}, stats, []any{&facility}, "learning-stats", `{"facility_id":3}`,
			`{"TotalCoursesOffered":5,"TotalStudentsEnrolled":6,"TotalHourlyActivity":7,"Insights":[{"course_name":"Math","total_students_completed":2,"total_students_enrolled":4,"completion_rate":0.5,"activity_hours":1.5}]}`},
		{"LastDemoSeed", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			last, found, err := p.LastDemoSeed(ctx)
			return []any{last, found}, err
		}, []any{day, true}, nil, "last-demo-seed", `{}`, `{"last":"2026-09-01T00:00:00Z","found":true}`},
		{"ResetDemoLearning", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return none(p.ResetDemoLearning(ctx, day))
		}, nil, []any{day}, "reset-demo-learning", `{"through":"2026-09-01T00:00:00Z"}`, `null`},
		{"SeedDemoLearning", func(ctx context.Context, p providerplatforms.ProviderPlatforms) (any, error) {
			return none(p.SeedDemoLearning(ctx, []uint{7, 8}))
		}, nil, []any{[]uint{7, 8}}, "seed-demo-learning", `{"user_ids":[7,8]}`, `null`},
	}
}

func TestRoundTrip(t *testing.T) {
	for _, tt := range calls() {
		t.Run(tt.name, func(t *testing.T) {
			c, f, _ := start(t)
			got, err := tt.do(context.Background(), c)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("answer = %#v, want %#v", got, tt.want)
			}
			method, args := f.asked()
			if method != tt.name || !reflect.DeepEqual(args, tt.args) {
				t.Errorf("asked %s%v, want %s%v", method, args, tt.name, tt.args)
			}
		})
	}
}

func TestErrorsReachCaller(t *testing.T) {
	for _, tt := range calls() {
		t.Run(tt.name, func(t *testing.T) {
			c, f, _ := start(t)
			f.err = errors.New("boom")
			if _, err := tt.do(context.Background(), c); err == nil || !strings.Contains(err.Error(), "boom") {
				t.Errorf("error = %v, want boom", err)
			}
		})
	}
}

func TestRejectsBadToken(t *testing.T) {
	_, f, url := start(t)
	wrong := providerplatforms.NewClient(url, "wrong", nil)
	for _, tt := range calls() {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.do(context.Background(), wrong); err == nil || !strings.Contains(err.Error(), "401") {
				t.Errorf("wrong token: error = %v, want 401", err)
			}
			resp, err := http.Post(url+"/internal/v1/provider-platforms/"+tt.path, "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("no token: status = %d, want 401", resp.StatusCode)
			}
		})
	}
	if method, _ := f.asked(); method != "" {
		t.Errorf("a rejected request reached %s", method)
	}
}

func TestBadArguments(t *testing.T) {
	_, _, url := start(t)
	req, _ := http.NewRequest(http.MethodPost, url+"/internal/v1/provider-platforms/provider", strings.NewReader(`{`))
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestWire pins the paths and JSON that running services depend on.
func TestWire(t *testing.T) {
	var gotPath, gotToken, gotBody string
	rec := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotPath, gotToken, gotBody = r.URL.Path, r.Header.Get("Authorization"), string(b)
		_, _ = io.WriteString(w, `{"message":"","data":null}`)
	}))
	defer rec.Close()
	recorder := providerplatforms.NewClient(rec.URL, token, nil)
	_, _, url := start(t)

	for _, tt := range calls() {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = tt.do(context.Background(), recorder)
			wantPath := "/internal/v1/provider-platforms/" + tt.path
			if gotPath != wantPath || strings.TrimSpace(gotBody) != tt.body || gotToken != "Bearer "+token {
				t.Errorf("sent %s %q token %q, want %s %q token %q", gotPath, gotBody, gotToken, wantPath, tt.body, "Bearer "+token)
			}

			req, _ := http.NewRequest(http.MethodPost, url+wantPath, strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			b, _ := io.ReadAll(resp.Body)
			want := `{"message":"","data":` + tt.answer + `}`
			if got := strings.TrimSpace(string(b)); got != want {
				t.Errorf("answer = %s, want %s", got, want)
			}
		})
	}
}
