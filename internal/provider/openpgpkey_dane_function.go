// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = &openpgpkeyDANEFunction{}

var openpgpkeyDANEAttrTypes = map[string]attr.Type{
	"domain":  types.StringType,
	"subname": types.StringType,
	"type":    types.StringType,
	"rdata":   types.StringType,
}

type openpgpkeyDANEFunction struct{}

func newOpenPGPKeyDANEFunction() function.Function {
	return &openpgpkeyDANEFunction{}
}

func (f *openpgpkeyDANEFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "openpgpkey_dane"
}

func (f *openpgpkeyDANEFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Compute the DNS record subname and RDATA for an OPENPGPKEY DANE record.",
		MarkdownDescription: "Computes the DNS record `subname` and `rdata` for publishing an OpenPGP public key " +
			"via DNS using the OPENPGPKEY resource record (TYPE 61) as defined in " +
			"[RFC 7929](https://datatracker.ietf.org/doc/html/rfc7929).\n\n" +
			"Given an email address, the DNS owner name is constructed by:\n\n" +
			"1. Taking the **local-part** (left of `@`) and converting it to **lowercase**.\n" +
			"2. Computing the **SHA-256 hash** and **truncating** to 28 octets (224 bits).\n" +
			"3. **Hex-encoding** the truncated hash (56 characters).\n\n" +
			"The resulting `subname` is `<56-char-hex>._openpgpkey`, ready to use with a `desec_record` resource.\n\n" +
			"The `rdata` is the validated base64-encoded OpenPGP Transferable Public Key, suitable for the " +
			"`records` attribute of a `desec_record` resource with `type = \"OPENPGPKEY\"`.\n\n" +
			"The function also returns the `domain` extracted from the email address.\n\n" +
			"The function returns an object with the following attributes:\n\n" +
			"- `domain` (String) — The domain part of the email address (right of `@`).\n" +
			"- `subname` (String) — The computed DNS subname in the form `<hash>._openpgpkey`.\n" +
			"- `type` (String) — The DNS record type, always `\"OPENPGPKEY\"`.\n" +
			"- `rdata` (String) — The base64-encoded OpenPGP public key.",

		Parameters: []function.Parameter{
			function.StringParameter{
				Name: "email",
				MarkdownDescription: "The email address to compute the OPENPGPKEY record for " +
					"(e.g. `\"hugh@example.com\"`). The local-part (left of `@`) is lowercased " +
					"before hashing per RFC 7929 §3.",
			},
			function.StringParameter{
				Name: "gpg_key",
				MarkdownDescription: "The base64-encoded OpenPGP Transferable Public Key. " +
					"This can be produced with:\n\n" +
					"```shell\n" +
					"gpg --export --export-options export-minimal,no-export-attributes \\\n" +
					"    user@example.com | base64\n" +
					"```",
			},
		},
		Return: function.ObjectReturn{
			AttributeTypes: openpgpkeyDANEAttrTypes,
		},
	}
}

func (f *openpgpkeyDANEFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var email, gpgKey string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &email, &gpgKey))
	if resp.Error != nil {
		return
	}

	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid email address: expected format \"local-part@domain\", got \""+email+"\"",
		))
		return
	}

	localPart := strings.ToLower(parts[0])
	domain := parts[1]

	hash := sha256.Sum256([]byte(localPart))
	truncated := hash[:28]
	hashHex := hex.EncodeToString(truncated)

	subname := hashHex + "._openpgpkey"

	if _, err := base64.StdEncoding.DecodeString(gpgKey); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			1,
			"Invalid GPG key: not valid base64: "+err.Error(),
		))
		return
	}

	result, diags := types.ObjectValue(openpgpkeyDANEAttrTypes, map[string]attr.Value{
		"domain":  types.StringValue(domain),
		"subname": types.StringValue(subname),
		"type":    types.StringValue("OPENPGPKEY"),
		"rdata":   types.StringValue(gpgKey),
	})
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
