// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNullableString(t *testing.T) {
	t.Run("non-null value returns pointer to value", func(t *testing.T) {
		s := types.StringValue("hello")
		got := nullableString(s)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != "hello" {
			t.Fatalf("expected %q, got %q", "hello", *got)
		}
	})

	t.Run("empty string returns pointer to empty string", func(t *testing.T) {
		s := types.StringValue("")
		got := nullableString(s)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != "" {
			t.Fatalf("expected empty string, got %q", *got)
		}
	})

	t.Run("null returns nil", func(t *testing.T) {
		s := types.StringNull()
		got := nullableString(s)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %q", *got)
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		s := types.StringUnknown()
		got := nullableString(s)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %q", *got)
		}
	})
}

func TestNullableBool(t *testing.T) {
	t.Run("true returns pointer to true", func(t *testing.T) {
		b := types.BoolValue(true)
		got := nullableBool(b)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != true {
			t.Fatalf("expected true, got %v", *got)
		}
	})

	t.Run("false returns pointer to false", func(t *testing.T) {
		b := types.BoolValue(false)
		got := nullableBool(b)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != false {
			t.Fatalf("expected false, got %v", *got)
		}
	})

	t.Run("null returns nil", func(t *testing.T) {
		b := types.BoolNull()
		got := nullableBool(b)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %v", *got)
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		b := types.BoolUnknown()
		got := nullableBool(b)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %v", *got)
		}
	})
}
