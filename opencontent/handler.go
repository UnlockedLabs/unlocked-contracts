package opencontent

import (
	"context"
	"net/http"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

// NewHandler serves oc's answers to core's questions, for Open Content to
// mount on its server:
//
//	mux.Handle("/internal/v1/open-content/", opencontent.NewHandler(answers, token))
//
// Only a request with token, the site's secret that core's Client also has,
// is answered; any other gets a 401, and so does every request when token is
// empty. Arguments that aren't valid JSON get a 400, and an error from oc a
// 500 with the error as the message, logged with slog.
func NewHandler(oc OpenContent, token string) http.Handler {
	mux := http.NewServeMux()
	rpc.Handle(mux, "opencontent", pathEndActivity, func(ctx context.Context, req activityRequest) (any, error) {
		oc.EndActivity(ctx, req.ActivityID)
		return nil, nil
	})
	rpc.Handle(mux, "opencontent", pathTopLibraries, func(ctx context.Context, req userRequest) ([]ContentUsage, error) {
		return oc.TopLibraries(ctx, req.UserID, req.FacilityID)
	})
	rpc.Handle(mux, "opencontent", pathRecentVideos, func(ctx context.Context, req userRequest) ([]ContentUsage, error) {
		return oc.RecentVideos(ctx, req.UserID)
	})
	rpc.Handle(mux, "opencontent", pathResourcesAccessed, func(ctx context.Context, req userRequest) (int64, error) {
		return oc.ResourcesAccessed(ctx, req.UserID)
	})
	rpc.Handle(mux, "opencontent", pathActivityEngagement, func(ctx context.Context, req userRequest) (ActivityEngagement, error) {
		return oc.ActivityEngagement(ctx, req.UserID)
	})
	rpc.Handle(mux, "opencontent", pathFacilityContent, func(ctx context.Context, req facilitiesRequest) (map[uint]FacilityContent, error) {
		return oc.FacilityContent(ctx, req.FacilityIDs)
	})
	rpc.Handle(mux, "opencontent", pathInteractions, func(ctx context.Context, req windowRequest) (Interactions, error) {
		return oc.Interactions(ctx, req.Window)
	})
	rpc.Handle(mux, "opencontent", pathAvgSessionMinutes, func(ctx context.Context, req windowRequest) (float64, error) {
		return oc.AvgSessionMinutes(ctx, req.Window)
	})
	rpc.Handle(mux, "opencontent", pathRepeatEngagement, func(ctx context.Context, req windowRequest) (RepeatEngagement, error) {
		return oc.RepeatEngagement(ctx, req.Window)
	})
	rpc.Handle(mux, "opencontent", pathLibraryViewsByCategory, func(ctx context.Context, req windowRequest) ([]CategoryViews, error) {
		return oc.LibraryViewsByCategory(ctx, req.Window)
	})
	rpc.Handle(mux, "opencontent", pathTopLibraryVisits, func(ctx context.Context, req windowRequest) ([]ContentVisits, error) {
		return oc.TopLibraryVisits(ctx, req.Window, req.Limit)
	})
	rpc.Handle(mux, "opencontent", pathTopVideoVisits, func(ctx context.Context, req windowRequest) ([]ContentVisits, error) {
		return oc.TopVideoVisits(ctx, req.Window, req.Limit)
	})
	rpc.Handle(mux, "opencontent", pathResetDemoActivity, func(ctx context.Context, req resetDemoRequest) (any, error) {
		return nil, oc.ResetDemoActivity(ctx, req.Before)
	})
	rpc.Handle(mux, "opencontent", pathSeedDemoActivity, func(ctx context.Context, req seedDemoRequest) (any, error) {
		return nil, oc.SeedDemoActivity(ctx, req.FacilityID, req.UserIDs)
	})
	return rpc.RequireToken(token, mux)
}
