resource "aws_iam_user" "instance_role" {
  name = "instance_role_${var.name}"
  path = "/instance_roles/"
}

resource "aws_iam_access_key" "instance_role" {
  user = aws_iam_user.instance_role.name
}

resource "google_secret_manager_secret" "secret" {
  provider  = google-beta
  secret_id = "instance_role_${var.name}_secret"

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "secret" {
  provider    = google-beta
  secret      = google_secret_manager_secret.secret.id
  secret_data = aws_iam_access_key.instance_role.secret
}

data "aws_secretsmanager_secret" "secrets" {
  for_each = toset(concat(var.secrets, var.default_secrets))
  name     = each.value
}

data "aws_iam_policy_document" "default_policy" {
  statement {
    sid       = "1"
    actions   = ["s3:GetObject"]
    resources = ["arn:aws:s3:::kite-deploys/*"]
  }

  statement {
    sid       = "2"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [for secret in data.aws_secretsmanager_secret.secrets : secret.arn]
  }

  dynamic statement {
    for_each = var.policy_statements
    content {
      sid = 3 + statement.key
      actions = statement.value["actions"]
      resources = statement.value["resources"]
    }
  }
}

resource "aws_iam_policy" "default_policy" {
  name   = "instance_role_${var.name}_policy"
  path   = "/"
  policy = data.aws_iam_policy_document.default_policy.json
}

resource "aws_iam_user_policy_attachment" "default_attachment" {
  user       = aws_iam_user.instance_role.name
  policy_arn = aws_iam_policy.default_policy.arn
}

resource "google_service_account" "sa" {
  provider   = google-beta
  account_id = "instance${replace(var.name, "_", "")}"
}

data "google_iam_policy" "secret_accessor" {
  binding {
    role = "roles/secretmanager.secretAccessor"
    members = [
      "serviceAccount:${google_service_account.sa.email}"
    ]
  }
}

resource "google_secret_manager_secret_iam_policy" "policy" {
  provider = google-beta

  secret_id   = google_secret_manager_secret.secret.secret_id
  policy_data = data.google_iam_policy.secret_accessor.policy_data
}
