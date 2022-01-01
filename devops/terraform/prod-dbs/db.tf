# Community database ----------------------------

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

resource "aws_db_subnet_group" "us-west-1-public-db-subnet" {
  name        = "us-west-1-public-db-subnet"
  description = "Public db subnet group (us-west-1b+c)"

  subnet_ids = ["${aws_subnet.us-west-1b-public-db.id}",
    "${aws_subnet.us-west-1c-public-db.id}",
  ]
}

resource "aws_db_instance" "community-prod-db" {
  identifier              = "community-prod-db"
  engine                  = "postgres"
  engine_version          = "9.4.7"
  backup_retention_period = 21

  apply_immediately = true

  storage_type      = "gp2"
  allocated_storage = 60

  name     = "${var.community_db_name}"
  username = "${var.community_db_username}"
  password = "${var.community_db_password}"

  instance_class    = "db.m3.large"
  availability_zone = "${var.aws_az}"

  vpc_security_group_ids = ["${aws_security_group.postgres.id}",
    "${aws_security_group.ssh.id}",
  ]

  db_subnet_group_name = "us-west-1-public-db-subnet"
  parameter_group_name = "kite-prod-postgres94"
  publicly_accessible  = 1
}

resource "aws_db_instance" "release-prod-db" {
  identifier              = "release-prod-db"
  engine                  = "postgres"
  engine_version          = "9.4.7"
  backup_retention_period = 21

  apply_immediately = true

  storage_type      = "gp2"
  allocated_storage = 5

  name     = "${var.release_db_name}"
  username = "${var.release_db_username}"
  password = "${var.release_db_password}"

  instance_class    = "db.t1.micro"
  availability_zone = "${var.aws_az}"

  vpc_security_group_ids = ["${aws_security_group.postgres.id}",
    "${aws_security_group.ssh.id}",
  ]

  db_subnet_group_name = "us-west-1-public-db-subnet"
  parameter_group_name = "kite-prod-postgres94"
  publicly_accessible  = 1
}

resource "aws_db_instance" "localfiles-prod-db" {
  identifier              = "localfiles-prod-db"
  engine                  = "postgres"
  engine_version          = "9.5.6"
  backup_retention_period = 21

  snapshot_identifier = "rds:localfiles-prod-db-2017-06-13-10-00"

  storage_type      = "gp2"
  allocated_storage = 250

  name     = "${var.localfiles_db_name}"
  username = "${var.localfiles_db_username}"
  password = "${var.localfiles_db_password}"

  instance_class    = "db.m4.xlarge"
  availability_zone = "${var.aws_az}"

  vpc_security_group_ids = ["${aws_security_group.postgres.id}",
    "${aws_security_group.ssh.id}",
  ]

  db_subnet_group_name = "us-west-1-public-db-subnet"
  parameter_group_name = "kite-prod-postgres95"
  publicly_accessible  = 1
}

resource "aws_db_instance" "events-prod-db" {
  identifier              = "events-prod-db"
  engine                  = "postgres"
  engine_version          = "9.4.7"
  backup_retention_period = 21

  apply_immediately = true

  storage_type      = "gp2"
  allocated_storage = 30

  name     = "${var.events_db_name}"
  username = "${var.events_db_username}"
  password = "${var.events_db_password}"

  instance_class    = "db.t2.small"
  availability_zone = "${var.aws_az}"

  vpc_security_group_ids = ["${aws_security_group.postgres.id}",
    "${aws_security_group.ssh.id}",
  ]

  db_subnet_group_name = "us-west-1-public-db-subnet"
  parameter_group_name = "kite-prod-postgres94"
  publicly_accessible  = 1
}
