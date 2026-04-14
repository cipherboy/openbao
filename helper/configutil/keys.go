// Copyright (c) 2026 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package configutil

import "github.com/hashicorp/hcl/hcl/ast"

// ParseEitherNamedKey is a helper for when an HCL configuration block can
// take two optionally named keys. These are of the form:
//
// block "<first:value>" "<second:value>" { ... }
// block "<first:value>" { second = "<value>" }
// block "<second:value>" { first = "<value>" }
// block { first = "value" second = "<value>" }
//
// This function returns false if the pattern is not observed, letting the
// caller return an appropriate error message.
func ParseEitherNamedKey(item *ast.ObjectItem, first *string, second *string) bool {
	switch {
	case *first != "" && *second != "":
	case *first != "" && *second == "" && len(item.Keys) == 1:
		*second = item.Keys[0].Token.Value().(string)
	case *first == "" && *second != "" && len(item.Keys) == 1:
		*second = item.Keys[0].Token.Value().(string)
	case *first == "" && *second == "" && len(item.Keys) == 2:
		*first = item.Keys[0].Token.Value().(string)
		*second = item.Keys[1].Token.Value().(string)
	default:
		return false
	}

	return true
}
