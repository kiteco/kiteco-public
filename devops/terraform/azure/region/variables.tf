variable region {}

variable az_subscription_id {}

variable az_client_id {}

variable az_client_secret {}

variable az_tenant_id {}

variable localfiles_db_password {}

variable az_ssl_storage_key {}

variable az_ssl_storage_user {}

variable az_state_storage_key {}

variable az_state_storage_user {}

variable "localfiles_db_name" {
  default = "localfiles"
}

variable "localfiles_db_username" {
  default = "kite"
}

variable "image_name_bastion" {
  default = "bastion.vhd"
}

variable "image_name_vpn" {
  default = "vpn.vhd"
}

variable "image_name_haproxy" {
  default = "haproxy.vhd"
}

variable "image_name_base" {
  default = "base.vhd"
}

variable "agw_prod_name" {
  default = "agw-prod"
}

variable "agw_staging_name" {
  default = "agw-staging"
}

variable ssh_pubkey {
  # default = "ssh-rsa XXXXXXX"
  default = "XXXXXXX"
}

variable ssh_pubkey_1 {
  default = "XXXXXXX"
}

variable ssh_metrics_pubkey {
  default = "XXXXXXX"
}

variable db_location {
  default = {
    "westus"     = "westus"
    "westus2"    = "westus"
    "eastus"     = "eastus"
    "westeurope" = "westeurope"
  }
}

variable db_name {
  default = {
    "westus2"    = "localfiles-db-server"
    "eastus"     = "localfiles-db-server-east"
    "westeurope" = "localfiles-db-server-westeurope"
  }
}

variable "resourcegroup_name" {
  default = {
    "westus2"    = "prod-westus2-0"
    "eastus"     = "prod-eastus-0"
    "westeurope" = "prod-westeurope-0"
  }
}

variable "storagedisks_acct_names" {
  default = {
    "westus2"    = "kitestoragedisksprod2"
    "eastus"     = "XXXXXXX"
    "westeurope" = "XXXXXXX"
  }
}

variable "azurerm_virtual_network_name" {
  default = "prod"
}

variable "vm_ip_list" {
  default = {
    dev.vpn = "10.47.0.7" #this vm is on dev subnet

    westus2.bastion    = "10.46.0.10"
    westus2.vpn-tunnel = "10.46.0.7"
    westus2.haproxy-lb = "10.46.0.20"
    westus2.haproxy-0 = "10.46.1.21"
    westus2.haproxy-1 = "10.46.1.22"

    eastus.bastion    = "10.48.0.10"
    eastus.vpn-tunnel = "10.48.0.7"

    westeurope.bastion    = "10.49.0.10"
    westeurope.vpn-tunnel = "10.49.0.7"
  }
}

variable "azurerm_virtual_network_address_space" {
  default = {
    "westus2"    = ["10.46.0.0/16"]
    "eastus"     = ["10.48.0.0/16"]
    "westeurope" = ["10.49.0.0/16"]
  }
}

variable "azurerm_virtual_network_address_space_start" {
  default = {
    "westus2"    = "10.46.0.0"
    "eastus"     = "10.48.0.0"
    "westeurope" = "10.49.0.0"
  }
}

variable "azurerm_virtual_network_address_space_end" {
  default = {
    "westus2"    = "10.46.255.255"
    "eastus"     = "10.48.255.255"
    "westeurope" = "10.49.255.255"
  }
}

variable "subnet_public_address_space" {
  default = {
    "westus2"    = "10.46.0.0/24"
    "eastus"     = "10.48.0.0/24"
    "westeurope" = "10.49.0.0/24"
  }
}

variable "subnet_agw_public_address_space" {
  default = {
    "westus2"    = "10.46.10.0/24"
    "eastus"     = "10.48.10.0/24"
    "westeurope" = "10.49.10.0/24"
  }
}

variable "subnet_private_address_space" {
  default = {
    "westus2"    = "10.46.1.0/24"
    "eastus"     = "10.48.1.0/24"
    "westeurope" = "10.49.1.0/24"
  }
}

variable "lb_ip_addr_0" {
  default = {
    "westus2"    = "10.46.0.40"
    "eastus"     = "10.48.0.40"
    "westeurope" = "10.49.0.40"
  }
}

variable "lb_ip_addr_1" {
  default = {
    "westus2"    = "10.46.0.41"
    "eastus"     = "10.48.0.41"
    "westeurope" = "10.49.0.41"
  }
}

variable "lb_ip_addr_2" {
  default = {
    "westus2"    = "10.46.0.42"
    "eastus"     = "10.48.0.42"
    "westeurope" = "10.49.0.42"
  }
}
