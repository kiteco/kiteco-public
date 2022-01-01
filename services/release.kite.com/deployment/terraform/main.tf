terraform {
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "services/release.kite.com"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

provider "google" {
  region  = var.gcp_region
  project = var.gcp_project
}

provider "google-beta" {
  region  = var.gcp_region
  project = var.gcp_project
}

resource "google_compute_global_address" "ip_address" {
  name = "release-kite-com-${terraform.workspace}"
}

