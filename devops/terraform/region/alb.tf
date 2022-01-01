# ALB ------------------------------------------

resource "aws_alb" "prod-alb" {
  name = "prod-alb"
  idle_timeout = 300

  subnets = [
    "${aws_subnet.az1-public.id}",
    "${aws_subnet.az2-public.id}",
  ]

  security_groups = [
    "${aws_security_group.https.id}",
  ]

  tags {
    Name = "prod-alb"
  }
}

resource "aws_alb_listener" "prod-alb-https-listener" {
  load_balancer_arn = "${aws_alb.prod-alb.arn}"
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2015-05"
  certificate_arn   = "${lookup(var.certificate_arn, var.region)}"

  default_action {
    target_group_arn = "${aws_alb_target_group.prod-alb-usernodes.arn}"
    type             = "forward"
  }
}

resource "aws_alb_target_group" "prod-alb-usernodes" {
  name                 = "prod-usernodes"
  port                 = "9090"
  protocol             = "HTTP"
  deregistration_delay = 0
  vpc_id               = "${aws_vpc.prod.id}"

  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 2
    interval            = 5
    path                = "/ready"
    port                = "9091"
    protocol            = "HTTP"
  }

  tags {
    Name = "prod-alb-usernodes-target-group"
  }
}

# Staging -------

resource "aws_alb" "staging-alb" {
  name = "staging-alb"
  idle_timeout = 300

  subnets = [
    "${aws_subnet.az1-public.id}",
    "${aws_subnet.az2-public.id}",
  ]

  security_groups = [
    "${aws_security_group.https.id}",
  ]

  tags {
    Name = "staging-alb"
  }
}

resource "aws_alb_listener" "staging-alb-https-listener" {
  load_balancer_arn = "${aws_alb.staging-alb.arn}"
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2015-05"
  certificate_arn   = "${lookup(var.certificate_arn, var.region)}"

  default_action {
    target_group_arn = "${aws_alb_target_group.staging-alb-usernodes.arn}"
    type             = "forward"
  }
}

resource "aws_alb_target_group" "staging-alb-usernodes" {
  name                 = "staging-usernodes"
  port                 = "9090"
  protocol             = "HTTP"
  deregistration_delay = 0
  vpc_id               = "${aws_vpc.prod.id}"

  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 2
    interval            = 5
    path                = "/ready"
    port                = "9091"
    protocol            = "HTTP"
  }

  tags {
    Name = "staging-alb-usernodes-target-group"
  }
}
