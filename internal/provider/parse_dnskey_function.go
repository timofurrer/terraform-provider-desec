// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = &parseDNSKeyFunction{}

var dnskeyAttrTypes = map[string]attr.Type{
	"flags":      types.Int64Type,
	"protocol":   types.Int64Type,
	"algorithm":  types.Int64Type,
	"public_key": types.StringType,
}

type parseDNSKeyFunction struct{}

func newParseDNSKeyFunction() function.Function {
	return &parseDNSKeyFunction{}
}

func (f *parseDNSKeyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "parse_dnskey"
}

func (f *parseDNSKeyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Parse a DNSKEY record string into its constituent fields.",
		MarkdownDescription: "Parses a DNSKEY record content string (as returned by the deSEC API in the " +
			"`keys[].dnskey` field of a domain) into its constituent fields per " +
			"[RFC 4034 §2](https://datatracker.ietf.org/doc/html/rfc4034#section-2).\n\n" +
			"The input string is expected to be space-delimited with the format " +
			"`<flags> <protocol> <algorithm> <public_key>`, for example " +
			"`\"257 3 13 MemvLhocKfZ8t/7kAef1UJ3cZGjkZLM3c+e76HZ2d2z7EW+6J2EbhHWAUcHhH/JZ5PtNi0GpEy5U56WvLswZAA==\"`.\n\n" +
			"The function returns an object with the following attributes:\n\n" +
			"- `flags` (Number) — DNSKEY flags field (e.g. 256 for ZSK, 257 for KSK).\n" +
			"- `protocol` (Number) — Must be 3 per RFC 4034.\n" +
			"- `algorithm` (Number) — DNSSEC algorithm number (e.g. 13 for ECDSAP256SHA256).\n" +
			"- `public_key` (String) — Base64-encoded public key material.",

		Parameters: []function.Parameter{
			function.StringParameter{
				Name: "dnskey",
				MarkdownDescription: "The DNSKEY record content string to parse, in the format " +
					"`<flags> <protocol> <algorithm> <public_key>`.",
			},
		},
		Return: function.ObjectReturn{
			AttributeTypes: dnskeyAttrTypes,
		},
	}
}

func (f *parseDNSKeyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var dnskey string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &dnskey))
	if resp.Error != nil {
		return
	}

	parts := strings.SplitN(dnskey, " ", 4)
	if len(parts) != 4 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DNSKEY record: expected 4 space-delimited fields (<flags> <protocol> <algorithm> <public_key>), got "+dnskey,
		))
		return
	}

	flags, err := parseInt64(parts[0])
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DNSKEY flags: "+err.Error(),
		))
		return
	}

	protocol, err := parseInt64(parts[1])
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DNSKEY protocol: "+err.Error(),
		))
		return
	}

	algorithm, err := parseInt64(parts[2])
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DNSKEY algorithm: "+err.Error(),
		))
		return
	}

	result, diags := types.ObjectValue(dnskeyAttrTypes, map[string]attr.Value{
		"flags":      types.Int64Value(flags),
		"protocol":   types.Int64Value(protocol),
		"algorithm":  types.Int64Value(algorithm),
		"public_key": types.StringValue(parts[3]),
	})
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
