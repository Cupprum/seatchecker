data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "seatchecker_lambda_role" {
  name               = "seatchecker_lambda_role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"]
}

data "archive_file" "lambda_seatchecker_zip" {
  type        = "zip"
  source_dir  = "/out/seatchecker"
  output_path = "/out/seatchecker.zip"
}

resource "aws_lambda_function" "seatchecker" {
  function_name    = "seatchecker"
  role             = aws_iam_role.seatchecker_lambda_role.arn
  filename         = data.archive_file.lambda_seatchecker_zip.output_path
  source_code_hash = data.archive_file.lambda_seatchecker_zip.output_base64sha256
  architectures    = ["arm64"]
  runtime          = "provided.al2023"
  handler          = "bootstrap"

  environment {
    variables = {
      OTEL_SERVICE_NAME           = "seatchecker"
      OTEL_EXPORTER_OTLP_PROTOCOL = "http/protobuf"
      OTEL_EXPORTER_OTLP_ENDPOINT = "https://api.eu1.honeycomb.io"
      OTEL_EXPORTER_OTLP_HEADERS  = "x-honeycomb-team=${var.honeycomb_api_key}"
    }
  }
}
