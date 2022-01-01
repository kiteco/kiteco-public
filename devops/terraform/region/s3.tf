resource "aws_s3_bucket" "localcontent-bucket" {
  bucket = "kite-local-content-${var.region}"
}

resource "aws_s3_bucket" "localsymbols-bucket" {
  bucket = "kite-local-symbols-${var.region}"
}

resource "aws_s3_bucket" "prod-data-bucket" {
  bucket = "kite-prod-data-${var.region}"
}
