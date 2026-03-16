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
var _ function.Function = &toPunycodeFunction{}

type toPunycodeFunction struct{}

func newToPunycodeFunction() function.Function {
	return &toPunycodeFunction{}
}

func (f *toPunycodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "to_punycode"
}

func (f *toPunycodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Convert a unicode domain name to its Punycode (ACE) representation.",
		MarkdownDescription: "Converts a domain name containing unicode characters to its " +
			"[Punycode](https://en.wikipedia.org/wiki/Punycode) (ACE) representation per RFC 5891. " +
			"Labels containing non-ASCII characters are encoded as `xn--` prefixed ASCII labels; " +
			"pure-ASCII labels are returned unchanged. A trailing dot (FQDN notation) is preserved.\n\n" +
			"This is useful when working with Internationalized Domain Names (IDN): the deSEC API " +
			"only accepts domain names in Punycode form, so this function can be used to convert a " +
			"human-readable unicode name before passing it to `desec_domain` or other resources.",

		Parameters: []function.Parameter{
			function.StringParameter{
				Name: "domain",
				MarkdownDescription: "The domain name to convert. Each label containing non-ASCII " +
					"characters is converted to its `xn--` Punycode form. Pure-ASCII labels are " +
					"returned unchanged. A trailing dot (FQDN notation) is preserved.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *toPunycodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var domain string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &domain))
	if resp.Error != nil {
		return
	}

	// Preserve a trailing dot (FQDN notation) and restore it after conversion.
	fqdn := strings.HasSuffix(domain, ".")
	name := strings.TrimSuffix(domain, ".")

	result, err := idna.Lookup.ToASCII(name)
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
