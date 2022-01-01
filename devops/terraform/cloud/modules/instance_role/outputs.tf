output "service_account_email" {
  value = google_service_account.sa.email
}

output "aws_acces_key_id" {
  value = aws_iam_access_key.instance_role.id
}

output "gcp_aws_secret_access_key" {
  value = google_secret_manager_secret.secret.secret_id
}
