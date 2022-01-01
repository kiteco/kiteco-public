# Mock updates release server ----------------------

resource "aws_instance" "mockrelease" {
    ami = var.aws_ubuntu_ami
    private_ip = "172.86.1.20"
    availability_zone = "us-west-1b"
    iam_instance_profile = "release"
    instance_type = "t2.micro"
    key_name = "kite-prod"
    subnet_id = aws_subnet.us-west-1b-private.id
    vpc_security_group_ids = [
        aws_security_group.all-vpn.id,
    ]
    tags = {
        Name = "mockrelease"
    }
}
