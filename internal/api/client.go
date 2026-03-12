// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// DefaultBaseURL is the default deSEC API base URL.
	DefaultBaseURL = "https://desec.io/api/v1"

	// defaultMaxRetries is the maximum number of times a 429 response is retried.
	defaultMaxRetries = 5
)

// ClientOption is a functional option for configuring a Client.
type ClientOption func(*Client)

// WithMaxRetries sets the maximum number of times a 429 response is retried.
func WithMaxRetries(n int) ClientOption {
	return func(c *Client) { c.maxRetries = n }
}

// WithSerializeRequests controls whether concurrent API requests are serialized
// per lock key to avoid hitting deSEC rate limits. When enabled (the default),
// domain-scoped requests are serialized per domain and global DNS requests share
// a single lock, preventing bursts that exhaust per-domain or global rate limits.
func WithSerializeRequests(v bool) ClientOption {
	return func(c *Client) { c.serializeRequests = v }
}

// Client is a deSEC REST API client.
type Client struct {
	BaseURL           string
	Token             string
	httpClient        *http.Client
	maxRetries        int
	serializeRequests bool
	mu                sync.Map // key -> *sync.Mutex
}

// NewClient creates a new deSEC API client.
func NewClient(baseURL, token string, opts ...ClientOption) *Client {
	c := &Client{
		BaseURL:           baseURL,
		Token:             token,
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		maxRetries:        defaultMaxRetries,
		serializeRequests: true,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// do executes an HTTP request with automatic rate-limit retry.
// On HTTP 429, it reads the Retry-After header and sleeps before retrying.
func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	ctx = tflog.NewSubsystem(ctx, "desec-api")

	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	for attempt := range c.maxRetries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Token "+c.Token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		tflog.SubsystemTrace(ctx, "desec-api", "sending API request", map[string]any{
			"http_method": method,
			"url":         c.BaseURL + path,
			"attempt":     attempt + 1,
		})

		resp, err := c.httpClient.Do(req)
		if err != nil {
			tflog.SubsystemDebug(ctx, "desec-api", "API request failed", map[string]any{
				"http_method": method,
				"url":         c.BaseURL + path,
				"error":       err.Error(),
			})
			return nil, fmt.Errorf("executing request: %w", err)
		}

		tflog.SubsystemTrace(ctx, "desec-api", "received API response", map[string]any{
			"http_method":      method,
			"url":              c.BaseURL + path,
			"http_status_code": resp.StatusCode,
		})

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// HTTP 429: parse Retry-After and sleep before retrying.
		_ = resp.Body.Close()

		waitSeconds := 1
		retryAfterPresent := false
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			retryAfterPresent = true
			if secs, parseErr := strconv.Atoi(ra); parseErr == nil && secs > 0 {
				waitSeconds = secs
			}
		}

		retryFields := map[string]any{
			"http_method": method,
			"url":         c.BaseURL + path,
			"attempt":     attempt + 1,
		}
		if retryAfterPresent {
			retryFields["retry_after_seconds"] = waitSeconds
		}
		tflog.SubsystemDebug(ctx, "desec-api", "rate limited, waiting before retry", retryFields)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(waitSeconds) * time.Second):
		}
	}

	return nil, fmt.Errorf("request rate-limited: still receiving HTTP 429 after %d retries", c.maxRetries)
}

// doLocked acquires a per-key mutex before calling do(), serializing concurrent
// requests that share the same lock key. Use the domain name as key for
// domain-scoped operations, and "" for global DNS operations (e.g. CreateDomain,
// ListDomains). This prevents bursts that would exhaust deSEC rate limit buckets.
// When serializeRequests is false the lock is skipped and do() is called directly.
func (c *Client) doLocked(ctx context.Context, method, path, key string, body any) (*http.Response, error) {
	if !c.serializeRequests {
		return c.do(ctx, method, path, body)
	}
	mu := c.getMutex(key)
	mu.Lock()
	defer mu.Unlock()
	return c.do(ctx, method, path, body)
}

// getMutex returns the mutex associated with key, creating it if necessary.
func (c *Client) getMutex(key string) *sync.Mutex {
	v, _ := c.mu.LoadOrStore(key, &sync.Mutex{})
	return v.(*sync.Mutex) //nolint:forcetypeassert
}

// checkResponse reads and parses an error body from an HTTP response.
// It returns nil if the response status code is one of the acceptable codes.
func checkResponse(resp *http.Response, acceptableCodes ...int) error {
	if slices.Contains(acceptableCodes, resp.StatusCode) {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return &APIError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}

// APIError represents an unexpected HTTP response from the deSEC API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("deSEC API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// IsNotFound returns true if the error (or any wrapped error) is a 404 Not Found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
