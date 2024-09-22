terraform {
  backend "s3" {
    bucket         = "seatchecker-terraform-state"
    key            = "backend/terraform.state"
    region         = "eu-central-1"
    acl            = "bucket-owner-full-control"
    dynamodb_table = "seatchecker-terraform-state-lock"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "eu-central-1"
  default_tags {
    tags = {
      Project = "seatchecker"
    }
  }
}