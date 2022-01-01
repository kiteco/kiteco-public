# DNS -------------------------------------------

resource "aws_instance" "dns" {
  ami                    = var.aws_ubuntu_ami
  availability_zone      = var.aws_az
  instance_type          = "t2.micro"
  key_name               = var.aws_key_name
  subnet_id              = aws_subnet.us-west-1b-private.id
  vpc_security_group_ids = [aws_security_group.ssh.id, aws_security_group.dns.id]
  private_ip             = "10.86.1.9"

  tags = {
    Name = "internal-dns"
  }
}

# Mock machine ---------------------------------

resource "aws_instance" "mock" {
  ami                    = var.aws_ubuntu_ami_dev
  availability_zone      = var.aws_az
  instance_type          = "t2.micro"
  key_name               = var.aws_key_name
  subnet_id              = aws_subnet.us-west-1b-private.id
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  private_ip             = "10.86.1.50"

  tags = {
    Name = "mock"
  }
}

