// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
)

// testAccDomainName returns a domain name suitable for use in acceptance tests.
// When using the fake backend, it returns a simple deterministic name based on
// the test name suffix. When using the real API, it returns a unique name with
// a random suffix to avoid conflicts.
func testAccDomainName(t *testing.T, suffix string) string {
	t.Helper()
	if useRealAPI() {
		// Use a hash of the test name to get a repeatable but unique name.
		return fmt.Sprintf("tf-acc-%s-%s.dedyn.io", suffix, sanitize(t.Name()))
	}
	return fmt.Sprintf("%s.example.com", suffix)
}

// sanitize removes characters that are not valid in domain names.
func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c+('a'-'A'))
		case c >= '0' && c <= '9':
			out = append(out, c)
		default:
			out = append(out, '-')
		}
	}
	// Trim trailing dashes and limit length to 30 characters.
	result := string(out)
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	if len(result) > 30 {
		result = result[len(result)-30:]
	}
	return result
}
