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

// rrsetSubnameForURL returns the subname to use in a URL path segment.
// The deSEC API uses "@" for the zone apex in individual endpoints.
// An empty subname or "@" is normalized to "@".
func rrsetSubnameForURL(subname string) string {
	if subname == "" || subname == "@" {
		return "@"
	}
	return subname
}

// rrsetSubnameForBody returns the subname to use in a JSON request body.
// The deSEC API expects an empty string for the zone apex in request bodies.
func rrsetSubnameForBody(subname string) string {
	if subname == "@" {
		return ""
	}
	return subname
}

// rrsetPath returns the URL path for a specific RRset.
func rrsetPath(domain, subname, rrtype string) string {
	return fmt.Sprintf("/domains/%s/rrsets/%s/%s/", domain, rrsetSubnameForURL(subname), rrtype)
}

// CreateRRset creates a new RRset within a domain.
func (c *Client) CreateRRset(ctx context.Context, domain, subname, rrtype string, ttl int, records []string) (*RRset, error) {
	body := map[string]any{
		"subname": rrsetSubnameForBody(subname),
		"type":    rrtype,
		"ttl":     ttl,
		"records": records,
	}
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/domains/%s/rrsets/", domain), body)
	if err != nil {
		return nil, fmt.Errorf("creating rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}

	var rrset RRset
	if err := json.NewDecoder(resp.Body).Decode(&rrset); err != nil {
		return nil, fmt.Errorf("decoding create rrset response: %w", err)
	}
	return &rrset, nil
}

// GetRRset retrieves a specific RRset by domain, subname, and type.
func (c *Client) GetRRset(ctx context.Context, domain, subname, rrtype string) (*RRset, error) {
	resp, err := c.do(ctx, http.MethodGet, rrsetPath(domain, subname, rrtype), nil)
	if err != nil {
		return nil, fmt.Errorf("getting rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("getting rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}

	var rrset RRset
	if err := json.NewDecoder(resp.Body).Decode(&rrset); err != nil {
		return nil, fmt.Errorf("decoding get rrset response: %w", err)
	}
	return &rrset, nil
}

// ListRRsets retrieves all RRsets in a domain, with optional subname and type filters.
func (c *Client) ListRRsets(ctx context.Context, domain, subname, rrtype string) ([]RRset, error) {
	basePath := fmt.Sprintf("/domains/%s/rrsets/", domain)

	// Build query parameters.
	params := url.Values{}
	if subname != "" {
		params.Set("subname", subname)
	}
	if rrtype != "" {
		params.Set("type", rrtype)
	}

	queryStr := ""
	if len(params) > 0 {
		queryStr = "?" + params.Encode()
	}

	var allRRsets []RRset
	cursor := ""
	firstPage := true

	for firstPage || cursor != "" {
		firstPage = false

		pagePath := basePath + queryStr
		if cursor != "" {
			if queryStr != "" {
				pagePath += "&cursor=" + url.QueryEscape(cursor)
			} else {
				pagePath += "?cursor=" + url.QueryEscape(cursor)
			}
		}

		resp, err := c.do(ctx, http.MethodGet, pagePath, nil)
		if err != nil {
			return nil, fmt.Errorf("listing rrsets for domain %q: %w", domain, err)
		}
		defer resp.Body.Close() //nolint:errcheck

		if err := checkResponse(resp, http.StatusOK); err != nil {
			return nil, fmt.Errorf("listing rrsets for domain %q: %w", domain, err)
		}

		var rrsets []RRset
		if err := json.NewDecoder(resp.Body).Decode(&rrsets); err != nil {
			return nil, fmt.Errorf("decoding list rrsets response: %w", err)
		}
		allRRsets = append(allRRsets, rrsets...)

		cursor = parseCursorNext(resp.Header.Get("Link"))
	}

	return allRRsets, nil
}

// UpdateRRset updates the TTL and records of an existing RRset.
func (c *Client) UpdateRRset(ctx context.Context, domain, subname, rrtype string, ttl int, records []string) (*RRset, error) {
	body := map[string]any{
		"ttl":     ttl,
		"records": records,
	}
	resp, err := c.do(ctx, http.MethodPatch, rrsetPath(domain, subname, rrtype), body)
	if err != nil {
		return nil, fmt.Errorf("updating rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("updating rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}

	var rrset RRset
	if err := json.NewDecoder(resp.Body).Decode(&rrset); err != nil {
		return nil, fmt.Errorf("decoding update rrset response: %w", err)
	}
	return &rrset, nil
}

// DeleteRRset deletes an RRset by domain, subname, and type.
func (c *Client) DeleteRRset(ctx context.Context, domain, subname, rrtype string) error {
	resp, err := c.do(ctx, http.MethodDelete, rrsetPath(domain, subname, rrtype), nil)
	if err != nil {
		return fmt.Errorf("deleting rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("deleting rrset %s/%s/%s: %w", domain, subname, rrtype, err)
	}
	return nil
}
