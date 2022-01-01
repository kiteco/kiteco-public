terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    google = {
      source = "hashicorp/google"
    }
  }
  backend "s3" {
    bucket               = "kite-terraform-state"
    workspace_key_prefix = "services/am.kite.com"
    key                  = "terraform.tfstate"
    region               = "us-west-1"
  }
}

provider "aws" {
  region = var.aws_region
}

provider "google" {
  region  = var.gcp_region
  project = var.gcp_project
}

resource "google_compute_global_address" "svc" {
  name = "am-kite-com-${terraform.workspace}"
}

resource "google_service_account" "default" {
  account_id   = "svc-am-kite-com-${terraform.workspace}"
  display_name = "svc-am-kite-com-${terraform.workspace}"
}

resource "google_project_iam_member" "default" {
  project = var.gcp_project
  role    = "roles/iam.serviceAccountTokenCreator"
  member  = "serviceAccount:svc-am-kite-com-${terraform.workspace}@${var.gcp_project}.iam.gserviceaccount.com"
}

resource "google_service_account_iam_binding" "default" {
  service_account_id = google_service_account.default.name
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.gcp_project}.svc.id.goog[am-kite-com-${terraform.workspace}/service]",
  ]
}

data "google_monitoring_notification_channel" "slack" {
  display_name = "#devops-notifications"
  type = "slack"
}

resource "google_monitoring_alert_policy" "status_code_alarm" {
  display_name = "am.kite.com 5xx Errors"
  combiner     = "OR"
  conditions {
    display_name = "Ingress Request 5xx Errors"
    condition_threshold {
      filter     = "metric.type=\"loadbalancing.googleapis.com/https/request_count\" resource.type=\"https_lb_rule\" resource.label.\"project_id\"=\"${var.gcp_project}\" resource.label.\"backend_target_name\"=monitoring.regex.full_match(\".*-am-kite-com-${terraform.workspace}-user-mux-9090-.*\") metric.label.\"response_code_class\"=\"500\""
      duration   = "600s"
      comparison = "COMPARISON_GT"
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["metric.label.response_code"]
        per_series_aligner   = "ALIGN_RATE"
      }
      trigger {
        count = "1"
      }
    }
  }
  notification_channels = [data.google_monitoring_notification_channel.slack.name]
  count                 = terraform.workspace == "prod" ? 1 : 0
}

resource "google_monitoring_alert_policy" "latency_alarm" {
  display_name = "am.kite.com High Latency"
  combiner     = "OR"
  conditions {
    display_name = "am.kite.com LB Latency [p50]"
    condition_threshold {
      filter     = "metric.type=\"loadbalancing.googleapis.com/https/total_latencies\" resource.type=\"https_lb_rule\" resource.label.\"project_id\"=\"${var.gcp_project}\" resource.label.\"backend_target_name\"=monitoring.regex.full_match(\".*-am-kite-com-${terraform.workspace}-user-mux-9090-.*\")"
      duration   = "600s"
      comparison = "COMPARISON_GT"
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_PERCENTILE_50"
        per_series_aligner   = "ALIGN_DELTA"
      }
      threshold_value = 500.0
      trigger {
        count = "1"
      }
    }
  }
  notification_channels = [data.google_monitoring_notification_channel.slack.name]
  count                 = terraform.workspace == "prod" ? 1 : 0
}

resource "google_monitoring_alert_policy" "error_logs_alarm" {
  display_name = "user-mux Error Logs"
  combiner     = "OR"
  conditions {
    display_name = "user-mux Error Logs"
    condition_threshold {
      filter     = "metric.type=\"logging.googleapis.com/user/user-mux-errors\" resource.type=\"k8s_container\""
      duration   = "600s"
      comparison = "COMPARISON_GT"
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_SUM"
        per_series_aligner   = "ALIGN_DELTA"
      }
      threshold_value = 10.0
      trigger {
        count = "1"
      }
    }
  }
  notification_channels = [data.google_monitoring_notification_channel.slack.name]
  count                 = terraform.workspace == "prod" ? 1 : 0
  depends_on = [
    google_logging_metric.error_logs_metric
  ]
}

resource "google_logging_metric" "error_logs_metric" {
  name = "user-mux-errors"
  filter   = "resource.type=\"k8s_container\" resource.labels.project_id=\"kite-dev-XXXXXXX\" resource.labels.location=\"${var.gcp_region}\" resource.labels.cluster_name=\"prod-us-west-1\" resource.labels.namespace_name=\"am-kite-com-${terraform.workspace}\" labels.k8s-pod/app=\"user-mux\" severity>=ERROR"
  metric_descriptor {
    metric_kind = "DELTA"
    display_name = "project/${var.gcp_project}/metricDescriptors/logging.googleapis.com/user/user-mux-errors"
    value_type  = "INT64"
    unit = "1"
  }
  count = terraform.workspace == "prod" ? 1 : 0
}

resource "aws_iam_role" "role" {
  name = "svc-am-kite-com-${terraform.workspace}"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "accounts.google.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "accounts.google.com:sub": "${google_service_account.default.unique_id}"
        }
      }
    }
  ]
}
EOF
}

data "aws_iam_policy_document" "policy" {
  statement {
    sid       = "1"
    actions   = ["rds-db:connect"]
    resources = ["arn:aws:rds:::db:community-prod-db"]
  }
  statement {
    sid       = "2"
    actions   = ["s3:GetObject"]
    resources = ["arn:aws:s3:::kite-data/swot-student-domains/*"]
  }
  statement {
    sid       = "3"
    actions   = ["s3:GetBucketLocation"]
    resources = ["arn:aws:s3:::kite-data"]
  }
}

resource "aws_iam_policy" "policy" {
  name   = "svc-am-kite-com-${terraform.workspace}"
  path   = "/"
  policy = data.aws_iam_policy_document.policy.json
}

resource "aws_iam_role_policy_attachment" "default_attachment" {
  role       = aws_iam_role.role.name
  policy_arn = aws_iam_policy.policy.arn
}
