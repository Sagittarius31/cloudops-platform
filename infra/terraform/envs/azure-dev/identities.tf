# Workload identities: Kubernetes service accounts / GitHub Actions get Azure permissions through
# federated credentials -- no client secrets anywhere.

# ---------- External Secrets Operator -> read Key Vault secrets ----------
resource "azurerm_user_assigned_identity" "eso" {
  name                = "${local.name}-eso"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  tags                = local.tags
}

resource "azurerm_federated_identity_credential" "eso" {
  name                = "external-secrets"
  resource_group_name = azurerm_resource_group.main.name
  parent_id           = azurerm_user_assigned_identity.eso.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = azurerm_kubernetes_cluster.main.oidc_issuer_url
  subject             = "system:serviceaccount:external-secrets:external-secrets"
}

resource "azurerm_role_assignment" "eso_kv_read" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.eso.principal_id
}

# ---------- Kyverno admission controller -> read ACR to verify image signatures ----------
resource "azurerm_user_assigned_identity" "kyverno" {
  name                = "${local.name}-kyverno"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  tags                = local.tags
}

resource "azurerm_federated_identity_credential" "kyverno" {
  name                = "kyverno-admission-controller"
  resource_group_name = azurerm_resource_group.main.name
  parent_id           = azurerm_user_assigned_identity.kyverno.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = azurerm_kubernetes_cluster.main.oidc_issuer_url
  subject             = "system:serviceaccount:kyverno:kyverno-admission-controller"
}

resource "azurerm_role_assignment" "kyverno_acr_pull" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.kyverno.principal_id
}

# ---------- GitHub Actions -> push images to ACR (OIDC federation) ----------
resource "azurerm_user_assigned_identity" "github" {
  name                = "${local.name}-github-ci"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  tags                = local.tags
}

resource "azurerm_federated_identity_credential" "github" {
  name                = "github-main"
  resource_group_name = azurerm_resource_group.main.name
  parent_id           = azurerm_user_assigned_identity.github.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = "https://token.actions.githubusercontent.com"
  subject             = "repo:${var.github_repo}:ref:refs/heads/main"
}

resource "azurerm_role_assignment" "github_acr_push" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPush"
  principal_id         = azurerm_user_assigned_identity.github.principal_id
}
