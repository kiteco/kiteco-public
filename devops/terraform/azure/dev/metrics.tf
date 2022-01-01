resource "azurerm_network_interface" "metrics" {
  name                      = "metrics-dev"
  location                  = "${var.region}"
  resource_group_name       = "${azurerm_resource_group.dev.name}"
  network_security_group_id = "${azurerm_network_security_group.sg_test.id}"

  ip_configuration {
    name                          = "metrics"
    subnet_id                     = "${azurerm_subnet.subnet_private.id}"
    private_ip_address_allocation = "static"
    private_ip_address            = "${var.vm_ip_list["${var.region}.metrics"]}"
  }
}

resource "azurerm_virtual_machine" "metrics" {
  name                         = "metrics"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.dev.name}"
  network_interface_ids        = ["${azurerm_network_interface.metrics.id}"]
  primary_network_interface_id = "${azurerm_network_interface.metrics.id}"
  vm_size                      = "Standard_A1"

  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name          = "os-disk"
    vhd_uri       = "${azurerm_storage_account.disks.primary_blob_endpoint}${azurerm_storage_container.disks.name}/${var.image_name_metrics}"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "metrics-${var.region}-dev"
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
