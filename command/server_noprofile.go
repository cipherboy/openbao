// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !memprofiler && !agent && !proxy

package command

func (c *ServerCommand) startMemProfiler() {
}
