# Optional managed PostgreSQL with PRIVATE access only (delegated subnet + private DNS).

resource "random_password" "db" {
  count   = var.enable_postgres ? 1 : 0
  length  = 32
  special = false # avoids URL-encoding problems in DATABASE_URL
}

resource "azurerm_subnet" "db" {
  count                = var.enable_postgres ? 1 : 0
  name                 = "postgres"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.20.16.0/24"]

  delegation {
    name = "flexible-server"

    service_delegation {
      name    = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

resource "azurerm_private_dns_zone" "postgres" {
  count               = var.enable_postgres ? 1 : 0
  name                = "${local.name}.private.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  count                 = var.enable_postgres ? 1 : 0
  name                  = "${local.name}-postgres"
  resource_group_name   = azurerm_resource_group.main.name
  private_dns_zone_name = azurerm_private_dns_zone.postgres[0].name
  virtual_network_id    = azurerm_virtual_network.main.id
}

resource "azurerm_postgresql_flexible_server" "main" {
  count               = var.enable_postgres ? 1 : 0
  name                = "${local.name}-pg"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  version                = "16"
  sku_name               = "B_Standard_B1ms"
  storage_mb             = 32768
  backup_retention_days  = 7
  administrator_login    = "sentinel"
  administrator_password = random_password.db[0].result

  delegated_subnet_id = azurerm_subnet.db[0].id
  private_dns_zone_id = azurerm_private_dns_zone.postgres[0].id

  tags = local.tags

  depends_on = [azurerm_private_dns_zone_virtual_network_link.postgres]

  lifecycle {
    ignore_changes = [zone] # Azure picks a zone; avoids perpetual drift
  }
}

resource "azurerm_postgresql_flexible_server_database" "sentinel" {
  count     = var.enable_postgres ? 1 : 0
  name      = "sentinel"
  server_id = azurerm_postgresql_flexible_server.main[0].id
  charset   = "UTF8"
  collation = "en_US.utf8"
}
