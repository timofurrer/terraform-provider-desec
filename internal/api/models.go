// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package api

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

// Token represents a deSEC API authentication token.
type Token struct {
	ID               string   `json:"id"`
	Created          string   `json:"created"`
	LastUsed         *string  `json:"last_used"`
	Owner            string   `json:"owner"`
	UserOverride     *string  `json:"user_override"`
	MFA              *bool    `json:"mfa"`
	MaxAge           *string  `json:"max_age"`
	MaxUnusedPeriod  *string  `json:"max_unused_period"`
	Name             string   `json:"name"`
	PermCreateDomain bool     `json:"perm_create_domain"`
	PermDeleteDomain bool     `json:"perm_delete_domain"`
	PermManageTokens bool     `json:"perm_manage_tokens"`
	AllowedSubnets   []string `json:"allowed_subnets"`
	AutoPolicy       bool     `json:"auto_policy"`
	IsValid          bool     `json:"is_valid"`
	// Secret is the token's secret value. Only present in the create response;
	// subsequent GET requests do not return it.
	Secret string `json:"token,omitempty"`
}

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
