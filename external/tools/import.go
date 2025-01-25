package tools

import (
	_ "github.com/openbao/openbao/external/v2/credential/aws"
	_ "github.com/openbao/openbao/external/v2/credential/github"
	_ "github.com/openbao/openbao/external/v2/credential/okta"
	_ "github.com/openbao/openbao/external/v2/database/hana"
	_ "github.com/openbao/openbao/external/v2/database/mongodb"
	_ "github.com/openbao/openbao/external/v2/database/mssql"
	_ "github.com/openbao/openbao/external/v2/database/redshift"
	_ "github.com/openbao/openbao/external/v2/helper/testhelpers/consul"
	_ "github.com/openbao/openbao/external/v2/helper/testhelpers/mongodb"
	_ "github.com/openbao/openbao/external/v2/helper/testhelpers/mssql"
	_ "github.com/openbao/openbao/external/v2/logical/aws"
	_ "github.com/openbao/openbao/external/v2/logical/consul"
	_ "github.com/openbao/openbao/external/v2/logical/nomad"
)

func init() {
	// This helper exists to force a dependency from the core Go module
	// onto external, allowing the main module to execute go tests within
	// this module without excessive hackery.
}
