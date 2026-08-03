terraform {
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
      version = "~> 0.104"
    }
  }
  backend "s3" {
    bucket   = "week2-tfstate"
    key      = "week-3/infra/terraform.tfstate"
    region   = "eu01"
    endpoints = {
      s3 = "https://object.storage.eu01.onstackit.cloud"
    }
    use_path_style              = true
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_s3_checksum            = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
  }
}


provider "stackit" {
  default_region           = "eu01"
  service_account_key_path = var.service_account_key_path
  private_key_path         = var.private_key_path
}