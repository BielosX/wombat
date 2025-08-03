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

provider "aws" {
  region = var.region
}

data "terraform_remote_state" "glue" {
  backend = "local"

  config = {
    path = "${path.module}/../glue/terraform.tfstate"
  }
}

data "terraform_remote_state" "poke_scraper" {
  backend = "local"

  config = {
    path = "${path.module}/../poke-scraper/infra/terraform.tfstate"
  }
}

locals {
  catalog_db_name = data.terraform_remote_state.glue.outputs.catalog_database
  bucket_name     = data.terraform_remote_state.poke_scraper.outputs.data_bucket_name
  bucket_arn      = data.terraform_remote_state.poke_scraper.outputs.data_bucket_arn
}