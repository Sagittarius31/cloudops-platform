output "resource_group" {
  value = azurerm_resource_group.main.name
}

output "cluster_name" {
  value = azurerm_kubernetes_cluster.main.name
}

output "acr_login_server" {
  value = azurerm_container_registry.main.login_server
}

output "configure_kubectl" {
  value = "az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${azurerm_kubernetes_cluster.main.name}"
}

output "tenant_id" {
  value = data.azurerm_client_config.current.tenant_id
}

output "subscription_id" {
  value = data.azurerm_client_config.current.subscription_id
}

output "key_vault_name" {
  value = azurerm_key_vault.main.name
}

output "key_vault_uri" {
  value = azurerm_key_vault.main.vault_uri
}

output "eso_client_id" {
  description = "Put in gitops/argocd/aks-apps/external-secrets.yaml and gitops/platform-aks."
  value       = azurerm_user_assigned_identity.eso.client_id
}

output "kyverno_client_id" {
  description = "Put in gitops/argocd/aks-apps/kyverno.yaml."
  value       = azurerm_user_assigned_identity.kyverno.client_id
}

output "github_client_id" {
  description = "Set as the AZURE_CLIENT_ID repository variable in GitHub."
  value       = azurerm_user_assigned_identity.github.client_id
}

output "postgres_fqdn" {
  value = var.enable_postgres ? azurerm_postgresql_flexible_server.main[0].fqdn : null
}
