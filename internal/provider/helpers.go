// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/txtrdata"
)

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// nullableString converts a types.String to a *string suitable for API calls.
// Null or unknown values become nil.
func nullableString(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	v := s.ValueString()
	return &v
}

// nullableBool converts a types.Bool to a *bool suitable for API calls.
// Null or unknown values become nil.
func nullableBool(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

// canonicalRecord returns a comparison key for a single record value of the
// given type. For TXT-like records it accounts for deSEC splitting
// character-strings longer than 255 bytes into multiple strings.
func canonicalRecord(rrtype, record string) string {
	if txtrdata.IsTXTLike(rrtype) {
		return txtrdata.Canonical(record)
	}
	return record
}

// recordsEquivalent reports whether a and b contain the same records
// (ignoring order) after canonicalization.
func recordsEquivalent(rrtype string, a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, r := range a {
		counts[canonicalRecord(rrtype, r)]++
	}
	for _, r := range b {
		k := canonicalRecord(rrtype, r)
		if counts[k] == 0 {
			return false
		}
		counts[k]--
	}
	return true
}

// reconcileRecords returns prior if it is equivalent to actual, so that the
// user's original representation (e.g. an unsplit long TXT string) is kept
// in state. Otherwise actual is returned.
func reconcileRecords(rrtype string, prior, actual []string) []string {
	if prior != nil && recordsEquivalent(rrtype, prior, actual) {
		return prior
	}
	return actual
}
