# Community database ----------------------------

resource "aws_db_subnet_group" "us-west-1-private-db-subnet" {
  name        = "us-west-1-private-db-subnet"
  description = "Private db subnet group (us-west-1b+v)"

  subnet_ids = [aws_subnet.us-west-1b-private-db.id,
    aws_subnet.us-west-1c-private-db.id,
  ]
}

resource "aws_db_instance" "curation-db" {
  identifier              = "curation-db"
  engine                  = "postgres"
  engine_version          = "9.4.20"
  backup_retention_period = 21

  storage_type        = "gp2"
  allocated_storage   = 5
  skip_final_snapshot = true

  name     = var.curation_db_name
  username = var.curation_db_username
  password = var.curation_db_password

  instance_class         = "db.t2.micro"
  availability_zone      = var.aws_az
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  db_subnet_group_name   = "us-west-1-private-db-subnet"
  publicly_accessible    = false
}

resource "aws_db_instance" "plugin-db" {
  identifier              = "plugin-db"
  engine                  = "postgres"
  engine_version          = "9.4.20"
  backup_retention_period = 21

  storage_type        = "gp2"
  allocated_storage   = 512
  skip_final_snapshot = true

  name     = var.plugin_db_name
  username = var.plugin_db_username
  password = var.plugin_db_password

  instance_class         = "db.t2.micro"
  availability_zone      = var.aws_az
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  db_subnet_group_name   = "us-west-1-private-db-subnet"
  publicly_accessible    = false
}

resource "aws_db_instance" "main-db" {
  identifier              = "main-db"
  backup_retention_period = 21
  snapshot_identifier     = "main-db-snapshot"

  storage_type        = "standard"
  skip_final_snapshot = false

  instance_class         = "db.t2.micro"
  availability_zone      = var.aws_az
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  db_subnet_group_name   = "us-west-1-private-db-subnet"
  publicly_accessible    = false
}

resource "aws_db_instance" "linkedin-db" {
  identifier              = "linkedin-db"
  engine                  = "postgres"
  engine_version          = "9.4.20"
  backup_retention_period = 21

  apply_immediately = true

  storage_type        = "gp2"
  allocated_storage   = 3072
  skip_final_snapshot = true

  name     = var.linkedin_db_name
  username = var.linkedin_db_username
  password = var.linkedin_db_password

  instance_class         = "db.t2.micro"
  availability_zone      = var.aws_az
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  db_subnet_group_name   = "us-west-1-private-db-subnet"
  publicly_accessible    = false
}
