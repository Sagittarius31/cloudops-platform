variable "region" {
  type    = string
  default = "ap-south-1"
}

variable "project" {
  type    = string
  default = "cloudops"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}

variable "kubernetes_version" {
  description = "EKS minor version. Check the EKS version calendar: versions past standard support cost ~6x for the control plane and old ones are force-upgraded. 1.35 is in standard support until Mar 2027."
  type        = string
  default     = "1.35"
}

variable "node_instance_types" {
  type    = list(string)
  default = ["t3.medium"]
}

variable "node_capacity_type" {
  description = "ON_DEMAND or SPOT. SPOT is much cheaper for a learning environment."
  type        = string
  default     = "SPOT"
}

variable "node_min_size" {
  type    = number
  default = 2
}

variable "node_max_size" {
  type    = number
  default = 4
}

variable "node_desired_size" {
  type    = number
  default = 2
}

variable "api_allowed_cidrs" {
  description = "CIDRs allowed to reach the public EKS API endpoint. Restrict to your own IP (x.x.x.x/32)."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "github_repo" {
  description = "GitHub repo allowed to push images via OIDC, as owner/name."
  type        = string
}

variable "create_github_oidc_provider" {
  description = "Set false if the GitHub OIDC provider already exists in this AWS account."
  type        = bool
  default     = true
}

variable "budget_limit_usd" {
  type    = number
  default = 50
}

variable "budget_email" {
  description = "Email that receives cost alerts."
  type        = string
}

variable "enable_rds" {
  description = "Create an RDS PostgreSQL instance (costs money while running). Needed for the External Secrets/RDS mode."
  type        = bool
  default     = false
}

variable "db_instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "db_multi_az" {
  type    = bool
  default = false
}

variable "single_nat_gateway" {
  description = "true = one shared NAT gateway (cheap, dev). false = one NAT per AZ (production availability)."
  type        = bool
  default     = true
}
