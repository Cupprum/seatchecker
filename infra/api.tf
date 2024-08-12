resource "aws_apigatewayv2_api" "seatchecker" {
  name          = "seatchecker_api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "trigger_step_function" {
  api_id                 = aws_apigatewayv2_api.seatchecker.id
  description            = "Invoke Step Functions"
  integration_type       = "AWS_PROXY"
  integration_subtype    = "StepFunctions-StartExecution"
  credentials_arn        = module.step-functions.role_arn // TODO: create this role, which is assumable by apigw, and allows to trigger stepfunction
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
