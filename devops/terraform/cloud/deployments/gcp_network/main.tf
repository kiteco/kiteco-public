terraform {
  backend "s3" {
    bucket = "kite-terraform-state"
    key    = "gcp_network/terraform.tfstate"
    region = "us-west-1"
  }
}

data "google_compute_network" "dev" {
  project = var.dev_project
  name    = "kite-dev"
}

data "google_compute_network" "prod" {
  project = var.prod_project
  name    = "kite-prod"
}

resource "google_compute_subnetwork" "dev" {
  project                  = var.dev_project
  region                   = var.dev_region
  name                     = "${data.google_compute_network.dev.name}-private-${var.dev_region}"
  network                  = data.google_compute_network.dev.self_link
  ip_cidr_range            = var.dev_cidr
  private_ip_google_access = false
}

resource "google_compute_router" "dev" {
  project = var.dev_project
  region  = var.dev_region
  name    = "kite-dev-us-west1"
  network = data.google_compute_network.dev.name

  bgp {
    asn               = 64512
    advertise_mode    = "CUSTOM"
    advertised_groups = ["ALL_SUBNETS"]

    dynamic "advertised_ip_ranges" {
      for_each = var.prod_subnets
      content {
        range = "10.201.${advertised_ip_ranges.value}.0/24"
      }
    }
  }
}

resource "google_compute_router_nat" "dev_nat" {
  project                = var.dev_project
  region                 = var.dev_region
  name                   = "${data.google_compute_network.dev.name}-${var.dev_region}"
  router                 = google_compute_router.dev.name
  nat_ip_allocate_option = "AUTO_ONLY"

  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetwork {
    name                    = google_compute_subnetwork.dev.self_link
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}

resource "google_compute_subnetwork" "prod" {
  for_each = var.prod_subnets

  project                  = var.prod_project
  region                   = each.key
  name                     = "${data.google_compute_network.prod.name}-private-${each.key}"
  network                  = data.google_compute_network.prod.self_link
  ip_cidr_range            = "10.201.${each.value}.0/24"
  private_ip_google_access = false
}

resource "google_compute_router" "prod" {
  for_each = var.prod_subnets

  project = var.prod_project
  name    = "${data.google_compute_network.prod.name}-${each.key}"
  region  = each.key
  network = data.google_compute_network.prod.self_link

  bgp {
    asn = 64510 + each.value
  }
}

resource "google_compute_router_nat" "prod" {
  for_each = var.prod_subnets

  project                = var.prod_project
  name                   = "${data.google_compute_network.prod.name}-${each.key}"
  router                 = google_compute_router.prod[each.key].name
  min_ports_per_vm       = 32000
  region                 = each.key
  nat_ip_allocate_option = "AUTO_ONLY"

  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetwork {
    name                    = google_compute_subnetwork.prod[each.key].self_link
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}
