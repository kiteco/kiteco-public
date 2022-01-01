variable "region" {
  type = string
}

variable "default_tags" {
  type = map
  default = {
    created-by = "terraform"
  }
}

variable "versions" {
  type = map
}

variable "ec2_prod_key_name" {
  type    = string
  default = "kite-prod"
}
