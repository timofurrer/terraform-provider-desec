// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"golang.org/x/net/idna"
)

// Ensure the implementation satisfies the desired interfaces.
var _ function.Function = &fromPunycodeFunction{}

type fromPunycodeFunction struct{}

func newFromPunycodeFunction() function.Function {
	return &fromPunycodeFunction{}
}

func (f *fromPunycodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "from_punycode"
}

func (f *fromPunycodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Convert a Punycode (ACE) domain name to its unicode representation.",
		MarkdownDescription: "Converts a domain name in [Punycode](https://en.wikipedia.org/wiki/Punycode) " +
			"(ACE) form to its unicode representation per RFC 5891. Labels beginning with `xn--` are " +
			"decoded to their unicode form; pure-ASCII labels that are not Punycode-encoded are returned " +
			"unchanged. A trailing dot (FQDN notation) is preserved.\n\n" +
			"This is the inverse of `to_punycode` and is useful for displaying " +
			"Internationalized Domain Names (IDN) in a human-readable form.",

		Parameters: []function.Parameter{
			function.StringParameter{
				Name: "domain",
				MarkdownDescription: "The domain name to convert. Labels beginning with `xn--` are " +
					"decoded to their unicode form. Pure-ASCII labels that are not Punycode-encoded " +
					"are returned unchanged. A trailing dot (FQDN notation) is preserved.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *fromPunycodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var domain string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &domain))
	if resp.Error != nil {
		return
	}

	// Preserve a trailing dot (FQDN notation) and restore it after conversion.
	fqdn := strings.HasSuffix(domain, ".")
	name := strings.TrimSuffix(domain, ".")

	// ToUnicode follows the RFC 5891 convention of returning the best-effort
	// unicode form. Invalid or unrecognised xn-- labels are passed through
	// unchanged — consistent with how browsers and DNS resolvers behave.
	result, err := idna.Lookup.ToUnicode(name)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(
			0,
			"Invalid domain name: "+err.Error(),
		))
		return
	}

	if fqdn {
		result += "."
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
