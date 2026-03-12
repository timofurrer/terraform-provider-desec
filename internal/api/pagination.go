// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"net/url"
	"regexp"
)

// linkNextRe matches the `next` relation in a Link header.
// Example: <https://desec.io/api/v1/domains/?cursor=abc>; rel="next".
var linkNextRe = regexp.MustCompile(`<[^>]+[?&]cursor=([^>&]+)[^>]*>;\s*rel="next"`)

// parseCursorNext extracts the cursor value for the next page from a Link header.
// Returns an empty string if there is no next page.
func parseCursorNext(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	m := linkNextRe.FindStringSubmatch(linkHeader)
	if len(m) < 2 {
		return ""
	}
	decoded, err := url.QueryUnescape(m[1])
	if err != nil {
		return m[1]
	}
	return decoded
}
