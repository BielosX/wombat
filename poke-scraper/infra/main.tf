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

data "aws_iam_policy_document" "assume_policy" {
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
  assume_role_policy = data.aws_iam_policy_document.assume_policy.json
}

resource "aws_iam_policy_attachment" "basic_execution" {
  name       = "basic-execution-role"
  roles      = [aws_iam_role.lambda_role.id]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

locals {
  lambdas = ["scheduler", "scraper"]
}

resource "aws_lambda_function" "scheduler" {
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