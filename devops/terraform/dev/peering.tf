resource "aws_vpc_peering_connection" "dev-prod-us-west-1" {
  peer_owner_id = var.aws_account_id
  peer_vpc_id   = var.prod_us_west_1_vpc_id
  vpc_id        = aws_vpc.dev.id
}
