# Bastion -------------------------------------------

resource "aws_instance" "bastion" {
    ami = var.aws_ubuntu_ami
    availability_zone = "us-west-1b"
    instance_type = "t2.micro"
    key_name = "kite-prod"
    subnet_id = aws_subnet.us-west-1b-public.id
    vpc_security_group_ids = [
        aws_security_group.ssh.id,
    ]
    tags = {
        Name = "bastion"
    }
}

resource "aws_eip" "bastion" {
    instance = aws_instance.bastion.id
    vpc=true
}

# Release -------------------------------------------

resource "aws_instance" "release" {
    ami = var.aws_ubuntu_ami
    private_ip = "172.86.0.21"
    availability_zone = "us-west-1b"
    iam_instance_profile = "release"
    instance_type = "t2.medium"
    cpu_credits = "unlimited"
    key_name = "kite-prod"
    subnet_id = aws_subnet.us-west-1b-public.id
    vpc_security_group_ids = [
        aws_security_group.https.id,
        aws_security_group.all-vpn.id,
    ]
    tags = {
        Name = "release"
    }
}

resource "aws_eip" "release" {
    instance = aws_instance.release.id
    vpc = true
}

resource "aws_instance" "stagingrelease" {
    ami = var.aws_ubuntu_ami
    private_ip = "172.86.1.21"
    availability_zone = "us-west-1b"
    iam_instance_profile = "release"
    instance_type = "t2.micro"
    key_name = "kite-prod"
    subnet_id = aws_subnet.us-west-1b-private.id
    vpc_security_group_ids = [
        aws_security_group.https.id,
        aws_security_group.all-vpn.id,
    ]
    tags = {
        Name = "stagingrelease"
    }
}

resource "aws_instance" "stagingrelease2" {
    ami = var.aws_ubuntu_ami
    private_ip = "172.86.0.22"
    availability_zone = "us-west-1b"
    iam_instance_profile = "release"
    instance_type = "t2.micro"
    key_name = "kite-prod"
    subnet_id = aws_subnet.us-west-1b-public.id
    vpc_security_group_ids = [
        aws_security_group.https.id,
        aws_security_group.all-vpn.id,
    ]
    tags = {
        Name = "stagingrelease2"
    }
}

resource "aws_eip" "stagingrelease" {
    instance = aws_instance.stagingrelease2.id
    vpc = true
}
