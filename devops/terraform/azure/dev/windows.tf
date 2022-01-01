resource "azurerm_public_ip" "recruiting" {
  name                         = "recruiting-ip"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.dev.name}"
  public_ip_address_allocation = "static"
}

resource "azurerm_network_interface" "recruiting" {
  name                      = "recruiting"
  location                  = "${var.region}"
  resource_group_name       = "${azurerm_resource_group.dev.name}"
  network_security_group_id = "${azurerm_network_security_group.sg_rdp.id}"

  ip_configuration {
    name                          = "recruiting"
    subnet_id                     = "${azurerm_subnet.subnet_public.id}"
    public_ip_address_id          = "${azurerm_public_ip.recruiting.id}"
    private_ip_address_allocation = "static"
    private_ip_address            = "${var.vm_ip_list["${var.region}.recruiting"]}"
  }
}

resource "azurerm_virtual_machine" "recruiting" {
  name                         = "recruiting"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.dev.name}"
  network_interface_ids        = ["${azurerm_network_interface.recruiting.id}"]
  primary_network_interface_id = "${azurerm_network_interface.recruiting.id}"
  vm_size                      = "Standard_B2s"

  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2016-Datacenter"
    version   = "latest"
  }

  storage_os_disk {
    name    = "os-disk"
    vhd_uri = "${azurerm_storage_account.disks.primary_blob_endpoint}${azurerm_storage_container.disks.name}/${var.image_name_recruiting}"

    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "recruiting"
    admin_username = "kite"
    admin_password = "${var.recruiting_password}"
  }

  os_profile_windows_config {
    provision_vm_agent = true
  }
}
