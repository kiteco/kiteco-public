resource "aws_key_pair" "kite-prod" {
  key_name   = "kite-prod"
  public_key = "ssh-rsa XXXXXXX"
}
