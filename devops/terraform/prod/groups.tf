resource "aws_security_group" "nat" {
    name = "nat"
    description = "Allow services from the private subnet through NAT"

    ingress {
        from_port = 0
        to_port = 65535
        protocol = "tcp"
        cidr_blocks = [aws_subnet.us-west-1b-private.cidr_block]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    vpc_id = aws_vpc.prod.id
}

# Allow ssh traffic from anywhere
resource "aws_security_group" "ssh" {
    name = "ssh"
    description = "Allow SSH traffic"

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    vpc_id = aws_vpc.prod.id
}


# Allow acess to usernode
resource "aws_security_group" "usernode" {
    name = "usernode"
    description = "Allow usernode traffic"

    ingress {
        from_port = 9090
        to_port = 9090
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    vpc_id = aws_vpc.prod.id
}

# Allow acess to usernode debug
resource "aws_security_group" "usernode-debug" {
    name = "usernode-debug"
    description = "Allow usernode debug traffic"

    ingress {
        from_port = 9091
        to_port = 9091
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    vpc_id = aws_vpc.prod.id
}

# Allow https
resource "aws_security_group" "https" {
    name = "https"
    description = "Allow HTTPS traffic"

    ingress {
        from_port = 443
        to_port = 443
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    vpc_id = aws_vpc.prod.id
}

# Allow all traffic if you are connected to the VPN.
resource "aws_security_group" "all-vpn" {
    name = "all-vpn"
    description = "Allow all traffic if you are on the VPN"

    ingress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["10.86.0.0/16"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

    vpc_id = aws_vpc.prod.id
}
