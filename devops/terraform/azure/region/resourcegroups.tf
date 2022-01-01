provider "azurerm" {
  version         = "~> 1.3"
  subscription_id = "${var.az_subscription_id}"
  client_id       = "${var.az_client_id}"
  client_secret   = "${var.az_client_secret}"
  tenant_id       = "${var.az_tenant_id}"
}

resource "azurerm_resource_group" "prod" {
  name     = "${var.resourcegroup_name[var.region]}"
  location = "${var.region}"
}
