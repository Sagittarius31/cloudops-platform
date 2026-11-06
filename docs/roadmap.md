# Build roadmap (8-12 weeks)

Rule: do not start a phase until the previous one is **deployed and working**. Record a screenshot
or short demo for each phase; they become your README and interview evidence.

| Phase | Goal | Done when |
|---|---|---|
| 1. Run locally | `make tidy`, `make test`, `make up`; add a target via curl | Targets show in Grafana, tests pass |
| 2. Understand the code | Read every file in `app/` and explain it aloud | You can explain each package without notes |
| 3. Push to GitHub + CI | CI green on a PR | All CI jobs pass (fix `terraform fmt`/lint issues) |
| 4. AWS foundation | Bootstrap state, apply `envs/dev` | `terraform output` shows cluster + ECR; `kubectl get nodes` works |
| 5. Image to ECR | Set `AWS_ROLE_ARN`/`AWS_REGION` repo variables | Merge to main pushes an image tagged with the SHA |
| 6. GitOps | Install ArgoCD, apply `root-app.yaml` | ArgoCD shows `sentinel` and `monitoring` Healthy/Synced |
| 7. Observability | Dashboards + alerts in the cluster | Stopping a target triggers `MonitoredTargetDown` |
| 8. Reliability | Run chaos experiments 1-4, fill `docs/chaos/` | Four written experiments with real observations |
| 9. Security hardening | NetworkPolicy on, Trivy exit-code 1, cosign signing, External Secrets + RDS | No HIGH/CRITICAL findings; DB password not in Git |
| 10. Multi-cloud (optional) | Apply `azure-dev`, push image via CI, deploy with `gitops/argocd/aks/root-app.yaml` | Same chart running on AKS; optional Key Vault + Flexible Server mode |
| 11. Per-user auth (optional) | Register an app in your IdP with read/write roles, set `auth.oidc.*` | A reader token can GET but gets 403 on POST; expired/wrong-audience tokens get 401 |
| 12. Public HTTPS (optional, needs a domain) | Enable `ingress`, set DNS, use staging then prod issuer | `curl https://<host>/healthz` works with a valid certificate |
| 13. Alerts to Slack | Set the real webhook secret, stop a target | A Slack message arrives for `MonitoredTargetDown` |

## Resume bullets (fill the numbers with YOUR measured values)
- Built and operated a Go monitoring service on AWS EKS provisioned with Terraform (VPC, EKS, ECR, IAM, Budgets) and delivered via GitHub Actions + ArgoCD GitOps.
- Defined SLOs with multi-window burn-rate alerting (Prometheus/Alertmanager/Grafana); validated with N chaos experiments (pod/node/database failure), recovering in <X>s.
- Hardened the pipeline: Trivy/Gitleaks scans, OIDC to AWS (no static keys), non-root read-only containers, NetworkPolicies; reduced dev cost by <Y>% using spot nodes and budgets.
