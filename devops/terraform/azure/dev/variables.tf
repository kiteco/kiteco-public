variable region {}

variable az_subscription_id {}

variable az_client_id {}

variable az_client_secret {}

variable az_tenant_id {}

# on quip - set using TF_VAR_localfiles_db_password
variable localfiles_db_password {}

# on quip - set using TF_VAR_recruiting_password
variable recruiting_password {}

variable "test_instance_count" {
  default = 2
}

variable "crawler_instance_count" {
  default = 5
}

variable "ml_training_instance_count" {
  default = 4
}

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

variable "image_name_vpntunnel" {
  default = "vpntunnel.vhd"
}

variable "image_name_dns" {
  default = "dns.vhd"
}

variable "image_name_test" {
  default = "test"
}

variable "image_name_import2" {
  default = "import2"
}

variable "image_name_metrics" {
  default = "metrics.vhd"
}

variable "image_name_base" {
  default = "base.vhd"
}

variable "image_name_recruiting" {
  default = "recruiting.vhd"
}

variable "image_name_ml" {
  default = "mltrain.vhd"
}

variable "image_name_ml-build" {
  default = "ml-build.vhd"
}

variable "other_vnet_region_addresses" {
  default = ["10.46.0.0/16", "10.48.0.0/16", "10.49.0.0/16"]
}

variable "vm_ip_list" {
  default = {
    westus2.bastion    = "10.47.0.10"
    westus2.vpn        = "10.47.0.4"
    westus2.vpn-tunnel = "10.47.0.7"
    westus2.recruiting = "10.47.0.8"
    westus2.dns        = "10.47.1.6"
    westus2.metrics    = "10.47.1.7"
    westus2.import2    = "10.47.1.9"
    westus2.ml-build   = "10.47.1.11"
  }
}

variable "vm_ip_prefix" {
  default = {
    westus2.test = "10.47.1.2" #2XX
    westus2.ml-training = "10.47.1." #150
    westus2.crawler = "10.47.0." #100
  }
}

variable ssh_pubkey {
  default = "XXXXXXX"
}

variable ssh_pubkey_1 {
  default = "XXXXXXX"
}

variable db_location {
  default = {
    "westus"  = "westus"
    "westus2" = "westus"
  }
}

variable "resourcegroup_name" {
  default = {
    "westus2" = "dev"
  }
}

variable "azurerm_virtual_network_name" {
  default = "dev"
}

variable "azurerm_virtual_network_address_space" {
  default = {
    "westus2" = ["10.47.0.0/16"]
  }
}

variable "subnet_public_address_space" {
  default = {
    "westus2" = "10.47.0.0/24"
  }
}

variable "subnet_private_address_space" {
  default = {
    "westus2" = "10.47.1.0/24"
  }
}
