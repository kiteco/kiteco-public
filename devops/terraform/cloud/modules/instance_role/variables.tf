variable "name" {
  type = string
}

variable "secrets" {
  type    = list(string)
  default = []
}

variable "default_secrets" {
  default = ["beats_elastic_auth_str"]
}

variable "policy_statements" {
  type = list(object({
    actions = list(string)
    resources = list(string)
  }))
  default = []
}
