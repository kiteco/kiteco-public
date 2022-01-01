terraform {
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/nchan"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

data "terraform_remote_state" "deployed" {
  backend   = "s3"
  workspace = terraform.workspace

  config = {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/nchan"
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
  name    = "nchan"
  secrets = [
    "ssl/star_kite_com/passphrase",
    "CUSTOMER_IO_API_KEY"
  ]
  policy_statements = [
    {
      actions   = ["s3:GetObject"]
      resources = [
        "arn:aws:s3:::XXXXXXX/ssl/star_kite_com.tar.gz",
        "arn:aws:s3:::XXXXXXX/htpasswd/nchan.htpasswd"
      ]
    },
    {
      actions = ["s3:ListAllMyBuckets"]
      resources = ["arn:aws:s3:::*"]
    },
    {
      actions = ["s3:ListBucket"]
      resources = ["arn:aws:s3:::kite-metrics/*"]
    },
    {
      actions = ["s3:GetObject"]
      resources = ["arn:aws:s3:::kite-metrics/enrichment/maxmind/raw/country/latest/*"]
    }
  ]
}

resource "google_compute_http_health_check" "default" {
  name               = "health-check-nchan"
  check_interval_sec = 1
  timeout_sec        = 1
  request_path       = "/.ping"
}

module "tcp" {
  source                = "../../modules/gcp_service_tcp"
  name                  = "nchan"
  network                   = data.google_compute_network.kite_prod.name
  regions               = var.gcp_regions
  health_checks         = [google_compute_http_health_check.default.id]
  versions              = local.versions
  port_range            = "443"
  service_account_email     = module.instance_role.service_account_email
}

module "https" {
  source                = "../../modules/gcp_service_https"
  name                  = "nchan"
  network               = data.google_compute_network.kite_prod.name
  service_account_email = module.instance_role.service_account_email
  health_check_url      = "/.ping"
  versions              = local.versions
  port                  = 9094
  certificate           = "rc-kite-com"
  instance_groups       = module.service.instance_groups
}

module "service" {
  source = "../../modules/gcp_service"

  name                      = "nchan"
  network                   = data.google_compute_network.kite_prod.name
  regions                   = var.gcp_regions
  versions                  = local.versions
  named_ports               = {http: 9094}
  aws_acces_key_id          = module.instance_role.aws_acces_key_id
  gcp_aws_secret_access_key = module.instance_role.gcp_aws_secret_access_key
  service_account_email     = module.instance_role.service_account_email

  target_pools              = module.tcp.target_pools

  min_replicas              = 2
  max_replicas              = 2
  machine_type              = "n2-standard-8"
}
