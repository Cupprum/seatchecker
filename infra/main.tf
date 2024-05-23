terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = "eu-central-1"
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "iam_for_lambda" {
  name               = "iam_for_lambda"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_lambda_function" "test_lambda" {
  # If the file is not in the current working directory you will need to include a
  # path.module in the filename.
  filename      = "seatChecker.zip"
  function_name = "seat_checker"
  role          = aws_iam_role.iam_for_lambda.arn
  handler       = "bootstrap"
  architectures = [ "arm64" ]

  runtime = "provided.al2023"

  environment {
    variables = {
      SEATCHECKER_EMAIL    = "xxx"
      SEATCHECKER_PASSWORD = "xxx"
    }
  }
}