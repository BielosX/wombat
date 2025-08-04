resource "aws_glue_catalog_database" "catalog_database" {
  name = "pokemon-crawler-db"
}

output "catalog_database" {
  value = aws_glue_catalog_database.catalog_database.name
}

data "aws_caller_identity" "current" {}

locals {
  data_bucket_arn  = data.terraform_remote_state.poke_scraper.outputs.data_bucket_arn
  data_bucket_name = data.terraform_remote_state.poke_scraper.outputs.data_bucket_name
  account_id       = data.aws_caller_identity.current.account_id
}


data "aws_iam_policy_document" "policy" {
  statement {
    effect  = "Allow"
    actions = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      local.data_bucket_arn,
      "${local.data_bucket_arn}/*"
    ]
  }
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = ["arn:aws:logs:${var.region}:${local.account_id}:log-group:/aws-glue/*"]
  }
}

resource "aws_iam_policy" "policy" {
  policy = data.aws_iam_policy_document.policy.json
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

resource "aws_iam_role_policy_attachment" "glue_service_role" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSGlueServiceRole"
  role       = aws_iam_role.crawler_role.id
}

resource "aws_iam_role_policy_attachment" "glue_policy" {
  policy_arn = aws_iam_policy.policy.arn
  role       = aws_iam_role.crawler_role.id
}

locals {
  name_to_prefix = {
    "csv" : "csv"
    "parquet" : "parquet"
    "etl-out-csv" : "etl_out/csv"
    "etl-out-parquet" : "etl_out/parquet"
  }
}

resource "aws_glue_crawler" "glue_crawler" {
  for_each      = local.name_to_prefix
  database_name = aws_glue_catalog_database.catalog_database.name
  name          = "pokemon-${each.key}-crawler"
  role          = aws_iam_role.crawler_role.arn
  classifiers   = [] // use builtin csv and parquet

  s3_target {
    path = "s3://${local.data_bucket_name}/pokemons/${each.value}/"
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