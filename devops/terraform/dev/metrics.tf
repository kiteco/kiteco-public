resource "aws_instance" "mixpanel-etl" {
  ami                    = "ami-03ba3948f6c37a4b0" // Ubuntu 18.04
  availability_zone      = var.aws_az
  instance_type          = "r5.2xlarge"
  ebs_optimized          = true
  key_name               = var.aws_key_name
  subnet_id              = aws_subnet.us-west-1b-private.id
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  private_ip             = "10.86.1.65"

  tags = {
    Name = "mixpanel-etl"
  }

  root_block_device {
    volume_type           = "gp2"
    volume_size           = 1024
    delete_on_termination = true
  }
}

resource "aws_instance" "metrics" {
  ami                    = var.aws_ubuntu_ami_dev_1604
  availability_zone      = var.aws_az
  instance_type          = "t2.medium"
  key_name               = var.aws_key_name
  subnet_id              = aws_subnet.us-west-1b-private.id
  vpc_security_group_ids = [aws_security_group.allow_all_vpn.id]
  private_ip             = "10.86.1.77"

  tags = {
    Name = "metrics"
  }
}


# # Lambda functions

resource "aws_iam_role" "lambda-telemetry-loader" {
  name = "lambda-telemetry-loader"
  path = "/service-role/"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

# Local Code Indexing
resource "aws_lambda_function" "index-telemetry-loader-segment" {
  function_name = "index-telemetry-loader-segment"
  handler = "s3toes.handler"
  role = aws_iam_role.lambda-telemetry-loader.arn
  runtime = "python3.6"
  timeout = 63
}

resource "aws_lambda_function" "index-telemetry-loader" {
  provider = aws.east

  function_name = "index-telemetry-loader"
  handler = "s3toes.handler"
  role = aws_iam_role.lambda-telemetry-loader.arn
  runtime = "python3.6"
  timeout = 63
}

# Offline Completion Latency
resource "aws_lambda_function" "completion-performance" {
  function_name = "completion-performance"
  handler = "main.handler"
  role = aws_iam_role.lambda-telemetry-loader.arn
  runtime = "python3.7"
  timeout = 3
}

# kite_status
resource "aws_lambda_function" "telemetry-loader" {
  provider = aws.east

  function_name = "telemetry-loader"
  role = aws_iam_role.lambda-telemetry-loader.arn

  handler = "s3toes.handler"
  runtime = "python3.7"
  memory_size = 256
  timeout = 300

  dead_letter_config {
    target_arn = "arn:aws:sqs:us-east-1:XXXXXXX:telemetry-loader-dl"
  }
}

resource "aws_s3_bucket_notification" "kite-metrics" {
  provider = aws.east
  bucket = "kite-metrics"
  lambda_function {
    lambda_function_arn = aws_lambda_function.telemetry-loader.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "segment-logs/Rgn399rf0J/"
  }
  lambda_function {
    lambda_function_arn = "arn:aws:lambda:us-east-1:XXXXXXX:function:telemetry-loader-elastic"
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "firehose/kite_status/"
  }
  lambda_function {
    lambda_function_arn = aws_lambda_function.index-telemetry-loader.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "firehose/client_events/"
  }
}
resource "aws_s3_bucket_notification" "kite-offline-metrics" {
  bucket = "kite-offline-metrics"
  lambda_function {
    lambda_function_arn = aws_lambda_function.completion-performance.arn
    events              = ["s3:ObjectCreated:*"]
    filter_suffix       = ".json.gz"
  }
}
resource "aws_s3_bucket_notification" "kite-segment-backend-http-requests" {
  bucket = "kite-segment-backend-http-requests"
  lambda_function {
    lambda_function_arn = aws_lambda_function.index-telemetry-loader-segment.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "segment-logs/"
  }
}
