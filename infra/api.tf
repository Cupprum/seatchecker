data "aws_iam_policy_document" "apigw_assume_role" {
  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["apigateway.amazonaws.com"]
    }
    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "iam_for_apigw" {
  name               = "iam_for_apigw"
  assume_role_policy = data.aws_iam_policy_document.apigw_assume_role.json
}

resource "aws_iam_policy" "iam_for_apigw" {
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = ["states:StartExecution"]
        Effect   = "Allow"
        Resource = module.step-functions.state_machine_arn
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "iam_for_apigw" {
  role       = aws_iam_role.iam_for_apigw.name
  policy_arn = aws_iam_policy.iam_for_apigw.arn
}


resource "aws_apigatewayv2_api" "seatchecker" {
  name          = "seatchecker_api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "trigger_step_function" {
  api_id                 = aws_apigatewayv2_api.seatchecker.id
  description            = "Invoke Step Functions"
  integration_type       = "AWS_PROXY"
  integration_subtype    = "StepFunctions-StartExecution"
  credentials_arn        = aws_iam_role.iam_for_apigw.arn
  payload_format_version = "1.0"
  request_parameters = {
    "StateMachineArn" = module.step-functions.state_machine_arn
    "Input"           = "$request.body",
  }
}

resource "aws_apigatewayv2_route" "trigger_step_function" {
  api_id    = aws_apigatewayv2_api.seatchecker.id
  route_key = "POST /test"
  target    = "integrations/${aws_apigatewayv2_integration.trigger_step_function.id}"
}

resource "aws_apigatewayv2_stage" "deployment" {
  api_id      = aws_apigatewayv2_api.seatchecker.id
  name        = "$default"
  auto_deploy = true
}
