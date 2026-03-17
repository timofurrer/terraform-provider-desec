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

var _ function.Function = &parseDSFunction{}

var dsAttrTypes = map[string]attr.Type{
	"key_tag":     types.Int64Type,
	"algorithm":   types.Int64Type,
	"digest_type": types.Int64Type,
	"digest":      types.StringType,
}

type parseDSFunction struct{}

func newParseDSFunction() function.Function {
	return &parseDSFunction{}
}

func (f *parseDSFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "parse_ds"
}

func (f *parseDSFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Parse a DS record string into its constituent fields.",
		MarkdownDescription: "Parses a DS (Delegation Signer) record content string (as returned by the deSEC API " +
			"in the `keys[].ds[]` field of a domain) into its constituent fields per " +
			"[RFC 4034 §5](https://datatracker.ietf.org/doc/html/rfc4034#section-5).\n\n" +
			"The input string is expected to be space-delimited with the format " +
			"`<key_tag> <algorithm> <digest_type> <digest>`, for example " +
			"`\"26064 13 2 ed8e5b1f29bb64d404e1c43ff7d15232289eb707e554808309bbd5f7fb4695d0\"`.\n\n" +
			"The function returns an object with the following attributes:\n\n" +
			"- `key_tag` (Number) — Key tag identifying the corresponding DNSKEY record.\n" +
			"- `algorithm` (Number) — DNSSEC algorithm number (e.g. 13 for ECDSAP256SHA256).\n" +
			"- `digest_type` (Number) — Digest algorithm (e.g. 2 for SHA-256, 4 for SHA-384).\n" +
			"- `digest` (String) — Hex-encoded digest of the DNSKEY record.",

		Parameters: []function.Parameter{
			function.StringParameter{
				Name: "ds",
				MarkdownDescription: "The DS record content string to parse, in the format " +
					"`<key_tag> <algorithm> <digest_type> <digest>`.",
			},
		},
		Return: function.ObjectReturn{
			AttributeTypes: dsAttrTypes,
		},
	}
}

func (f *parseDSFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ds string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ds))
	if resp.Error != nil {
		return
	}

	parts := strings.SplitN(ds, " ", 4)
	if len(parts) != 4 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DS record: expected 4 space-delimited fields (<key_tag> <algorithm> <digest_type> <digest>), got "+ds,
		))
		return
	}

	keyTag, err := parseInt64(parts[0])
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DS key_tag: "+err.Error(),
		))
		return
	}

	algorithm, err := parseInt64(parts[1])
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DS algorithm: "+err.Error(),
		))
		return
	}

	digestType, err := parseInt64(parts[2])
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid DS digest_type: "+err.Error(),
		))
		return
	}

	result, diags := types.ObjectValue(dsAttrTypes, map[string]attr.Value{
		"key_tag":     types.Int64Value(keyTag),
		"algorithm":   types.Int64Value(algorithm),
		"digest_type": types.Int64Value(digestType),
		"digest":      types.StringValue(parts[3]),
	})
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
