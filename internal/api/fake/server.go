// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

// Package fake provides an in-memory fake deSEC API server for use in tests.
package fake

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	testToken  = "test-token"
	pageSize   = 500
	minimumTTL = 3600
)

// domain is the in-memory representation of a domain.
type domain struct {
	Created    string `json:"created"`
	Keys       []key  `json:"keys,omitempty"`
	MinimumTTL int    `json:"minimum_ttl"`
	Name       string `json:"name"`
	Published  string `json:"published"`
	Touched    string `json:"touched"`
}

type key struct {
	DNSKey  string   `json:"dnskey"`
	DS      []string `json:"ds"`
	Managed bool     `json:"managed"`
}

// rrset is the in-memory representation of a resource record set.
type rrset struct {
	Created string   `json:"created"`
	Domain  string   `json:"domain"`
	Subname string   `json:"subname"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Records []string `json:"records"`
	TTL     int      `json:"ttl"`
	Touched string   `json:"touched"`
}

// Server is a fake deSEC API server backed by in-memory state.
type Server struct {
	mu      sync.RWMutex
	domains map[string]*domain           // key: domain name
	rrsets  map[string]map[string]*rrset // key: domain name -> "subname/type"
	srv     *httptest.Server
}

// NewServer creates and starts a new fake deSEC API server.
// The returned Server must be closed with Close() when done.
func NewServer() *Server {
	s := &Server{
		domains: make(map[string]*domain),
		rrsets:  make(map[string]map[string]*rrset),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/domains/", s.handleDomains)
	mux.HandleFunc("/api/v1/domains", s.handleDomains)

	s.srv = httptest.NewServer(mux)
	return s
}

// URL returns the base URL of the fake server (e.g., "http://127.0.0.1:PORT/api/v1").
func (s *Server) URL() string {
	return s.srv.URL + "/api/v1"
}

// Token returns the authentication token expected by the fake server.
func (s *Server) Token() string {
	return testToken
}

// Close shuts down the fake server.
func (s *Server) Close() {
	s.srv.Close()
}

// authenticate checks the Authorization header and writes 401 if invalid.
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "Token "+testToken {
		http.Error(w, `{"detail":"Invalid token."}`, http.StatusUnauthorized)
		return false
	}
	return true
}

// writeJSON writes a JSON-encoded value with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// now returns the current time in ISO 8601 format.
func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// handleDomains routes domain-related requests.
func (s *Server) handleDomains(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(w, r) {
		return
	}

	// Strip /api/v1 prefix and split path.
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	// path is now like /domains/, /domains/{name}/, /domains/{name}/rrsets/, etc.

	parts := strings.Split(strings.Trim(path, "/"), "/")
	// parts[0] = "domains"
	// parts[1] = domain name (optional)
	// parts[2] = "rrsets" (optional)
	// parts[3] = subname (optional)
	// parts[4] = type (optional)

	switch {
	case len(parts) == 1:
		// /domains/
		switch r.Method {
		case http.MethodGet:
			s.listDomains(w, r)
		case http.MethodPost:
			s.createDomain(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	case len(parts) == 2:
		// /domains/{name}/
		domainName := parts[1]
		switch r.Method {
		case http.MethodGet:
			s.getDomain(w, r, domainName)
		case http.MethodDelete:
			s.deleteDomain(w, r, domainName)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	case len(parts) == 3 && parts[2] == "rrsets":
		// /domains/{name}/rrsets/
		domainName := parts[1]
		switch r.Method {
		case http.MethodGet:
			s.listRRsets(w, r, domainName)
		case http.MethodPost:
			s.createRRset(w, r, domainName)
		case http.MethodPatch, http.MethodPut:
			s.bulkUpdateRRsets(w, r, domainName)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	case len(parts) == 5 && parts[2] == "rrsets":
		// /domains/{name}/rrsets/{subname}/{type}/
		domainName := parts[1]
		subname := parts[3]
		rrtype := parts[4]
		// Normalize "@" to empty string for storage.
		if subname == "@" {
			subname = ""
		}
		switch r.Method {
		case http.MethodGet:
			s.getRRset(w, r, domainName, subname, rrtype)
		case http.MethodPatch, http.MethodPut:
			s.updateRRset(w, r, domainName, subname, rrtype)
		case http.MethodDelete:
			s.deleteRRset(w, r, domainName, subname, rrtype)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	case len(parts) == 3 && parts[2] == "zonefile":
		// /domains/{name}/zonefile/
		domainName := parts[1]
		switch r.Method {
		case http.MethodGet:
			s.getZonefile(w, r, domainName)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	default:
		http.NotFound(w, r)
	}
}

// ---- Domain handlers ----

func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ownsQname := r.URL.Query().Get("owns_qname")

	// Collect and sort domains by creation time (reverse chronological).
	all := make([]*domain, 0, len(s.domains))
	for _, d := range s.domains {
		all = append(all, d)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Created > all[j].Created
	})

	// Filter by owns_qname if provided.
	if ownsQname != "" {
		var filtered []*domain
		for _, d := range all {
			if domainOwnsQname(d.Name, ownsQname) {
				filtered = append(filtered, d)
				break // at most one domain can be responsible
			}
		}
		all = filtered
	}

	// Strip keys (not returned in list endpoint).
	result := make([]domain, 0, len(all))
	for _, d := range all {
		stripped := *d
		stripped.Keys = nil
		result = append(result, stripped)
	}

	// Pagination.
	cursor := r.URL.Query().Get("cursor")
	page, nextCursor := paginate(result, cursor)

	if nextCursor != "" {
		linkURL := *r.URL
		q := linkURL.Query()
		q.Set("cursor", nextCursor)
		linkURL.RawQuery = q.Encode()
		w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, s.srv.URL+linkURL.String()))
	}

	writeJSON(w, http.StatusOK, page)
}

func (s *Server) createDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"name":["This field is required."]}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.domains[req.Name]; exists {
		http.Error(w, `{"name":["This field must be unique."]}`, http.StatusBadRequest)
		return
	}

	ts := now()
	d := &domain{
		Created:    ts,
		MinimumTTL: minimumTTL,
		Name:       req.Name,
		Published:  ts,
		Touched:    ts,
		Keys: []key{
			{
				DNSKey:  "257 3 13 FakeKey==",
				DS:      []string{"12345 13 2 fakeds256hash", "12345 13 4 fakeds384hash"},
				Managed: true,
			},
		},
	}
	s.domains[req.Name] = d
	s.rrsets[req.Name] = make(map[string]*rrset)

	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) getDomain(w http.ResponseWriter, _ *http.Request, name string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.domains[name]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) deleteDomain(w http.ResponseWriter, _ *http.Request, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.domains, name)
	delete(s.rrsets, name)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getZonefile(w http.ResponseWriter, _ *http.Request, domainName string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	var sb strings.Builder
	for _, rs := range s.rrsets[domainName] {
		fmt.Fprintf(&sb, "%s\t%d\tIN\t%s\t%s\n", rs.Name, rs.TTL, rs.Type, strings.Join(rs.Records, " "))
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sb.String())) //nolint:errcheck
}

// ---- RRset handlers ----

func (s *Server) listRRsets(w http.ResponseWriter, r *http.Request, domainName string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	filterSubname := r.URL.Query().Get("subname")
	filterType := r.URL.Query().Get("type")
	hasSubnameFilter := r.URL.Query().Has("subname")

	all := make([]*rrset, 0)
	for _, rs := range s.rrsets[domainName] {
		if filterType != "" && rs.Type != filterType {
			continue
		}
		if hasSubnameFilter && rs.Subname != filterSubname {
			continue
		}
		all = append(all, rs)
	}

	// Sort for deterministic output.
	sort.Slice(all, func(i, j int) bool {
		if all[i].Subname != all[j].Subname {
			return all[i].Subname < all[j].Subname
		}
		return all[i].Type < all[j].Type
	})

	result := make([]rrset, 0, len(all))
	for _, rs := range all {
		result = append(result, *rs)
	}

	cursor := r.URL.Query().Get("cursor")
	page, nextCursor := paginate(result, cursor)

	if nextCursor != "" {
		linkURL := *r.URL
		q := linkURL.Query()
		q.Set("cursor", nextCursor)
		linkURL.RawQuery = q.Encode()
		w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, s.srv.URL+linkURL.String()))
	}

	writeJSON(w, http.StatusOK, page)
}

func (s *Server) createRRset(w http.ResponseWriter, r *http.Request, domainName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	var req struct {
		Subname string   `json:"subname"`
		Type    string   `json:"type"`
		TTL     int      `json:"ttl"`
		Records []string `json:"records"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"detail":"Parse error."}`, http.StatusBadRequest)
		return
	}

	if req.Type == "" || req.TTL == 0 || req.Records == nil {
		http.Error(w, `{"detail":"Missing required fields."}`, http.StatusBadRequest)
		return
	}

	rrKey := rrsetKey(req.Subname, req.Type)
	if _, exists := s.rrsets[domainName][rrKey]; exists {
		http.Error(w, `{"detail":"RRset with this domain, subname and type already exists."}`, http.StatusBadRequest)
		return
	}

	ts := now()
	rs := &rrset{
		Created: ts,
		Domain:  domainName,
		Subname: req.Subname,
		Name:    rrsetName(req.Subname, domainName),
		Type:    req.Type,
		Records: req.Records,
		TTL:     req.TTL,
		Touched: ts,
	}
	s.rrsets[domainName][rrKey] = rs

	writeJSON(w, http.StatusCreated, rs)
}

func (s *Server) getRRset(w http.ResponseWriter, _ *http.Request, domainName, subname, rrtype string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	rs, ok := s.rrsets[domainName][rrsetKey(subname, rrtype)]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, rs)
}

func (s *Server) updateRRset(w http.ResponseWriter, r *http.Request, domainName, subname, rrtype string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	rs, ok := s.rrsets[domainName][rrsetKey(subname, rrtype)]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	var req struct {
		TTL     *int     `json:"ttl"`
		Records []string `json:"records"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"detail":"Parse error."}`, http.StatusBadRequest)
		return
	}

	if req.TTL != nil {
		rs.TTL = *req.TTL
	}
	if req.Records != nil {
		if len(req.Records) == 0 {
			// Deleting via empty records.
			delete(s.rrsets[domainName], rrsetKey(subname, rrtype))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		rs.Records = req.Records
	}
	rs.Touched = now()

	writeJSON(w, http.StatusOK, rs)
}

func (s *Server) deleteRRset(w http.ResponseWriter, _ *http.Request, domainName, subname, rrtype string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	delete(s.rrsets[domainName], rrsetKey(subname, rrtype))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) bulkUpdateRRsets(w http.ResponseWriter, r *http.Request, domainName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	var reqs []struct {
		Subname string   `json:"subname"`
		Type    string   `json:"type"`
		TTL     *int     `json:"ttl"`
		Records []string `json:"records"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		http.Error(w, `{"detail":"Parse error."}`, http.StatusBadRequest)
		return
	}

	ts := now()
	results := make([]rrset, 0, len(reqs))

	for _, req := range reqs {
		rrKey := rrsetKey(req.Subname, req.Type)
		existing, exists := s.rrsets[domainName][rrKey]

		if req.Records != nil && len(req.Records) == 0 {
			// Delete.
			delete(s.rrsets[domainName], rrKey)
			continue
		}

		if exists {
			if req.TTL != nil {
				existing.TTL = *req.TTL
			}
			if req.Records != nil {
				existing.Records = req.Records
			}
			existing.Touched = ts
			results = append(results, *existing)
		} else {
			if req.Type == "" || req.Records == nil {
				continue
			}
			ttl := 3600
			if req.TTL != nil {
				ttl = *req.TTL
			}
			rs := &rrset{
				Created: ts,
				Domain:  domainName,
				Subname: req.Subname,
				Name:    rrsetName(req.Subname, domainName),
				Type:    req.Type,
				Records: req.Records,
				TTL:     ttl,
				Touched: ts,
			}
			s.rrsets[domainName][rrKey] = rs
			results = append(results, *rs)
		}
	}

	writeJSON(w, http.StatusOK, results)
}

// ---- helpers ----

// rrsetKey returns the map key for an rrset.
func rrsetKey(subname, rrtype string) string {
	return subname + "/" + rrtype
}

// rrsetName returns the full DNS name for an rrset.
func rrsetName(subname, domainName string) string {
	if subname == "" {
		return domainName + "."
	}
	return subname + "." + domainName + "."
}

// domainOwnsQname returns true if the domain is the authoritative zone for qname.
func domainOwnsQname(domainName, qname string) bool {
	qname = strings.TrimSuffix(qname, ".")
	domainName = strings.TrimSuffix(domainName, ".")
	if qname == domainName {
		return true
	}
	return strings.HasSuffix(qname, "."+domainName)
}

// paginate returns a page of items starting from the cursor, and the next cursor.
// Items must be a slice type. This implementation uses a simple integer index cursor.
func paginate[T any](items []T, cursor string) ([]T, string) {
	start := 0
	if cursor != "" {
		decoded, err := url.QueryUnescape(cursor)
		if err == nil {
			var idx int
			if _, err := fmt.Sscanf(decoded, "%d", &idx); err == nil && idx >= 0 && idx < len(items) {
				start = idx
			}
		}
	}

	end := start + pageSize
	if end >= len(items) {
		return items[start:], ""
	}

	nextCursor := fmt.Sprintf("%d", end)
	return items[start:end], nextCursor
}
