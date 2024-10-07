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

resource "aws_iam_role" "seatchecker_apigw_role" {
  name               = "seatchecker_apigw_role"
  assume_role_policy = data.aws_iam_policy_document.apigw_assume_role.json
}

resource "aws_iam_policy" "seatchecker_apigw_role" {
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = ["states:StartExecution"]
        Effect   = "Allow"
        Resource = module.step-functions.state_machine_arn
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "seatchecker_apigw_role" {
  role       = aws_iam_role.seatchecker_apigw_role.name
  policy_arn = aws_iam_policy.seatchecker_apigw_role.arn
}


resource "aws_apigatewayv2_api" "seatchecker_api" {
  name          = "seatchecker_api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "trigger_step_function" {
  api_id                 = aws_apigatewayv2_api.seatchecker_api.id
  description            = "Invoke Step Functions"
  integration_type       = "AWS_PROXY"
  integration_subtype    = "StepFunctions-StartExecution"
  credentials_arn        = aws_iam_role.seatchecker_apigw_role.arn
  payload_format_version = "1.0"
  request_parameters = {
    "StateMachineArn" = module.step-functions.state_machine_arn
    "Input"           = "$request.body",
  }
}

resource "aws_apigatewayv2_route" "trigger_step_function" {
  api_id    = aws_apigatewayv2_api.seatchecker_api.id
  route_key = "POST /start"
  target    = "integrations/${aws_apigatewayv2_integration.trigger_step_function.id}"
}

resource "aws_apigatewayv2_integration" "stop_step_function" {
  api_id                 = aws_apigatewayv2_api.seatchecker_api.id
  description            = "Stop Step Functions Execution"
  integration_type       = "AWS_PROXY"
  integration_subtype    = "StepFunctions-StopExecution"
  credentials_arn        = aws_iam_role.seatchecker_apigw_role.arn
  payload_format_version = "1.0"
  request_parameters = {
    "ExecutionArn" = "$request.body.executionArn",
  }
}

resource "aws_apigatewayv2_route" "stop_step_function" {
  api_id    = aws_apigatewayv2_api.seatchecker_api.id
  route_key = "POST /stop"
  target    = "integrations/${aws_apigatewayv2_integration.stop_step_function.id}"
}

resource "aws_apigatewayv2_stage" "deployment" {
  api_id      = aws_apigatewayv2_api.seatchecker_api.id
  name        = "$default"
  auto_deploy = true
}
