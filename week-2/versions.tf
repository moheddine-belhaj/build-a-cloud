terraform {
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
      version = "~> 0.104"
    }
  }
}

provider "stackit" {
  region = "eu01"
}