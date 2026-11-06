# Security model

| Layer | Control | Where |
|---|---|---|
| Code | Dependency vuln scan (govulncheck), secret scan (Gitleaks) | `.github/workflows/ci.yml` |
| API | Static full-access and read-only tokens (SHA-256 + constant-time compare) **and** optional per-user OIDC JWTs (RS256 only, issuer/audience/expiry checks, JWKS cache) with **separate read and write roles**; 401 vs 403 are distinct; refuses to start with no auth; 1 MB body limit | `app/internal/api/auth.go`, `app/internal/auth/jwt.go` |
| Outbound | SSRF protection in two layers: URL validation at creation, and a dial-time guard on the resolved IP that blocks loopback/private/link-local/metadata addresses (covers redirects and DNS rebinding) | `app/internal/api/validate.go`, `app/internal/netguard` |
| Image | Distroless, non-root, Trivy scan fails build on fixable HIGH/CRITICAL, SBOM artifact | `app/Dockerfile`, CI |
| Supply chain | Keyless cosign signature by digest from CI; Kyverno verifies it (Audit -> Enforce) | CI `publish`, `gitops/policies/verify-signed-images.yaml` |
| Pod | runAsNonRoot, read-only root FS, drop ALL caps, seccomp RuntimeDefault, no SA token | `charts/sentinel/templates/deployment.yaml` |
| Edge | HTTPS via Traefik + cert-manager (Let's Encrypt); the Ingress routes only `/api` and `/healthz`, so `/metrics` and `/readyz` are never public | `charts/sentinel/templates/ingress.yaml` |
| Network | Default-deny NetworkPolicies; DB reachable only from app pods; egress to public internet only | `charts/sentinel/templates/networkpolicy.yaml` |
| Secrets | AWS: RDS-managed password + API token in Secrets Manager. Azure: DATABASE_URL + API token in Key Vault (RBAC). Synced by External Secrets; not in Git | `rds.tf`, `secrets.tf`, `keyvault.tf`, `externalsecret.yaml` |
| Cloud identity | No static cloud credentials anywhere: GitHub OIDC to AWS and to Azure, EKS Pod Identity, AKS Workload Identity (ESO, Kyverno), managed-identity ACR pull | `github_oidc.tf`, `pod_identity.tf`, `azure-dev/identities.tf` |
| Infra | Encrypted S3 state with locking, ECR immutable tags + scan on push, private subnets, restricted API CIDRs | Terraform |

## Verify a signed image yourself
```bash
cosign verify \
  --certificate-identity "https://github.com/YOUR_GITHUB_USERNAME/cloudops-platform/.github/workflows/ci.yml@refs/heads/main" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  <ecr-registry>/sentinel@sha256:<digest>
```

## Set / rotate the API token (production path)
```bash
aws secretsmanager put-secret-value --secret-id <api_token_secret_name> --secret-string "$(openssl rand -hex 24)"
# External Secrets refreshes hourly; force it: kubectl -n sentinel annotate externalsecret <release>-app force-sync=$(date +%s) --overwrite
# Then restart pods to pick it up: kubectl -n sentinel rollout restart deploy/<release>
```
To rotate without downtime, set two comma-separated tokens, move clients to the new one, then drop the old.

## Per-user auth (OIDC)
Set `auth.oidc.*` in the chart (issuer, audience, jwksUrl, optional requiredRole). Tokens must be RS256 JWTs whose
`iss`, `aud` and `exp` validate; The role names are matched against a `roles` array or a space-separated `scope`/`scp` claim.
Static tokens keep working for machine clients (CI, k6).

**Authorization:** `auth.oidc.readRole` / `writeRole` map to the read (GET) and write (POST/DELETE) endpoints; a writer can also read; empty role = any valid token. Static `API_TOKENS` have full access, `API_READ_TOKENS` are read-only (writes return 403). Example for Microsoft Entra ID:
`issuer=https://login.microsoftonline.com/<tenant>/v2.0`, `jwksUrl=https://login.microsoftonline.com/<tenant>/discovery/v2.0/keys`,
`audience=<API app registration client id>`. Register the API/app roles in your IdP yourself; this repo does not create them.

## Honest gaps
- Authorization is two-level (read/write), not per-resource or per-tenant; all authenticated users see all targets.
- RDS: set `externalSecrets.sslMode=verify-full` (CA bundle is baked into the image) to verify the server certificate; default is `require`. Azure Flexible Server uses `require`.
- Kyverno ECR credential helper and keyless verification must be validated in your cluster before switching to Enforce.
- Trivy IaC scan is report-only until findings are triaged.
