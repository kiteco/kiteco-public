resource "azurerm_virtual_network" "prod" {
  name                = "${var.azurerm_virtual_network_name}"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  address_space       = "${var.azurerm_virtual_network_address_space[var.region]}"
  location            = "${var.region}"

  # dns_servers         = ["10.0.0.4", "10.0.0.5"]
}

resource "azurerm_subnet" "subnet_agw_public" {
  name                 = "subnet-agw-public"
  resource_group_name  = "${azurerm_resource_group.prod.name}"
  virtual_network_name = "${azurerm_virtual_network.prod.name}"
  address_prefix       = "${var.subnet_agw_public_address_space[var.region]}"

  # network_security_group_id = "${azurerm_network_security_group.sg_deny.id}"
}

resource "azurerm_subnet" "subnet_public" {
  name                 = "subnet-public"
  resource_group_name  = "${azurerm_resource_group.prod.name}"
  virtual_network_name = "${azurerm_virtual_network.prod.name}"
  address_prefix       = "${var.subnet_public_address_space[var.region]}"
  route_table_id       = "${azurerm_route_table.public_routes.id}"

  # network_security_group_id = "${azurerm_network_security_group.sg_deny.id}"
}

resource "azurerm_route_table" "public_routes" {
  name                = "routes-prod"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.prod.name}"

  route {
    name                   = "02_forward-to-region-dev"
    address_prefix         = "10.47.0.0/16"
    next_hop_type          = "virtualappliance"
    next_hop_in_ip_address = "${var.vm_ip_list["${var.region}.vpn-tunnel"]}"
  }

  # TODO not sure if this rule actually works, need to test
  route {
    name                   = "03_forward-to-vpn-clients"
    address_prefix         = "10.45.0.0/16"
    next_hop_type          = "virtualappliance"
    next_hop_in_ip_address = "${var.vm_ip_list["dev.vpn"]}"
  }
}

resource "azurerm_subnet" "subnet_private" {
  name                 = "subnet-private"
  resource_group_name  = "${azurerm_resource_group.prod.name}"
  virtual_network_name = "${azurerm_virtual_network.prod.name}"
  address_prefix       = "${var.subnet_private_address_space[var.region]}"
  route_table_id       = "${azurerm_route_table.private_routes.id}"

  service_endpoints    = ["Microsoft.Storage"]

  # network_security_group_id = "${azurerm_network_security_group.sg_deny.id}"
  #TODO: add route tables
}

resource "azurerm_route_table" "private_routes" {
  name                = "routes-dev-private"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.prod.name}"

  route {
    name                   = "02_forward-to-region-dev"
    address_prefix         = "10.47.0.0/16"
    next_hop_type          = "virtualappliance"
    next_hop_in_ip_address = "${var.vm_ip_list["${var.region}.vpn-tunnel"]}"
  }

  route {
    name                   = "03_forward-to-vpn-clients"
    address_prefix         = "10.45.0.0/16"
    next_hop_type          = "virtualappliance"
    next_hop_in_ip_address = "${var.vm_ip_list["dev.vpn"]}"
  }
}

resource "azurerm_public_ip" "prod_app_gateway" {
  name                         = "prod-app-gateway-ip"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  public_ip_address_allocation = "dynamic"
}

resource "azurerm_public_ip" "staging_app_gateway" {
  name                         = "staging-app-gateway-ip"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  public_ip_address_allocation = "dynamic"
}
