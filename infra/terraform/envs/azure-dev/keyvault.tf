data "azurerm_client_config" "current" {}

resource "azurerm_key_vault" "main" {
  name                       = var.kv_name
  location                   = azurerm_resource_group.main.location
  resource_group_name        = azurerm_resource_group.main.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  enable_rbac_authorization  = true # access via Azure RBAC roles, not legacy access policies
  soft_delete_retention_days = 7
  purge_protection_enabled   = false # dev convenience; enable in production
  tags                       = local.tags
}

# Whoever runs Terraform may write secrets (needed to create the secrets below).
# RBAC can take a minute to propagate: if the first apply fails on a secret, run it again.
resource "azurerm_role_assignment" "kv_admin" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Secrets Officer"
  principal_id         = data.azurerm_client_config.current.object_id
}

# API token: created with a deliberately INVALID placeholder (shorter than the 16-char minimum, so the
# app refuses to start until you set a real value). Set it out-of-band so it never lands in state:
#   az keyvault secret set --vault-name <kv_name> --name sentinel-api-token --value "$(openssl rand -hex 24)"
resource "azurerm_key_vault_secret" "api_token" {
  name         = "sentinel-api-token"
  value        = "CHANGE-ME"
  key_vault_id = azurerm_key_vault.main.id

  lifecycle {
    ignore_changes = [value]
  }

  depends_on = [azurerm_role_assignment.kv_admin]
}

# Full DATABASE_URL for the Flexible Server. NOTE: this value (it contains the generated DB password)
# is stored in Terraform state; the state container is encrypted and access-controlled, but treat it as secret.
resource "azurerm_key_vault_secret" "database_url" {
  count        = var.enable_postgres ? 1 : 0
  name         = "sentinel-database-url"
  value        = "postgres://sentinel:${random_password.db[0].result}@${azurerm_postgresql_flexible_server.main[0].fqdn}:5432/sentinel?sslmode=require"
  key_vault_id = azurerm_key_vault.main.id

  depends_on = [azurerm_role_assignment.kv_admin]
}

# Slack incoming-webhook URL for Alertmanager (placeholder until you set the real one):
#   az keyvault secret set --vault-name <kv_name> --name alertmanager-slack-webhook --value "https://hooks.slack.com/services/..."
resource "azurerm_key_vault_secret" "slack_webhook" {
  name         = "alertmanager-slack-webhook"
  value        = "https://example.invalid/replace-me"
  key_vault_id = azurerm_key_vault.main.id

  lifecycle {
    ignore_changes = [value]
  }

  depends_on = [azurerm_role_assignment.kv_admin]
}
