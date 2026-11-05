package programs

import (
	"context"
	"net/http"
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

// Client asks Programs' service core's questions over HTTP. It is safe for
// concurrent use.
type Client struct {
	c *rpc.Client
}

var _ Programs = (*Client)(nil)

// NewClient returns a Client that asks Programs at baseURL, such as
// "http://programs:8080", with the shared service token. A nil client means
// http.DefaultClient; each question times out after 10 seconds either way, and
// sooner if the caller's context ends.
func NewClient(baseURL, token string, client *http.Client) *Client {
	return &Client{rpc.NewClient("programs", baseURL, token, client)}
}

func (c *Client) EnrolledUserIDs(ctx context.Context, classID uint) ([]uint, error) {
	return rpc.Ask[[]uint](ctx, c.c, pathEnrolledUserIDs, rpc.CallTimeout, classRequest{ClassID: classID})
}

func (c *Client) EligibleUserIDs(ctx context.Context, classID uint) ([]uint, error) {
	return rpc.Ask[[]uint](ctx, c.c, pathEligibleUserIDs, rpc.CallTimeout, classRequest{ClassID: classID})
}

func (c *Client) TransferConflicts(ctx context.Context, userID, toFacilityID uint) ([]ResidentTransferProgramConflicts, error) {
	return rpc.Ask[[]ResidentTransferProgramConflicts](ctx, c.c, pathTransferConflicts, rpc.CallTimeout, userRequest{UserID: userID, FacilityID: toFacilityID})
}

func (c *Client) EndEnrollmentsForTransfer(ctx context.Context, userID, fromFacilityID uint) error {
	_, err := rpc.Ask[any](ctx, c.c, pathEndEnrollmentsForTransfer, rpc.CallTimeout, userRequest{UserID: userID, FacilityID: fromFacilityID})
	return err
}

func (c *Client) WithdrawEnrollments(ctx context.Context, userID uint) error {
	_, err := rpc.Ask[any](ctx, c.c, pathWithdrawEnrollments, rpc.CallTimeout, userRequest{UserID: userID})
	return err
}

func (c *Client) ResidentPrograms(ctx context.Context, userID uint) ([]ResidentProgramClassInfo, error) {
	return rpc.Ask[[]ResidentProgramClassInfo](ctx, c.c, pathResidentPrograms, rpc.CallTimeout, userRequest{UserID: userID})
}

func (c *Client) FacilityPrograms(ctx context.Context, facilityIDs []uint) (map[uint]FacilityPrograms, error) {
	return rpc.Ask[map[uint]FacilityPrograms](ctx, c.c, pathFacilityPrograms, rpc.CallTimeout, facilitiesRequest{FacilityIDs: facilityIDs})
}

func (c *Client) ClassHistoryCreatedAt(ctx context.Context, ids []uint) (map[uint]time.Time, error) {
	return rpc.Ask[map[uint]time.Time](ctx, c.c, pathClassHistoryCreatedAt, rpc.CallTimeout, idsRequest{IDs: ids})
}

func (c *Client) EngagementOverview(ctx context.Context, start, end *time.Time, facilityID *uint) (ProgramEngagementOverview, error) {
	return rpc.Ask[ProgramEngagementOverview](ctx, c.c, pathEngagementOverview, rpc.CallTimeout, overviewRequest{Start: start, End: end, FacilityID: facilityID})
}

func (c *Client) SecondProgramEnrollment(ctx context.Context, facilityID *uint) ([]SecondProgramEnrollmentRow, error) {
	return rpc.Ask[[]SecondProgramEnrollmentRow](ctx, c.c, pathSecondProgramEnrollment, rpc.CallTimeout, facilityRequest{FacilityID: facilityID})
}

func (c *Client) CompletionMatrix(ctx context.Context, facilityID *uint) ([]ProgramCompletionMatrixCell, error) {
	return rpc.Ask[[]ProgramCompletionMatrixCell](ctx, c.c, pathCompletionMatrix, rpc.CallTimeout, facilityRequest{FacilityID: facilityID})
}

func (c *Client) LoadDistribution(ctx context.Context, facilityID *uint) (ProgramLoadDistribution, error) {
	return rpc.Ask[ProgramLoadDistribution](ctx, c.c, pathLoadDistribution, rpc.CallTimeout, facilityRequest{FacilityID: facilityID})
}

func (c *Client) EnrollmentByProgramType(ctx context.Context, facilityID *uint) ([]ProgramTypeEnrollment, error) {
	return rpc.Ask[[]ProgramTypeEnrollment](ctx, c.c, pathEnrollmentByProgramType, rpc.CallTimeout, facilityRequest{FacilityID: facilityID})
}

func (c *Client) ClassMetrics(ctx context.Context, facilityID *uint) (ClassDashboardMetrics, error) {
	return rpc.Ask[ClassDashboardMetrics](ctx, c.c, pathClassMetrics, rpc.CallTimeout, facilityRequest{FacilityID: facilityID})
}

func (c *Client) FacilityHealth(ctx context.Context, facilityID *uint, days int, tz string) ([]FacilityHealthSummary, error) {
	return rpc.Ask[[]FacilityHealthSummary](ctx, c.c, pathFacilityHealth, rpc.CallTimeout, healthRequest{FacilityID: facilityID, Days: days, TZ: tz})
}
