terraform {
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/metrics-collector"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

data "terraform_remote_state" "deployed" {
  backend   = "s3"
  workspace = terraform.workspace

  config = {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/metrics-collector"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

provider "aws" {
  region = var.region
}

data "aws_vpc" "prod" {
  filter {
    name   = "tag:Name"
    values = ["kite-prod"]
  }
}

data "aws_ami" "kite_base" {
  most_recent = true
  name_regex  = "^kite_base_bionic_[0-9]+$"
  owners      = ["self"]
}

data "aws_subnet_ids" "public" {
  vpc_id = data.aws_vpc.prod.id

  tags = {
    tier = "public"
  }
}

data "aws_subnet_ids" "private" {
  vpc_id = data.aws_vpc.prod.id

  tags = {
    tier = "private"
  }
}

locals {
  deployed_versions = data.terraform_remote_state.deployed.outputs.versions
  versions_raw = {
    for color, version in var.versions : color => lookup(local.deployed_versions, version, version)
  }
  versions          = { for color, version in local.versions_raw : color => version if color != version }
  colors_by_version = zipmap(values(local.versions), keys(local.versions))
}

### Security Groups ###
data "aws_security_group" "https" {
  name = "https"
}

data "aws_security_group" "ssh" {
  name = "ssh"
}

data "aws_security_group" "all_vpn" {
  name = "all-vpn"
}

resource "aws_security_group" "collector" {
  name        = "metrics-collector"
  description = "Allow metrics-collector traffic"

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = data.aws_vpc.prod.id
}

#######

resource "aws_alb" "blue" {
  name         = "metrics-collector-alb-blue"
  idle_timeout = 300

  subnets = data.aws_subnet_ids.public.ids

  security_groups = [
    data.aws_security_group.https.id
  ]

  tags = merge(var.default_tags, { "Name" : "metrics-collector-alb" })
}

data "aws_acm_certificate" "star_kite_com" {
  domain   = "*.kite.com"
  statuses = ["ISSUED"]
}

resource "aws_alb_listener" "prod-alb-https-listener" {
  for_each = { for k, v in local.versions : k => v if k == "blue" }

  load_balancer_arn = aws_alb.blue.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2015-05"
  certificate_arn   = data.aws_acm_certificate.star_kite_com.arn

  default_action {
    target_group_arn = aws_alb_target_group.collector[each.value].arn
    type             = "forward"
  }
}

resource "aws_alb_target_group" "collector" {
  for_each = local.colors_by_version

  name_prefix          = "metcol"
  port                 = "8080"
  protocol             = "HTTP"
  deregistration_delay = 300
  vpc_id               = data.aws_vpc.prod.id

  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 2
    interval            = 5
    path                = "/.ping"
    port                = "8080"
    protocol            = "HTTP"
  }

  tags = merge(var.default_tags, { "Name" : "metrics-collector-${each.value}", "ReleaseVersion" : each.key, "ReleasePhase" : each.value })
}


resource "aws_launch_template" "collector" {
  for_each = local.colors_by_version

  name_prefix   = "metrics-collector-"
  image_id      = data.aws_ami.kite_base.id
  instance_type = "c5.large"
  key_name      = var.ec2_prod_key_name

  vpc_security_group_ids = [
    data.aws_security_group.ssh.id,
    data.aws_security_group.all_vpn.id,
    aws_security_group.collector.id,
  ]

  block_device_mappings {
    device_name = "/dev/sda1"

    ebs {
      volume_size = 20
    }
  }

  user_data = base64encode(templatefile("${path.module}/../userdata.tmpl",
  { node_name : "metrics-collector", release_version : each.key, region : var.region }))

  iam_instance_profile {
    arn = "arn:aws:iam::XXXXXXX:instance-profile/metrics_collector_profile"
  }

  tags = merge(var.default_tags, { "Name" : "metrics-collector-template", "ReleaseVersion" : each.key, "ReleasePhase" : each.value })

  tag_specifications {
    resource_type = "instance"
    tags          = merge(var.default_tags, { "Name" : "metrics-collector", "ReleaseVersion" : each.key, "ReleasePhase" : each.value })
  }
  lifecycle {
    ignore_changes: 'ALL'
  }
}


resource "aws_autoscaling_group" "collector" {
  for_each = local.colors_by_version

  name                      = replace(substr("metrics-collector-${each.key}", 0, 255), "/[^0-9a-zA-Z-]/", "-")
  max_size                  = 2
  min_size                  = 2
  health_check_grace_period = 300
  health_check_type         = "ELB"
  desired_capacity          = 2
  force_delete              = true
  placement_group           = aws_placement_group.collector.id
  target_group_arns         = [aws_alb_target_group.collector[each.key].arn]
  vpc_zone_identifier       = data.aws_subnet_ids.private.ids

  launch_template {
    id      = aws_launch_template.collector[each.key].id
    version = aws_launch_template.collector[each.key].version
  }

  timeouts {
    delete = "15m"
  }

  dynamic "tag" {
    for_each = merge(var.default_tags, { "Name" : "metrics-collector-asg", "ReleaseVersion" : each.key, "ReleasePhase" : each.value })
    content {
      key                 = tag.key
      value               = tag.value
      propagate_at_launch = false
    }
  }

}

resource "aws_placement_group" "collector" {
  name     = "metrics-collector"
  strategy = "partition"
}
