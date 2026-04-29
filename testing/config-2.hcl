ui            = true
cluster_addr  = "https://127.0.0.1:8301"
api_addr      = "http://127.0.0.1:8300"
log_level = "trace"
raw_storage_endpoint = true
enable_grpc_invalidation = true

storage "postgresql" {
  connection_url = "postgres://username:password@127.0.0.1:5432/postgres"
  max_connection_retries = 50
  ha_enabled = true
}

listener "tcp" {
  address       = "127.0.0.1:8300"
  tls_disable   = true
}

seal "static" {
  current_key_id = "20250630"
  current_key = "542a91c2aff86797e8dfa16adf86dcbea5706410791b319798936a1b08e4ef8b"
}

audit "file" "to-stdout" {
  description = "This audit device should never fail."
  options {
    file_path = "/dev/stdout"
    log_raw = "true"
  }
}

initialize "identity" {
  request "mountuserpass" {
    operation = "update"
    path = "sys/auth/userpass"
    data = {
      type = "userpass"
      path = "userpass/"
      description = "admin"
    }
  }

  request "userpassaddadmin" {
    operation = "update"
    path = "auth/userpass/users/admin"
    data = {
      "password" = "admin"
      "token_policies" = ["superuser"]
    }
  }
}

initialize "policy" {
  request "addsuperuserpolicy" {
    operation = "update"
    path = "sys/policies/acl/superuser"
    data = {
      policy = <<EOP
path "*" {
  capabilities = ["create", "update", "read", "delete", "list", "scan", "sudo"]
}
EOP
    }
  }
}

initialize "secrets" {
  request "mountkvv2" {
    operation = "update"
    path = "sys/mounts/secrets"
    data = {
      type = "kv-v2"
    }
  }
}
