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
