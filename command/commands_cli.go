// Copyright (c) OpenBao, Inc.
// SPDX-License-Identifier: MPL-2.0

// See note in commands_unified.go; this file is for cli builds.
//go:build !agent && cli && !proxy && !server

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
		"audit": func() (cli.Command, error) {
			return &AuditCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"audit disable": func() (cli.Command, error) {
			return &AuditDisableCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"audit enable": func() (cli.Command, error) {
			return &AuditEnableCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"audit list": func() (cli.Command, error) {
			return &AuditListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"auth tune": func() (cli.Command, error) {
			return &AuthTuneCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"auth": func() (cli.Command, error) {
			return &AuthCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"auth disable": func() (cli.Command, error) {
			return &AuthDisableCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"auth enable": func() (cli.Command, error) {
			return &AuthEnableCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"auth help": func() (cli.Command, error) {
			return &AuthHelpCommand{
				BaseCommand: getBaseCommand(),
				Handlers:    loginHandlers,
			}, nil
		},
		"auth list": func() (cli.Command, error) {
			return &AuthListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"auth move": func() (cli.Command, error) {
			return &AuthMoveCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"debug": func() (cli.Command, error) {
			return &DebugCommand{
				BaseCommand: getBaseCommand(),
				ShutdownCh:  MakeShutdownCh(),
			}, nil
		},
		"delete": func() (cli.Command, error) {
			return &DeleteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"lease": func() (cli.Command, error) {
			return &LeaseCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"lease renew": func() (cli.Command, error) {
			return &LeaseRenewCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"lease lookup": func() (cli.Command, error) {
			return &LeaseLookupCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"lease revoke": func() (cli.Command, error) {
			return &LeaseRevokeCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"list": func() (cli.Command, error) {
			return &ListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"login": func() (cli.Command, error) {
			return &LoginCommand{
				BaseCommand: getBaseCommand(),
				Handlers:    loginHandlers,
			}, nil
		},
		"namespace": func() (cli.Command, error) {
			return &NamespaceCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"namespace list": func() (cli.Command, error) {
			return &NamespaceListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"namespace lookup": func() (cli.Command, error) {
			return &NamespaceLookupCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"namespace create": func() (cli.Command, error) {
			return &NamespaceCreateCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"namespace patch": func() (cli.Command, error) {
			return &NamespacePatchCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"namespace delete": func() (cli.Command, error) {
			return &NamespaceDeleteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"namespace lock": func() (cli.Command, error) {
			return &NamespaceAPILockCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"namespace unlock": func() (cli.Command, error) {
			return &NamespaceAPIUnlockCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator": func() (cli.Command, error) {
			return &OperatorCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator generate-root": func() (cli.Command, error) {
			return &OperatorGenerateRootCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator init": func() (cli.Command, error) {
			return &OperatorInitCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator key-status": func() (cli.Command, error) {
			return &OperatorKeyStatusCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator migrate": func() (cli.Command, error) {
			return &OperatorMigrateCommand{
				BaseCommand:      getBaseCommand(),
				PhysicalBackends: physicalBackends,
				ShutdownCh:       MakeShutdownCh(),
			}, nil
		},
		"operator raft": func() (cli.Command, error) {
			return &OperatorRaftCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft autopilot get-config": func() (cli.Command, error) {
			return &OperatorRaftAutopilotGetConfigCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft autopilot set-config": func() (cli.Command, error) {
			return &OperatorRaftAutopilotSetConfigCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft autopilot state": func() (cli.Command, error) {
			return &OperatorRaftAutopilotStateCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft list-peers": func() (cli.Command, error) {
			return &OperatorRaftListPeersCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft join": func() (cli.Command, error) {
			return &OperatorRaftJoinCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft remove-peer": func() (cli.Command, error) {
			return &OperatorRaftRemovePeerCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft snapshot": func() (cli.Command, error) {
			return &OperatorRaftSnapshotCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft snapshot restore": func() (cli.Command, error) {
			return &OperatorRaftSnapshotRestoreCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator raft snapshot save": func() (cli.Command, error) {
			return &OperatorRaftSnapshotSaveCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator rekey": func() (cli.Command, error) {
			return &OperatorRekeyCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator rotate": func() (cli.Command, error) {
			return &OperatorRotateCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator seal": func() (cli.Command, error) {
			return &OperatorSealCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator step-down": func() (cli.Command, error) {
			return &OperatorStepDownCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator unseal": func() (cli.Command, error) {
			return &OperatorUnsealCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"operator members": func() (cli.Command, error) {
			return &OperatorMembersCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"patch": func() (cli.Command, error) {
			return &PatchCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"path-help": func() (cli.Command, error) {
			return &PathHelpCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"pki": func() (cli.Command, error) {
			return &PKICommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"pki health-check": func() (cli.Command, error) {
			return &PKIHealthCheckCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"pki issue": func() (cli.Command, error) {
			return &PKIIssueCACommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"pki list-intermediates": func() (cli.Command, error) {
			return &PKIListIntermediateCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"pki reissue": func() (cli.Command, error) {
			return &PKIReIssueCACommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"pki verify-sign": func() (cli.Command, error) {
			return &PKIVerifySignCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"plugin": func() (cli.Command, error) {
			return &PluginCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"plugin deregister": func() (cli.Command, error) {
			return &PluginDeregisterCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"plugin info": func() (cli.Command, error) {
			return &PluginInfoCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"plugin list": func() (cli.Command, error) {
			return &PluginListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"plugin register": func() (cli.Command, error) {
			return &PluginRegisterCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"plugin reload": func() (cli.Command, error) {
			return &PluginReloadCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"plugin reload-status": func() (cli.Command, error) {
			return &PluginReloadStatusCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"policy": func() (cli.Command, error) {
			return &PolicyCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"policy delete": func() (cli.Command, error) {
			return &PolicyDeleteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"policy fmt": func() (cli.Command, error) {
			return &PolicyFmtCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"policy list": func() (cli.Command, error) {
			return &PolicyListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"policy read": func() (cli.Command, error) {
			return &PolicyReadCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"policy write": func() (cli.Command, error) {
			return &PolicyWriteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"print": func() (cli.Command, error) {
			return &PrintCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"print token": func() (cli.Command, error) {
			return &PrintTokenCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"read": func() (cli.Command, error) {
			return &ReadCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"secrets": func() (cli.Command, error) {
			return &SecretsCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"secrets disable": func() (cli.Command, error) {
			return &SecretsDisableCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"secrets enable": func() (cli.Command, error) {
			return &SecretsEnableCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"secrets list": func() (cli.Command, error) {
			return &SecretsListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"secrets move": func() (cli.Command, error) {
			return &SecretsMoveCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"secrets tune": func() (cli.Command, error) {
			return &SecretsTuneCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"ssh": func() (cli.Command, error) {
			return &SSHCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &StatusCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"transform": func() (cli.Command, error) {
			return &TransformCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"transform import": func() (cli.Command, error) {
			return &TransformImportCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"transform import-version": func() (cli.Command, error) {
			return &TransformImportVersionCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"transit": func() (cli.Command, error) {
			return &TransitCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"transit import": func() (cli.Command, error) {
			return &TransitImportCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"transit import-version": func() (cli.Command, error) {
			return &TransitImportVersionCommand{
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
		"unwrap": func() (cli.Command, error) {
			return &UnwrapCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &VersionCommand{
				VersionInfo: version.GetVersion(),
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"version-history": func() (cli.Command, error) {
			return &VersionHistoryCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"write": func() (cli.Command, error) {
			return &WriteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv": func() (cli.Command, error) {
			return &KVCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv put": func() (cli.Command, error) {
			return &KVPutCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv patch": func() (cli.Command, error) {
			return &KVPatchCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv rollback": func() (cli.Command, error) {
			return &KVRollbackCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv get": func() (cli.Command, error) {
			return &KVGetCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv delete": func() (cli.Command, error) {
			return &KVDeleteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv list": func() (cli.Command, error) {
			return &KVListCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv destroy": func() (cli.Command, error) {
			return &KVDestroyCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv undelete": func() (cli.Command, error) {
			return &KVUndeleteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv enable-versioning": func() (cli.Command, error) {
			return &KVEnableVersioningCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv metadata": func() (cli.Command, error) {
			return &KVMetadataCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv metadata put": func() (cli.Command, error) {
			return &KVMetadataPutCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv metadata patch": func() (cli.Command, error) {
			return &KVMetadataPatchCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv metadata get": func() (cli.Command, error) {
			return &KVMetadataGetCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"kv metadata delete": func() (cli.Command, error) {
			return &KVMetadataDeleteCommand{
				BaseCommand: getBaseCommand(),
			}, nil
		},
		"monitor": func() (cli.Command, error) {
			return &MonitorCommand{
				BaseCommand: getBaseCommand(),
				ShutdownCh:  MakeShutdownCh(),
			}, nil
		},
	}

	return commands
}
