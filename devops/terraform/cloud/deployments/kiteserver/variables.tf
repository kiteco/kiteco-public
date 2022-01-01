variable "gcp_regions" {
  default = ["europe-west2", "us-west1", "us-east1", "asia-southeast1"]
}

variable "gcp_project" {
  default = "kite-prod-XXXXXXX"
}

variable aws_region {
  default = "us-west-1"
}

variable "versions" {
  type = map(string)
}

variable service_name {
  default = "kiteserver"
}
