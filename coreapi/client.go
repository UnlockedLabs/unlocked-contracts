package coreapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

type usersClient struct{ c *rpc.Client }

type liveProvidersClient struct{ c *rpc.Client }

// NewUsersClient returns a Users that asks core at baseURL, such as
// "http://server:8080", with the shared service token. A nil client means
// http.DefaultClient; each call times out after 10 seconds either way.
func NewUsersClient(baseURL, token string, client *http.Client) Users {
	return usersClient{rpc.NewClient("coreapi", baseURL, token, client)}
}

// NewLiveProvidersClient returns a LiveProviders that asks core at baseURL.
// See NewUsersClient.
func NewLiveProvidersClient(baseURL, token string, client *http.Client) LiveProviders {
	return liveProvidersClient{rpc.NewClient("coreapi", baseURL, token, client)}
}

func (u usersClient) Get(ctx context.Context, id uint) (User, error) {
	return rpc.Ask[User](ctx, u.c, pathGet, rpc.CallTimeout, idRequest{ID: id})
}

func (u usersClient) Create(ctx context.Context, nu NewUser) (User, error) {
	return rpc.Ask[User](ctx, u.c, pathCreate, rpc.CallTimeout, createRequest{User: nu})
}

func (u usersClient) CreateLogin(ctx context.Context, userID uint) (string, error) {
	return rpc.Ask[string](ctx, u.c, pathCreateLogin, rpc.CallTimeout, idRequest{ID: userID})
}

func (u usersClient) Delete(ctx context.Context, id uint) error {
	_, err := rpc.Ask[any](ctx, u.c, pathDelete, rpc.CallTimeout, idRequest{ID: id})
	return err
}

func (l liveProvidersClient) Users(ctx context.Context, providerID uint) ([]ProviderUser, bool, error) {
	a, err := rpc.Ask[liveUsersAnswer](ctx, l.c, pathLiveUsers, rpc.CallTimeout, providerRequest{ProviderID: providerID})
	return a.Users, a.OK, err
}

func (l liveProvidersClient) MappingsChanged(providerID, userID uint) {
	_, err := rpc.Ask[any](context.Background(), l.c, pathMappingsChanged, rpc.CallTimeout, mappingsRequest{ProviderID: providerID, UserID: userID})
	if err != nil {
		slog.Error("coreapi: mappings changed", "provider_id", providerID, "user_id", userID, "err", err)
	}
}
