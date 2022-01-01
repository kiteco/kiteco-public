variable "name" {
  type = string
}

variable "network" {
  type = string
}

variable "service_account_email" {
  type = string
}

variable "health_check_url" {
  type = string
}

variable "versions" {
  type = map
}

variable "port" {
  type = number
}

variable "instance_groups" {
  type = map(set(string))
}

variable "certificate" {
  type    = string
  default = "star-kite-com"
}