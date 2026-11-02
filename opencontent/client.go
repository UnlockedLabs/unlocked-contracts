package opencontent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// callTimeout bounds each question, whatever client the Client was given.
// The Knowledge Center numbers over every facility are the slowest.
const callTimeout = 10 * time.Second

// demoTimeout bounds the demo seed and reset, which write months of activity
// one row at a time.
const demoTimeout = 5 * time.Minute

// Client asks Open Content's service core's questions over HTTP. It is safe
// for concurrent use.
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

var _ OpenContent = (*Client)(nil)

// NewClient returns a Client that asks Open Content at baseURL, such as
// "http://open-content:8080", sending token, the site's secret that Open
// Content's handler also has. A nil client means http.DefaultClient; each
// question times out after 10 seconds either way, the demo seed and reset
// after 5 minutes.
func NewClient(baseURL, token string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), token: token, client: client}
}

func (c *Client) EndActivity(ctx context.Context, activityID int64) {
	if _, err := ask[any](ctx, c, pathEndActivity, callTimeout, activityRequest{ActivityID: activityID}); err != nil {
		slog.Error("opencontent: end activity", "activity_id", activityID, "err", err)
	}
}

func (c *Client) TopLibraries(ctx context.Context, userID, facilityID uint) ([]ContentUsage, error) {
	return ask[[]ContentUsage](ctx, c, pathTopLibraries, callTimeout, userRequest{UserID: userID, FacilityID: facilityID})
}

func (c *Client) RecentVideos(ctx context.Context, userID uint) ([]ContentUsage, error) {
	return ask[[]ContentUsage](ctx, c, pathRecentVideos, callTimeout, userRequest{UserID: userID})
}

func (c *Client) ResourcesAccessed(ctx context.Context, userID uint) (int64, error) {
	return ask[int64](ctx, c, pathResourcesAccessed, callTimeout, userRequest{UserID: userID})
}

func (c *Client) ActivityEngagement(ctx context.Context, userID uint) (ActivityEngagement, error) {
	return ask[ActivityEngagement](ctx, c, pathActivityEngagement, callTimeout, userRequest{UserID: userID})
}

func (c *Client) FacilityContent(ctx context.Context, facilityIDs []uint) (map[uint]FacilityContent, error) {
	return ask[map[uint]FacilityContent](ctx, c, pathFacilityContent, callTimeout, facilitiesRequest{FacilityIDs: facilityIDs})
}

func (c *Client) Interactions(ctx context.Context, w Window) (Interactions, error) {
	return ask[Interactions](ctx, c, pathInteractions, callTimeout, windowRequest{Window: w})
}

func (c *Client) AvgSessionMinutes(ctx context.Context, w Window) (float64, error) {
	return ask[float64](ctx, c, pathAvgSessionMinutes, callTimeout, windowRequest{Window: w})
}

func (c *Client) RepeatEngagement(ctx context.Context, w Window) (RepeatEngagement, error) {
	return ask[RepeatEngagement](ctx, c, pathRepeatEngagement, callTimeout, windowRequest{Window: w})
}

func (c *Client) LibraryViewsByCategory(ctx context.Context, w Window) ([]CategoryViews, error) {
	return ask[[]CategoryViews](ctx, c, pathLibraryViewsByCategory, callTimeout, windowRequest{Window: w})
}

func (c *Client) TopLibraryVisits(ctx context.Context, w Window, limit int) ([]ContentVisits, error) {
	return ask[[]ContentVisits](ctx, c, pathTopLibraryVisits, callTimeout, windowRequest{Window: w, Limit: limit})
}

func (c *Client) TopVideoVisits(ctx context.Context, w Window, limit int) ([]ContentVisits, error) {
	return ask[[]ContentVisits](ctx, c, pathTopVideoVisits, callTimeout, windowRequest{Window: w, Limit: limit})
}

func (c *Client) ResetDemoActivity(ctx context.Context, before time.Time) error {
	_, err := ask[any](ctx, c, pathResetDemoActivity, demoTimeout, resetDemoRequest{Before: before})
	return err
}

func (c *Client) SeedDemoActivity(ctx context.Context, facilityID uint, userIDs []uint) error {
	_, err := ask[any](ctx, c, pathSeedDemoActivity, demoTimeout, seedDemoRequest{FacilityID: facilityID, UserIDs: userIDs})
	return err
}

// ask posts args to path and returns the answer. Any status but 200 is an
// error, with Open Content's message when it sent one.
func ask[Resp any](ctx context.Context, c *Client, path string, timeout time.Duration, args any) (Resp, error) {
	var zero Resp
	body, err := json.Marshal(args)
	if err != nil {
		return zero, fmt.Errorf("opencontent: write arguments for %s: %w", path, err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("opencontent: build request for %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("opencontent: ask %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var answer envelope[Resp]
	decodeErr := json.NewDecoder(resp.Body).Decode(&answer)
	switch {
	case resp.StatusCode != http.StatusOK && decodeErr == nil && answer.Message != "":
		return zero, fmt.Errorf("opencontent: %s answered %s: %s", path, resp.Status, answer.Message)
	case resp.StatusCode != http.StatusOK:
		return zero, fmt.Errorf("opencontent: %s answered %s", path, resp.Status)
	case decodeErr != nil:
		return zero, fmt.Errorf("opencontent: read answer from %s: %w", path, decodeErr)
	}
	return answer.Data, nil
}
