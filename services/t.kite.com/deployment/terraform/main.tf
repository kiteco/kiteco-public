terraform {
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "services/t.kite.com"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

provider "aws" {
  region = var.region
}

provider "aws" {
  region = "us-west-1"
  alias = "uswest1"
}

resource "aws_ecs_cluster" "service" {
  name = var.service_name
  capacity_providers = ["FARGATE"]
}

resource "aws_iam_role" "execution" {
  name = "services-${var.service_name}-ecs-execution-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "execution" {
  name = "services-${var.service_name}-ecs-execution-policy"
  role = aws_iam_role.execution.id

  policy = <<-EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetResourcePolicy",
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret",
        "secretsmanager:ListSecretVersionIds"
      ],
      "Resource": [
        "${data.aws_secretsmanager_secret.elastic_conn_str.arn}"
      ]
    }
  ]
}
EOF
}

resource "aws_iam_role" "task" {
  name = "services-${var.service_name}-ecs-task-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "kinesis" {
  name = "services-${var.service_name}-ecs-task-policy-kinesis"
  role = aws_iam_role.task.id

  policy = <<-EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "firehose:PutRecord",
                "firehose:PutRecordBatch"
            ],
            "Effect": "Allow",
            "Resource": [
                "arn:aws:firehose:us-east-1:XXXXXXX:deliverystream/*"
            ]
        }
    ]
}
EOF
}

data "aws_vpc" "kite_prod" {
  filter {
    name   = "tag:Name"
    values = ["kite-prod"]
  }
}

data "aws_subnet" "private1" {
  vpc_id = data.aws_vpc.kite_prod.id
  filter {
    name   = "tag:Name"
    values = ["az1-private"]
  }
}

data "aws_subnet" "private2" {
  vpc_id = data.aws_vpc.kite_prod.id
  filter {
    name   = "tag:Name"
    values = ["az2-private"]
  }
}

data "aws_subnet" "public1" {
  vpc_id = data.aws_vpc.kite_prod.id
  filter {
    name   = "tag:Name"
    values = ["az1-public"]
  }
}

data "aws_subnet" "public2" {
  vpc_id = data.aws_vpc.kite_prod.id
  filter {
    name   = "tag:Name"
    values = ["az2-public"]
  }
}

resource "aws_security_group" "private" {
  name = "service-${var.service_name}-private"
  description = "Internal firewall for service ${var.service_name}"
  vpc_id = data.aws_vpc.kite_prod.id

  ingress {
    from_port = var.webserver_port
    to_port = var.webserver_port
    protocol = "TCP"

    self = true
    security_groups = [aws_security_group.public.id]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "public" {
  name = "service-${var.service_name}-public"
  description = "Public load-balancer firewall for service ${var.service_name}"
  vpc_id = data.aws_vpc.kite_prod.id

  ingress {
    from_port = 443
    to_port = 443
    protocol = "TCP"

    self = true
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# Verify the image is published
data "aws_ecr_image" "webserver" {
  provider = aws.uswest1

  repository_name = var.webserver_repository_name
  image_tag       = var.tag
}

data "aws_ecr_image" "fluentd" {
  provider = aws.uswest1

  repository_name = var.fluentd_repository_name
  image_tag       = var.tag
}

data "aws_secretsmanager_secret" "elastic_conn_str" {
  provider = aws.uswest1
  name     = "ELASTIC_CONN_STR"
}

resource "aws_ecs_task_definition" "service" {
  family = var.service_name
  container_definitions = jsonencode(
    [
      {
        "name" = "webserver",
        "image" = "${data.aws_ecr_image.webserver.registry_id}.dkr.ecr.us-west-1.amazonaws.com/${var.webserver_repository_name}:${var.tag}",
        "portMappings" = [
          {
            "containerPort" = var.webserver_port,
            "hostPort" = var.webserver_port,
            "protocol" = "tcp"
          }
        ],
        "cpu": 0,
        "essential" = true,
        "environment" = [],
        "secrets" = [],
        "volumesFrom": [],
        "mountPoints": [],
        "healthCheck": {
          "command": [ "CMD-SHELL", "curl -f http://localhost:${var.webserver_port}/.ping || exit 1" ],
          "interval": 30,
          "retries": 3,
          "timeout": 5
        },
        "logConfiguration": {
            "logDriver": "awsfirelens"
        }
      },
      {
        "name" = "fluentd",
        "image" = "${data.aws_ecr_image.fluentd.registry_id}.dkr.ecr.us-west-1.amazonaws.com/${var.fluentd_repository_name}:${var.tag}",
        "cpu": 0,
        "essential" = true,
        "environment" = [],
        "secrets" = [
          { "name" = "ELASTIC_CONN_STR", "valueFrom" = data.aws_secretsmanager_secret.elastic_conn_str.arn },
        ],
        "portMappings": [],
        "volumesFrom": [],
        "mountPoints": [],
        "firelensConfiguration": {
            "type": "fluentd",
            "options": {
                "config-file-type": "file",
                "config-file-value": "/root/fluent.conf"
            }
        },
        "logConfiguration": {
            "logDriver": "awslogs",
            "options": {
                "awslogs-group": "firelens-container",
                "awslogs-region": "us-east-1",
                "awslogs-create-group": "true",
                "awslogs-stream-prefix": "firelens"
            }
        }
      }
    ]
  )

  requires_compatibilities = ["FARGATE"]
  network_mode = "awsvpc"

  execution_role_arn = aws_iam_role.execution.arn
  task_role_arn = aws_iam_role.task.arn

  cpu = var.cpu * 1024
  memory = var.memory * 1024
}

resource "aws_appautoscaling_target" "service" {
  max_capacity       = 4
  min_capacity       = 1
  resource_id        = "service/${aws_ecs_cluster.service.name}/${aws_ecs_service.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "service_policy_cpu" {
  name               = "cpu-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.service.resource_id
  scalable_dimension = aws_appautoscaling_target.service.scalable_dimension
  service_namespace  = aws_appautoscaling_target.service.service_namespace

  target_tracking_scaling_policy_configuration {
   predefined_metric_specification {
     predefined_metric_type = "ECSServiceAverageCPUUtilization"
   }

   target_value       = 70
   scale_in_cooldown  = 60
   scale_out_cooldown = 120
  }
}

resource "aws_ecs_service" "service" {
  name = var.service_name
  cluster = aws_ecs_cluster.service.arn
  launch_type = "FARGATE"
  platform_version = "LATEST"
  task_definition = aws_ecs_task_definition.service.arn
  desired_count = 1

  network_configuration {
    subnets = [data.aws_subnet.private1.id]
    security_groups = [aws_security_group.private.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.service.arn
    container_name = "webserver"
    container_port = var.webserver_port
  }

  enable_ecs_managed_tags = true
  propagate_tags = "TASK_DEFINITION"

  lifecycle {
    ignore_changes = [desired_count]
  }
}

resource "aws_lb" "service" {
  name = var.service_name
  subnets = [data.aws_subnet.public1.id, data.aws_subnet.public2.id]
  load_balancer_type = "application"
  internal = false
  security_groups = [aws_security_group.public.id]
}

resource "aws_lb_target_group" "service" {
  name = var.service_name
  port = var.webserver_port
  protocol = "HTTP"
  vpc_id = data.aws_vpc.kite_prod.id
  target_type = "ip"

  health_check {
    path = "/.ping"
    matcher = "200"
    interval = 300
  }
}

data "aws_acm_certificate" "kite_com" {
  domain   = "*.kite.com"
  statuses = ["ISSUED"]
}

resource "aws_lb_listener" "service" {
  load_balancer_arn = aws_lb.service.arn
  port = 443
  protocol = "HTTPS"

  default_action {
    target_group_arn = aws_lb_target_group.service.arn
    type = "forward"
  }

  certificate_arn = data.aws_acm_certificate.kite_com.arn
}
