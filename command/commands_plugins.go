// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !agent && !proxy

package command

import (
	"github.com/openbao/openbao/audit"
	"github.com/openbao/openbao/builtin/plugin"
	"github.com/openbao/openbao/sdk/logical"
	"github.com/openbao/openbao/sdk/physical"

	/*
		The builtinplugins package is initialized here because it, in turn,
		initializes the database plugins.
		They register multiple database drivers for the "database/sql" package.
	*/
	_ "github.com/openbao/openbao/helper/builtinplugins"

	auditFile "github.com/openbao/openbao/builtin/audit/file"
	auditSocket "github.com/openbao/openbao/builtin/audit/socket"
	auditSyslog "github.com/openbao/openbao/builtin/audit/syslog"

	logicalDb "github.com/openbao/openbao/builtin/logical/database"
	logicalKv "github.com/openbao/openbao/builtin/logical/kv"

	physRaft "github.com/openbao/openbao/physical/raft"
	physFile "github.com/openbao/openbao/sdk/physical/file"
	physInmem "github.com/openbao/openbao/sdk/physical/inmem"

	sr "github.com/openbao/openbao/serviceregistration"
	ksr "github.com/openbao/openbao/serviceregistration/kubernetes"
)

var (
	auditBackends = map[string]audit.Factory{
		"file":   auditFile.Factory,
		"socket": auditSocket.Factory,
		"syslog": auditSyslog.Factory,
	}

	credentialBackends = map[string]logical.Factory{
		"plugin": plugin.Factory,
	}

	logicalBackends = map[string]logical.Factory{
		"plugin":   plugin.Factory,
		"database": logicalDb.Factory,
		// This is also available in the plugin catalog, but is here due to the need to
		// automatically mount it.
		"kv": logicalKv.Factory,
	}

	physicalBackends = map[string]physical.Factory{
		"file_transactional":     physFile.NewTransactionalFileBackend,
		"file":                   physFile.NewFileBackend,
		"inmem_ha":               physInmem.NewInmemHA,
		"inmem_transactional_ha": physInmem.NewTransactionalInmemHA,
		"inmem_transactional":    physInmem.NewTransactionalInmem,
		"inmem":                  physInmem.NewInmem,
		"raft":                   physRaft.NewRaftBackend,
	}

	serviceRegistrations = map[string]sr.Factory{
		"kubernetes": ksr.NewServiceRegistration,
	}
)
