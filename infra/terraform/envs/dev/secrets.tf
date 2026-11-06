# Secret *container* for the API bearer token. The VALUE is set out-of-band so it never lands in
# Terraform state or Git:
#   aws secretsmanager put-secret-value --secret-id <api_token_secret_name> \
#     --secret-string "$(openssl rand -hex 24)"
resource "aws_secretsmanager_secret" "api_token" {
  name                    = "${local.name}/sentinel/api-token"
  description             = "Bearer token(s) accepted by the Sentinel API (comma-separated)"
  recovery_window_in_days = 0 # dev: allows immediate re-create after destroy
}

# Slack incoming-webhook URL for Alertmanager. Created with a harmless placeholder so the Alertmanager pod
# can start; alerts are not delivered until you set the real URL:
#   aws secretsmanager put-secret-value --secret-id <slack_webhook_secret_name> --secret-string "https://hooks.slack.com/services/..."
resource "aws_secretsmanager_secret" "slack_webhook" {
  name                    = "${local.name}/alertmanager/slack-webhook"
  description             = "Slack webhook URL used by Alertmanager"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "slack_webhook_placeholder" {
  secret_id     = aws_secretsmanager_secret.slack_webhook.id
  secret_string = "https://example.invalid/replace-me"

  lifecycle {
    ignore_changes = [secret_string]
  }
}
