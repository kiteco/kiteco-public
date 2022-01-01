locals {
  versions             = [for color, version in var.versions : { color : color, version : version }]
  region_versions      = [for product in setproduct(var.regions, local.versions) : merge(product[1], { region : product[0] })]
  version_key_prefixes = { for cfg in local.versions : cfg.version => replace(substr(cfg.version, 0, 16), ".", "-") }
  resources            = { for cfg in local.region_versions : "${local.version_key_prefixes[cfg.version]}-${cfg.region}" => cfg }
}

data "google_compute_subnetwork" "private" {
  for_each = toset(var.regions)
  region   = each.value
  name     = "${var.network}-private-${each.value}"
}

data "google_compute_image" "kite_base" {
  name = var.kite_base_image
}

// Check that the Puppet build exists
data "aws_s3_bucket_object" "puppet" {
  for_each = var.versions

  bucket = "kite-deploys"
  key    = "v${each.value}/puppet.tar.gz"
}

data "google_compute_zones" "available" {
  for_each = toset(var.regions)

  region=each.value
}

resource "google_compute_instance_template" "default" {
  for_each = local.resources

  name_prefix = "${var.name}-${each.key}"
  region      = each.value.region

  machine_type   = var.machine_type
  enable_display = false

  // Create a new boot disk from an image
  disk {
    source_image = data.google_compute_image.kite_base.self_link
    disk_size_gb = var.boot_disk_size_gb
    auto_delete  = true
    boot         = true
  }

  network_interface {
    subnetwork = data.google_compute_subnetwork.private[each.value.region].self_link
  }

  service_account {
    email  = var.service_account_email
    scopes = ["cloud-platform"]
  }

  scheduling {
    on_host_maintenance = var.guest_accelerator_type != null ? "TERMINATE" : "MIGRATE"
  }

  dynamic "guest_accelerator" {
    for_each = var.guest_accelerator_type != null ? [var.guest_accelerator_type] : []
    content {
      type = guest_accelerator.value
      count = 1
    }
  }

  metadata_startup_script = templatefile("${path.module}/../../templates/userdata-gcp.tmpl",
  { node_name : var.name, release_version : each.value.version, aws_acces_key_id : var.aws_acces_key_id, gcp_aws_secret_access_key : var.gcp_aws_secret_access_key, ubuntu_release: var.ubuntu_release })

  lifecycle {
    ignore_changes = all
  }
}

resource "google_compute_region_instance_group_manager" "default" {
  for_each = local.resources

  name                      = "${var.name}-${each.key}"
  region                    = each.value.region
  distribution_policy_zones = var.guest_accelerator_type == null ? null : setintersection(var.gpu_zones[each.value.region][var.guest_accelerator_type], data.google_compute_zones.available[each.value.region].names)
  base_instance_name        = var.name

  dynamic "named_port" {
    for_each = var.named_ports
    content {
      name = named_port.key
      port = named_port.value
    }
  }

  target_pools = var.target_pools != null ? [var.target_pools[each.value.version][each.value.region]]: null

  version {
    instance_template = google_compute_instance_template.default[each.key].self_link
  }

  # lifecycle {
  #   ignore_changes = all
  # }
}

resource "google_compute_region_autoscaler" "default" {
  for_each = local.resources

  name   = "${var.name}-${each.key}"
  region = each.value.region
  target = google_compute_region_instance_group_manager.default[each.key].id

  autoscaling_policy {
    max_replicas    = var.max_replicas
    min_replicas    = var.min_replicas
    cooldown_period = 60

    cpu_utilization {
      target = 0.8
    }
  }
}
