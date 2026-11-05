package providerplatforms

import (
	"context"
	"net/http"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

// NewHandler serves pp's answers to core's questions, for Provider Platforms
// to mount on its server:
//
//	mux.Handle("/internal/v1/provider-platforms/", providerplatforms.NewHandler(answers, token))
//
// A call without the shared service token gets a 401, and an empty token
// rejects every call. Arguments that aren't valid JSON get a 400, and an
// error from pp a 500 with the error as the message, logged with slog.
func NewHandler(pp ProviderPlatforms, token string) http.Handler {
	mux := http.NewServeMux()
	handle(mux, pathProvider, func(ctx context.Context, req idRequest) (Provider, error) {
		return pp.Provider(ctx, req.ID)
	})
	handle(mux, pathActiveProviders, func(ctx context.Context, _ noRequest) ([]Provider, error) {
		return pp.ActiveProviders(ctx)
	})
	handle(mux, pathUserMappings, func(ctx context.Context, req userRequest) ([]Mapping, error) {
		return pp.UserMappings(ctx, req.UserID)
	})
	handle(mux, pathMappedUserIDs, func(ctx context.Context, req providerRequest) ([]uint, error) {
		return pp.MappedUserIDs(ctx, req.ProviderID)
	})
	handle(mux, pathDeleteUserMappings, func(ctx context.Context, req userRequest) (any, error) {
		return nil, pp.DeleteUserMappings(ctx, req.UserID)
	})
	handle(mux, pathMappedUsers, func(ctx context.Context, req mappedRequest) ([]MappedUser, error) {
		return pp.MappedUsers(ctx, req.ProviderID, req.ExternalUserIDs, req.FacilityID)
	})
	handle(mux, pathCountMapped, func(ctx context.Context, req mappedRequest) (int64, error) {
		return pp.CountMapped(ctx, req.ProviderID, req.ExternalUserIDs)
	})
	handle(mux, pathCountMappedAtFacility, func(ctx context.Context, req mappedRequest) (int64, error) {
		return pp.CountMappedAtFacility(ctx, req.ProviderID, req.ExternalUserIDs, req.FacilityID)
	})
	handle(mux, pathCountMappedPerFacility, func(ctx context.Context, req mappedRequest) (map[uint]int64, error) {
		return pp.CountMappedPerFacility(ctx, req.ProviderID, req.ExternalUserIDs)
	})
	handle(mux, pathAddUserToProviders, func(ctx context.Context, req addToProvidersRequest) (any, error) {
		return nil, pp.AddUserToProviders(ctx, req.User, req.ProviderIDs)
	})
	handle(mux, pathAddUserToKolibri, func(ctx context.Context, req addToKolibriRequest) (any, error) {
		return nil, pp.AddUserToKolibri(ctx, req.User)
	})
	handle(mux, pathSetupKolibri, func(ctx context.Context, _ noRequest) (any, error) {
		return nil, pp.SetupKolibri(ctx)
	})
	handle(mux, pathLearningStats, func(ctx context.Context, req statsRequest) (LearningStats, error) {
		return pp.LearningStats(ctx, req.FacilityID)
	})
	handle(mux, pathLastDemoSeed, func(ctx context.Context, _ noRequest) (lastDemoSeedAnswer, error) {
		last, found, err := pp.LastDemoSeed(ctx)
		return lastDemoSeedAnswer{Last: last, Found: found}, err
	})
	handle(mux, pathResetDemoLearning, func(ctx context.Context, req resetDemoRequest) (any, error) {
		return nil, pp.ResetDemoLearning(ctx, req.Through)
	})
	handle(mux, pathSeedDemoLearning, func(ctx context.Context, req seedDemoRequest) (any, error) {
		return nil, pp.SeedDemoLearning(ctx, req.UserIDs)
	})
	return rpc.RequireToken(token, mux)
}

func handle[Req, Resp any](mux *http.ServeMux, path string, answer func(context.Context, Req) (Resp, error)) {
	rpc.Handle(mux, "providerplatforms", path, answer)
}
