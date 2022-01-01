variable "region" {}

variable "access_key" {}

variable "secret_key" {}

variable "localfiles_db_password" {}

variable "localfiles_db_name" {
  default = "localfiles"
}

variable "localfiles_db_username" {
  default = "kite"
}

variable "key_name" {
  default = "kite-prod"
}

variable "nat_ami" {
  default = {
    us-west-1      = "ami-XXXXXXX"
    us-west-2      = "ami-XXXXXXX"
    us-east-1      = "ami-XXXXXXX"
    eu-west-1      = "ami-XXXXXXX"
    ap-southeast-1 = "ami-XXXXXXX"
  }
}

variable "ubuntu_ami" {
  default = {
    us-west-1      = "ami-XXXXXXX"
    us-west-2      = "ami-XXXXXXX"
    us-east-1      = "ami-XXXXXXX"
    eu-west-1      = "ami-XXXXXXX"
    ap-southeast-1 = "ami-XXXXXXX"
  }
}

variable "vpc_cidrblock" {
  default = {
    us-west-1      = "172.86.0.0/16"
    us-west-2      = "172.87.0.0/16"
    us-east-1      = "172.88.0.0/16"
    eu-west-1      = "172.89.0.0/16"
    ap-southeast-1 = "172.90.0.0/16"
  }
}

variable "az1" {
  default = {
    us-west-1      = "us-west-1b"
    us-west-2      = "us-west-2a"
    us-east-1      = "us-east-1a"
    eu-west-1      = "eu-west-1a"
    ap-southeast-1 = "ap-southeast-1a"
  }
}

variable "az1_public_cidrblock" {
  default = {
    us-west-1      = "172.86.0.0/24"
    us-west-2      = "172.87.0.0/24"
    us-east-1      = "172.88.0.0/24"
    eu-west-1      = "172.89.0.0/24"
    ap-southeast-1 = "172.90.0.0/24"
  }
}

variable "az1_private_cidrblock" {
  default = {
    us-west-1      = "172.86.1.0/24"
    us-west-2      = "172.87.1.0/24"
    us-east-1      = "172.88.1.0/24"
    eu-west-1      = "172.89.1.0/24"
    ap-southeast-1 = "172.90.1.0/24"
  }
}

variable "az2" {
  default = {
    us-west-1      = "us-west-1c"
    us-west-2      = "us-west-2b"
    us-east-1      = "us-east-1b"
    eu-west-1      = "eu-west-1b"
    ap-southeast-1 = "ap-southeast-1b"
  }
}

variable "az2_public_cidrblock" {
  default = {
    us-west-1      = "172.86.20.0/24"
    us-west-2      = "172.87.20.0/24"
    us-east-1      = "172.88.20.0/24"
    eu-west-1      = "172.89.20.0/24"
    ap-southeast-1 = "172.90.20.0/24"
  }
}

variable "az2_private_cidrblock" {
  default = {
    us-west-1      = "172.86.21.0/24"
    us-west-2      = "172.87.21.0/24"
    us-east-1      = "172.88.21.0/24"
    eu-west-1      = "172.89.21.0/24"
    ap-southeast-1 = "172.89.21.0/24"
  }
}

variable "db_az1" {
  default = {
    us-west-1      = "us-west-1b"
    us-west-2      = "us-west-2a"
    us-east-1      = "us-east-1a"
    eu-west-1      = "eu-west-1a"
    ap-southeast-1 = "ap-southeast-1a"
  }
}

variable "db_az1_cidrblock" {
  default = {
    us-west-1      = "172.86.200.0/24"
    us-west-2      = "172.87.200.0/24"
    us-east-1      = "172.88.200.0/24"
    eu-west-1      = "172.89.200.0/24"
    ap-southeast-1 = "172.90.200.0/24"
  }
}

variable "db_az2" {
  default = {
    us-west-1      = "us-west-1c"
    us-west-2      = "us-west-2b"
    us-east-1      = "us-east-1b"
    eu-west-1      = "eu-west-1b"
    ap-southeast-1 = "ap-southeast-1b"
  }
}

variable "db_az2_cidrblock" {
  default = {
    us-west-1      = "172.86.201.0/24"
    us-west-2      = "172.87.201.0/24"
    us-east-1      = "172.88.201.0/24"
    eu-west-1      = "172.89.201.0/24"
    ap-southeast-1 = "172.90.201.0/24"
  }
}

variable "certificate_arn" {
  default = {
    us-west-1      = "arn:aws:acm:us-west-1:XXXXXXX:certificate/XXXXXXX"
    us-west-2      = "arn:aws:acm:us-west-2:XXXXXXX:certificate/XXXXXXX"
    us-east-1      = "arn:aws:acm:us-east-1:XXXXXXX:certificate/XXXXXXX"
    eu-west-1      = "arn:aws:acm:eu-west-1:XXXXXXX:certificate/XXXXXXX"
    ap-southeast-1 = "arn:aws:acm:ap-southeast-1:XXXXXXX:certificate/XXXXXXX"
  }
}

variable "localfiles_snapshot" {
  default = {
    us-east-1 = "rds:localfiles-prod-db-2017-06-08-09-08"
    eu-west-1 = "rds:localfiles-prod-db-2017-06-08-00-07"
  }
}
