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

data "aws_iam_policy" "lambda_basic_execution_policy" {
  arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_flow_log_cloudwatch" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = data.aws_iam_policy.lambda_basic_execution_policy.arn
}

data "archive_file" "lambda_seatchecker_zip" {
  type        = "zip"
  source_dir  = "/out/seatchecker"
  output_path = "/out/seatchecker.zip"
}

resource "aws_lambda_function" "seatchecker" {
  function_name    = "seatchecker"
  role             = aws_iam_role.iam_for_lambda.arn
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
