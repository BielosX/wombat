terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      version = ">= 6.5.0"
      source  = "hashicorp/aws"
    }
  }
}

variable "zip_path" {
  type    = string
  default = ""
}

variable "region" {
  type = string
}

provider "aws" {
  region = var.region
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

resource "aws_s3_bucket" "data_warehouse" {
  bucket_prefix = "data-warehouse-"
  force_destroy = true
}

output "data_bucket_arn" {
  value = aws_s3_bucket.data_warehouse.arn
}

output "data_bucket_name" {
  value = aws_s3_bucket.data_warehouse.id
}

resource "aws_s3_bucket_public_access_block" "data_warehouse_block" {
  bucket                  = aws_s3_bucket.data_warehouse.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "data_warehouse_ownership" {
  bucket = aws_s3_bucket.data_warehouse.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }

}

data "aws_iam_policy_document" "bucket_access" {
  statement {
    effect    = "Allow"
    actions   = ["s3:PutObject"]
    resources = ["${aws_s3_bucket.data_warehouse.arn}/*"]
  }
}

resource "aws_iam_policy" "bucket_access" {
  policy = data.aws_iam_policy_document.bucket_access.json
}

resource "aws_iam_role" "lambda_role" {
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_policy.json
}

resource "aws_iam_role_policy_attachment" "basic_execution" {
  role       = aws_iam_role.lambda_role.id
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "bucket_access" {
  role       = aws_iam_role.lambda_role.id
  policy_arn = aws_iam_policy.bucket_access.arn
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
  timeout          = 60 * 2
  memory_size      = 512
  environment {
    variables = {
      BUCKET_NAME = aws_s3_bucket.data_warehouse.id
    }
  }
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
  policy = data.aws_iam_policy_document.sfn_role_policy.json
}

resource "aws_iam_role_policy_attachment" "sfn_policy" {
  role       = aws_iam_role.sfn_role.id
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