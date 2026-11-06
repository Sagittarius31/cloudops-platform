# Placeholders to fill in (and where each value comes from)

Fill your GitHub username with one command: `make set-github GITHUB_USER=yourname`.
Everything else comes from `terraform output` after you apply, or from your own accounts.

| Placeholder | File(s) | Value comes from |
|---|---|---|
| `YOUR_GITHUB_USERNAME` | `gitops/**`, `infra/**/terraform.tfvars.example`, `charts` | Your GitHub username (`make set-github`) |
| `REPLACE_WITH_ECR_URL` | `charts/sentinel/values-dev.yaml` | `terraform output ecr_repository_url` (AWS) |
| `REPLACE_WITH_GIT_SHA` | `charts/sentinel/values-dev.yaml` | The image tag CI pushed (the commit SHA) |
| `REPLACE_WITH_rds_address`, `..._rds_master_secret_arn`, `..._api_token_secret_name` | `charts/sentinel/values-rds.yaml` | `terraform output` (AWS, only with `enable_rds=true`) |
| `REPLACE_WITH_slack_webhook_secret_name` | `gitops/monitoring-config/alertmanager-secret.yaml` | `terraform output slack_webhook_secret_name` |
| `REPLACE_WITH_ACR_LOGIN_SERVER`, `REPLACE_WITH_TAG` | `charts/sentinel/values-aks.yaml` | `terraform output acr_login_server` (Azure) and your image tag |
| `REPLACE_WITH_ESO_CLIENT_ID`, `REPLACE_WITH_KYVERNO_CLIENT_ID`, `REPLACE_WITH_TENANT_ID` | `gitops/argocd/aks-apps/*.yaml`, `gitops/platform-aks/` | `terraform output` (Azure) |
| `REPLACE_WITH_KEY_VAULT_NAME` | `gitops/platform-aks/clustersecretstore.yaml` | `terraform output key_vault_name` |
| `REPLACE_WITH_YOUR_EMAIL` | `gitops/platform*/clusterissuers.yaml` | Your email (Let's Encrypt notices) |
| `REPLACE_UNIQUE_STORAGE_ACCOUNT_NAME` | `infra/terraform/envs/azure-dev/backend.hcl.example` | A globally unique name you choose (Azure only) |
| `YOURNAME` | `infra/terraform/envs/azure-dev/terraform.tfvars.example` | A unique suffix you choose (Azure only) |

Azure-only rows can be ignored if you only use AWS, and vice versa. Find anything left with:
`grep -rn "REPLACE_\|YOUR_GITHUB" gitops charts infra`
