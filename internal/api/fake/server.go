// Copyright Timo Furrer 2026
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

	"github.com/google/uuid"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

const (
	testToken  = "test-token"
	pageSize   = 500
	minimumTTL = 3600
)

// isDefaultPolicy returns true when all scope fields are nil (the catch-all policy).
func isDefaultPolicy(p *api.TokenPolicy) bool {
	return p.Domain == nil && p.Subname == nil && p.Type == nil
}

// Server is a fake deSEC API server backed by in-memory state.
type Server struct {
	mu       sync.RWMutex
	domains  map[string]*api.Domain                 // key: domain name
	rrsets   map[string]map[string]*api.RRset       // key: domain name -> "subname/type"
	tokens   map[string]*api.Token                  // key: token ID
	secrets  map[string]string                      // key: token ID -> raw secret value
	policies map[string]map[string]*api.TokenPolicy // key: token ID -> policy ID
	srv      *httptest.Server
}

// NewServer creates and starts a new fake deSEC API server.
// The returned Server must be closed with Close() when done.
func NewServer() *Server {
	s := &Server{
		domains:  make(map[string]*api.Domain),
		rrsets:   make(map[string]map[string]*api.RRset),
		tokens:   make(map[string]*api.Token),
		secrets:  make(map[string]string),
		policies: make(map[string]map[string]*api.TokenPolicy),
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/domains/", s.requireAuthentication(http.HandlerFunc(s.listDomains)))
	mux.Handle("POST /api/v1/domains/", s.requireAuthentication(http.HandlerFunc(s.createDomain)))
	mux.Handle("GET /api/v1/domains/{name}/", s.requireAuthentication(http.HandlerFunc(s.getDomain)))
	mux.Handle("DELETE /api/v1/domains/{name}/", s.requireAuthentication(http.HandlerFunc(s.deleteDomain)))
	mux.Handle("GET /api/v1/domains/{name}/zonefile/", s.requireAuthentication(http.HandlerFunc(s.getZonefile)))
	mux.Handle("GET /api/v1/domains/{name}/rrsets/", s.requireAuthentication(http.HandlerFunc(s.listRRsets)))
	mux.Handle("POST /api/v1/domains/{name}/rrsets/", s.requireAuthentication(http.HandlerFunc(s.createRRset)))
	mux.Handle("PATCH /api/v1/domains/{name}/rrsets/", s.requireAuthentication(http.HandlerFunc(s.bulkUpdateRRsets)))
	mux.Handle("PUT /api/v1/domains/{name}/rrsets/", s.requireAuthentication(http.HandlerFunc(s.bulkUpdateRRsets)))
	mux.Handle("GET /api/v1/domains/{name}/rrsets/{subname}/{type}/", s.requireAuthentication(http.HandlerFunc(s.getRRset)))
	mux.Handle("PATCH /api/v1/domains/{name}/rrsets/{subname}/{type}/", s.requireAuthentication(http.HandlerFunc(s.updateRRset)))
	mux.Handle("PUT /api/v1/domains/{name}/rrsets/{subname}/{type}/", s.requireAuthentication(http.HandlerFunc(s.updateRRset)))
	mux.Handle("DELETE /api/v1/domains/{name}/rrsets/{subname}/{type}/", s.requireAuthentication(http.HandlerFunc(s.deleteRRset)))
	mux.Handle("GET /api/v1/auth/tokens/", s.requireAuthentication(http.HandlerFunc(s.listTokens)))
	mux.Handle("POST /api/v1/auth/tokens/", s.requireAuthentication(http.HandlerFunc(s.createToken)))
	mux.Handle("GET /api/v1/auth/tokens/{id}/", s.requireAuthentication(http.HandlerFunc(s.getToken)))
	mux.Handle("PATCH /api/v1/auth/tokens/{id}/", s.requireAuthentication(http.HandlerFunc(s.updateToken)))
	mux.Handle("DELETE /api/v1/auth/tokens/{id}/", s.requireAuthentication(http.HandlerFunc(s.deleteToken)))
	mux.Handle("GET /api/v1/auth/tokens/{id}/policies/rrsets/", s.requireAuthentication(http.HandlerFunc(s.listTokenPolicies)))
	mux.Handle("POST /api/v1/auth/tokens/{id}/policies/rrsets/", s.requireAuthentication(http.HandlerFunc(s.createTokenPolicy)))
	mux.Handle("GET /api/v1/auth/tokens/{id}/policies/rrsets/{policy_id}/", s.requireAuthentication(http.HandlerFunc(s.getTokenPolicy)))
	mux.Handle("PATCH /api/v1/auth/tokens/{id}/policies/rrsets/{policy_id}/", s.requireAuthentication(http.HandlerFunc(s.updateTokenPolicy)))
	mux.Handle("DELETE /api/v1/auth/tokens/{id}/policies/rrsets/{policy_id}/", s.requireAuthentication(http.HandlerFunc(s.deleteTokenPolicy)))

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
// It accepts the hardcoded test token as well as any token dynamically created
// via the API (i.e. tokens stored in the in-memory tokens map).
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	bearer, ok := strings.CutPrefix(authHeader, "Token ")
	if !ok {
		http.Error(w, `{"detail":"Invalid token."}`, http.StatusUnauthorized)
		return false
	}
	if bearer == testToken {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, secret := range s.secrets {
		if secret == bearer {
			return true
		}
	}
	http.Error(w, `{"detail":"Invalid token."}`, http.StatusUnauthorized)
	return false
}

// requireAuthentication wraps a handlerFunc to enforce token authentication.
func (s *Server) requireAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticate(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	}
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

// ---- Domain handlers ----

func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ownsQname := r.URL.Query().Get("owns_qname")

	// Collect and sort domains by creation time (reverse chronological).
	all := make([]*api.Domain, 0, len(s.domains))
	for _, d := range s.domains {
		all = append(all, d)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Created > all[j].Created
	})

	// Filter by owns_qname if provided.
	if ownsQname != "" {
		var filtered []*api.Domain
		for _, d := range all {
			if domainOwnsQname(d.Name, ownsQname) {
				filtered = append(filtered, d)
				break // at most one domain can be responsible
			}
		}
		all = filtered
	}

	// Strip keys (not returned in list endpoint).
	result := make([]api.Domain, 0, len(all))
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
	var req api.CreateDomainOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"name":["This field is required."]}`, http.StatusBadRequest)
		return
	}

	// Reject domain names with non-ASCII characters, mirroring real deSEC
	// behaviour: IDN domains must be supplied in Punycode form.
	for label := range strings.SplitSeq(strings.TrimSuffix(req.Name, "."), ".") {
		for i := 0; i < len(label); i++ {
			if label[i] > 127 {
				http.Error(w, `{"name":["Enter a valid value."]}`, http.StatusBadRequest)
				return
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.domains[req.Name]; exists {
		http.Error(w, `{"name":["This field must be unique."]}`, http.StatusBadRequest)
		return
	}

	ts := now()
	d := &api.Domain{
		Created:    ts,
		MinimumTTL: minimumTTL,
		Name:       req.Name,
		Published:  ts,
		Touched:    ts,
		Keys: []api.Key{
			{
				DNSKey:  "257 3 13 bm90YXJlYWxrZXk=",
				DS:      []string{"12345 13 2 aabbccddeeff00112233445566778899aabbccddeeff001122334455667788", "12345 13 4 aabbccddeeff00112233445566778899aabbccddeeff001122334455667788990011223344556677889900aabbccdd"},
				Managed: true,
			},
		},
	}
	s.domains[req.Name] = d
	s.rrsets[req.Name] = map[string]*api.RRset{
		rrsetKey("", "NS"): {
			Created: ts,
			Domain:  req.Name,
			Subname: "",
			Name:    req.Name + ".",
			Type:    "NS",
			TTL:     minimumTTL,
			Records: []string{"ns1.desec.io.", "ns2.desec.org."},
			Touched: ts,
		},
	}

	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) getDomain(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.domains[name]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.domains, name)
	delete(s.rrsets, name)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getZonefile(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("name")

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

func (s *Server) listRRsets(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("name")

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	filterSubname := r.URL.Query().Get("subname")
	filterType := r.URL.Query().Get("type")
	hasSubnameFilter := r.URL.Query().Has("subname")

	all := make([]*api.RRset, 0)
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

	result := make([]api.RRset, 0, len(all))
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

func (s *Server) createRRset(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("name")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	var req api.CreateRRsetOptions
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
	rs := &api.RRset{
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

func (s *Server) getRRset(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("name")
	subname := r.PathValue("subname")
	if subname == "@" {
		subname = ""
	}
	rrtype := r.PathValue("type")

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

func (s *Server) updateRRset(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("name")
	subname := r.PathValue("subname")
	if subname == "@" {
		subname = ""
	}
	rrtype := r.PathValue("type")

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

	// *int needed to distinguish absent field from zero in PATCH.
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

func (s *Server) deleteRRset(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("name")
	subname := r.PathValue("subname")
	if subname == "@" {
		subname = ""
	}
	rrtype := r.PathValue("type")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	delete(s.rrsets[domainName], rrsetKey(subname, rrtype))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) bulkUpdateRRsets(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("name")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	// *int needed to distinguish absent field from zero in PATCH/PUT.
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
	results := make([]api.RRset, 0, len(reqs))

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
			rs := &api.RRset{
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

// ---- Token handlers ----

func (s *Server) listTokens(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]*api.Token, 0, len(s.tokens))
	for _, t := range s.tokens {
		all = append(all, t)
	}
	// Sort by creation time, then by ID as a stable tiebreaker.
	sort.Slice(all, func(i, j int) bool {
		if all[i].Created != all[j].Created {
			return all[i].Created < all[j].Created
		}
		return all[i].ID < all[j].ID
	})

	result := make([]api.Token, 0, len(all))
	for _, t := range all {
		result = append(result, *t)
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

func (s *Server) createToken(w http.ResponseWriter, r *http.Request) {
	// Use value-type bools so we can detect absent fields and apply API defaults.
	var req struct {
		Name             string   `json:"name"`
		PermCreateDomain bool     `json:"perm_create_domain"`
		PermDeleteDomain bool     `json:"perm_delete_domain"`
		PermManageTokens bool     `json:"perm_manage_tokens"`
		AllowedSubnets   []string `json:"allowed_subnets"`
		AutoPolicy       bool     `json:"auto_policy"`
		MaxAge           *string  `json:"max_age"`
		MaxUnusedPeriod  *string  `json:"max_unused_period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"detail":"Parse error."}`, http.StatusBadRequest)
		return
	}

	allowedSubnets := req.AllowedSubnets
	if allowedSubnets == nil {
		allowedSubnets = []string{"0.0.0.0/0", "::/0"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.NewString()
	secret := "fake-token-" + id[:8]
	ts := now()
	t := &api.Token{
		ID:               id,
		Created:          ts,
		LastUsed:         nil,
		Owner:            "test@example.com",
		UserOverride:     nil,
		MFA:              nil,
		MaxAge:           req.MaxAge,
		MaxUnusedPeriod:  req.MaxUnusedPeriod,
		Name:             req.Name,
		PermCreateDomain: req.PermCreateDomain,
		PermDeleteDomain: req.PermDeleteDomain,
		PermManageTokens: req.PermManageTokens,
		AllowedSubnets:   allowedSubnets,
		AutoPolicy:       req.AutoPolicy,
		IsValid:          true,
	}
	s.tokens[id] = t
	s.secrets[id] = secret

	// Include secret in the create response only; Secret has omitempty so it
	// is suppressed in all other responses.
	resp := *t
	resp.Secret = secret
	writeJSON(w, http.StatusCreated, &resp)
}

func (s *Server) getToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tokens[id]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}
	// Secret field is empty string on stored token; omitempty suppresses it.
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) updateToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tokens[id]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	// We use json.RawMessage to distinguish between a field being absent vs. null.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, `{"detail":"Parse error."}`, http.StatusBadRequest)
		return
	}

	if v, ok := raw["name"]; ok {
		_ = json.Unmarshal(v, &t.Name)
	}
	if v, ok := raw["perm_create_domain"]; ok {
		_ = json.Unmarshal(v, &t.PermCreateDomain)
	}
	if v, ok := raw["perm_delete_domain"]; ok {
		_ = json.Unmarshal(v, &t.PermDeleteDomain)
	}
	if v, ok := raw["perm_manage_tokens"]; ok {
		_ = json.Unmarshal(v, &t.PermManageTokens)
	}
	if v, ok := raw["allowed_subnets"]; ok {
		_ = json.Unmarshal(v, &t.AllowedSubnets)
	}
	if v, ok := raw["auto_policy"]; ok {
		_ = json.Unmarshal(v, &t.AutoPolicy)
	}
	if v, ok := raw["max_age"]; ok {
		_ = json.Unmarshal(v, &t.MaxAge)
	}
	if v, ok := raw["max_unused_period"]; ok {
		_ = json.Unmarshal(v, &t.MaxUnusedPeriod)
	}

	writeJSON(w, http.StatusOK, t)
}

func (s *Server) deleteToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, id)
	delete(s.secrets, id)
	delete(s.policies, id) // cascade-delete all policies for this token
	w.WriteHeader(http.StatusNoContent)
}

// ---- Token policy handlers ----

func (s *Server) listTokenPolicies(w http.ResponseWriter, r *http.Request) {
	tokenID := r.PathValue("id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.tokens[tokenID]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	all := make([]*api.TokenPolicy, 0)
	for _, p := range s.policies[tokenID] {
		all = append(all, p)
	}
	// Stable sort: default policy first, then by ID.
	sort.Slice(all, func(i, j int) bool {
		iDef := isDefaultPolicy(all[i])
		jDef := isDefaultPolicy(all[j])
		if iDef != jDef {
			return iDef // default policy sorts first
		}
		return all[i].ID < all[j].ID
	})

	result := make([]api.TokenPolicy, 0, len(all))
	for _, p := range all {
		result = append(result, *p)
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

func (s *Server) createTokenPolicy(w http.ResponseWriter, r *http.Request) {
	tokenID := r.PathValue("id")

	var req api.CreateTokenPolicyOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"detail":"Parse error."}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tokens[tokenID]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	if s.policies[tokenID] == nil {
		s.policies[tokenID] = make(map[string]*api.TokenPolicy)
	}

	incoming := &api.TokenPolicy{Domain: req.Domain, Subname: req.Subname, Type: req.Type}

	// Enforce: a specific policy requires a default policy to already exist.
	if !isDefaultPolicy(incoming) {
		hasDefault := false
		for _, p := range s.policies[tokenID] {
			if isDefaultPolicy(p) {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			http.Error(w, `{"detail":"A default policy must exist before specific policies can be created."}`, http.StatusBadRequest)
			return
		}
	}

	id := uuid.NewString()
	p := &api.TokenPolicy{
		ID:        id,
		Domain:    req.Domain,
		Subname:   req.Subname,
		Type:      req.Type,
		PermWrite: req.PermWrite,
	}
	s.policies[tokenID][id] = p

	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getTokenPolicy(w http.ResponseWriter, r *http.Request) {
	tokenID := r.PathValue("id")
	policyID := r.PathValue("policy_id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.tokens[tokenID]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	p, ok := s.policies[tokenID][policyID]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (s *Server) updateTokenPolicy(w http.ResponseWriter, r *http.Request) {
	tokenID := r.PathValue("id")
	policyID := r.PathValue("policy_id")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tokens[tokenID]; !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	p, ok := s.policies[tokenID][policyID]
	if !ok {
		http.Error(w, `{"detail":"Not found."}`, http.StatusNotFound)
		return
	}

	var req api.UpdateTokenPolicyOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"detail":"Parse error."}`, http.StatusBadRequest)
		return
	}

	p.PermWrite = req.PermWrite
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) deleteTokenPolicy(w http.ResponseWriter, r *http.Request) {
	tokenID := r.PathValue("id")
	policyID := r.PathValue("policy_id")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tokens[tokenID]; !ok {
		// Token not found — treat as success per deSEC semantics.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	p, ok := s.policies[tokenID][policyID]
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Enforce: the default policy cannot be deleted while specific policies exist.
	if isDefaultPolicy(p) {
		for _, other := range s.policies[tokenID] {
			if !isDefaultPolicy(other) {
				http.Error(w, `{"detail":"Cannot delete the default policy while specific policies exist."}`, http.StatusBadRequest)
				return
			}
		}
	}

	delete(s.policies[tokenID], policyID)
	w.WriteHeader(http.StatusNoContent)
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
