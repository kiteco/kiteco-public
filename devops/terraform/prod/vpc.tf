# This terraform file sets up AWS VPC and an OpenVPN endpoint.

provider "aws" {
    region = "us-west-1"
}

provider "aws" {
    alias = "east"
    region = "us-east-1"
}

# VPC setup -------------------------------------

resource "aws_vpc" "prod" {
    cidr_block = "172.86.0.0/16"
    tags = {
        Name = "kite-prod"
    }
}

# Internet gateway ------------------------------

resource "aws_internet_gateway" "prod" {
    vpc_id = aws_vpc.prod.id
}

# NAT ------------------------------------------

resource "aws_instance" "nat" {
    ami = var.aws_nat_ami
    availability_zone = "us-west-1b"
    instance_type = "m1.small"
    key_name = "kite-prod"
    vpc_security_group_ids = [aws_security_group.nat.id]
    subnet_id = aws_subnet.us-west-1b-public.id
    source_dest_check = false
    tags = {
        Name = "nat"
    }
}

resource "aws_eip" "nat" {
    instance = aws_instance.nat.id
    vpc = true
}

# Routing table for public subnets --------------

resource "aws_route_table" "us-west-1-public" {
    vpc_id = aws_vpc.prod.id

    route {
        cidr_block = "0.0.0.0/0"
        gateway_id = aws_internet_gateway.prod.id
    }
    route {
        cidr_block = "10.86.0.0/16"
        vpc_peering_connection_id = var.dev_vpc_peering_connection_id
    }
}

resource "aws_route_table_association" "us-west-1b-public" {
    subnet_id = aws_subnet.us-west-1b-public.id
    route_table_id = aws_route_table.us-west-1-public.id
}

resource "aws_route_table_association" "us-west-1c-public" {
    subnet_id = aws_subnet.us-west-1c-public.id
    route_table_id = aws_route_table.us-west-1-public.id
}


# Routing table for private subnets -------------

resource "aws_route_table" "us-west-1-private" {
    vpc_id = aws_vpc.prod.id

    route {
        cidr_block = "0.0.0.0/0"
        instance_id = aws_instance.nat.id
    }
    route {
        cidr_block = "10.86.0.0/16"
        vpc_peering_connection_id = var.dev_vpc_peering_connection_id
    }
}

resource "aws_route_table_association" "us-west-1b-private" {
    subnet_id = aws_subnet.us-west-1b-private.id
    route_table_id = aws_route_table.us-west-1-private.id
}

# Subnets ---------------------------------------

resource "aws_subnet" "us-west-1b-public" {
    vpc_id = aws_vpc.prod.id
    cidr_block = "172.86.0.0/24"
    availability_zone = "us-west-1b"
}

resource "aws_subnet" "us-west-1c-public" {
    vpc_id = aws_vpc.prod.id
    cidr_block = "172.86.20.0/24"
    availability_zone = "us-west-1c"
}

resource "aws_subnet" "us-west-1b-private" {
    vpc_id = aws_vpc.prod.id
    cidr_block = "172.86.1.0/24"
    availability_zone = "us-west-1b"
}
