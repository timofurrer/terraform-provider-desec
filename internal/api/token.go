// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// tokenRequest is the JSON body for creating or updating a token.
type tokenRequest struct {
	Name             string   `json:"name"`
	PermCreateDomain bool     `json:"perm_create_domain"`
	PermDeleteDomain bool     `json:"perm_delete_domain"`
	PermManageTokens bool     `json:"perm_manage_tokens"`
	AllowedSubnets   []string `json:"allowed_subnets"`
	AutoPolicy       bool     `json:"auto_policy"`
	MaxAge           *string  `json:"max_age"`
	MaxUnusedPeriod  *string  `json:"max_unused_period"`
}

// CreateToken creates a new API token with the given configuration.
// The returned Token includes the secret value in the Secret field,
// which is only available at creation time.
func (c *Client) CreateToken(
	ctx context.Context,
	name string,
	permCreateDomain, permDeleteDomain, permManageTokens bool,
	allowedSubnets []string,
	autoPolicy bool,
	maxAge, maxUnusedPeriod *string,
) (*Token, error) {
	body := tokenRequest{
		Name:             name,
		PermCreateDomain: permCreateDomain,
		PermDeleteDomain: permDeleteDomain,
		PermManageTokens: permManageTokens,
		AllowedSubnets:   allowedSubnets,
		AutoPolicy:       autoPolicy,
		MaxAge:           maxAge,
		MaxUnusedPeriod:  maxUnusedPeriod,
	}

	resp, err := c.do(ctx, http.MethodPost, "/auth/tokens/", body)
	if err != nil {
		return nil, fmt.Errorf("creating token: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating token: %w", err)
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding create token response: %w", err)
	}
	return &token, nil
}

// GetToken retrieves a token by its ID.
// Note: the response does not include the token's secret value.
func (c *Client) GetToken(ctx context.Context, id string) (*Token, error) {
	resp, err := c.do(ctx, http.MethodGet, "/auth/tokens/"+url.PathEscape(id)+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("getting token %q: %w", id, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("getting token %q: %w", id, err)
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding get token response: %w", err)
	}
	return &token, nil
}

// ListTokens retrieves all tokens for the authenticated account.
func (c *Client) ListTokens(ctx context.Context) ([]Token, error) {
	var allTokens []Token
	cursor := ""
	firstPage := true

	for firstPage || cursor != "" {
		firstPage = false

		path := "/auth/tokens/"
		if cursor != "" {
			path += "?cursor=" + url.QueryEscape(cursor)
		}

		resp, err := c.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("listing tokens: %w", err)
		}
		defer resp.Body.Close() //nolint:errcheck

		if err := checkResponse(resp, http.StatusOK); err != nil {
			return nil, fmt.Errorf("listing tokens: %w", err)
		}

		var tokens []Token
		if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
			return nil, fmt.Errorf("decoding list tokens response: %w", err)
		}
		allTokens = append(allTokens, tokens...)

		cursor = parseCursorNext(resp.Header.Get("Link"))
	}

	return allTokens, nil
}

// UpdateToken modifies an existing token by its ID.
// Note: the response does not include the token's secret value.
func (c *Client) UpdateToken(
	ctx context.Context,
	id string,
	name string,
	permCreateDomain, permDeleteDomain, permManageTokens bool,
	allowedSubnets []string,
	autoPolicy bool,
	maxAge, maxUnusedPeriod *string,
) (*Token, error) {
	body := tokenRequest{
		Name:             name,
		PermCreateDomain: permCreateDomain,
		PermDeleteDomain: permDeleteDomain,
		PermManageTokens: permManageTokens,
		AllowedSubnets:   allowedSubnets,
		AutoPolicy:       autoPolicy,
		MaxAge:           maxAge,
		MaxUnusedPeriod:  maxUnusedPeriod,
	}

	resp, err := c.do(ctx, http.MethodPatch, "/auth/tokens/"+url.PathEscape(id)+"/", body)
	if err != nil {
		return nil, fmt.Errorf("updating token %q: %w", id, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("updating token %q: %w", id, err)
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding update token response: %w", err)
	}
	return &token, nil
}

// DeleteToken deletes a token by its ID.
// Returns nil even if the token was not found (the API returns 204 in both cases).
func (c *Client) DeleteToken(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/auth/tokens/"+url.PathEscape(id)+"/", nil)
	if err != nil {
		return fmt.Errorf("deleting token %q: %w", id, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("deleting token %q: %w", id, err)
	}
	return nil
}
