terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "eu-central-1"
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com", "states.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "iam_for_lambda" {
  name               = "iam_for_lambda"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

data "aws_iam_policy" "lambda_basic_execution_policy" {
  arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_flow_log_cloudwatch" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = data.aws_iam_policy.lambda_basic_execution_policy.arn
}

resource "aws_lambda_function" "seatchecker" {
  # If the file is not in the current working directory you will need to include a
  # path.module in the filename.
  filename      = "/out/seatchecker.zip"
  function_name = "seatchecker"
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


resource "aws_lambda_function" "notifier" {
  # If the file is not in the current working directory you will need to include a
  # path.module in the filename.
  filename      = "/out/notifier.zip"
  function_name = "notifier"
  role          = aws_iam_role.iam_for_lambda.arn
  handler       = "bootstrap"
  architectures = [ "arm64" ]

  runtime = "provided.al2023"
}
