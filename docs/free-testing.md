# Testing everything for free, without a big laptop

Do these in order; stop at the first one that fails and fix it before continuing.

## Level 0 -- zero install, zero disk (GitHub Actions)
1. Create a GitHub repo (public repos get free standard runners; private repos have a monthly free allowance -- check your plan) and push this project.
2. Actions tab -> **bootstrap-go-sum** -> Run workflow. It generates and commits `app/go.sum`.
3. Push any small change. The **ci** workflow then runs: `go vet`, `go test -race`, `govulncheck`, Helm lint/render (all modes),
   `terraform fmt` + `validate` (AWS and Azure), Gitleaks, Trivy and the image build.
4. Read the failures in the Actions logs, fix, push again. This alone finds most compile/config errors.

## Level 1 -- full stack in the browser (GitHub Codespaces)
Personal accounts get 120 core-hours and 15 GB per month free (about 60 hours on a 2-core machine; confirm on your billing page).
1. Repo -> Code -> Codespaces -> Create. The `.devcontainer` installs Go, Docker, Terraform, kubectl and Helm.
2. `make test`, then `make up` (API + Postgres + Prometheus + Grafana). Open the forwarded ports 8080 / 3000 / 9090.
3. **Stop the codespace when you finish** (stopped codespaces use no compute hours; their disk still counts toward storage, so delete old ones).

## Level 2 -- real Kubernetes, still free (kind in Codespaces)
Choose a 4-core machine for this (costs 4 core-hours per hour, so about 30 hours/month free).
```bash
make kind-up        # cluster + image
make kind-deploy    # your Helm chart with in-cluster Postgres
make kind-forward   # then in a second terminal:
curl -H "Authorization: Bearer change-me-dev-only-token-0123" localhost:8080/api/v1/targets
make kind-down
```
This tests the chart, probes, secrets and rollouts. It does not test EKS/AKS, ArgoCD sync, External Secrets, Kyverno or ingress.

## Level 3 -- real cloud (this is where money starts)
- EKS and AKS are not free: the EKS control plane and the NAT gateway bill by the hour even when idle. Free-credit offers change; check what your account currently gets.
- Before applying anything: set the Terraform `budget_email`, restrict `api_allowed_cidrs` to your IP, and plan to `terraform destroy` the same day.
- Do Level 0-2 first so the cloud session is short and you are only fixing cloud-specific problems.

## If you only have a tiny laptop
Skip Docker entirely. Level 0 needs nothing installed, and Level 1 runs everything in the browser.
