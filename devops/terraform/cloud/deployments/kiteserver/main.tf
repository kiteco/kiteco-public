terraform {
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/kiteserver"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

data "terraform_remote_state" "deployed" {
  backend   = "s3"
  workspace = terraform.workspace

  config = {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/kiteserver"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

locals {
  deployed_versions = length(data.terraform_remote_state.deployed.outputs) > 0 ? data.terraform_remote_state.deployed.outputs.versions : {}
  versions_raw = {
    for color, version in var.versions : color => lookup(local.deployed_versions, version, version)
  }
  versions = { for color, version in local.versions_raw : color => version if color != version && length(setintersection(["blue", "green", "gray"], [color, version])) < 2 }
  lbs = { for color, version in local.versions : color => version if color == "blue" }

  backends = {
    for cfg in flatten([
      for color, version in local.versions: [
        for group in module.service.instance_groups[version]: {version: version, group: group}
      ] if color != "gray"
    ]): cfg.group => cfg.version
  }
}


provider "aws" {
  region = var.aws_region
}

provider "google" {
  region  = var.gcp_regions[0]
  project = var.gcp_project
}

provider "google-beta" {
  region  = var.gcp_regions[0]
  project = var.gcp_project
}

data "google_compute_network" "kite_prod" {
  name = "kite-prod"
}

module "instance_role" {
  source  = "../../modules/instance_role"
  name    = var.service_name
}

module "service" {
  source = "../../modules/gcp_service"

  name                      = var.service_name
  network                   = data.google_compute_network.kite_prod.name
  regions                   = var.gcp_regions
  versions                  = local.versions
  named_ports               = {
    http: 8500,
    https: 9500,
    envoyadmin: 9901
  }
  aws_acces_key_id          = module.instance_role.aws_acces_key_id
  gcp_aws_secret_access_key = module.instance_role.gcp_aws_secret_access_key
  service_account_email     = module.instance_role.service_account_email
  boot_disk_size_gb         = 20

  min_replicas              = 1
  max_replicas              = 1
  machine_type              = "n1-standard-8"
  kite_base_image           = "nvidia-docker-puppet-1602655040"

  guest_accelerator_type    = "nvidia-tesla-t4"
  ubuntu_release            = "focal"
}

resource "google_compute_global_address" "default" {
  for_each = local.lbs

  name     = "${var.service_name}-${each.key}"
}

resource "google_compute_global_forwarding_rule" "default" {
  for_each = local.lbs

  name       = "${var.service_name}-${each.key}"
  port_range = "443"
  target     = google_compute_target_https_proxy.default[each.key].self_link
  ip_address = google_compute_global_address.default[each.key].address
}

data "google_compute_ssl_certificate" "tf_serving_1" {
  name = "cloud-kite-com"
}

resource "google_compute_target_https_proxy" "default" {
  for_each = local.lbs

  name             = "${var.service_name}-${each.key}"
  url_map          = google_compute_url_map.default[each.key].self_link
  ssl_certificates = [data.google_compute_ssl_certificate.tf_serving_1.self_link]
}

resource "google_compute_url_map" "default" {
  for_each = local.lbs

  name            = "${var.service_name}-${each.key}"
  default_service = google_compute_backend_service.http2[each.key].self_link

  host_rule {
    hosts        = ["*"]
    path_matcher = "all"
  }

  path_matcher {
    name            = "all"
    default_service = google_compute_backend_service.http2[each.key].self_link

    path_rule {
      paths   = ["/api/*", "/model-assets/*"]
      service = google_compute_backend_service.http[each.key].self_link
    }
  }
}

resource "google_compute_backend_service" "http" {
  for_each = local.lbs

  name             = "${var.service_name}-http-${each.key}"
  health_checks    = [google_compute_health_check.default.id]
  protocol         = "HTTP"
  session_affinity = "NONE"
  timeout_sec      = 30

  dynamic "backend" {
    for_each = local.backends

    content {
      group           = backend.key
      capacity_scaler = backend.value == each.value ? 1 : 0
      balancing_mode  = "UTILIZATION"
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_backend_service" "http2" {
  for_each = local.lbs

  name             = "${var.service_name}-http2-${each.key}"
  health_checks    = [google_compute_health_check.default.id]
  protocol         = "HTTP2"
  session_affinity = "NONE"
  timeout_sec      = 30
  port_name        = "https"

  dynamic "backend" {
    for_each = local.backends

    content {
      group           = backend.key
      capacity_scaler = backend.value == each.value ? 1 : 0
      balancing_mode  = "UTILIZATION"
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_health_check" "default" {
  name               = "health-check-${var.service_name}"
  check_interval_sec = 1
  timeout_sec        = 1

  tcp_health_check {
    port_name          = "https"
    port_specification = "USE_NAMED_PORT"
  }
}

resource "google_compute_firewall" "default" {
  name    = "svc-${var.service_name}"
  network = data.google_compute_network.kite_prod.name

  allow {
    protocol = "tcp"
    ports    = [8500,9500]
  }

  source_ranges           = ["35.191.0.0/16", "130.211.0.0/22"]
  target_service_accounts = [module.instance_role.service_account_email]
}
