# This terraform file sets up AWS VPC and an OpenVPN endpoint.

provider "aws" {
  profile = "default"
  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
  region     = "us-west-1"
}

provider "aws" {
  alias = "east"
  profile = "default"
  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
  region     = "us-east-1"
}

# VPC setup -------------------------------------

resource "aws_vpc" "dev" {
  cidr_block = "10.86.0.0/16"

  tags = {
    Name = "kite-dev"
  }
}

# Internet gateway ------------------------------

resource "aws_internet_gateway" "dev" {
  vpc_id = aws_vpc.dev.id
}

# NAT ------------------------------------------

resource "aws_eip" "nat_dev" {
  vpc  = true
  tags = {
    Name = "NAT gateway"
  }
}

resource "aws_nat_gateway" "dev" {
  allocation_id = aws_eip.nat_dev.id
  subnet_id     = aws_subnet.us-west-1b-public.id
}

# Routing table for public subnets --------------

resource "aws_route_table" "us-west-1-public" {
  vpc_id = aws_vpc.dev.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.dev.id
  }

  route {
    cidr_block                = "172.86.0.0/16"
    vpc_peering_connection_id = aws_vpc_peering_connection.dev-prod-us-west-1.id
  }

  route {
    cidr_block  = "172.87.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }

  route {
    cidr_block  = "172.88.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }

  route {
    cidr_block  = "172.89.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }

  route {
    cidr_block  = "172.90.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }
}

resource "aws_route_table_association" "us-west-1b-public" {
  subnet_id      = aws_subnet.us-west-1b-public.id
  route_table_id = aws_route_table.us-west-1-public.id
}

# Routing table for private subnets -------------

resource "aws_route_table" "us-west-1-private" {
  vpc_id = aws_vpc.dev.id

  route {
    cidr_block  = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.dev.id
  }

  route {
    cidr_block                = "172.86.0.0/16"
    vpc_peering_connection_id = aws_vpc_peering_connection.dev-prod-us-west-1.id
  }

  route {
    cidr_block  = "172.87.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }

  route {
    cidr_block  = "172.88.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }

  route {
    cidr_block  = "172.89.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }

  route {
    cidr_block  = "172.90.0.0/16"
    instance_id = aws_instance.vpn-tunnel.id
  }
}

resource "aws_route_table_association" "us-west-1b-private" {
  subnet_id      = aws_subnet.us-west-1b-private.id
  route_table_id = aws_route_table.us-west-1-private.id
}

resource "aws_route_table_association" "us-west-1b-private-db" {
  subnet_id      = aws_subnet.us-west-1b-private-db.id
  route_table_id = aws_route_table.us-west-1-private.id
}

resource "aws_route_table_association" "us-west-1c-private-db" {
  subnet_id      = aws_subnet.us-west-1c-private-db.id
  route_table_id = aws_route_table.us-west-1-private.id
}

# Subnets ---------------------------------------

resource "aws_subnet" "us-west-1b-public" {
  vpc_id            = aws_vpc.dev.id
  cidr_block        = "10.86.0.0/24"
  availability_zone = var.aws_az
  tags              = {
    "Name" = "public"
  }
}

resource "aws_subnet" "us-west-1b-private" {
  vpc_id            = aws_vpc.dev.id
  cidr_block        = "10.86.1.0/24"
  availability_zone = var.aws_az
  tags              = {
    "Name" = "private"
  }
}

resource "aws_subnet" "us-west-1b-private-db" {
  vpc_id            = aws_vpc.dev.id
  cidr_block        = "10.86.2.0/24"
  availability_zone = "us-west-1b"
  tags              = {
    "Name" = "private-db"
  }
}

resource "aws_subnet" "us-west-1c-private-db" {
  vpc_id            = aws_vpc.dev.id
  cidr_block        = "10.86.3.0/24"
  availability_zone = "us-west-1c"
  tags              = {
    "Name" = "private-db"
  }
}
