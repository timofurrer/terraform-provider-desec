// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
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
