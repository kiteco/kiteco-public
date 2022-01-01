variable "dev_region" {
  default = "us-west1"
}

variable "dev_cidr" {
  default = "10.201.1.0/24"
}

variable "prod_subnets" {
  default = {
    "us-west1": 5,
    "us-east1": 6,
    "asia-southeast1": 7,
    "europe-west3": 8,
    "europe-west1": 9,
    "europe-west2": 10,
  }
}

variable "dev_project" {
  default = "kite-dev-XXXXXXX"
}

variable "prod_project" {
  default = "kite-prod-XXXXXXX"
}
