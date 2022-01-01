resource "aws_db_parameter_group" "us-west-1-db-params" {
  name        = "kite-prod-postgres94"
  family      = "postgres9.4"
  description = "Postgres 9.4 production parameter group"

  # Require SSL connections
  parameter {
    name  = "ssl"
    value = "1"
  }

  parameter {
    name  = "max_connections"
    value = "10000"
  }

  parameter {
    name  = "statement_timeout"
    value = "300000"
  }

  parameter {
    name  = "shared_preload_libraries"
    value = "pg_stat_statements"
  }
}

resource "aws_db_parameter_group" "us-west-1-db-params-psql-95" {
  name        = "kite-prod-postgres95"
  family      = "postgres9.5"
  description = "Postgres 9.5 production parameter group"

  # Require SSL connections
  parameter {
    name  = "ssl"
    value = "1"
  }

  parameter {
    name  = "max_connections"
    value = "10000"
  }

  parameter {
    name  = "statement_timeout"
    value = "300000"
  }

  parameter {
    name  = "shared_preload_libraries"
    value = "pg_stat_statements"
  }
}

resource "aws_db_instance" "localfiles2-prod-db" {
  identifier              = "localfiles2-prod-db"
  engine                  = "postgres"
  engine_version          = "9.5.6"
  backup_retention_period = 21

  snapshot_identifier = "${lookup(var.localfiles_snapshot, var.region)}"

  storage_type      = "gp2"
  allocated_storage = 250

  name     = "${var.localfiles_db_name}"
  username = "${var.localfiles_db_username}"
  password = "${var.localfiles_db_password}"

  instance_class    = "db.m4.xlarge"
  availability_zone = "${lookup(var.db_az1, var.region)}"

  vpc_security_group_ids = ["${aws_security_group.postgres.id}",
    "${aws_security_group.ssh.id}",
  ]

  db_subnet_group_name = "db-az1-az2-private-subnet-group"
  parameter_group_name = "kite-prod-postgres95"
}
