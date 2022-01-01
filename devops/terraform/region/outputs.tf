output "vpc-id" {
  value = "${aws_vpc.prod.id}"
}

output "availability_zone" {
  value = "${lookup(var.az1, var.region)}"
}

output "ami" {
  value = "${lookup(var.ubuntu_ami, var.region)}"
}

output "public-subnet-id" {
  value = "${aws_subnet.az1-public.id}"
}

output "private-subnet-id" {
  value = "${aws_subnet.az1-private.id}"
}

output "allow-all-vpn" {
  value = "${aws_security_group.all-vpn.id}"
}

output "allow-http" {
  value = "${aws_security_group.http.id}"
}

output "allow-https" {
  value = "${aws_security_group.https.id}"
}

output "allow-ssh" {
  value = "${aws_security_group.ssh.id}"
}

output "allow-usernode" {
  value = "${aws_security_group.usernode.id}"
}

output "allow-usernode-debug" {
  value = "${aws_security_group.usernode-debug.id}"
}
