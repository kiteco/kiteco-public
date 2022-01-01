# This terraform file sets up AWS VPC and an OpenVPN endpoint.

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  region     = "${var.aws_region}"
}

# VPC setup -------------------------------------

resource "aws_vpc" "prod-dbs" {
  cidr_block           = "172.76.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags {
    Name = "kite-prod-dbs"
  }
}

# Internet gateway ------------------------------

resource "aws_internet_gateway" "prod-dbs" {
  vpc_id = "${aws_vpc.prod-dbs.id}"
}

# NAT ------------------------------------------

resource "aws_instance" "nat" {
  ami                    = "${var.aws_nat_ami}"
  availability_zone      = "${var.aws_az}"
  instance_type          = "m1.small"
  key_name               = "${var.aws_key_name}"
  vpc_security_group_ids = ["${aws_security_group.nat.id}"]
  subnet_id              = "${aws_subnet.us-west-1b-public.id}"
  source_dest_check      = false

  tags {
    Name = "nat"
  }
}

resource "aws_eip" "nat" {
  instance = "${aws_instance.nat.id}"
  vpc      = true
}

# Routing table for public subnets --------------

resource "aws_route_table" "us-west-1-public" {
  vpc_id = "${aws_vpc.prod-dbs.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.prod-dbs.id}"
  }
}

resource "aws_route_table_association" "us-west-1b-public" {
  subnet_id      = "${aws_subnet.us-west-1b-public.id}"
  route_table_id = "${aws_route_table.us-west-1-public.id}"
}

resource "aws_route_table_association" "us-west-1b-public-db" {
  subnet_id      = "${aws_subnet.us-west-1b-public-db.id}"
  route_table_id = "${aws_route_table.us-west-1-public.id}"
}

resource "aws_route_table_association" "us-west-1c-public-db" {
  subnet_id      = "${aws_subnet.us-west-1c-public-db.id}"
  route_table_id = "${aws_route_table.us-west-1-public.id}"
}

# Routing table for private subnets -------------

resource "aws_route_table" "us-west-1-private" {
  vpc_id = "${aws_vpc.prod-dbs.id}"

  route {
    cidr_block  = "0.0.0.0/0"
    instance_id = "${aws_instance.nat.id}"
  }
}

resource "aws_route_table_association" "us-west-1b-private" {
  subnet_id      = "${aws_subnet.us-west-1b-private.id}"
  route_table_id = "${aws_route_table.us-west-1-private.id}"
}

# Subnets ---------------------------------------

resource "aws_subnet" "us-west-1b-public" {
  vpc_id            = "${aws_vpc.prod-dbs.id}"
  cidr_block        = "172.76.0.0/24"
  availability_zone = "${var.aws_az}"
}

resource "aws_subnet" "us-west-1b-private" {
  vpc_id            = "${aws_vpc.prod-dbs.id}"
  cidr_block        = "172.76.1.0/24"
  availability_zone = "${var.aws_az}"
}

resource "aws_subnet" "us-west-1b-public-db" {
  vpc_id            = "${aws_vpc.prod-dbs.id}"
  cidr_block        = "172.76.2.0/24"
  availability_zone = "us-west-1b"
}

resource "aws_subnet" "us-west-1c-public-db" {
  vpc_id            = "${aws_vpc.prod-dbs.id}"
  cidr_block        = "172.76.3.0/24"
  availability_zone = "us-west-1c"
}
