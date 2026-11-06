// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

// Package txtrdata parses and formats the presentation format of TXT-like
// record data (a sequence of character-strings).
//
// deSEC accepts TXT character-strings longer than the 255 bytes allowed by
// the DNS wire format and splits them into multiple character-strings
// within the same record. This package provides the helpers needed to
// recognize that a user-supplied value and the value returned by deSEC are
// equivalent.
package txtrdata

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxStringLength is the maximum length in bytes of a single DNS character-string.
const MaxStringLength = 255

// IsTXTLike reports whether the given record type uses TXT-style rdata.
func IsTXTLike(rrtype string) bool {
	switch strings.ToUpper(rrtype) {
	case "TXT", "SPF":
		return true
	}
	return false
}

// Parse decodes TXT presentation format into its character-strings.
// Both quoted and unquoted tokens are accepted. Escape sequences (`\X` and
// `\DDD`) are decoded.
func Parse(s string) ([][]byte, error) {
	var out [][]byte
	i := 0
	for {
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		if i >= len(s) {
			return out, nil
		}

		quoted := s[i] == '"'
		if quoted {
			i++
		}

		var cur []byte
		closed := false
		for i < len(s) {
			c := s[i]
			if quoted && c == '"' {
				i++
				closed = true
				break
			}
			if !quoted && isSpace(c) {
				break
			}
			if c == '\\' {
				if i+1 >= len(s) {
					return nil, fmt.Errorf("dangling escape at end of %q", s)
				}
				if i+3 < len(s) && isDigit(s[i+1]) && isDigit(s[i+2]) && isDigit(s[i+3]) {
					n, err := strconv.Atoi(s[i+1 : i+4])
					if err != nil || n > 255 {
						return nil, fmt.Errorf("invalid decimal escape %q", s[i:i+4])
					}
					cur = append(cur, byte(n))
					i += 4
					continue
				}
				cur = append(cur, s[i+1])
				i += 2
				continue
			}
			cur = append(cur, c)
			i++
		}
		if quoted && !closed {
			return nil, fmt.Errorf("unterminated quoted string in %q", s)
		}
		out = append(out, cur)
	}
}

// Split splits every character-string longer than MaxStringLength bytes into
// consecutive chunks of at most MaxStringLength bytes, the same way deSEC does.
func Split(strs [][]byte) [][]byte {
	out := make([][]byte, 0, len(strs))
	for _, s := range strs {
		if len(s) <= MaxStringLength {
			out = append(out, s)
			continue
		}
		for start := 0; start < len(s); start += MaxStringLength {
			end := min(start+MaxStringLength, len(s))
			out = append(out, s[start:end])
		}
	}
	return out
}

// Format encodes character-strings as quoted TXT presentation format,
// separated by single spaces.
func Format(strs [][]byte) string {
	var sb strings.Builder
	for i, s := range strs {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteByte('"')
		for _, c := range s {
			switch {
			case c == '"' || c == '\\':
				sb.WriteByte('\\')
				sb.WriteByte(c)
			case c < 0x20 || c >= 0x7f:
				fmt.Fprintf(&sb, "\\%03d", c)
			default:
				sb.WriteByte(c)
			}
		}
		sb.WriteByte('"')
	}
	return sb.String()
}

// Canonical returns a comparison key for TXT rdata: the character-strings
// after splitting at MaxStringLength bytes. Two values with the same key are
// served identically by deSEC. If s cannot be parsed, s is returned as-is.
func Canonical(s string) string {
	strs, err := Parse(s)
	if err != nil {
		return s
	}
	return Format(Split(strs))
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
