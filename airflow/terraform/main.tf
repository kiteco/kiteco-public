terraform {
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "deployments/airflow"
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

resource "aws_ecs_cluster" "airflow" {
  name = var.service_name
  capacity_providers = ["FARGATE"]
}

resource "aws_iam_role" "airflow_task_execution" {
  name = "instance_role_airflow"

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

resource "aws_iam_role_policy" "airflow_task_execution" {
  name = "airflow-execution-policy"
  role = aws_iam_role.airflow_task_execution.id

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
                "${data.aws_secretsmanager_secret.sql_alchemy_conn_str.arn}",
                "${data.aws_secretsmanager_secret.result_db_uri.arn}"
            ]
        }
    ]
}
EOF
}

resource "aws_iam_role" "airflow_task" {
  name = "airflow-container-role"

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

resource "aws_iam_role_policy_attachment" "airflow-ecr" {
  role       = aws_iam_role.airflow_task.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_role_policy_attachment" "airflow-sm" {
  role       = aws_iam_role.airflow_task.name
  policy_arn = "arn:aws:iam::aws:policy/SecretsManagerReadWrite"
}

resource "aws_iam_role_policy_attachment" "airflow-s3" {
  role       = aws_iam_role.airflow_task.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
}

resource "aws_iam_role_policy_attachment" "airflow-athena" {
  role       = aws_iam_role.airflow_task.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonAthenaFullAccess"
}

resource "aws_iam_role_policy_attachment" "airflow-ecs" {
  role       = aws_iam_role.airflow_task.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonECS_FullAccess"
}

resource "aws_iam_role_policy" "airflow-cloudwatch" {
  name = "airflow-cloudwatch-policy"
  role = aws_iam_role.airflow_task.id

  policy = <<-EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents",
                "logs:DescribeLogStreams"
            ],
            "Resource": [
                "arn:aws:logs:*:*:*"
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

data "aws_security_group" "vpn" {
  name   = "all-vpn"
  vpc_id = data.aws_vpc.kite_prod.id
}

resource "aws_security_group" "airflow" {
  name = "Airflow"
  description = "Airflow test security group"
  vpc_id = data.aws_vpc.kite_prod.id

  ingress {
    from_port = 8080
    to_port = 8080
    protocol = "TCP"

    self = true
    security_groups = [data.aws_security_group.vpn.id]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

data "aws_secretsmanager_secret" "sql_alchemy_conn_str" {
  provider = aws.uswest1
  name     = "airflow/db_uri"
}

data "aws_secretsmanager_secret" "result_db_uri" {
  provider = aws.uswest1
  name     = "airflow/result_db_uri"
}

# Verify the image is published
data "aws_ecr_image" "airflow" {
  provider = aws.uswest1

  repository_name = var.repository_name
  image_tag       = var.tag
}

resource "aws_ecs_task_definition" "airflow" {
  for_each = var.tasks

  family = each.key
  container_definitions = jsonencode(
    [
      {
        "name" = each.key,
        "image" = "${data.aws_ecr_image.airflow.registry_id}.dkr.ecr.us-west-1.amazonaws.com/${var.repository_name}:${var.tag}",
        "portMappings" = [
          {
            "containerPort" = each.value.port,
            "protocol" = "tcp"
          }
        ],
        "essential" = true,
        "entryPoint" = ["airflow", each.key],
        "environment" = [
          { "name" = "AIRFLOW_VAR_ENV", "value" = "production" },
        ],
        "secrets" = [
          { "name" = "AIRFLOW__CORE__SQL_ALCHEMY_CONN", "valueFrom" = data.aws_secretsmanager_secret.sql_alchemy_conn_str.arn },
          { "name" = "AIRFLOW__CELERY__RESULT_BACKEND", "valueFrom" = data.aws_secretsmanager_secret.result_db_uri.arn }
        ],
        "logConfiguration" = {
          "logDriver" = "awslogs",
          "options" = {
            "awslogs-create-group" = "true",
            "awslogs-region" = var.region,
            "awslogs-group" = "/ecs/airflow/${each.key}",
            "awslogs-stream-prefix" = "ecs"
          }
        }
      }
    ]
  )

  requires_compatibilities = ["FARGATE"]
  network_mode = "awsvpc"

  execution_role_arn = aws_iam_role.airflow_task_execution.arn
  task_role_arn = aws_iam_role.airflow_task.arn

  cpu = each.value.cpu
  memory = each.value.memory
}

resource "aws_appautoscaling_target" "worker" {
  max_capacity       = 8
  min_capacity       = 1
  resource_id        = "service/${aws_ecs_cluster.airflow.name}/${aws_ecs_service.airflow["worker"].name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "worker_policy_memory" {
  name               = "memory-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.worker.resource_id
  scalable_dimension = aws_appautoscaling_target.worker.scalable_dimension
  service_namespace  = aws_appautoscaling_target.worker.service_namespace

  target_tracking_scaling_policy_configuration {
   predefined_metric_specification {
     predefined_metric_type = "ECSServiceAverageMemoryUtilization"
   }

   target_value       = 70
   scale_in_cooldown  = 60
   scale_out_cooldown = 120
  }
}

resource "aws_appautoscaling_policy" "worker_policy_cpu" {
  name               = "cpu-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.worker.resource_id
  scalable_dimension = aws_appautoscaling_target.worker.scalable_dimension
  service_namespace  = aws_appautoscaling_target.worker.service_namespace

  target_tracking_scaling_policy_configuration {
   predefined_metric_specification {
     predefined_metric_type = "ECSServiceAverageCPUUtilization"
   }

   target_value       = 70
   scale_in_cooldown  = 60
   scale_out_cooldown = 120
  }
}

resource "aws_ecs_service" "airflow" {
  for_each = var.tasks

  name = each.key
  cluster = aws_ecs_cluster.airflow.arn
  launch_type = "FARGATE"
  platform_version = "LATEST"
  task_definition = aws_ecs_task_definition.airflow[each.key].arn
  desired_count = 1

  network_configuration {
    subnets = [data.aws_subnet.private1.id]
    security_groups = [aws_security_group.airflow.id]
  }

  dynamic "load_balancer" {
    for_each = each.value.load_balancer ? [1] : []

    content {
      target_group_arn = aws_lb_target_group.airflow.arn
      container_name = each.key
      container_port = each.value.port
    }
  }

  enable_ecs_managed_tags = true
  propagate_tags = "TASK_DEFINITION"

  lifecycle {
    ignore_changes = [desired_count]
  }
}

resource "aws_lb" "airflow" {
  name = "airflow"
  subnets = [data.aws_subnet.private1.id, data.aws_subnet.private2.id]
  load_balancer_type = "application"
  internal = true
  security_groups = [data.aws_security_group.vpn.id]
}

resource "aws_lb_target_group" "airflow" {
  name = "airflow"
  port = var.webserver_port
  protocol = "HTTP"
  vpc_id = data.aws_vpc.kite_prod.id
  target_type = "ip"

  health_check {
    path = "/health"
    matcher = "200"
    interval = 300
  }
}

data "aws_acm_certificate" "kite_dev" {
  domain   = "*.kite.dev"
  statuses = ["ISSUED"]
}

resource "aws_lb_listener" "airflow" {
  load_balancer_arn = aws_lb.airflow.arn
  port = 443
  protocol = "HTTPS"

  default_action {
    target_group_arn = aws_lb_target_group.airflow.arn
    type = "forward"
  }

  certificate_arn = data.aws_acm_certificate.kite_dev.arn
}
