output "cluster_name" {
  value = module.eks.cluster_name
}

output "ecr_repository_url" {
  value = aws_ecr_repository.sentinel.repository_url
}

output "github_ci_role_arn" {
  description = "Set this as the AWS_ROLE_ARN repository variable in GitHub."
  value       = aws_iam_role.github_ci.arn
}

output "configure_kubectl" {
  value = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}

output "api_token_secret_name" {
  description = "Set its value with `aws secretsmanager put-secret-value` (see secrets.tf)."
  value       = aws_secretsmanager_secret.api_token.name
}

output "rds_address" {
  value = var.enable_rds ? aws_db_instance.main[0].address : null
}

output "rds_master_secret_arn" {
  value = var.enable_rds ? aws_db_instance.main[0].master_user_secret[0].secret_arn : null
}

output "slack_webhook_secret_name" {
  description = "Put in gitops/monitoring-config/alertmanager-secret.yaml, then set the real URL (see secrets.tf)."
  value       = aws_secretsmanager_secret.slack_webhook.name
}
