// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TokenPolicy represents a scoping policy for a deSEC API token.
// The combination of (Domain, Subname, Type) identifies the RRset scope;
// all three being nil denotes the default (catch-all) policy.
type TokenPolicy struct {
	ID        string  `json:"id"`
	Domain    *string `json:"domain"`
	Subname   *string `json:"subname"`
	Type      *string `json:"type"`
	PermWrite bool    `json:"perm_write"`
}

// CreateTokenPolicyOptions are the body parameters for CreateTokenPolicy.
// Domain, Subname and Type are sent as explicit JSON null when the pointer is nil,
// which is what the deSEC API requires for the default (catch-all) policy.
type CreateTokenPolicyOptions struct {
	Domain    *string `json:"domain"`
	Subname   *string `json:"subname"`
	Type      *string `json:"type"`
	PermWrite bool    `json:"perm_write"`
}

// UpdateTokenPolicyOptions are the body parameters for UpdateTokenPolicy.
// Only PermWrite is mutable after creation.
type UpdateTokenPolicyOptions struct {
	PermWrite bool `json:"perm_write"`
}

// tokenPoliciesPath returns the base path for a token's policy collection.
func tokenPoliciesPath(tokenID string) string {
	return "/auth/tokens/" + url.PathEscape(tokenID) + "/policies/rrsets/"
}

// tokenPolicyPath returns the path for a specific token policy.
func tokenPolicyPath(tokenID, policyID string) string {
	return tokenPoliciesPath(tokenID) + url.PathEscape(policyID) + "/"
}

// CreateTokenPolicy creates a new scoping policy for the given token.
// Pass nil for Domain, Subname, and/or Type to create the default policy.
func (c *Client) CreateTokenPolicy(ctx context.Context, tokenID string, opts CreateTokenPolicyOptions) (*TokenPolicy, error) {
	resp, err := c.do(ctx, http.MethodPost, tokenPoliciesPath(tokenID), opts)
	if err != nil {
		return nil, fmt.Errorf("creating token policy for token %q: %w", tokenID, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating token policy for token %q: %w", tokenID, err)
	}

	var policy TokenPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("decoding create token policy response: %w", err)
	}
	return &policy, nil
}

// GetTokenPolicy retrieves a specific policy for the given token.
func (c *Client) GetTokenPolicy(ctx context.Context, tokenID, policyID string) (*TokenPolicy, error) {
	resp, err := c.do(ctx, http.MethodGet, tokenPolicyPath(tokenID, policyID), nil)
	if err != nil {
		return nil, fmt.Errorf("getting token policy %q for token %q: %w", policyID, tokenID, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("getting token policy %q for token %q: %w", policyID, tokenID, err)
	}

	var policy TokenPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("decoding get token policy response: %w", err)
	}
	return &policy, nil
}

// ListTokenPolicies retrieves all scoping policies for the given token.
func (c *Client) ListTokenPolicies(ctx context.Context, tokenID string) ([]TokenPolicy, error) {
	var allPolicies []TokenPolicy
	cursor := ""
	firstPage := true

	for firstPage || cursor != "" {
		firstPage = false

		path := tokenPoliciesPath(tokenID)
		if cursor != "" {
			path += "?cursor=" + url.QueryEscape(cursor)
		}

		resp, err := c.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("listing token policies for token %q: %w", tokenID, err)
		}
		defer resp.Body.Close() //nolint:errcheck

		if err := checkResponse(resp, http.StatusOK); err != nil {
			return nil, fmt.Errorf("listing token policies for token %q: %w", tokenID, err)
		}

		var policies []TokenPolicy
		if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
			return nil, fmt.Errorf("decoding list token policies response: %w", err)
		}
		allPolicies = append(allPolicies, policies...)

		cursor = parseCursorNext(resp.Header.Get("Link"))
	}

	return allPolicies, nil
}

// UpdateTokenPolicy modifies the perm_write flag of an existing token policy.
// The domain, subname, and type fields are immutable after creation.
func (c *Client) UpdateTokenPolicy(ctx context.Context, tokenID, policyID string, opts UpdateTokenPolicyOptions) (*TokenPolicy, error) {
	resp, err := c.do(ctx, http.MethodPatch, tokenPolicyPath(tokenID, policyID), opts)
	if err != nil {
		return nil, fmt.Errorf("updating token policy %q for token %q: %w", policyID, tokenID, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("updating token policy %q for token %q: %w", policyID, tokenID, err)
	}

	var policy TokenPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("decoding update token policy response: %w", err)
	}
	return &policy, nil
}

// DeleteTokenPolicy deletes a token policy by its ID.
func (c *Client) DeleteTokenPolicy(ctx context.Context, tokenID, policyID string) error {
	resp, err := c.do(ctx, http.MethodDelete, tokenPolicyPath(tokenID, policyID), nil)
	if err != nil {
		return fmt.Errorf("deleting token policy %q for token %q: %w", policyID, tokenID, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("deleting token policy %q for token %q: %w", policyID, tokenID, err)
	}
	return nil
}
