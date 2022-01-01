# OpenVPN ---------------------------------------

resource "aws_instance" "openvpn" {
  ami                    = var.aws_ubuntu_ami
  availability_zone      = var.aws_az
  instance_type          = "t2.micro"
  key_name               = var.aws_key_name
  vpc_security_group_ids = [aws_security_group.openvpn.id, aws_security_group.ssh.id]
  subnet_id              = aws_subnet.us-west-1b-public.id

  # Needed for OpenVPN
  source_dest_check = false

  tags = {
    Name = "openvpn"
  }
}

resource "aws_eip" "openvpn" {
  instance = aws_instance.openvpn.id
  vpc      = true
}

# Labeling -------------------------------------------
# NOTE: This is curation!

resource "aws_instance" "labeling" {
  ami               = var.aws_ubuntu_ami_dev
  availability_zone = var.aws_az
  instance_type     = "t2.micro"
  key_name          = var.aws_key_name
  subnet_id         = aws_subnet.us-west-1b-public.id

  vpc_security_group_ids = [
    aws_security_group.labeling.id,
    aws_security_group.allow_all_vpn.id,
  ]

  private_ip = "10.86.0.20"

  tags = {
    Name = "curation"
  }
}

resource "aws_eip" "labeling" {
  instance = aws_instance.labeling.id
  vpc      = true
}

# Plugin gateway --------------------------------

resource "aws_instance" "plugins-gateway" {
  ami               = var.aws_ubuntu_ami_dev
  availability_zone = var.aws_az
  instance_type     = "t2.small"
  key_name          = var.aws_key_name
  subnet_id         = aws_subnet.us-west-1b-public.id

  vpc_security_group_ids = [
    aws_security_group.allow_all_vpn.id,
    aws_security_group.public_https.id,
    aws_security_group.public_http.id,
  ]

  private_ip = "10.86.0.22"

  tags = {
    Name = "plugins"
  }
}

resource "aws_eip" "plugins-gateway" {
  instance = aws_instance.plugins-gateway.id
  vpc      = true
}

# VPN Tunnel ------------------------------------

resource "aws_instance" "vpn-tunnel" {
  ami               = var.aws_ubuntu_ami_dev
  availability_zone = var.aws_az
  instance_type     = "t2.micro"
  key_name          = var.aws_key_name
  subnet_id         = aws_subnet.us-west-1b-public.id
  source_dest_check = false

  vpc_security_group_ids = [
    aws_security_group.ssh.id,
    aws_security_group.vpn-tunnel.id,
  ]

  tags = {
    Name = "vpn-tunnel"
  }
}

resource "aws_eip" "vpn-tunnel" {
  instance = aws_instance.vpn-tunnel.id
  vpc      = true
}
