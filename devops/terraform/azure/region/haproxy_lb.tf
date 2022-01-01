data "template_file" "haproxy-cloudinit-template" {
  template = "${file("haproxy-init.conf")}"

  vars {
    HAPROXY_CONFIG_BODY = "${indent(8, file("haproxy.cfg"))}"
    BOOT_SCRIPT_BODY = "${indent(8, file("haproxy-boot.sh"))}"
    HAPROXY_CTL_SCRIPT_BODY = "${indent(8, file("haproxy-deploy-ctl.sh"))}"
    STATS_SCRIPT_BODY = "${indent(8, file("haproxy_get_release_ips.sh"))}"
    AZ_SSL_KEY = "${var.az_ssl_storage_key}"
    AZ_SSL_USER = "${var.az_ssl_storage_user}"
    AZ_STATE_KEY = "${var.az_state_storage_key}"
    AZ_STATE_USER = "${var.az_state_storage_user}"
    REGION = "${var.region}"
    METRICS_SSH_KEY = "${var.ssh_metrics_pubkey}"
  }
}

# data "template_cloudinit_config" "haproxy-cloudinit" {
#   gzip          = true
#   base64_encode = true

#   part {
#     filename     = "init.cfg"
#     content_type = "text/cloud-config"
#     content      = "${data.template_file.haproxy-cloudinit-template.rendered}"
#   }

# }

resource "azurerm_public_ip" "haproxy-lb-prod" {
  name                         = "haproxy-lb-ip-prod"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  public_ip_address_allocation = "static"
}

resource "azurerm_public_ip" "haproxy-lb-staging" {
  name                         = "haproxy-lb-ip-staging"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  public_ip_address_allocation = "static"
}

# resource "azurerm_network_interface" "haproxy-0" {
#   name                      = "haproxy-0-prod"
#   location                  = "${var.region}"
#   resource_group_name       = "${azurerm_resource_group.prod.name}"
#   network_security_group_id = "${azurerm_network_security_group.sg_https.id}"
#   enable_ip_forwarding      = true

  
# }

resource "azurerm_virtual_machine_scale_set" "haproxy-prod" {
  name                         = "haproxy-prod"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  upgrade_policy_mode = "Manual"

  sku {
    name     = "Standard_A0"
    tier     = "Standard"
    capacity = 2
  }

  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_profile_os_disk {
    name           = "os-disk"
    vhd_containers = ["${azurerm_storage_account.proddisks.primary_blob_endpoint}${azurerm_storage_container.proddisks.name}/${var.image_name_haproxy}"]
    caching        = "ReadWrite"
    create_option  = "FromImage"
  }

  network_profile {
    name    = "haproxy-network-profile"
    primary = true

    ip_configuration {
      name                          = "haproxy-pool-cfg"
      subnet_id                     = "${azurerm_subnet.subnet_private.id}"
      load_balancer_backend_address_pool_ids = ["${azurerm_lb_backend_address_pool.haproxy-lbpool-prod.id}"]
    }
  }

  os_profile {
    computer_name_prefix  = "haproxy"
    admin_username        = "ubuntu"
    admin_password        = ""
    custom_data           = "${data.template_file.haproxy-cloudinit-template.rendered}"
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

resource "azurerm_virtual_machine_scale_set" "haproxy-staging" {
  name                         = "haproxy-staging"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.prod.name}"
  upgrade_policy_mode = "Manual"

  sku {
    name     = "Standard_A0"
    tier     = "Standard"
    capacity = 2
  }

  storage_profile_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_profile_os_disk {
    name           = "os-disk"
    vhd_containers = ["${azurerm_storage_account.proddisks.primary_blob_endpoint}${azurerm_storage_container.proddisks.name}/${var.image_name_haproxy}"]
    caching        = "ReadWrite"
    create_option  = "FromImage"
  }

  network_profile {
    name    = "haproxy-network-profile"
    primary = true

    ip_configuration {
      name                          = "haproxy-pool-cfg"
      subnet_id                     = "${azurerm_subnet.subnet_private.id}"
      load_balancer_backend_address_pool_ids = ["${azurerm_lb_backend_address_pool.haproxy-lbpool-staging.id}"]
    }
  }

  os_profile {
    computer_name_prefix  = "haproxy"
    admin_username        = "ubuntu"
    admin_password        = ""
    custom_data           = "${data.template_file.haproxy-cloudinit-template.rendered}"
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



resource "azurerm_lb" "haproxy-staging" {
  name                = "haproxy-staging"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  location            = "${var.region}"
  # sku                 = "Standard"

  frontend_ip_configuration {
    name                          = "haproxy-frontend"
    public_ip_address_id          = "${azurerm_public_ip.haproxy-lb-staging.id}"
  }
}

resource "azurerm_lb" "haproxy-prod" {
  name                = "haproxy-prod"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  location            = "${var.region}"
  # sku                 = "Standard"

  frontend_ip_configuration {
    name                          = "haproxy-frontend"
    public_ip_address_id          = "${azurerm_public_ip.haproxy-lb-prod.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "haproxy-lbpool-staging" {
  name                = "haproxy-pool"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  loadbalancer_id     = "${azurerm_lb.haproxy-staging.id}"
}

resource "azurerm_lb_backend_address_pool" "haproxy-lbpool-prod" {
  name                = "haproxy-pool"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  loadbalancer_id     = "${azurerm_lb.haproxy-prod.id}"
}

resource "azurerm_lb_probe" "haproxy-probe-staging" {
  name                = "haproxy-probe"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  loadbalancer_id     = "${azurerm_lb.haproxy-staging.id}"
  protocol            = "Tcp"
  port                = 443
  interval_in_seconds = 5
  number_of_probes    = 3
}

resource "azurerm_lb_probe" "haproxy-probe-prod" {
  name                = "haproxy-probe"
  resource_group_name = "${azurerm_resource_group.prod.name}"
  loadbalancer_id     = "${azurerm_lb.haproxy-prod.id}"
  protocol            = "Tcp"
  port                = 443
  interval_in_seconds = 5
  number_of_probes    = 3
}

resource "azurerm_lb_rule" "haproxy-rule-staging" {
  name                           = "haproxy-rule"
  resource_group_name            = "${azurerm_resource_group.prod.name}"
  loadbalancer_id                = "${azurerm_lb.haproxy-staging.id}"
  frontend_ip_configuration_name = "${azurerm_lb.haproxy-staging.frontend_ip_configuration.0.name}"
  protocol                       = "Tcp"
  frontend_port                  = 443
  backend_port                   = 443
  backend_address_pool_id        = "${azurerm_lb_backend_address_pool.haproxy-lbpool-staging.id}"
  probe_id                       = "${azurerm_lb_probe.haproxy-probe-staging.id}"
}

resource "azurerm_lb_rule" "haproxy-rule-prod" {
  name                           = "haproxy-rule"
  resource_group_name            = "${azurerm_resource_group.prod.name}"
  loadbalancer_id                = "${azurerm_lb.haproxy-prod.id}"
  frontend_ip_configuration_name = "${azurerm_lb.haproxy-prod.frontend_ip_configuration.0.name}"
  protocol                       = "Tcp"
  frontend_port                  = 443
  backend_port                   = 443
  backend_address_pool_id        = "${azurerm_lb_backend_address_pool.haproxy-lbpool-prod.id}"
  probe_id                       = "${azurerm_lb_probe.haproxy-probe-prod.id}"
}
