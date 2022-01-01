provider "azurerm" {
  subscription_id = "${var.az_subscription_id}"
  client_id       = "${var.az_client_id}"
  client_secret   = "${var.az_client_secret}"
  tenant_id       = "${var.az_tenant_id}"
  version         = "~> 1.3"
}

resource "azurerm_resource_group" "dev" {
  name     = "${var.resourcegroup_name[var.region]}"
  location = "${var.region}"
}
