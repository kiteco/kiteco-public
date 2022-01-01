variable "name" {
  type = string
}

variable "versions" {
  type = map
}

variable "port_range" {
  type = string
}

variable "port" {
  type = string
}

variable "port_name" {
  type = string
}

variable "network" {
  type = string
}

variable "instance_groups" {
  type = map(set(string))
}

variable "service_account_email" {
  type = string
}
