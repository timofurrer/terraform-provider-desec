// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/go-querystring/query"
)

// RRset represents a DNS Resource Record Set within a domain.
type RRset struct {
	Created string   `json:"created"`
	Domain  string   `json:"domain"`
	Subname string   `json:"subname"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Records []string `json:"records"`
	TTL     int      `json:"ttl"`
	Touched string   `json:"touched"`
}

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

// CreateRRsetOptions are the body parameters for CreateRRset.
type CreateRRsetOptions struct {
	Subname string   `json:"subname"`
	Type    string   `json:"type"`
	TTL     int      `json:"ttl"`
	Records []string `json:"records"`
}

// CreateRRset creates a new RRset within a domain.
func (c *Client) CreateRRset(ctx context.Context, domain string, opts CreateRRsetOptions) (*RRset, error) {
	// Normalise subname for the JSON body (apex must be an empty string).
	opts.Subname = rrsetSubnameForBody(opts.Subname)
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/domains/%s/rrsets/", domain), opts)
	if err != nil {
		return nil, fmt.Errorf("creating rrset %s/%s/%s: %w", domain, opts.Subname, opts.Type, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating rrset %s/%s/%s: %w", domain, opts.Subname, opts.Type, err)
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

// ListRRsetsOptions are the optional filter parameters for ListRRsets.
type ListRRsetsOptions struct {
	// Subname, when non-nil, filters results to RRsets with this subname.
	Subname *string `url:"subname,omitempty"`
	// Type, when non-nil, filters results to RRsets of this DNS type.
	Type *string `url:"type,omitempty"`
}

// ListRRsets retrieves all RRsets in a domain, with optional subname and type filters.
func (c *Client) ListRRsets(ctx context.Context, domain string, opts ListRRsetsOptions) ([]RRset, error) {
	basePath := fmt.Sprintf("/domains/%s/rrsets/", domain)

	params, err := query.Values(opts)
	if err != nil {
		return nil, fmt.Errorf("encoding list rrsets query params: %w", err)
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

// UpdateRRsetOptions are the body parameters for UpdateRRset.
type UpdateRRsetOptions struct {
	TTL     int      `json:"ttl"`
	Records []string `json:"records"`
}

// UpdateRRset updates the TTL and records of an existing RRset.
func (c *Client) UpdateRRset(ctx context.Context, domain, subname, rrtype string, opts UpdateRRsetOptions) (*RRset, error) {
	resp, err := c.do(ctx, http.MethodPatch, rrsetPath(domain, subname, rrtype), opts)
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
