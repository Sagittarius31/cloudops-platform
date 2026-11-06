# CloudOps Platform — Sentinel

A Go infrastructure-monitoring service plus the production platform around it, on **AWS EKS and Azure AKS**:
Terraform, GitHub Actions CI with security gates and keyless image signing, ArgoCD GitOps,
External Secrets (AWS Secrets Manager + RDS, Azure Key Vault + Flexible Server), Kyverno policies, OIDC per-user auth, Prometheus/Grafana with SLO burn-rate alerts,
default-deny NetworkPolicies, chaos experiments, runbooks and interview prep.

Docs: `docs/versions.md` (pinned versions + why) · `docs/architecture.md` · `docs/security.md` · `docs/slo.md` · `docs/multi-cloud.md` · `docs/cost.md` · `docs/roadmap.md` · `docs/interview-prep.md`

## Status (read this first)
Everything is **written**; nothing has been **compiled or deployed** by the author (no Go/Terraform/Docker/cloud access when it was generated).
Expect small fixes on first run; debugging them is part of the learning.

| Area | Checked so far | You must verify |
|---|---|---|
| Go app + tests (auth read/write, OIDC, migrations, SSRF, checker) | Reviewed by hand | `make tidy && make test` |
| Docker Compose stack | YAML syntax | `make up`, open Grafana |
| Prometheus rules / dashboard | YAML/JSON syntax | Load in Prometheus/Grafana |
| Helm chart (3 secret modes) | YAML syntax of values | `make helm-lint helm-modes` |
| Terraform AWS + Azure | Reviewed by hand | `make fmt`, `make tf-validate`, then `plan` |
| CI (AWS + Azure publish), GitOps, Kyverno, External Secrets (AWS + Azure) | YAML syntax | Run in real clusters; keep policies in Audit until clean |

**Placeholders:** `make set-github GITHUB_USER=yourname` fills your username everywhere; the rest are listed in `docs/placeholders.md`.

**Short on laptop space or money?** See `docs/free-testing.md` (GitHub Actions + Codespaces + kind, all free-tier).

## Quickstart (local, no cloud)
```bash
make tidy          # creates app/go.sum -> commit it
make test
make up            # API :8080, Prometheus :9090, Grafana :3000 (admin/admin)

TOKEN=local-dev-token-0123456789   # full access; API_READ_TOKENS gives read-only tokens
curl -X POST localhost:8080/api/v1/targets -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"self","url":"http://api:8080/healthz"}'
curl -H "Authorization: Bearer $TOKEN" localhost:8080/api/v1/targets
curl -H "Authorization: Bearer $TOKEN" "localhost:8080/api/v1/targets/1/summary?window=1h"
```
`/healthz`, `/readyz`, `/metrics` need no token; `/api/v1/*` does. The service refuses to start without `API_TOKENS`.

## Deploy to AWS (after local works)
1. `make fmt` (so CI's `terraform fmt -check` passes), push to GitHub, get CI green.
2. `cd infra/terraform/bootstrap && terraform init && terraform apply` (remote state).
3. `infra/terraform/envs/dev`: copy `backend.hcl.example` -> `backend.hcl`, `terraform.tfvars.example` -> `terraform.tfvars`, edit, then
   `terraform init -backend-config=backend.hcl && terraform plan && terraform apply`.
4. Run `terraform output configure_kubectl`. (EKS 1.35 is set by default; check `docs/versions.md` for newer supported versions.) In GitHub repo **variables** set `AWS_ROLE_ARN` (from output) and `AWS_REGION`.
5. Set the API token value (stays out of Git/state):
   `aws secretsmanager put-secret-value --secret-id <api_token_secret_name> --secret-string "$(openssl rand -hex 24)"`
6. Merge to `main`: CI pushes and **signs** the image. Put the ECR URL + SHA tag in `charts/sentinel/values-dev.yaml`.
7. Replace `YOUR_GITHUB_USERNAME` in `gitops/**` and `charts` files, then `make argocd-install` and
   `kubectl apply -f gitops/argocd/root-app.yaml`. Sync waves install External Secrets/Kyverno first, then monitoring, then the app.
8. Production-style database: `terraform apply -var enable_rds=true`, fill `charts/sentinel/values-rds.yaml` from outputs, add it to the `valueFiles` in `gitops/argocd/apps/sentinel.yaml`.
9. **Always `terraform destroy` after each session** (EKS control plane + NAT gateway cost money while idle).

Azure (AKS, ACR, Key Vault, optional Flexible Server, CI push via OIDC): see `docs/multi-cloud.md`.

## Auth options
- **Static tokens:** `API_TOKENS` (read + write) and `API_READ_TOKENS` (read-only; writes return 403) for machines/CI.
- **Per-user JWTs** from your OIDC provider: `OIDC_ISSUER`, `OIDC_AUDIENCE`, `OIDC_JWKS_URL`, plus optional `OIDC_READ_ROLE` / `OIDC_WRITE_ROLE`.
- All can be enabled together; with none configured the service refuses to start.

## HTTPS and alerts
Traefik + cert-manager (Let's Encrypt) and Slack alert routing are included; both need things only you have
(a domain, a Slack webhook). See "HTTPS ingress and alert routing" in `docs/multi-cloud.md`.

## Security notes
- Never commit `*.tfvars`, `backend.hcl` or real passwords. The dev Postgres password and dev token in `values.yaml` are placeholders for Mode A only.
- Keep `ALLOW_PRIVATE_TARGETS=false` outside local dev (SSRF protection).
- Kyverno policies start in `Audit`; confirm signed images pass before switching to `Enforce`.
