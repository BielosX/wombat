terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      version = ">= 6.5.0"
      source  = "hashicorp/aws"
    }
  }
}

variable "region" {
  type = string
}

variable "source_bucket" {
  type    = string
  default = ""
}

provider "aws" {
  region = var.region
}

resource "aws_glue_catalog_database" "catalog_database" {
  name = "pokemon-crawler-db"
}

data "aws_s3_bucket" "data_bucket" {
  bucket = var.source_bucket
}

output "source_bucket" {
  value = data.aws_s3_bucket.data_bucket.id
}

data "aws_iam_policy_document" "s3_policy" {
  statement {
    effect  = "Allow"
    actions = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      data.aws_s3_bucket.data_bucket.arn,
      "${data.aws_s3_bucket.data_bucket.arn}/*"
    ]
  }
}

resource "aws_iam_policy" "policy" {
  policy = data.aws_iam_policy_document.s3_policy.json
}

data "aws_iam_policy_document" "glue_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["glue.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role" "crawler_role" {
  assume_role_policy = data.aws_iam_policy_document.glue_assume_role.json
}

resource "aws_iam_policy_attachment" "policy_attachment" {
  name       = "s3-policy-attachment"
  policy_arn = aws_iam_policy.policy.arn
  roles      = [aws_iam_role.crawler_role.id]
}

resource "aws_iam_policy_attachment" "glue_service_role" {
  name       = "glue-service-role"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSGlueServiceRole"
  roles      = [aws_iam_role.crawler_role.id]
}

resource "aws_glue_crawler" "glue_crawler" {
  for_each      = toset(["csv", "parquet"])
  database_name = aws_glue_catalog_database.catalog_database.name
  name          = "pokemon-${each.value}-crawler"
  role          = aws_iam_role.crawler_role.arn
  classifiers   = [] // use builtin csv and parquet

  s3_target {
    path = "s3://${data.aws_s3_bucket.data_bucket.id}/pokemons/${each.value}/"
  }

  schema_change_policy {
    delete_behavior = "LOG"
    update_behavior = "UPDATE_IN_DATABASE"
  }

  configuration = jsonencode(
    {
      Version = 1.0,
      CrawlerOutput = {
        Partitions = {
          AddOrUpdateBehavior = "InheritFromTable"
        }
      }
      Grouping = {
        TableGroupingPolicy = "CombineCompatibleSchemas"
      }
      CreatePartitionIndex = true
    }
  )
}