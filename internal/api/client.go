// Copyright (c) Timo Furrer
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
	"time"
)

const (
	// DefaultBaseURL is the default deSEC API base URL.
	DefaultBaseURL = "https://desec.io/api/v1"

	// defaultMaxRetries is the maximum number of times a 429 response is retried.
	defaultMaxRetries = 5
)

// Client is a deSEC REST API client.
type Client struct {
	BaseURL    string
	Token      string
	httpClient *http.Client
	maxRetries int
}

// NewClient creates a new deSEC API client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: defaultMaxRetries,
	}
}

// do executes an HTTP request with automatic rate-limit retry.
// On HTTP 429, it reads the Retry-After header and sleeps before retrying.
func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	for range c.maxRetries {
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

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// HTTP 429: parse Retry-After and sleep before retrying.
		_ = resp.Body.Close()

		waitSeconds := 1
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, parseErr := strconv.Atoi(ra); parseErr == nil && secs > 0 {
				waitSeconds = secs
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(waitSeconds) * time.Second):
		}
	}

	return nil, fmt.Errorf("request rate-limited: still receiving HTTP 429 after %d retries", c.maxRetries)
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
