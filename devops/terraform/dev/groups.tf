resource "aws_security_group" "nat" {
  name        = "nat"
  description = "Allow services from the private subnet through NAT"

  ingress {
    from_port   = 0
    to_port     = 65535
    protocol    = "tcp"
    cidr_blocks = [aws_subnet.us-west-1b-private.cidr_block]
  }

  vpc_id = aws_vpc.dev.id
}

# Allow all traffic
resource "aws_security_group" "allow_all" {
  name        = "allow_all"
  description = "Allow all traffic"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Allow all traffic if you are connected to the VPN. This is used
# for public services so we only expose ports that are needed
# for services, but allow internal users to connect (e.g ssh)
resource "aws_security_group" "allow_all_vpn" {
  name        = "allow_all_vpn"
  description = "Allow all traffic if you are on the VPN"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["10.86.0.0/16"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Allow ssh traffic from anywhere
resource "aws_security_group" "ssh" {
  name        = "ssh"
  description = "Allow SSH traffic"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Allow openvpn traffic from anywhere
resource "aws_security_group" "openvpn" {
  name        = "openvpn"
  description = "Allow OpenVPN traffic"

  ingress {
    from_port   = 1194
    to_port     = 1194
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Allow DNS traffic from anywhere
resource "aws_security_group" "dns" {
  name        = "dns"
  description = "Allow DNS traffic"

  ingress {
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 53
    to_port     = 53
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Public is the set of ports we want exposed to everyone
# on public services. Currently, this is just 9090 for
# user-node.
resource "aws_security_group" "public" {
  name        = "public"
  description = "Allowed ports for public services"

  ingress {
    from_port   = 9090
    to_port     = 9090
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

resource "aws_security_group" "public_http" {
  name        = "public_http"
  description = "Allowed ports for public services"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

resource "aws_security_group" "public_https" {
  name        = "public_https"
  description = "Allowed ports for public services"

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Labeling is the set of ports we want exposed to the public
# for labeling services.
resource "aws_security_group" "labeling" {
  name        = "labeling"
  description = "Allowed ports for labeling services"

  ingress {
    from_port   = 4040
    to_port     = 4040
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Relative title labeler
  ingress {
    from_port   = 8090
    to_port     = 8090
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 3003
    to_port     = 3003
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Staging/dev curation app
  ingress {
    from_port   = 8888
    to_port     = 8888
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 2020
    to_port     = 2020
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Nginx
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Ports open for the ranking service
resource "aws_security_group" "ranking" {
  name        = "ranking"
  description = "Allowed ports for ranking services"

  # Nginx
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 3012
    to_port     = 3012
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}

# Allow vpn-tunnel traffic
resource "aws_security_group" "vpn-tunnel" {
  name        = "vpn-tunnel"
  description = "Allow vpn-tunnel traffic"

  ingress {
    from_port   = 500
    to_port     = 500
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 4500
    to_port     = 4500
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 50
    to_port     = 50
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 51
    to_port     = 51
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["10.86.0.0/16"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = aws_vpc.dev.id
}
