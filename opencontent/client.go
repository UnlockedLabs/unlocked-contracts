package opencontent

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

// Client asks Open Content's service core's questions over HTTP. It is safe
// for concurrent use.
type Client struct {
	rpc *rpc.Client
}

var _ OpenContent = (*Client)(nil)

// NewClient returns a Client that asks Open Content at baseURL, such as
// "http://open-content:8080", sending token, the site's secret that Open
// Content's handler also has. A nil client means http.DefaultClient; each
// question times out after 10 seconds either way, the demo seed and reset
// after 5 minutes.
func NewClient(baseURL, token string, client *http.Client) *Client {
	return &Client{rpc: rpc.NewClient("opencontent", baseURL, token, client)}
}

func (c *Client) EndActivity(ctx context.Context, activityID int64) {
	if _, err := rpc.Ask[any](ctx, c.rpc, pathEndActivity, rpc.CallTimeout, activityRequest{ActivityID: activityID}); err != nil {
		slog.Error("opencontent: end activity", "activity_id", activityID, "err", err)
	}
}

func (c *Client) TopLibraries(ctx context.Context, userID, facilityID uint) ([]ContentUsage, error) {
	return rpc.Ask[[]ContentUsage](ctx, c.rpc, pathTopLibraries, rpc.CallTimeout, userRequest{UserID: userID, FacilityID: facilityID})
}

func (c *Client) RecentVideos(ctx context.Context, userID uint) ([]ContentUsage, error) {
	return rpc.Ask[[]ContentUsage](ctx, c.rpc, pathRecentVideos, rpc.CallTimeout, userRequest{UserID: userID})
}

func (c *Client) ResourcesAccessed(ctx context.Context, userID uint) (int64, error) {
	return rpc.Ask[int64](ctx, c.rpc, pathResourcesAccessed, rpc.CallTimeout, userRequest{UserID: userID})
}

func (c *Client) ActivityEngagement(ctx context.Context, userID uint) (ActivityEngagement, error) {
	return rpc.Ask[ActivityEngagement](ctx, c.rpc, pathActivityEngagement, rpc.CallTimeout, userRequest{UserID: userID})
}

func (c *Client) FacilityContent(ctx context.Context, facilityIDs []uint) (map[uint]FacilityContent, error) {
	return rpc.Ask[map[uint]FacilityContent](ctx, c.rpc, pathFacilityContent, rpc.CallTimeout, facilitiesRequest{FacilityIDs: facilityIDs})
}

func (c *Client) Interactions(ctx context.Context, w Window) (Interactions, error) {
	return rpc.Ask[Interactions](ctx, c.rpc, pathInteractions, rpc.CallTimeout, windowRequest{Window: w})
}

func (c *Client) AvgSessionMinutes(ctx context.Context, w Window) (float64, error) {
	return rpc.Ask[float64](ctx, c.rpc, pathAvgSessionMinutes, rpc.CallTimeout, windowRequest{Window: w})
}

func (c *Client) RepeatEngagement(ctx context.Context, w Window) (RepeatEngagement, error) {
	return rpc.Ask[RepeatEngagement](ctx, c.rpc, pathRepeatEngagement, rpc.CallTimeout, windowRequest{Window: w})
}

func (c *Client) LibraryViewsByCategory(ctx context.Context, w Window) ([]CategoryViews, error) {
	return rpc.Ask[[]CategoryViews](ctx, c.rpc, pathLibraryViewsByCategory, rpc.CallTimeout, windowRequest{Window: w})
}

func (c *Client) TopLibraryVisits(ctx context.Context, w Window, limit int) ([]ContentVisits, error) {
	return rpc.Ask[[]ContentVisits](ctx, c.rpc, pathTopLibraryVisits, rpc.CallTimeout, windowRequest{Window: w, Limit: limit})
}

func (c *Client) TopVideoVisits(ctx context.Context, w Window, limit int) ([]ContentVisits, error) {
	return rpc.Ask[[]ContentVisits](ctx, c.rpc, pathTopVideoVisits, rpc.CallTimeout, windowRequest{Window: w, Limit: limit})
}

func (c *Client) ResetDemoActivity(ctx context.Context, before time.Time) error {
	_, err := rpc.Ask[any](ctx, c.rpc, pathResetDemoActivity, rpc.DemoTimeout, resetDemoRequest{Before: before})
	return err
}

func (c *Client) SeedDemoActivity(ctx context.Context, facilityID uint, userIDs []uint) error {
	_, err := rpc.Ask[any](ctx, c.rpc, pathSeedDemoActivity, rpc.DemoTimeout, seedDemoRequest{FacilityID: facilityID, UserIDs: userIDs})
	return err
}
