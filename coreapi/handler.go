package coreapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
	"github.com/UnlockedLabs/unlocked-contracts/users"
)

// NewHandler serves core's answers to a product service, for core to mount on
// its internal server:
//
//	mux.Handle("/internal/v1/core/", coreapi.NewHandler(users, live, token))
//
// A call without the shared service token gets a 401, and an empty token
// rejects every call. Arguments that aren't valid JSON, or a create with any
// role but student or empty, get a 400, a missing user (an error wrapping
// ErrUserNotFound) a 404, and any other error a 500 with the error as the
// message, logged with slog.
func NewHandler(u Users, l LiveProviders, token string) http.Handler {
	mux := http.NewServeMux()
	rpc.Handle(mux, "coreapi", pathGet, func(ctx context.Context, req idRequest) (User, error) {
		return u.Get(ctx, req.ID)
	})
	rpc.Handle(mux, "coreapi", pathCreate, func(ctx context.Context, req createRequest) (User, error) {
		if role := req.User.Role; role != "" && role != users.Student {
			return User{}, fmt.Errorf("%w: create only accepts the student role, not %q", rpc.ErrInvalid, role)
		}
		return u.Create(ctx, req.User)
	})
	rpc.Handle(mux, "coreapi", pathCreateLogin, func(ctx context.Context, req idRequest) (string, error) {
		return u.CreateLogin(ctx, req.ID)
	})
	rpc.Handle(mux, "coreapi", pathDelete, func(ctx context.Context, req idRequest) (any, error) {
		return nil, u.Delete(ctx, req.ID)
	})
	rpc.Handle(mux, "coreapi", pathLiveUsers, func(ctx context.Context, req providerRequest) (liveUsersAnswer, error) {
		users, ok, err := l.Users(ctx, req.ProviderID)
		return liveUsersAnswer{Users: users, OK: ok}, err
	})
	rpc.Handle(mux, "coreapi", pathMappingsChanged, func(_ context.Context, req mappingsRequest) (any, error) {
		l.MappingsChanged(req.ProviderID, req.UserID)
		return nil, nil
	})
	return rpc.RequireToken(token, mux)
}
