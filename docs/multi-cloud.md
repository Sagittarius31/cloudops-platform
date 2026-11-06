# Multi-cloud: one chart, two clouds

| Concern | AWS (EKS) | Azure (AKS) |
|---|---|---|
| IaC | `infra/terraform/envs/dev` | `infra/terraform/envs/azure-dev` |
| Registry | ECR (immutable tags, scan on push) | ACR (managed-identity pull via `AcrPull`) |
| Workload identity | EKS Pod Identity | AKS Workload Identity (OIDC issuer enabled) |
| NetworkPolicy engine | VPC CNI network policy | Calico |
| Secrets (prod path) | Secrets Manager + External Secrets (`source: rds`) | Key Vault + External Secrets (`source: direct`, `values-aks-kv.yaml`) |
| Database | In-cluster (Mode A) or RDS (Mode B) | In-cluster (Mode A) or Azure Flexible Server, private access (`enable_postgres`) |
| GitOps entry | `gitops/argocd/root-app.yaml` | `gitops/argocd/aks/root-app.yaml` |
| Chart values | `values-dev.yaml` (+ `values-rds.yaml`) | `values-aks.yaml` |

Same Helm chart, same app, same dashboards and alerts on both clusters; only the infrastructure and a values file differ.

## AKS bring-up
```bash
# one-time remote state (names must be unique)
az group create -n cloudops-tfstate-rg -l centralindia
az storage account create -n <uniquename> -g cloudops-tfstate-rg -l centralindia --sku Standard_LRS
az storage container create -n tfstate --account-name <uniquename>

cd infra/terraform/envs/azure-dev
cp backend.hcl.example backend.hcl && cp terraform.tfvars.example terraform.tfvars   # edit both
terraform init -backend-config=backend.hcl && terraform apply
$(terraform output -raw configure_kubectl)

# CI push to ACR: set these GitHub repo VARIABLES from `terraform output`
#   AZURE_CLIENT_ID (=github_client_id), AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID, ACR_NAME, ACR_LOGIN_SERVER
# (manual alternative: az acr login -n <acr_name>; docker build/push <acr_login_server>/sentinel:<tag>)

# GitOps
make argocd-install
kubectl apply -f gitops/argocd/aks/root-app.yaml
```
Edit `values-aks.yaml` (ACR server + tag) and `YOUR_GITHUB_USERNAME` first.

## Production-style AKS (Key Vault + managed PostgreSQL)
```bash
terraform apply -var enable_postgres=true            # run twice if the first apply races Key Vault RBAC propagation
az keyvault secret set --vault-name <key_vault_name> --name sentinel-api-token --value "$(openssl rand -hex 24)"
```
Then fill the `REPLACE_*` values from `terraform output` in `gitops/argocd/aks-apps/{external-secrets,kyverno}.yaml` and
`gitops/platform-aks/clustersecretstore.yaml`, and add `values-aks-kv.yaml` to the valueFiles of `gitops/argocd/aks-apps/sentinel.yaml`.
The placeholder token (`CHANGE-ME`) is intentionally invalid: pods crash-loop until you set a real value.

## HTTPS ingress and alert routing (both clouds)
- **Ingress:** Traefik + cert-manager are installed by ArgoCD. Replace the email in `gitops/platform*/clusterissuers.yaml`,
  then in your values set `ingress.enabled=true`, `ingress.host=<your-domain>`. Get the load-balancer address with
  `kubectl -n traefik get svc traefik` and point a DNS record at it. Start with `letsencrypt-staging`, then switch
  `ingress.clusterIssuer` to `letsencrypt-prod`. (This needs a domain you own; the repo cannot provide one.)
- **Alerts:** Alertmanager sends `critical` and `warning` alerts to Slack via a webhook kept in the cloud secret store
  (`slack_webhook_secret_name` on AWS, `alertmanager-slack-webhook` in Key Vault), synced by External Secrets.
  Terraform creates it with a placeholder; deliveries start once you set the real URL.
