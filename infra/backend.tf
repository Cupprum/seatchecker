terraform {
  backend "s3" {
    bucket         = "seatchecker-terraform-state"
    key            = "backend/terraform.state"
    region         = "eu-central-1"
    acl            = "bucket-owner-full-control"
    dynamodb_table = "seatchecker-terraform-state-lock"
  }
}