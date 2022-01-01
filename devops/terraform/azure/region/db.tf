# https://github.com/terraform-providers/terraform-provider-azurerm/pull/210/files
# currently requires building azurerm provider from source

resource "azurerm_postgresql_server" "localfiles" {
  name                = "${var.db_name[var.region]}"
  location            = "${var.db_location[var.region]}"
  resource_group_name = "${azurerm_resource_group.prod.name}"

  sku {
    name     = "PGSQLS200"
    capacity = 200
    tier     = "Standard"
  }

  administrator_login          = "${var.localfiles_db_username}"
  administrator_login_password = "${var.localfiles_db_password}"
  version                      = "9.6"
  storage_mb                   = 512000
  ssl_enforcement              = "Enabled"
}

resource "azurerm_postgresql_database" "localfiles" {
  name                = "${var.localfiles_db_name}"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  server_name         = "${azurerm_postgresql_server.localfiles.name}"
  charset             = "UTF8"
  collation           = "English_United States.1252"
}

resource "azurerm_postgresql_firewall_rule" "localfiles" {
  name                = "localfiles-db-allow-inet"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  server_name         = "${azurerm_postgresql_server.localfiles.name}"

  # start_ip_address    = "${var.azurerm_virtual_network_address_space_start[var.region]}"
  # end_ip_address      = "${var.azurerm_virtual_network_address_space_end[var.region]}"
  start_ip_address = "0.0.0.0"

  end_ip_address = "255.255.255.255"
}

resource "azurerm_postgresql_configuration" "stmt_timeout" {
  name                = "statement_timeout"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  server_name         = "${azurerm_postgresql_server.localfiles.name}"
  value               = "300000"
}

resource "azurerm_postgresql_configuration" "log_min_messages" {
  name                = "log_min_messages"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  server_name         = "${azurerm_postgresql_server.localfiles.name}"
  value               = "PANIC"
}

resource "azurerm_postgresql_configuration" "log_min_error_statement" {
  name                = "log_min_error_statement"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  server_name         = "${azurerm_postgresql_server.localfiles.name}"
  value               = "PANIC"
}
