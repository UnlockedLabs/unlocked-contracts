package features

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CacheTTL is how long a Checker uses an answer before asking core again, so a
// change in Feature Control reaches every module within this time.
const CacheTTL = 30 * time.Second

// checkPath is core's feature-check endpoint (EN-188). Only reachable from
// inside the cluster.
const checkPath = "/internal/v1/features"

// Checker answers "is this feature on at this facility?" by asking core, which
// owns the rules (statewide defaults, facility overrides, sub-features off when
// their parent is off). It caches each facility's answer for CacheTTL. It is
// safe for concurrent use.
//
// It fails safe: when core can't answer, it keeps using the last answer it
// got, however old, and with no answer yet every feature reads as off. A core
// outage never turns on a feature that should be off. After a failed call it
// waits CacheTTL before asking again, so an outage doesn't slow every request.
type Checker struct {
	baseURL string
	client  *http.Client
	now     func() time.Time

	mu    sync.Mutex
	cache map[uint]answer
}

type answer struct {
	on      map[Access]bool
	fetched time.Time
}

// NewChecker returns a Checker that asks core at baseURL, such as
// "http://server:8080". A nil client gets a 2-second timeout.
func NewChecker(baseURL string, client *http.Client) *Checker {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	return &Checker{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  client,
		now:     time.Now,
		cache:   map[uint]answer{},
	}
}

// Enabled reports whether f is on at the facility. The bool is always safe to
// act on. A non-nil error means core couldn't answer just now and the bool
// came from the last answer (or is false if there was none): log it, but
// don't fail the request because of it.
func (c *Checker) Enabled(ctx context.Context, facilityID uint, f Access) (bool, error) {
	c.mu.Lock()
	cached, ok := c.cache[facilityID]
	c.mu.Unlock()
	if ok && c.now().Sub(cached.fetched) < CacheTTL {
		return cached.on[f], nil
	}

	on, err := c.fetch(ctx, facilityID)
	if err != nil {
		// A canceled caller says nothing about core, so only a real failure
		// starts the wait before the next try.
		if ctx.Err() == nil {
			c.store(facilityID, cached.on)
		}
		return cached.on[f], err
	}
	c.store(facilityID, on)
	return on[f], nil
}

func (c *Checker) store(facilityID uint, on map[Access]bool) {
	c.mu.Lock()
	c.cache[facilityID] = answer{on: on, fetched: c.now()}
	c.mu.Unlock()
}

func (c *Checker) fetch(ctx context.Context, facilityID uint) (map[Access]bool, error) {
	url := fmt.Sprintf("%s%s?facility_id=%d", c.baseURL, checkPath, facilityID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("features: build request for facility %d: %w", facilityID, err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("features: ask core about facility %d: %w", facilityID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("features: core answered %s for facility %d", resp.Status, facilityID)
	}

	// Core wraps every response in {"message", "data"}.
	var body struct {
		Data struct {
			FacilityID uint     `json:"facility_id"`
			Features   []Access `json:"features"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("features: read core's answer for facility %d: %w", facilityID, err)
	}
	if body.Data.FacilityID != facilityID {
		return nil, fmt.Errorf("features: asked about facility %d, core answered for %d", facilityID, body.Data.FacilityID)
	}
	on := make(map[Access]bool, len(body.Data.Features))
	for _, f := range body.Data.Features {
		on[f] = true
	}
	return on, nil
}
