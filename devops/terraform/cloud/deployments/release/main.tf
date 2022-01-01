terraform {
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/release"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

data "terraform_remote_state" "deployed" {
  backend   = "s3"
  workspace = terraform.workspace

  config = {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/release"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

locals {
  deployed_versions = length(data.terraform_remote_state.deployed.outputs) > 0 ? data.terraform_remote_state.deployed.outputs.versions : {}
  versions_raw = {
    for color, version in var.versions : color => lookup(local.deployed_versions, version, version)
  }
  versions = { for color, version in local.versions_raw : color => version if color != version }
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
  name    = "release"
  secrets = ["RELEASE_DB_URI", "ROLLBAR_TOKEN"]
}

module "https" {
  source                = "../../modules/gcp_service_https"
  name                  = "release"
  network               = data.google_compute_network.kite_prod.name
  service_account_email = module.instance_role.service_account_email
  health_check_url      = "/appcast.xml"
  versions              = local.versions
  port                  = 9093
  certificate           = "release-kite-com-2"
  instance_groups       = module.service.instance_groups
}

module "service" {
  source = "../../modules/gcp_service"

  name                      = "release"
  network                   = data.google_compute_network.kite_prod.name
  regions                   = var.gcp_regions
  versions                  = local.versions
  named_ports               = {http: 9093}
  aws_acces_key_id          = module.instance_role.aws_acces_key_id
  gcp_aws_secret_access_key = module.instance_role.gcp_aws_secret_access_key
  service_account_email     = module.instance_role.service_account_email
  min_replicas              = 1
  max_replicas              = 1
}

module "gcp_deployment" {
  source = "../../modules/gcp_deployment"
}
