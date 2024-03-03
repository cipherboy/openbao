// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mitchellh/cli"

	credCert "github.com/openbao/openbao/builtin/credential/cert"
	credOIDC "github.com/openbao/openbao/builtin/credential/jwt"
	credKerb "github.com/openbao/openbao/builtin/credential/kerberos"
	credLdap "github.com/openbao/openbao/builtin/credential/ldap"
	credToken "github.com/openbao/openbao/builtin/credential/token"
	credUserpass "github.com/openbao/openbao/builtin/credential/userpass"
)

const (
	// EnvVaultCLINoColor is an env var that toggles colored UI output.
	EnvVaultCLINoColor = `BAO_CLI_NO_COLOR`
	// EnvVaultFormat is the output format
	EnvVaultFormat = `BAO_FORMAT`
	// EnvVaultDetailed is to output detailed information (e.g., ListResponseWithInfo).
	EnvVaultDetailed = `BAO_DETAILED`
	// EnvVaultLogFormat is used to specify the log format. Supported values are "standard" and "json"
	EnvVaultLogFormat = "BAO_LOG_FORMAT"
	// EnvVaultLogLevel is used to specify the log level applied to logging
	// Supported log levels: Trace, Debug, Error, Warn, Info
	EnvVaultLogLevel = "BAO_LOG_LEVEL"

	// flagNameAddress is the flag used in the base command to read in the
	// address of the Vault server.
	flagNameAddress = "address"
	// flagnameCACert is the flag used in the base command to read in the CA
	// cert.
	flagNameCACert = "ca-cert"
	// flagnameCAPath is the flag used in the base command to read in the CA
	// cert path.
	flagNameCAPath = "ca-path"
	// flagNameClientCert is the flag used in the base command to read in the
	// client key
	flagNameClientKey = "client-key"
	// flagNameClientCert is the flag used in the base command to read in the
	// client cert
	flagNameClientCert = "client-cert"
	// flagNameTLSSkipVerify is the flag used in the base command to read in
	// the option to ignore TLS certificate verification.
	flagNameTLSSkipVerify = "tls-skip-verify"
	// flagTLSServerName is the flag used in the base command to read in
	// the TLS server name.
	flagTLSServerName = "tls-server-name"
	// flagNameAuditNonHMACRequestKeys is the flag name used for auth/secrets enable
	flagNameAuditNonHMACRequestKeys = "audit-non-hmac-request-keys"
	// flagNameAuditNonHMACResponseKeys is the flag name used for auth/secrets enable
	flagNameAuditNonHMACResponseKeys = "audit-non-hmac-response-keys"
	// flagNameDescription is the flag name used for tuning the secret and auth mount description parameter
	flagNameDescription = "description"
	// flagListingVisibility is the flag to toggle whether to show the mount in the UI-specific listing endpoint
	flagNameListingVisibility = "listing-visibility"
	// flagNamePassthroughRequestHeaders is the flag name used to set passthrough request headers to the backend
	flagNamePassthroughRequestHeaders = "passthrough-request-headers"
	// flagNameAllowedResponseHeaders is used to set allowed response headers from a plugin
	flagNameAllowedResponseHeaders = "allowed-response-headers"
	// flagNameTokenType is the flag name used to force a specific token type
	flagNameTokenType = "token-type"
	// flagNameAllowedManagedKeys is the flag name used for auth/secrets enable
	flagNameAllowedManagedKeys = "allowed-managed-keys"
	// flagNamePluginVersion selects what version of a plugin should be used.
	flagNamePluginVersion = "plugin-version"
	// flagNameUserLockoutThreshold is the flag name used for tuning the auth mount lockout threshold parameter
	flagNameUserLockoutThreshold = "user-lockout-threshold"
	// flagNameUserLockoutDuration is the flag name used for tuning the auth mount lockout duration parameter
	flagNameUserLockoutDuration = "user-lockout-duration"
	// flagNameUserLockoutCounterResetDuration is the flag name used for tuning the auth mount lockout counter reset parameter
	flagNameUserLockoutCounterResetDuration = "user-lockout-counter-reset-duration"
	// flagNameUserLockoutDisable is the flag name used for tuning the auth mount disable lockout parameter
	flagNameUserLockoutDisable = "user-lockout-disable"
	// flagNameDisableRedirects is used to prevent the client from honoring a single redirect as a response to a request
	flagNameDisableRedirects = "disable-redirects"
	// flagNameCombineLogs is used to specify whether log output should be combined and sent to stdout
	flagNameCombineLogs = "combine-logs"
	// flagDisableGatedLogs is used to disable gated logs and immediately show the vault logs as they become available
	flagDisableGatedLogs = "disable-gated-logs"
	// flagNameLogFile is used to specify the path to the log file that Vault should use for logging
	flagNameLogFile = "log-file"
	// flagNameLogRotateBytes is the flag used to specify the number of bytes a log file should be before it is rotated.
	flagNameLogRotateBytes = "log-rotate-bytes"
	// flagNameLogRotateDuration is the flag used to specify the duration after which a log file should be rotated.
	flagNameLogRotateDuration = "log-rotate-duration"
	// flagNameLogRotateMaxFiles is the flag used to specify the maximum number of older/archived log files to keep.
	flagNameLogRotateMaxFiles = "log-rotate-max-files"
	// flagNameLogFormat is the flag used to specify the log format. Supported values are "standard" and "json"
	flagNameLogFormat = "log-format"
	// flagNameLogLevel is used to specify the log level applied to logging
	// Supported log levels: Trace, Debug, Error, Warn, Info
	flagNameLogLevel = "log-level"
)

func initCommands(ui, serverCmdUi cli.Ui, runOpts *RunOptions) map[string]cli.CommandFactory {
	loginHandlers := map[string]LoginHandler{
		"cert":     &credCert.CLIHandler{},
		"kerberos": &credKerb.CLIHandler{},
		"ldap":     &credLdap.CLIHandler{},
		"oidc":     &credOIDC.CLIHandler{},
		"radius": &credUserpass.CLIHandler{
			DefaultMount: "radius",
		},
		"token": &credToken.CLIHandler{},
		"userpass": &credUserpass.CLIHandler{
			DefaultMount: "userpass",
		},
	}

	getBaseCommand := func() *BaseCommand {
		return &BaseCommand{
			UI:          ui,
			tokenHelper: runOpts.TokenHelper,
			flagAddress: runOpts.Address,
			client:      runOpts.Client,
		}
	}

	commands := getCommandsForBinary(ui, serverCmdUi, runOpts, getBaseCommand, loginHandlers)
	return commands
}

// MakeShutdownCh returns a channel that can be used for shutdown
// notifications for commands. This channel will send a message for every
// SIGINT or SIGTERM received.
func MakeShutdownCh() chan struct{} {
	resultCh := make(chan struct{})

	shutdownCh := make(chan os.Signal, 4)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownCh
		close(resultCh)
	}()
	return resultCh
}

// MakeSighupCh returns a channel that can be used for SIGHUP
// reloading. This channel will send a message for every
// SIGHUP received.
func MakeSighupCh() chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, syscall.SIGHUP)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()
	return resultCh
}
