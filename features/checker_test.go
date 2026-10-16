package features

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const facility uint = 7

// fakeCore answers like core's /internal/v1/features. Its reply is set per
// call; it counts how many times it was asked. If hold is set, it doesn't
// answer until hold is closed.
type fakeCore struct {
	mu    sync.Mutex
	reply string
	calls atomic.Int32
	hold  chan struct{}
}

// Replies a fake core can give.
const (
	openContentOn = "open content on"
	allOff        = "all off"
	serverError   = "500"
	notFound      = "400 facilities not found"
	badJSON       = "bad json"
	otherFacility = "other facility"
)

func (f *fakeCore) set(reply string) {
	f.mu.Lock()
	f.reply = reply
	f.mu.Unlock()
}

func (f *fakeCore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	if f.hold != nil {
		<-f.hold
	}
	f.mu.Lock()
	reply := f.reply
	f.mu.Unlock()
	if r.URL.Path != checkPath || r.URL.Query().Get("facility_id") != fmt.Sprint(facility) {
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusTeapot)
		return
	}
	var body string
	switch reply {
	case openContentOn:
		body = fmt.Sprintf(`{"message":"Data fetched successfully","data":{"facility_id":%d,"features":["open_content","helpful_links"]}}`, facility)
	case allOff:
		body = fmt.Sprintf(`{"message":"Data fetched successfully","data":{"facility_id":%d,"features":[]}}`, facility)
	case serverError:
		http.Error(w, `{"message":"error getting facility_feature_flags","data":null}`, http.StatusInternalServerError)
		return
	case notFound:
		http.Error(w, `{"message":"facilities not found","data":null}`, http.StatusBadRequest)
		return
	case badJSON:
		body = `<html>not json</html>`
	case otherFacility:
		body = `{"message":"Data fetched successfully","data":{"facility_id":8,"features":["open_content"]}}`
	}
	_, _ = w.Write([]byte(body))
}

func TestCheckerEnabled(t *testing.T) {
	// Each step moves the clock, sets core's reply, then asks about a feature.
	type step struct {
		after     time.Duration
		core      string
		feature   Access
		want      bool
		wantErr   bool
		wantCalls int32 // total calls to core after this step
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "core's answer decides on and off",
			steps: []step{
				{core: openContentOn, feature: OpenContent, want: true, wantCalls: 1},
				{core: openContentOn, feature: HelpfulLinks, want: true, wantCalls: 1},
				{core: openContentOn, feature: LearningRecord, want: false, wantCalls: 1},
			},
		},
		{
			name: "an answer is reused for 30 seconds",
			steps: []step{
				{core: openContentOn, feature: OpenContent, want: true, wantCalls: 1},
				{after: 29 * time.Second, core: allOff, feature: OpenContent, want: true, wantCalls: 1},
			},
		},
		{
			name: "after 30 seconds core is asked again",
			steps: []step{
				{core: openContentOn, feature: OpenContent, want: true, wantCalls: 1},
				{after: CacheTTL, core: allOff, feature: OpenContent, want: false, wantCalls: 2},
			},
		},
		{
			name: "core down with an answer keeps the last answer",
			steps: []step{
				{core: openContentOn, feature: OpenContent, want: true, wantCalls: 1},
				{after: CacheTTL, core: serverError, feature: OpenContent, want: true, wantErr: true, wantCalls: 2},
				{after: 10 * time.Minute, core: serverError, feature: OpenContent, want: true, wantErr: true, wantCalls: 3},
			},
		},
		{
			name: "core down with no answer reads off",
			steps: []step{
				{core: serverError, feature: OpenContent, want: false, wantErr: true, wantCalls: 1},
			},
		},
		{
			name: "after a failure core isn't asked again for 30 seconds",
			steps: []step{
				{core: serverError, feature: OpenContent, want: false, wantErr: true, wantCalls: 1},
				{after: 10 * time.Second, core: openContentOn, feature: OpenContent, want: false, wantCalls: 1},
				{after: 20 * time.Second, core: openContentOn, feature: OpenContent, want: true, wantCalls: 2},
			},
		},
		{
			name: "an unknown facility reads off",
			steps: []step{
				{core: notFound, feature: OpenContent, want: false, wantErr: true, wantCalls: 1},
			},
		},
		{
			name: "a reply that isn't JSON counts as core down",
			steps: []step{
				{core: openContentOn, feature: OpenContent, want: true, wantCalls: 1},
				{after: CacheTTL, core: badJSON, feature: OpenContent, want: true, wantErr: true, wantCalls: 2},
			},
		},
		{
			name: "an answer for another facility is rejected",
			steps: []step{
				{core: otherFacility, feature: OpenContent, want: false, wantErr: true, wantCalls: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core := &fakeCore{}
			srv := httptest.NewServer(core)
			defer srv.Close()

			now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
			c := NewChecker(srv.URL+"/", nil)
			c.now = func() time.Time { return now }

			for i, s := range tt.steps {
				now = now.Add(s.after)
				core.set(s.core)
				got, err := c.Enabled(context.Background(), facility, s.feature)
				if got != s.want {
					t.Errorf("step %d: Enabled(%s) = %v, want %v", i, s.feature, got, s.want)
				}
				if (err != nil) != s.wantErr {
					t.Errorf("step %d: err = %v, wantErr %v", i, err, s.wantErr)
				}
				if calls := core.calls.Load(); calls != s.wantCalls {
					t.Errorf("step %d: core was called %d times, want %d", i, calls, s.wantCalls)
				}
			}
		})
	}
}

// Core not running at all: the call fails and the feature reads off.
func TestCheckerCoreUnreachable(t *testing.T) {
	srv := httptest.NewServer(&fakeCore{})
	url := srv.URL
	srv.Close()

	got, err := NewChecker(url, nil).Enabled(context.Background(), facility, OpenContent)
	if got || err == nil {
		t.Fatalf("Enabled = %v, %v; want false and an error", got, err)
	}
}

// A caller that gives up only stops waiting itself: the call to core goes on,
// and everyone else waiting on it gets core's answer.
func TestCheckerCanceledCallerDoesNotAffectOthers(t *testing.T) {
	core := &fakeCore{hold: make(chan struct{})}
	core.set(openContentOn)
	srv := httptest.NewServer(core)
	defer srv.Close()
	c := NewChecker(srv.URL, nil)

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error)
	go func() {
		got, err := c.Enabled(ctx, facility, OpenContent)
		if got {
			t.Error("canceled caller: Enabled = true, want false")
		}
		first <- err
	}()
	for core.calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}

	second := make(chan bool)
	go func() {
		got, err := c.Enabled(context.Background(), facility, OpenContent)
		if err != nil {
			t.Errorf("waiting caller: err = %v, want nil", err)
		}
		second <- got
	}()

	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Errorf("canceled caller: err = %v, want context.Canceled", err)
	}
	close(core.hold)
	if !<-second {
		t.Error("waiting caller: Enabled = false, want true")
	}
	if calls := core.calls.Load(); calls != 1 {
		t.Errorf("core was called %d times, want 1", calls)
	}
}

// Many requests at once share one call to core (run with -race). Because only
// one call per facility runs at a time, a slow failure can't overwrite a newer
// answer.
func TestCheckerConcurrent(t *testing.T) {
	core := &fakeCore{hold: make(chan struct{})}
	core.set(openContentOn)
	srv := httptest.NewServer(core)
	defer srv.Close()
	c := NewChecker(srv.URL, nil)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if got, err := c.Enabled(context.Background(), facility, OpenContent); !got || err != nil {
				t.Errorf("Enabled = %v, %v; want true and no error", got, err)
			}
		})
	}
	for core.calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(core.hold)
	wg.Wait()
	if calls := core.calls.Load(); calls != 1 {
		t.Errorf("core was called %d times, want 1", calls)
	}
}
