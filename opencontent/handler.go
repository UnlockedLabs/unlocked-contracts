package opencontent

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
)

// maxRequestBytes bounds a question's arguments. The largest, a demo seed's
// user ids, is far smaller.
const maxRequestBytes = 1 << 20

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
	handle(mux, pathEndActivity, func(ctx context.Context, req activityRequest) (any, error) {
		oc.EndActivity(ctx, req.ActivityID)
		return nil, nil
	})
	handle(mux, pathTopLibraries, func(ctx context.Context, req userRequest) ([]ContentUsage, error) {
		return oc.TopLibraries(ctx, req.UserID, req.FacilityID)
	})
	handle(mux, pathRecentVideos, func(ctx context.Context, req userRequest) ([]ContentUsage, error) {
		return oc.RecentVideos(ctx, req.UserID)
	})
	handle(mux, pathResourcesAccessed, func(ctx context.Context, req userRequest) (int64, error) {
		return oc.ResourcesAccessed(ctx, req.UserID)
	})
	handle(mux, pathActivityEngagement, func(ctx context.Context, req userRequest) (ActivityEngagement, error) {
		return oc.ActivityEngagement(ctx, req.UserID)
	})
	handle(mux, pathFacilityContent, func(ctx context.Context, req facilitiesRequest) (map[uint]FacilityContent, error) {
		return oc.FacilityContent(ctx, req.FacilityIDs)
	})
	handle(mux, pathInteractions, func(ctx context.Context, req windowRequest) (Interactions, error) {
		return oc.Interactions(ctx, req.Window)
	})
	handle(mux, pathAvgSessionMinutes, func(ctx context.Context, req windowRequest) (float64, error) {
		return oc.AvgSessionMinutes(ctx, req.Window)
	})
	handle(mux, pathRepeatEngagement, func(ctx context.Context, req windowRequest) (RepeatEngagement, error) {
		return oc.RepeatEngagement(ctx, req.Window)
	})
	handle(mux, pathLibraryViewsByCategory, func(ctx context.Context, req windowRequest) ([]CategoryViews, error) {
		return oc.LibraryViewsByCategory(ctx, req.Window)
	})
	handle(mux, pathTopLibraryVisits, func(ctx context.Context, req windowRequest) ([]ContentVisits, error) {
		return oc.TopLibraryVisits(ctx, req.Window, req.Limit)
	})
	handle(mux, pathTopVideoVisits, func(ctx context.Context, req windowRequest) ([]ContentVisits, error) {
		return oc.TopVideoVisits(ctx, req.Window, req.Limit)
	})
	handle(mux, pathResetDemoActivity, func(ctx context.Context, req resetDemoRequest) (any, error) {
		return nil, oc.ResetDemoActivity(ctx, req.Before)
	})
	handle(mux, pathSeedDemoActivity, func(ctx context.Context, req seedDemoRequest) (any, error) {
		return nil, oc.SeedDemoActivity(ctx, req.FacilityID, req.UserIDs)
	})
	return requireToken(token, mux)
}

// requireToken passes on only the requests that carry token as a bearer
// token. With an empty token it passes on none, so a site without its secret
// fails closed.
func requireToken(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			slog.Warn("opencontent: refuse a question without the token", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			writeJSON(w, http.StatusUnauthorized, envelope[any]{Message: "missing or wrong token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handle serves one question on path: it reads the arguments, asks answer,
// and writes what it returns.
func handle[Req, Resp any](mux *http.ServeMux, path string, answer func(context.Context, Req) (Resp, error)) {
	mux.HandleFunc("POST "+path, func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, envelope[any]{Message: "read arguments: " + err.Error()})
			return
		}
		resp, err := answer(r.Context(), req)
		if err != nil {
			slog.Error("opencontent: answer core", "path", path, "err", err)
			writeJSON(w, http.StatusInternalServerError, envelope[any]{Message: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, envelope[Resp]{Data: resp})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("opencontent: write answer", "err", err)
	}
}
