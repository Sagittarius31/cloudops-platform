variable "location" {
  type    = string
  default = "centralindia"
}

variable "project" {
  type    = string
  default = "cloudops"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "acr_name" {
  description = "Globally unique Azure Container Registry name: 5-50 lowercase letters/digits only."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9]{5,50}$", var.acr_name))
    error_message = "acr_name must be 5-50 lowercase letters and digits."
  }
}

variable "node_vm_size" {
  type    = string
  default = "Standard_B2s"
}

variable "node_min_count" {
  type    = number
  default = 1
}

variable "node_max_count" {
  type    = number
  default = 3
}

variable "api_authorized_ip_ranges" {
  description = "CIDRs allowed to reach the AKS API server. Restrict to your own IP (x.x.x.x/32)."
  type        = list(string)
  default     = []
}

variable "kv_name" {
  description = "Globally unique Key Vault name: 3-24 chars, letters/digits/hyphens."
  type        = string

  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z0-9-]{1,22}[a-zA-Z0-9]$", var.kv_name))
    error_message = "kv_name must be 3-24 characters: letters, digits, hyphens; start with a letter."
  }
}

variable "enable_postgres" {
  description = "Create Azure Database for PostgreSQL Flexible Server (private access). Costs money while running."
  type        = bool
  default     = false
}

variable "github_repo" {
  description = "GitHub repo allowed to push images via OIDC, as owner/name."
  type        = string
}
