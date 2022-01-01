output "vpc-name" {
  value = "${azurerm_virtual_network.prod.name}"
}

output "app-rg" {
  value = "${azurerm_resource_group.prod.name}"
}

output "subnet-agw-public" {
  value = "${azurerm_subnet.subnet_agw_public.name}"
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

output "ip-prod-app-gateway" {
  value = "${azurerm_public_ip.prod_app_gateway.name}"
}

output "ip-staging-app-gateway" {
  value = "${azurerm_public_ip.staging_app_gateway.name}"
}

output "vhd-container-path" {
  value = "${azurerm_storage_account.proddisks.primary_blob_endpoint}${azurerm_storage_container.proddisks.name}/${var.image_name_base}"
}

output "vhd-container-name" {
  value = "${azurerm_storage_container.proddisks.name}"
}

output "provisioning-container-path" {
  value = "${azurerm_storage_account.proddisks.primary_blob_endpoint}${azurerm_storage_container.prodprovision.name}"
}

output "provisioning-container-name" {
  value = "${azurerm_storage_container.prodprovision.name}"
}

output "vhd-storage-name" {
  value = "${azurerm_storage_account.proddisks.name}"
}

output "agw-prod-name" {
  value = "${var.agw_prod_name}"
}

output "agw-staging-name" {
  value = "${var.agw_staging_name}"
}

output "agw-prod-pool" {
  value = "${var.agw_prod_name}-pool"
}

output "agw-staging-pool" {
  value = "${var.agw_staging_name}-pool"
}
