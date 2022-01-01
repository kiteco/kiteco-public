# Bastion ---------------------------------------

resource "aws_instance" "bastion" {
  ami               = "${lookup(var.ubuntu_ami, var.region)}"
  availability_zone = "${lookup(var.az1, var.region)}"
  instance_type     = "t2.micro"
  key_name          = "${var.key_name}"
  subnet_id         = "${aws_subnet.az1-public.id}"

  vpc_security_group_ids = [
    "${aws_security_group.ssh.id}",
  ]

  tags {
    Name = "bastion"
  }
}

resource "aws_eip" "bastion" {
  instance = "${aws_instance.bastion.id}"
  vpc      = true
}

# VPN Tunnel ------------------------------------

resource "aws_instance" "vpn-tunnel" {
  ami               = "${lookup(var.ubuntu_ami, var.region)}"
  availability_zone = "${lookup(var.az1, var.region)}"
  instance_type     = "t2.micro"
  key_name          = "${var.key_name}"
  subnet_id         = "${aws_subnet.az1-public.id}"
  source_dest_check = false

  vpc_security_group_ids = [
    "${aws_security_group.ssh.id}",
    "${aws_security_group.vpn-tunnel.id}",
  ]

  tags {
    Name = "vpn-tunnel"
  }
}

resource "aws_eip" "vpn-tunnel" {
  instance = "${aws_instance.vpn-tunnel.id}"
  vpc      = true
}
