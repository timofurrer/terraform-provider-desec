// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// CreateDomain creates a new domain (DNS zone) in deSEC.
func (c *Client) CreateDomain(ctx context.Context, name string) (*Domain, error) {
	body := map[string]string{"name": name}
	resp, err := c.do(ctx, http.MethodPost, "/domains/", body)
	if err != nil {
		return nil, fmt.Errorf("creating domain %q: %w", name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if err := checkResponse(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating domain %q: %w", name, err)
	}

	var domain Domain
	if err := json.NewDecoder(resp.Body).Decode(&domain); err != nil {
		return nil, fmt.Errorf("decoding create domain response: %w", err)
	}
	return &domain, nil
}

// GetDomain retrieves a domain by name.
func (c *Client) GetDomain(ctx context.Context, name string) (*Domain, error) {
	resp, err := c.do(ctx, http.MethodGet, "/domains/"+name+"/", nil)
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

// ListDomains retrieves all domains. If ownsQname is non-empty, only the domain
// responsible for that DNS name is returned.
func (c *Client) ListDomains(ctx context.Context, ownsQname string) ([]Domain, error) {
	path := "/domains/"
	if ownsQname != "" {
		path += "?owns_qname=" + url.QueryEscape(ownsQname)
	}

	var allDomains []Domain
	cursor := ""
	firstPage := true

	for firstPage || cursor != "" {
		firstPage = false

		pagePath := path
		if cursor != "" {
			if ownsQname != "" {
				pagePath += "&cursor=" + url.QueryEscape(cursor)
			} else {
				pagePath += "?cursor=" + url.QueryEscape(cursor)
			}
		}

		resp, err := c.do(ctx, http.MethodGet, pagePath, nil)
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
	resp, err := c.do(ctx, http.MethodDelete, "/domains/"+name+"/", nil)
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
	resp, err := c.do(ctx, http.MethodGet, "/domains/"+name+"/zonefile/", nil)
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
