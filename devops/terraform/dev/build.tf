# Build machine for user-node

resource "aws_instance" "build" {
  ami                    = var.aws_ubuntu_ami_dev
  availability_zone      = var.aws_az
  instance_type          = "t2.medium"
  key_name               = var.aws_key_name
  subnet_id              = aws_subnet.us-west-1b-private.id
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  private_ip             = "10.86.1.40"

  root_block_device {
    volume_type           = "gp2"
    volume_size           = 1024
    delete_on_termination = false
  }

  tags = {
    Name = "build"
  }
}

# Concourse (Build Server)

resource "aws_instance" "concourse-master" {
  ami                    = "ami-0dd655843c87b6930" # Ubuntu 18.04
  availability_zone      = var.aws_az
  instance_type          = "t2.micro"
  key_name               = var.aws_key_name
  subnet_id              = aws_subnet.us-west-1b-public.id
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  private_ip             = "10.86.0.122"

  root_block_device {
    volume_type           = "gp2"
    volume_size           = 8
    delete_on_termination = true
  }

  tags = {
    Name = "concourse"
  }
}

resource "aws_instance" "concourse-worker" {
  ami                    = "ami-0dd655843c87b6930" # Ubuntu 18.04
  availability_zone      = var.aws_az
  instance_type          = "t2.large"
  key_name               = var.aws_key_name
  subnet_id              = aws_subnet.us-west-1b-private.id
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  private_ip             = "10.86.1.173"

  root_block_device {
    volume_type           = "gp2"
    volume_size           = 8
    delete_on_termination = true
  }

  ebs_block_device {
    device_name           = "/dev/sdf"
    volume_type           = "gp2"
    volume_size           = 100
    delete_on_termination = false
  }

  tags = {
    Name = "concourse-worker-linux0"
  }
}
