resource "azurerm_network_interface" "test-mgmt" {
  name                      = "test-dev-mgmt-${count.index}"
  location                  = "${var.region}"
  resource_group_name       = "${azurerm_resource_group.dev.name}"
  network_security_group_id = "${azurerm_network_security_group.sg_test.id}"
  count                     = "${var.test_instance_count}"

  ip_configuration {
    name                          = "test-dev-mgmt"
    subnet_id                     = "${azurerm_subnet.subnet_private.id}"
    private_ip_address            = "${var.vm_ip_prefix["${var.region}.test"]}${format("%02d", count.index)}"
    private_ip_address_allocation = "static"
  }
}

resource "azurerm_virtual_machine" "test" {
  name                = "test-${count.index}"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"

  network_interface_ids        = ["${element(azurerm_network_interface.test-mgmt.*.id, count.index)}"]
  primary_network_interface_id = "${element(azurerm_network_interface.test-mgmt.*.id, count.index)}"
  vm_size                      = "Standard_A4m_v2"
  count                        = "${var.test_instance_count}"

  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name          = "os-disk"
    vhd_uri       = "${azurerm_storage_account.disks.primary_blob_endpoint}${azurerm_storage_container.disks.name}/${var.image_name_test}-${count.index}.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"

    disk_size_gb = "80"
  }

  os_profile {
    computer_name  = "test-${var.region}-dev-${count.index}"
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
