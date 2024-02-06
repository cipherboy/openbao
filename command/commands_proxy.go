// Copyright (c) OpenBao, Inc.
// SPDX-License-Identifier: MPL-2.0

// See note in commands_unified.go; this file is for server builds.
//go:build !agent && !cli && proxy && !server

package command

import (
	"github.com/mitchellh/cli"
	"github.com/openbao/openbao/version"

	/*
		The builtinplugins package is initialized here because it, in turn,
		initializes the database plugins.
		They register multiple database drivers for the "database/sql" package.
	*/
	_ "github.com/openbao/openbao/helper/builtinplugins"
)

func getCommandsForBinary(ui, serverCmdUi cli.Ui, runOpts *RunOptions, getBaseCommand func() *BaseCommand, loginHandlers map[string]LoginHandler) map[string]cli.CommandFactory {
	commands := map[string]cli.CommandFactory{
		"login": func() (cli.Command, error) {
			return &LoginCommand{
				BaseCommand: getBaseCommand(),
				Handlers:    loginHandlers,
			}, nil
		},
		"proxy": func() (cli.Command, error) {
			return &ProxyCommand{
				BaseCommand: &BaseCommand{
					UI: serverCmdUi,
				},
				ShutdownCh: MakeShutdownCh(),
				SighupCh:   MakeSighupCh(),
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &StatusCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"token": func() (cli.Command, error) {
			return &TokenCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"token create": func() (cli.Command, error) {
			return &TokenCreateCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"token capabilities": func() (cli.Command, error) {
			return &TokenCapabilitiesCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"token lookup": func() (cli.Command, error) {
			return &TokenLookupCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"token renew": func() (cli.Command, error) {
			return &TokenRenewCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"token revoke": func() (cli.Command, error) {
			return &TokenRevokeCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &VersionCommand{
				VersionInfo: version.GetVersion(),
				BaseCommand: getBaseCommand(),
			}, nil
		},
	}

	return commands
}
