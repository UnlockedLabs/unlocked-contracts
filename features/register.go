package features

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Feature is a feature a product registers with core.
type Feature struct {
	// Name is permanent: lowercase letters, digits and underscores, starting
	// with a letter, up to 64 characters.
	Name Access
	// Title and Description are what Feature Control shows. Title is required,
	// up to 100 characters.
	Title       string
	Description string
}

// Register adds f to core's list of features, or updates its title and
// description if it is already there. A new feature starts off at every
// facility; registering again never turns it on or changes any facility.
//
// Call it at startup, before serving, and fail startup if it returns an error,
// so the product restarts until core answers. It asks core once at coreURL,
// such as "http://server:8080". A nil client means http.DefaultClient; the call
// times out after 2 seconds either way.
func Register(ctx context.Context, coreURL string, client *http.Client, f Feature) error {
	if client == nil {
		client = http.DefaultClient
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	body, err := json.Marshal(struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}{f.Title, f.Description})
	if err != nil {
		return fmt.Errorf("features: register %s: %w", f.Name, err)
	}
	target := strings.TrimSuffix(coreURL, "/") + checkPath + "/" + url.PathEscape(string(f.Name))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("features: register %s: build request: %w", f.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("features: register %s: %w", f.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	// Core wraps every response in {"message", "data"}; the message says what
	// was wrong, such as a bad name.
	var reply struct {
		Message string `json:"message"`
	}
	if json.NewDecoder(resp.Body).Decode(&reply) == nil && reply.Message != "" {
		return fmt.Errorf("features: register %s: core answered %s: %s", f.Name, resp.Status, reply.Message)
	}
	return fmt.Errorf("features: register %s: core answered %s", f.Name, resp.Status)
}
