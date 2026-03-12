// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// testRunIDs caches a per-test random hex ID so that all testAccDomainName
// calls within the same test share the same random component.
var testRunIDs sync.Map

// testAccDomainName returns a domain name suitable for use in acceptance tests.
// When using the fake backend, it returns a simple deterministic name based on
// the test name suffix. When using the real API, it returns a unique name with
// a random suffix to avoid conflicts between runs.
func testAccDomainName(t *testing.T, suffix string) string {
	t.Helper()
	if useRealAPI() {
		// Generate (or reuse) a per-test random 6-hex-char ID so that all
		// domains created by the same test share the same random component,
		// but different runs produce different names and don't collide.
		id, _ := testRunIDs.LoadOrStore(t.Name(), randomHex(3))
		idStr, ok := id.(string)
		if !ok {
			t.Fatalf("testRunIDs: unexpected non-string value %T", id)
		}
		return fmt.Sprintf("tf-acc-%s-%s-%s.dedyn.io", suffix, sanitize(t.Name()), idStr)
	}
	return fmt.Sprintf("%s.example.com", suffix)
}

// randomHex returns a random lowercase hex string of n bytes (2*n characters).
func randomHex(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.IntN(256))
	}
	return hex.EncodeToString(b)
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

// tokenListContains returns a statecheck.StateCheck that verifies that the
// "tokens" list attribute of a desec_tokens data source contains at least one
// token whose "name" attribute equals wantName. This is safe to use against a
// real deSEC account that may have additional tokens beyond the ones created by
// the test.
func tokenListContains(resourceAddr, wantName string) statecheck.StateCheck {
	return tokenListContainsCheck{resourceAddr: resourceAddr, wantName: wantName}
}

type tokenListContainsCheck struct {
	resourceAddr string
	wantName     string
}

func (c tokenListContainsCheck) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	if req.State == nil || req.State.Values == nil || req.State.Values.RootModule == nil {
		resp.Error = fmt.Errorf("state has no root module")
		return
	}
	var res *tfjson.StateResource
	for _, r := range req.State.Values.RootModule.Resources {
		if r.Address == c.resourceAddr {
			res = r
			break
		}
	}
	if res == nil {
		resp.Error = fmt.Errorf("resource %q not found in state", c.resourceAddr)
		return
	}
	tokens, ok := res.AttributeValues["tokens"]
	if !ok {
		resp.Error = fmt.Errorf("resource %q has no attribute \"tokens\"", c.resourceAddr)
		return
	}
	list, ok := tokens.([]any)
	if !ok {
		resp.Error = fmt.Errorf("resource %q attribute \"tokens\" is not a list", c.resourceAddr)
		return
	}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if name, ok := m["name"].(string); ok && name == c.wantName {
			return
		}
	}
	resp.Error = fmt.Errorf("resource %q: no token with name %q found in tokens list", c.resourceAddr, c.wantName)
}
