// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"io"
	"net"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-secure-stdlib/reloadutil"
	"github.com/openbao/openbao/internalshared/configutil"
	"github.com/openbao/openbao/internalshared/listenerutil"
)

func unixListenerFactory(l *configutil.Listener, _ io.Writer, ui cli.Ui) (net.Listener, map[string]string, reloadutil.ReloadFunc, error) {
	addr := l.Address
	if addr == "" {
		addr = "/run/vault.sock"
	}

	var cfg *listenerutil.UnixSocketsConfig
	if l.SocketMode != "" &&
		l.SocketUser != "" &&
		l.SocketGroup != "" {
		cfg = &listenerutil.UnixSocketsConfig{
			Mode:  l.SocketMode,
			User:  l.SocketUser,
			Group: l.SocketGroup,
		}
	}

	ln, err := listenerutil.UnixSocketListener(addr, cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	return ln, map[string]string{}, nil, nil
}
