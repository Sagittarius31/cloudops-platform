terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.72"
    }
  }

  # Partial backend config. Initialise with:
  #   terraform init -backend-config=backend.hcl
  backend "s3" {}
}
