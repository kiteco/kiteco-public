resource "azurerm_network_interface" "ml-training" {
  name                      = "ml-training-${count.index}"
  location                  = "${var.region}"
  resource_group_name       = "${azurerm_resource_group.dev.name}"
  network_security_group_id = "${azurerm_network_security_group.sg_test.id}"
  count                     = "${var.ml_training_instance_count}"

  ip_configuration {
    name                          = "ml-training"
    subnet_id                     = "${azurerm_subnet.subnet_private.id}"
    private_ip_address            = "${var.vm_ip_prefix["${var.region}.ml-training"]}${format("%03d", count.index + 150)}"
    private_ip_address_allocation = "static"
  }
}

resource "azurerm_virtual_machine" "ml-training" {
  name                = "ml-training-${count.index}"
  location            = "${var.region}"
  resource_group_name = "${azurerm_resource_group.dev.name}"

  network_interface_ids        = ["${element(azurerm_network_interface.ml-training.*.id, count.index)}"]
  primary_network_interface_id = "${element(azurerm_network_interface.ml-training.*.id, count.index)}"
  vm_size                      = "standard_ds4_v2"
  count                        = "${var.ml_training_instance_count}"

  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name          = "ml-os-disk-${count.index}"
    caching       = "ReadWrite"
    create_option = "FromImage"

    create_option     = "FromImage"
    managed_disk_type = "Standard_LRS"
  }

  storage_data_disk {
    name          = "ml-data-${count.index}"
    create_option = "Empty"
    disk_size_gb  = "512"
    lun           = "1"

    managed_disk_type = "Premium_LRS"
  }

  os_profile {
    computer_name  = "ml-training-${var.region}-dev-${count.index}"
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
