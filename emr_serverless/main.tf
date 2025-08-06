terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      version = ">= 6.7.0"
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

data "terraform_remote_state" "poke_scraper" {
  backend = "local"

  config = {
    path = "${path.module}/../poke-scraper/infra/terraform.tfstate"
  }
}