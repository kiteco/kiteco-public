resource "aws_ecs_task_definition" "monetizable" {
  family = "monetizable"
  container_definitions = jsonencode(
    [
      {
        "name" = "monetizable",
        "image" = "${data.aws_ecr_image.airflow.registry_id}.dkr.ecr.us-west-1.amazonaws.com/kite-airflow-monetizable:${var.tag}",
        "essential" = true,
        "logConfiguration" = {
          "logDriver" = "awslogs",
          "options" = {
            "awslogs-create-group" = "true",
            "awslogs-region" = var.region,
            "awslogs-group" = "/ecs/airflow/monetizable",
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

  cpu = 1 * 1024.0
  memory = 2 * 1024.0
}
