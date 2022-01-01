#allow all
resource "azurerm_network_security_group" "sg_all" {
  name                = "sg-all"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_all_inbound" {
  name                       = "allow all inbound"
  priority                   = 4096
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "*"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_all.name}"
}

# ssh only traffic
resource "azurerm_network_security_group" "sg_ssh" {
  name                = "sg-ssh"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_ssh_inbound_ssh" {
  name                       = "ssh allow ssh inbound"
  priority                   = 600
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "TCP"
  source_port_range          = "*"
  destination_port_range     = "22"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_ssh.name}"
}

# usernode traffic
resource "azurerm_network_security_group" "sg_usernode" {
  name                = "sg-usernode"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_usernode_inbound" {
  name                       = "allow all inbound"
  priority                   = 500
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "9090"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_usernode.name}"
}

# usernode debug traffic
resource "azurerm_network_security_group" "sg_usernode_debug" {
  name                = "sg-usernode-debug"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_usernode_debug_inbound" {
  name                       = "allow all inbound"
  priority                   = 400
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "9091"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_usernode_debug.name}"
}

# http traffic
resource "azurerm_network_security_group" "sg_http" {
  name                = "sg-http"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_http_inbound" {
  name                       = "allow all inbound"
  priority                   = 300
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "80"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_http.name}"
}

# http traffic
resource "azurerm_network_security_group" "sg_https" {
  name                = "sg-https"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_https_inbound" {
  name                       = "allow all inbound"
  priority                   = 200
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "443"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_https.name}"
}

# vpn traffic
resource "azurerm_network_security_group" "sg_vpn" {
  name                = "sg-vpn"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_vpn_inbound_0" {
  name                       = "allow vpn inbound 0"
  priority                   = 100
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "1194"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_vpn.name}"
}

# dns traffic
resource "azurerm_network_security_group" "sg_dns" {
  name                = "sg-dns"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_dns_inbound_0" {
  name                       = "allow dns inbound 0"
  priority                   = 800
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "53"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_dns.name}"
}

# vpn tunnel traffic
resource "azurerm_network_security_group" "sg_vpn_tunnel" {
  name                = "sg-vpn-tunnel"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_vpn_tunnel_inbound_0" {
  name                       = "allow vpn inbound 0"
  priority                   = 900
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "udp"
  source_port_range          = "*"
  destination_port_range     = "500"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_vpn_tunnel.name}"
}

resource "azurerm_network_security_rule" "sg_vpn_tunnel_inbound_1" {
  name                       = "allow vpn inbound 1"
  priority                   = 901
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "udp"
  source_port_range          = "*"
  destination_port_range     = "4500"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_vpn_tunnel.name}"
}

resource "azurerm_network_security_rule" "sg_vpn_tunnel_inbound_2" {
  name                       = "allow vpn inbound 2"
  priority                   = 902
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "tcp"
  source_port_range          = "*"
  destination_port_range     = "50"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_vpn_tunnel.name}"
}

resource "azurerm_network_security_rule" "sg_vpn_tunnel_inbound_3" {
  name                       = "allow vpn inbound 3"
  priority                   = 903
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "tcp"
  source_port_range          = "*"
  destination_port_range     = "51"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_vpn_tunnel.name}"
}

# test traffic
resource "azurerm_network_security_group" "sg_test" {
  name                = "sg-test"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_test_inbound" {
  name                       = "allow ssh inbound"
  priority                   = 1000
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "22"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_test.name}"
}

resource "azurerm_network_security_rule" "sg_test_inbound_1" {
  name                       = "allow all inbound from internal"
  priority                   = 1001
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "*"
  source_address_prefix      = "${var.subnet_public_address_space[var.region]}"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_test.name}"
}

resource "azurerm_network_security_rule" "sg_test_inbound_2" {
  name                       = "allow all inbound from internal private"
  priority                   = 1002
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "*"
  source_address_prefix      = "${var.subnet_private_address_space[var.region]}"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_test.name}"
}

# rdp traffic
resource "azurerm_network_security_group" "sg_rdp" {
  name                = "sg-rdp"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"
}

resource "azurerm_network_security_rule" "sg_rdp_inbound" {
  name                       = "allow rdp inbound"
  priority                   = 1000
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "TCP"
  source_port_range          = "*"
  destination_port_range     = "3389"
  source_address_prefix      = "*"
  destination_address_prefix = "*"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_rdp.name}"
}

resource "azurerm_network_security_rule" "sg_rdp_outbound" {
  name                       = "deny vnet outbound"
  priority                   = 1000
  direction                  = "Outbound"
  access                     = "Deny"
  protocol                   = "*"
  source_port_range          = "*"
  destination_port_range     = "*"
  source_address_prefix      = "VirtualNetwork"
  destination_address_prefix = "VirtualNetwork"

  resource_group_name         = "${azurerm_resource_group.dev.name}"
  network_security_group_name = "${azurerm_network_security_group.sg_rdp.name}"
}
