# This terraform file sets up AWS VPC and an OpenVPN endpoint.

provider "aws" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region     = "${var.region}"
}

# VPC setup -------------------------------------

resource "aws_vpc" "prod" {
  cidr_block = "${lookup(var.vpc_cidrblock, var.region)}"

  tags {
    Name = "kite-prod"
  }
}

# Internet gateway ------------------------------

resource "aws_internet_gateway" "prod" {
  vpc_id = "${aws_vpc.prod.id}"
}

# Routing table for public subnets --------------

resource "aws_route_table" "public" {
  vpc_id = "${aws_vpc.prod.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.prod.id}"
  }

  route {
    cidr_block  = "10.86.0.0/16"
    instance_id = "${aws_instance.vpn-tunnel.id}"
  }

  tags {
    Name = "public"
  }
}

resource "aws_route_table_association" "az1-public" {
  subnet_id      = "${aws_subnet.az1-public.id}"
  route_table_id = "${aws_route_table.public.id}"
}

resource "aws_route_table_association" "az2-public" {
  subnet_id      = "${aws_subnet.az2-public.id}"
  route_table_id = "${aws_route_table.public.id}"
}

# Routing table for private subnets -------------

resource "aws_route_table" "private" {
  vpc_id = "${aws_vpc.prod.id}"

  route {
    cidr_block  = "0.0.0.0/0"
    instance_id = "${aws_instance.nat.id}"
  }

  route {
    cidr_block  = "10.86.0.0/16"
    instance_id = "${aws_instance.vpn-tunnel.id}"
  }

  tags {
    Name = "private"
  }
}

resource "aws_route_table_association" "az1-private" {
  subnet_id      = "${aws_subnet.az1-private.id}"
  route_table_id = "${aws_route_table.private.id}"
}

resource "aws_route_table_association" "db-az1-private" {
  subnet_id      = "${aws_subnet.db-az1-private.id}"
  route_table_id = "${aws_route_table.private.id}"
}

resource "aws_route_table_association" "db-az2-private" {
  subnet_id      = "${aws_subnet.db-az2-private.id}"
  route_table_id = "${aws_route_table.private.id}"
}

# Subnets ---------------------------------------

resource "aws_subnet" "az1-public" {
  vpc_id            = "${aws_vpc.prod.id}"
  cidr_block        = "${lookup(var.az1_public_cidrblock, var.region)}"
  availability_zone = "${lookup(var.az1, var.region)}"

  tags {
    Name = "az1-public"
  }
}

resource "aws_subnet" "az2-public" {
  vpc_id            = "${aws_vpc.prod.id}"
  cidr_block        = "${lookup(var.az2_public_cidrblock, var.region)}"
  availability_zone = "${lookup(var.az2, var.region)}"

  tags {
    Name = "az2-public"
  }
}

resource "aws_subnet" "az1-private" {
  vpc_id            = "${aws_vpc.prod.id}"
  cidr_block        = "${lookup(var.az1_private_cidrblock, var.region)}"
  availability_zone = "${lookup(var.az1, var.region)}"

  tags {
    Name = "az1-private"
  }
}

# Database subnets ------------------------------

resource "aws_subnet" "db-az1-private" {
  vpc_id            = "${aws_vpc.prod.id}"
  cidr_block        = "${lookup(var.db_az1_cidrblock, var.region)}"
  availability_zone = "${lookup(var.db_az1, var.region)}"

  tags {
    Name = "db-az1-private"
  }
}

resource "aws_subnet" "db-az2-private" {
  vpc_id            = "${aws_vpc.prod.id}"
  cidr_block        = "${lookup(var.db_az2_cidrblock, var.region)}"
  availability_zone = "${lookup(var.db_az2, var.region)}"

  tags {
    Name = "db-az2-private"
  }
}

resource "aws_db_subnet_group" "db-az1-az2-private-subnet-group" {
  name        = "db-az1-az2-private-subnet-group"
  description = "Private DB subnet group"

  subnet_ids = ["${aws_subnet.db-az1-private.id}",
    "${aws_subnet.db-az2-private.id}",
  ]
}

# NAT ------------------------------------------

resource "aws_instance" "nat" {
  ami                    = "${lookup(var.nat_ami, var.region)}"
  availability_zone      = "${lookup(var.az1, var.region)}"
  instance_type          = "m1.small"
  key_name               = "${var.key_name}"
  vpc_security_group_ids = ["${aws_security_group.nat.id}"]
  subnet_id              = "${aws_subnet.az1-public.id}"
  source_dest_check      = false

  tags {
    Name = "nat"
  }
}

resource "aws_eip" "nat" {
  instance = "${aws_instance.nat.id}"
  vpc      = true
}
