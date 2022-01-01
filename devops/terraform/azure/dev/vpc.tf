resource "azurerm_virtual_network" "dev" {
  name                = "${var.azurerm_virtual_network_name}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
  address_space       = "${var.azurerm_virtual_network_address_space[var.region]}"
  location            = "${var.region}"

  # dns_servers         = ["10.0.0.4", "10.0.0.5"]
}

resource "azurerm_subnet" "subnet_public" {
  name                 = "subnet-public"
  resource_group_name  = "${azurerm_resource_group.dev.name}"
  virtual_network_name = "${azurerm_virtual_network.dev.name}"
  address_prefix       = "${var.subnet_public_address_space[var.region]}"
  route_table_id       = "${azurerm_route_table.public_routes.id}"

  # network_security_group_id = "${azurerm_network_security_group.sg_deny.id}"
}

resource "azurerm_route_table" "public_routes" {
  name                = "routes-dev"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_route" "forward-to-other-region-public" {
  name                = "forward-to-other-region-${count.index}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
  route_table_name    = "${azurerm_route_table.public_routes.name}"

  address_prefix         = "${element(var.other_vnet_region_addresses, count.index)}"
  next_hop_type          = "virtualappliance"
  next_hop_in_ip_address = "${var.vm_ip_list["${var.region}.vpn-tunnel"]}"
  count                  = "${length(var.other_vnet_region_addresses)}"
}

resource "azurerm_route" "forward-to-vpn-public" {
  name                = "forward-to-vpn-clients"
  resource_group_name = "${azurerm_resource_group.dev.name}"
  route_table_name    = "${azurerm_route_table.public_routes.name}"

  address_prefix         = "10.45.0.0/16"
  next_hop_type          = "virtualappliance"
  next_hop_in_ip_address = "${var.vm_ip_list["${var.region}.vpn"]}"
}

resource "azurerm_subnet" "subnet_private" {
  name                 = "subnet-private"
  resource_group_name  = "${azurerm_resource_group.dev.name}"
  virtual_network_name = "${azurerm_virtual_network.dev.name}"
  address_prefix       = "${var.subnet_private_address_space[var.region]}"
  route_table_id       = "${azurerm_route_table.private_routes.id}"

  service_endpoints    = ["Microsoft.Storage"]
  # network_security_group_id = "${azurerm_network_security_group.sg_deny.id}"
  #TODO: add route tables
}

resource "azurerm_route_table" "private_routes" {
  name                = "routes-dev-private"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_route" "forward-to-other-region-private" {
  name                = "forward-to-other-region-${count.index}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
  route_table_name    = "${azurerm_route_table.private_routes.name}"

  address_prefix         = "${element(var.other_vnet_region_addresses, count.index)}"
  next_hop_type          = "virtualappliance"
  next_hop_in_ip_address = "${var.vm_ip_list["${var.region}.vpn-tunnel"]}"
  count                  = "${length(var.other_vnet_region_addresses)}"
}

resource "azurerm_route" "forward-to-vpn-private" {
  name                = "forward-to-vpn-clients"
  resource_group_name = "${azurerm_resource_group.dev.name}"
  route_table_name    = "${azurerm_route_table.private_routes.name}"

  address_prefix         = "10.45.0.0/16"
  next_hop_type          = "virtualappliance"
  next_hop_in_ip_address = "${var.vm_ip_list["${var.region}.vpn"]}"
}
