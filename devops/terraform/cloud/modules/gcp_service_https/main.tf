locals {
  lbs = { for color, version in var.versions : color => version if color == "blue" }
}

resource "google_compute_global_address" "default" {
  for_each = local.lbs

  name     = "${var.name}-${each.key}"
}

resource "google_compute_global_forwarding_rule" "default" {
  for_each = local.lbs

  name       = "${var.name}-${each.key}"
  target     = google_compute_target_https_proxy.default[each.key].self_link
  port_range = "443"
  ip_address = google_compute_global_address.default[each.key].address
}

data "google_compute_ssl_certificate" "star_kite_com" {
  name = var.certificate
}

resource "google_compute_target_https_proxy" "default" {
  for_each = local.lbs

  name             = "${var.name}-${each.key}"
  url_map          = google_compute_url_map.default[each.key].self_link
  ssl_certificates = [data.google_compute_ssl_certificate.star_kite_com.self_link]
}

resource "google_compute_url_map" "default" {
  for_each = local.lbs

  name            = "${var.name}-${each.key}"
  default_service = google_compute_backend_service.default[each.value].self_link
}

resource "google_compute_backend_service" "default" {
  for_each = {for color, version in var.versions: version => color}

  name             = "${var.name}-${replace(each.key, ".", "-")}"
  health_checks    = [google_compute_health_check.default.id]
  protocol         = "HTTP"
  session_affinity = "NONE"
  timeout_sec      = 30

  dynamic "backend" {
    for_each = var.instance_groups[each.key]
    content {
      group          = backend.value
      balancing_mode = "UTILIZATION"
    }
  }
}

resource "google_compute_health_check" "default" {
  name               = "health-check-${var.name}"
  check_interval_sec = 1
  timeout_sec        = 1

  http_health_check {
    port_name          = "http"
    port_specification = "USE_NAMED_PORT"
    request_path       = var.health_check_url
  }

  lifecycle {
    ignore_changes = all
  }
}

resource "google_compute_firewall" "default" {
  name    = "svc-${var.name}"
  network = var.network

  allow {
    protocol = "tcp"
    ports    = [var.port]
  }

  source_ranges           = ["35.191.0.0/16", "130.211.0.0/22"]
  target_service_accounts = [var.service_account_email]
}
