terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      version = ">= 6.5.0"
      source  = "hashicorp/aws"
    }
  }
}

provider "aws" {
  region = "eu-west-1"
}

variable "zip_path" {
  type    = string
  default = ""
}

data "aws_iam_policy_document" "lambda_assume_policy" {
  statement {
    actions = ["sts:AssumeRole"]
    effect  = "Allow"
    principals {
      identifiers = ["lambda.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role" "lambda_role" {
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_policy.json
}

resource "aws_iam_policy_attachment" "basic_execution" {
  name       = "basic-execution-role"
  roles      = [aws_iam_role.lambda_role.id]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

locals {
  lambdas = ["scheduler", "scraper"]
}

resource "aws_lambda_function" "functions" {
  for_each         = toset(local.lambdas)
  function_name    = "poke-${each.value}"
  handler          = each.value
  role             = aws_iam_role.lambda_role.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  filename         = var.zip_path
  source_code_hash = filebase64sha256(var.zip_path)
  timeout          = 60 * 3
  memory_size      = 512
}

data "aws_iam_policy_document" "sfn_assume_policy" {
  statement {
    actions = ["sts:AssumeRole"]
    effect  = "Allow"
    principals {
      identifiers = ["states.amazonaws.com"]
      type        = "Service"
    }
  }
}

data "aws_iam_policy_document" "sfn_role_policy" {
  statement {
    actions = ["lambda:InvokeFunction"]
    effect  = "Allow"
    resources = [
      aws_lambda_function.functions["scheduler"].arn,
      aws_lambda_function.functions["scraper"].arn
    ]
  }
}

resource "aws_iam_role" "sfn_role" {
  assume_role_policy = data.aws_iam_policy_document.sfn_assume_policy.json
}

resource "aws_iam_policy" "invoke_lambdas" {
  name   = "invoke-lambdas"
  policy = data.aws_iam_policy_document.sfn_role_policy.json
}

resource "aws_iam_policy_attachment" "sfn_policy" {
  name       = "sfn-policy"
  roles      = [aws_iam_role.sfn_role.id]
  policy_arn = aws_iam_policy.invoke_lambdas.arn
}

resource "aws_sfn_state_machine" "state_machine" {
  name = "poke-scraper-sfn"
  definition = templatefile("${path.module}/sfn_definition.json", {
    scheduler_arn : aws_lambda_function.functions["scheduler"].arn
    scraper_arn : aws_lambda_function.functions["scraper"].arn
  })
  role_arn = aws_iam_role.sfn_role.arn
}