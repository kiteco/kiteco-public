output "vpc-name" {
  value = "${azurerm_virtual_network.dev.name}"
}

output "app-rg" {
  value = "${azurerm_resource_group.dev.name}"
}

output "subnet-public" {
  value = "${azurerm_subnet.subnet_public.name}"
}

output "subnet-private" {
  value = "${azurerm_subnet.subnet_private.id}"
}

output "sg-ssh-id" {
  value = "${azurerm_network_security_group.sg_ssh.id}"
}

output "sg-usernode-id" {
  value = "${azurerm_network_security_group.sg_usernode.id}"
}

output "sg-usernode-debug-id" {
  value = "${azurerm_network_security_group.sg_usernode_debug.id}"
}

output "sg-http-id" {
  value = "${azurerm_network_security_group.sg_http.id}"
}

output "sg-https-id" {
  value = "${azurerm_network_security_group.sg_https.id}"
}

output "vhd-container-path" {
  value = "${azurerm_storage_account.disks.primary_blob_endpoint}${azurerm_storage_container.disks.name}/${var.image_name_base}"
}

output "vhd-container-name" {
  value = "${azurerm_storage_container.disks.name}"
}

output "provisioning-container-path" {
  value = "${azurerm_storage_account.disks.primary_blob_endpoint}${azurerm_storage_container.provision.name}"
}

output "provisioning-container-name" {
  value = "${azurerm_storage_container.provision.name}"
}

output "vhd-storage-name" {
  value = "${azurerm_storage_account.disks.name}"
}
