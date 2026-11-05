package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// CallTimeout bounds each call, whatever client the Client was given.
const CallTimeout = 10 * time.Second

// DemoTimeout bounds the demo seed and reset, which write months of data one
// row at a time.
const DemoTimeout = 5 * time.Minute

const maxRequestBytes = 1 << 20

// ErrNotFound is what a handler's answer wraps to say 404, and what a Client
// returns for a 404.
var ErrNotFound = errors.New("not found")

// ErrInvalid is what a handler's answer wraps to say 400.
var ErrInvalid = errors.New("invalid argument")

// Envelope is every answer, in core's JSON shape: the answer in data, or what
// went wrong in message.
type Envelope[T any] struct {
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// Client posts calls to one service. It is safe for concurrent use.
type Client struct {
	pkg     string
	baseURL string
	token   string
	http    *http.Client
}

// NewClient returns a Client for the service at baseURL. pkg prefixes its
// errors. A nil client means http.DefaultClient.
func NewClient(pkg, baseURL, token string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{pkg: pkg, baseURL: strings.TrimSuffix(baseURL, "/"), token: token, http: client}
}

// Ask posts args to path and returns the answer. Any status but 200 is an
// error, with the service's message when it sent one.
func Ask[Resp any](ctx context.Context, c *Client, path string, timeout time.Duration, args any) (Resp, error) {
	var zero Resp
	body, err := json.Marshal(args)
	if err != nil {
		return zero, fmt.Errorf("%s: write arguments for %s: %w", c.pkg, path, err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("%s: build request for %s: %w", c.pkg, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	SetToken(req, c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return zero, fmt.Errorf("%s: ask %s: %w", c.pkg, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var answer Envelope[Resp]
	decodeErr := json.NewDecoder(resp.Body).Decode(&answer)
	switch {
	case resp.StatusCode == http.StatusNotFound && decodeErr == nil && answer.Message != "":
		return zero, fmt.Errorf("%s: %s: %w: %s", c.pkg, path, ErrNotFound, answer.Message)
	case resp.StatusCode != http.StatusOK && decodeErr == nil && answer.Message != "":
		return zero, fmt.Errorf("%s: %s answered %s: %s", c.pkg, path, resp.Status, answer.Message)
	case resp.StatusCode != http.StatusOK:
		return zero, fmt.Errorf("%s: %s answered %s", c.pkg, path, resp.Status)
	case decodeErr != nil:
		return zero, fmt.Errorf("%s: read answer from %s: %w", c.pkg, path, decodeErr)
	}
	return answer.Data, nil
}

// Handle serves one call on path: it reads the arguments, asks answer, and
// writes what it returns. An error wrapping ErrNotFound is a 404, one wrapping
// ErrInvalid a 400, any other a 500 logged with slog.
func Handle[Req, Resp any](mux *http.ServeMux, pkg, path string, answer func(context.Context, Req) (Resp, error)) {
	mux.HandleFunc("POST "+path, func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes)).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, Envelope[any]{Message: "read arguments: " + err.Error()})
			return
		}
		resp, err := answer(r.Context(), req)
		switch {
		case errors.Is(err, ErrNotFound):
			WriteJSON(w, http.StatusNotFound, Envelope[any]{Message: err.Error()})
		case errors.Is(err, ErrInvalid):
			WriteJSON(w, http.StatusBadRequest, Envelope[any]{Message: err.Error()})
		case err != nil:
			slog.Error(pkg+": answer", "path", path, "err", err)
			WriteJSON(w, http.StatusInternalServerError, Envelope[any]{Message: err.Error()})
		default:
			WriteJSON(w, http.StatusOK, Envelope[Resp]{Data: resp})
		}
	})
}

// WriteJSON writes body as JSON with status.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("rpc: write answer", "err", err)
	}
}
