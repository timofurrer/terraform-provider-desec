// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/go-querystring/query"
)

// Key represents a DNSSEC public key associated with a domain.
type Key struct {
	DNSKey  string   `json:"dnskey"`
	DS      []string `json:"ds"`
	Managed bool     `json:"managed"`
}

// Domain represents a DNS zone managed by deSEC.
type Domain struct {
	Created    string `json:"created"`
	Keys       []Key  `json:"keys,omitempty"`
	MinimumTTL int    `json:"minimum_ttl"`
	Name       string `json:"name"`
	Published  string `json:"published"`
	Touched    string `json:"touched"`
}

// CreateDomainOptions are the body parameters for CreateDomain.
type CreateDomainOptions struct {
	Name string `json:"name"`
}

// CreateDomain creates a new domain (DNS zone) in deSEC.
func (c *Client) CreateDomain(ctx context.Context, opts CreateDomainOptions) (*Domain, error) {
	resp, err := c.doLocked(ctx, http.MethodPost, "/domains/", "", opts)
	if err != nil {
		return nil, fmt.Errorf("creating domain %q: %w", opts.Name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating domain %q: %w", opts.Name, err)
	}

	var domain Domain
	if err := json.NewDecoder(resp.Body).Decode(&domain); err != nil {
		return nil, fmt.Errorf("decoding create domain response: %w", err)
	}
	return &domain, nil
}

// GetDomain retrieves a domain by name.
func (c *Client) GetDomain(ctx context.Context, name string) (*Domain, error) {
	resp, err := c.doLocked(ctx, http.MethodGet, "/domains/"+name+"/", name, nil)
	if err != nil {
		return nil, fmt.Errorf("getting domain %q: %w", name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("getting domain %q: %w", name, err)
	}

	var domain Domain
	if err := json.NewDecoder(resp.Body).Decode(&domain); err != nil {
		return nil, fmt.Errorf("decoding get domain response: %w", err)
	}
	return &domain, nil
}

// ListDomainsOptions are the optional parameters for ListDomains.
type ListDomainsOptions struct {
	// OwnsQname, when non-nil, filters the result to the single domain
	// responsible for that DNS query name.
	OwnsQname *string `url:"owns_qname,omitempty"`
}

// ListDomains retrieves all domains. If opts.OwnsQname is set, only the domain
// responsible for that DNS name is returned.
func (c *Client) ListDomains(ctx context.Context, opts ListDomainsOptions) ([]Domain, error) {
	params, err := query.Values(opts)
	if err != nil {
		return nil, fmt.Errorf("encoding list domains query params: %w", err)
	}

	basePath := "/domains/"
	queryStr := ""
	if len(params) > 0 {
		queryStr = "?" + params.Encode()
	}

	var allDomains []Domain
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

		resp, err := c.doLocked(ctx, http.MethodGet, pagePath, "", nil)
		if err != nil {
			return nil, fmt.Errorf("listing domains: %w", err)
		}
		defer resp.Body.Close() //nolint:errcheck

		if err := checkResponse(resp, http.StatusOK); err != nil {
			return nil, fmt.Errorf("listing domains: %w", err)
		}

		var domains []Domain
		if err := json.NewDecoder(resp.Body).Decode(&domains); err != nil {
			return nil, fmt.Errorf("decoding list domains response: %w", err)
		}
		allDomains = append(allDomains, domains...)

		cursor = parseCursorNext(resp.Header.Get("Link"))
	}

	return allDomains, nil
}

// DeleteDomain deletes a domain by name.
func (c *Client) DeleteDomain(ctx context.Context, name string) error {
	resp, err := c.doLocked(ctx, http.MethodDelete, "/domains/"+name+"/", name, nil)
	if err != nil {
		return fmt.Errorf("deleting domain %q: %w", name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("deleting domain %q: %w", name, err)
	}
	return nil
}

// GetZonefile retrieves the zonefile for a domain.
func (c *Client) GetZonefile(ctx context.Context, name string) (string, error) {
	resp, err := c.doLocked(ctx, http.MethodGet, "/domains/"+name+"/zonefile/", name, nil)
	if err != nil {
		return "", fmt.Errorf("getting zonefile for %q: %w", name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusOK); err != nil {
		return "", fmt.Errorf("getting zonefile for %q: %w", name, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading zonefile: %w", err)
	}
	return string(body), nil
}
