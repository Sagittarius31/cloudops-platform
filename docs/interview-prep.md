# Interview prep: questions this project lets you answer

Answer each in your own words from what you actually built and measured.

1. **Walk me through the architecture.** App -> Postgres, Prometheus scrape, CI -> ECR -> ArgoCD -> EKS (see `docs/architecture.md`).
2. **Why GitOps instead of `kubectl apply` from CI?** Git is the source of truth, drift self-heals, rollback = git revert, no cluster credentials in CI.
3. **How do you avoid static cloud credentials?** GitHub OIDC for CI, EKS Pod Identity for workloads, managed identity for ACR pulls.
4. **Where do secrets live?** Secrets Manager (RDS-managed password), synced by External Secrets; none in Git or Terraform state.
5. **Explain your SLO and alerting.** 99.5% availability, multi-window burn-rate alerts (`docs/slo.md`); why not alert on raw error spikes?
6. **A pod is CrashLoopBackOff. What do you do?** `describe`, `logs --previous`, readiness vs liveness, DB reachability (`docs/runbooks/api-down.md`).
7. **What did chaos testing show?** Your own results from `docs/chaos/` (be specific: time to recover, which alert fired).
8. **Liveness vs readiness probe: why separate?** `/healthz` never touches the DB (avoid restart storms), `/readyz` does (remove from Service).
9. **How do rolling updates stay zero-downtime?** `maxUnavailable: 0`, readiness gating, PDB, multiple replicas, zone spread.
10. **How is the supply chain protected?** Trivy gate, SBOM, immutable ECR tags, cosign keyless signing, Kyverno verification.
11. **What is SSRF and how does your app mitigate it?** User-supplied URLs fetched server-side. I validate at creation, but the real control is a dial-time check on the resolved IP (so redirects and DNS rebinding can't bypass it), plus NetworkPolicy egress limits. Be ready to explain why validating only the URL string is not enough.
12. **Why a NetworkPolicy and what did you have to enable?** Default-deny; EKS needs VPC CNI network policy turned on, AKS uses Calico.
13. **Terraform state: how is it protected?** S3 versioning + encryption + public-access block, DynamoDB locking, no secrets in state.
14. **How would you scale this?** Split checker from API, partition targets, cardinality limits on metrics, RDS read replicas, per-AZ NAT.
15. **What would you do differently in production?** Per-AZ NAT, Multi-AZ RDS with PITR, OIDC/JWT auth, Enforce Kyverno, verify-full TLS to RDS, Karpenter.
16. **How does your API authenticate users, not just services?** OIDC JWT validation: RS256 only, JWKS cached and refreshed with a throttle, issuer/audience/expiry enforced, optional role; static tokens remain for machine clients. Be ready to explain why `alg=none` and HS256 confusion are blocked.
17. **How do two clouds share one chart?** Same Helm chart; per-cloud Terraform and values overlays; secrets via External Secrets with a store per cloud (`rds` vs `direct` source).
18. **Why federated identity instead of a service principal secret?** Short-lived tokens, nothing to rotate or leak; trust is scoped to a repo + branch or a Kubernetes service account.
19. **Why did you pick Traefik, and what happened to ingress-nginx?** It was retired in March 2026 (no security fixes), so I chose an actively maintained controller; the long-term direction is Gateway API.
20. **Why does your Ingress only route `/api` and `/healthz`?** `/metrics` is unauthenticated by design for Prometheus; exposing only needed paths keeps it internal.
21. **How do migrations stay safe with multiple replicas?** A Postgres advisory lock serialises runners; each migration runs in a transaction and is recorded in `schema_migrations`.
22. **How do you handle Kubernetes version support?** Track the EKS/AKS calendars, never run past standard support (extended support costs ~6x), upgrade one minor at a time, and re-pin add-on versions (`docs/versions.md`).
23. **What is the difference between `sslmode=require` and `verify-full`?** `require` encrypts but trusts any certificate; `verify-full` also checks the CA chain and hostname, defeating man-in-the-middle attacks.
24. **401 vs 403?** 401 = not authenticated (missing/invalid token); 403 = authenticated but not allowed (e.g. read-only token trying to write).
