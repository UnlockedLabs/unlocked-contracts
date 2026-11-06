package programs

import (
	"context"
	"net/http"
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

// NewHandler serves p's answers to core's questions, for Programs to mount on
// its server:
//
//	mux.Handle("/internal/v1/programs/", programs.NewHandler(answers, token))
//
// A call without the shared service token gets a 401, and an empty token
// rejects every call. Arguments that aren't valid JSON get a 400, and an
// error from p a 500 with the error as the message, logged with slog.
func NewHandler(p Programs, token string) http.Handler {
	mux := http.NewServeMux()
	rpc.Handle(mux, "programs", pathEnrolledUserIDs, func(ctx context.Context, req classRequest) ([]uint, error) {
		return p.EnrolledUserIDs(ctx, req.ClassID)
	})
	rpc.Handle(mux, "programs", pathEligibleUserIDs, func(ctx context.Context, req classRequest) ([]uint, error) {
		return p.EligibleUserIDs(ctx, req.ClassID)
	})
	rpc.Handle(mux, "programs", pathTransferConflicts, func(ctx context.Context, req userRequest) ([]ResidentTransferProgramConflicts, error) {
		return p.TransferConflicts(ctx, req.UserID, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathEndEnrollmentsForTransfer, func(ctx context.Context, req userRequest) (any, error) {
		return nil, p.EndEnrollmentsForTransfer(ctx, req.UserID, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathWithdrawEnrollments, func(ctx context.Context, req userRequest) (any, error) {
		return nil, p.WithdrawEnrollments(ctx, req.UserID)
	})
	rpc.Handle(mux, "programs", pathResidentPrograms, func(ctx context.Context, req userRequest) ([]ResidentProgramClassInfo, error) {
		return p.ResidentPrograms(ctx, req.UserID)
	})
	rpc.Handle(mux, "programs", pathFacilityPrograms, func(ctx context.Context, req facilitiesRequest) (map[uint]FacilityPrograms, error) {
		return p.FacilityPrograms(ctx, req.FacilityIDs)
	})
	rpc.Handle(mux, "programs", pathClassHistoryCreatedAt, func(ctx context.Context, req idsRequest) (map[uint]time.Time, error) {
		return p.ClassHistoryCreatedAt(ctx, req.IDs)
	})
	rpc.Handle(mux, "programs", pathEngagementOverview, func(ctx context.Context, req overviewRequest) (ProgramEngagementOverview, error) {
		return p.EngagementOverview(ctx, req.Start, req.End, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathSecondProgramEnrollment, func(ctx context.Context, req facilityRequest) ([]SecondProgramEnrollmentRow, error) {
		return p.SecondProgramEnrollment(ctx, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathCompletionMatrix, func(ctx context.Context, req facilityRequest) ([]ProgramCompletionMatrixCell, error) {
		return p.CompletionMatrix(ctx, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathLoadDistribution, func(ctx context.Context, req facilityRequest) (ProgramLoadDistribution, error) {
		return p.LoadDistribution(ctx, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathEnrollmentByProgramType, func(ctx context.Context, req facilityRequest) ([]ProgramTypeEnrollment, error) {
		return p.EnrollmentByProgramType(ctx, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathClassMetrics, func(ctx context.Context, req facilityRequest) (ClassDashboardMetrics, error) {
		return p.ClassMetrics(ctx, req.FacilityID)
	})
	rpc.Handle(mux, "programs", pathFacilityHealth, func(ctx context.Context, req healthRequest) ([]FacilityHealthSummary, error) {
		return p.FacilityHealth(ctx, req.FacilityID, req.Days, req.TZ)
	})
	rpc.Handle(mux, "programs", pathLiveUsers, func(ctx context.Context, req providerRequest) (liveUsersAnswer, error) {
		users, ok, err := p.LiveUsers(ctx, req.ProviderID)
		return liveUsersAnswer{Users: users, OK: ok}, err
	})
	rpc.Handle(mux, "programs", pathMappingsChanged, func(_ context.Context, req mappingsRequest) (any, error) {
		p.MappingsChanged(req.ProviderID, req.UserID)
		return nil, nil
	})
	return rpc.RequireToken(token, mux)
}
