terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.60"
    }
  }

  # Partial backend config. Initialise with:
  #   terraform init -backend-config=backend.hcl
  backend "s3" {}
}
