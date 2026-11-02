package providerplatforms

import (
	"context"
	"net/http"
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/coreapi"
	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

// Client asks Provider Platforms' service core's questions over HTTP. It is
// safe for concurrent use.
type Client struct {
	c *rpc.Client
}

var _ ProviderPlatforms = (*Client)(nil)

// NewClient returns a Client that asks Provider Platforms at baseURL, such as
// "http://provider-platforms:8080", with the shared service token. A nil
// client means http.DefaultClient; each question times out after 10 seconds
// either way, the demo seed and reset after 5 minutes, and sooner if the
// caller's context ends.
func NewClient(baseURL, token string, client *http.Client) *Client {
	return &Client{rpc.NewClient("providerplatforms", baseURL, token, client)}
}

func (c *Client) Provider(ctx context.Context, id uint) (Provider, error) {
	return rpc.Ask[Provider](ctx, c.c, pathProvider, rpc.CallTimeout, idRequest{ID: id})
}

func (c *Client) ActiveProviders(ctx context.Context) ([]Provider, error) {
	return rpc.Ask[[]Provider](ctx, c.c, pathActiveProviders, rpc.CallTimeout, noRequest{})
}

func (c *Client) UserMappings(ctx context.Context, userID uint) ([]Mapping, error) {
	return rpc.Ask[[]Mapping](ctx, c.c, pathUserMappings, rpc.CallTimeout, userRequest{UserID: userID})
}

func (c *Client) MappedUserIDs(ctx context.Context, providerID uint) ([]uint, error) {
	return rpc.Ask[[]uint](ctx, c.c, pathMappedUserIDs, rpc.CallTimeout, providerRequest{ProviderID: providerID})
}

func (c *Client) DeleteUserMappings(ctx context.Context, userID uint) error {
	_, err := rpc.Ask[any](ctx, c.c, pathDeleteUserMappings, rpc.CallTimeout, userRequest{UserID: userID})
	return err
}

func (c *Client) MappedUsers(ctx context.Context, providerID uint, externalUserIDs []string, facilityID uint) ([]MappedUser, error) {
	return rpc.Ask[[]MappedUser](ctx, c.c, pathMappedUsers, rpc.CallTimeout, mappedRequest{providerID, externalUserIDs, facilityID})
}

func (c *Client) CountMapped(ctx context.Context, providerID uint, externalUserIDs []string) (int64, error) {
	return rpc.Ask[int64](ctx, c.c, pathCountMapped, rpc.CallTimeout, mappedRequest{ProviderID: providerID, ExternalUserIDs: externalUserIDs})
}

func (c *Client) CountMappedAtFacility(ctx context.Context, providerID uint, externalUserIDs []string, facilityID uint) (int64, error) {
	return rpc.Ask[int64](ctx, c.c, pathCountMappedAtFacility, rpc.CallTimeout, mappedRequest{providerID, externalUserIDs, facilityID})
}

func (c *Client) CountMappedPerFacility(ctx context.Context, providerID uint, externalUserIDs []string) (map[uint]int64, error) {
	return rpc.Ask[map[uint]int64](ctx, c.c, pathCountMappedPerFacility, rpc.CallTimeout, mappedRequest{ProviderID: providerID, ExternalUserIDs: externalUserIDs})
}

func (c *Client) AddUserToProviders(ctx context.Context, user coreapi.User, providerIDs []uint) error {
	_, err := rpc.Ask[any](ctx, c.c, pathAddUserToProviders, rpc.CallTimeout, addToProvidersRequest{user, providerIDs})
	return err
}

func (c *Client) AddUserToKolibri(ctx context.Context, user coreapi.User) error {
	_, err := rpc.Ask[any](ctx, c.c, pathAddUserToKolibri, rpc.CallTimeout, addToKolibriRequest{user})
	return err
}

func (c *Client) SetupKolibri(ctx context.Context) error {
	_, err := rpc.Ask[any](ctx, c.c, pathSetupKolibri, rpc.CallTimeout, noRequest{})
	return err
}

func (c *Client) LearningStats(ctx context.Context, facilityID *uint) (LearningStats, error) {
	return rpc.Ask[LearningStats](ctx, c.c, pathLearningStats, rpc.CallTimeout, statsRequest{facilityID})
}

func (c *Client) LastDemoSeed(ctx context.Context) (time.Time, bool, error) {
	a, err := rpc.Ask[lastDemoSeedAnswer](ctx, c.c, pathLastDemoSeed, rpc.CallTimeout, noRequest{})
	return a.Last, a.Found, err
}

func (c *Client) ResetDemoLearning(ctx context.Context, through time.Time) error {
	_, err := rpc.Ask[any](ctx, c.c, pathResetDemoLearning, rpc.DemoTimeout, resetDemoRequest{through})
	return err
}

func (c *Client) SeedDemoLearning(ctx context.Context, userIDs []uint) error {
	_, err := rpc.Ask[any](ctx, c.c, pathSeedDemoLearning, rpc.DemoTimeout, seedDemoRequest{userIDs})
	return err
}
