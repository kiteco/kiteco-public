resource "azurerm_storage_account" "proddisks" {
  name                = "${var.storagedisks_acct_names[var.region]}"
  resource_group_name = "${azurerm_resource_group.prod.name}"

  location                 = "${var.region}"
  account_tier             = "Standard"
  account_replication_type = "LRS"
  enable_blob_encryption   = "true"
}

resource "azurerm_storage_container" "proddisks" {
  name                  = "vhds"
  resource_group_name   = "${azurerm_resource_group.prod.name}"
  storage_account_name  = "${azurerm_storage_account.proddisks.name}"
  container_access_type = "private"
}

resource "azurerm_storage_container" "prodprovision" {
  name                  = "provisioning"
  resource_group_name   = "${azurerm_resource_group.prod.name}"
  storage_account_name  = "${azurerm_storage_account.proddisks.name}"
  container_access_type = "private"
}

resource "azurerm_public_ip" "bastion" {
  name                         = "bastion-ip"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  public_ip_address_allocation = "static"
}

resource "azurerm_network_interface" "bastion" {
  name                      = "bastion-prod"
  location                  = "${var.region}"
  resource_group_name       = "${azurerm_resource_group.prod.name}"
  network_security_group_id = "${azurerm_network_security_group.sg_ssh.id}"

  ip_configuration {
    name                          = "bastion"
    subnet_id                     = "${azurerm_subnet.subnet_public.id}"
    public_ip_address_id          = "${azurerm_public_ip.bastion.id}"
    private_ip_address_allocation = "static"
    private_ip_address            = "${var.vm_ip_list["${var.region}.bastion"]}"
  }
}

resource "azurerm_virtual_machine" "bastion" {
  name                         = "bastion"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  network_interface_ids        = ["${azurerm_network_interface.bastion.id}"]
  primary_network_interface_id = "${azurerm_network_interface.bastion.id}"
  vm_size                      = "Standard_A0"

  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name          = "os-disk"
    vhd_uri       = "${azurerm_storage_account.proddisks.primary_blob_endpoint}${azurerm_storage_container.proddisks.name}/${var.image_name_bastion}"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "bastion-${var.region}-prod"
    admin_username = "ubuntu"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/ubuntu/.ssh/authorized_keys"
      key_data = "${var.ssh_pubkey}"
    }

    ssh_keys {
      path     = "/home/ubuntu/.ssh/authorized_keys"
      key_data = "${var.ssh_pubkey_1}"
    }
  }
}

resource "azurerm_public_ip" "vpn-tunnel" {
  name                         = "vpn-tunnel-ip"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  public_ip_address_allocation = "static"
}

resource "azurerm_network_interface" "vpn-tunnel" {
  name                      = "vpn-tunnel-prod"
  location                  = "${var.region}"
  resource_group_name       = "${azurerm_resource_group.prod.name}"
  network_security_group_id = "${azurerm_network_security_group.sg_vpn_tunnel.id}"
  enable_ip_forwarding      = true

  ip_configuration {
    name                          = "vpn-tunnel"
    subnet_id                     = "${azurerm_subnet.subnet_public.id}"
    public_ip_address_id          = "${azurerm_public_ip.vpn-tunnel.id}"
    private_ip_address_allocation = "static"
    private_ip_address            = "${var.vm_ip_list["${var.region}.vpn-tunnel"]}"
  }
}

resource "azurerm_virtual_machine" "vpn-tunnel" {
  name                         = "vpn-tunnel"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  network_interface_ids        = ["${azurerm_network_interface.vpn-tunnel.id}"]
  primary_network_interface_id = "${azurerm_network_interface.vpn-tunnel.id}"
  vm_size                      = "Standard_A0"

  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name          = "os-disk"
    vhd_uri       = "${azurerm_storage_account.proddisks.primary_blob_endpoint}${azurerm_storage_container.proddisks.name}/${var.image_name_vpn}"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "vpn-tunnel"
    admin_username = "ubuntu"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/ubuntu/.ssh/authorized_keys"
      key_data = "${var.ssh_pubkey}"
    }

    ssh_keys {
      path     = "/home/ubuntu/.ssh/authorized_keys"
      key_data = "${var.ssh_pubkey_1}"
    }
  }
}
