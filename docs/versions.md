# Version pins and why (verified 5 Nov 2026 via public release pages)

Versions rot quickly. Re-check this table before every deployment and after any long break.

| Component | Pinned | Why / source of truth |
|---|---|---|
| EKS Kubernetes | `1.35` (`kubernetes_version`) | In standard support until Mar 2027. **1.30 and older are past extended support**; versions in extended support cost ~6x for the control plane. Check the EKS version calendar. |
| EKS node AMI | `AL2023_x86_64_STANDARD` | AL2 AMIs are not available for current Kubernetes versions. |
| AKS Kubernetes | provider default (unset) | AKS picks a supported default; set `kubernetes_version` only deliberately. |
| Go | `1.26` (go.mod, Dockerfile) | Go supports only the two newest majors; 1.24 and older are EOL. When 1.27 ships, move to 1.27 or stay on 1.26 only while it is supported. |
| golang-jwt | `v5.2.2` | Fixes a header-parsing memory-allocation issue present in earlier 5.x. |
| ingress controller | Traefik chart `41.5.0` | **ingress-nginx was retired in March 2026** (no more security fixes). |
| cert-manager | chart `v1.20.2` | `crds.enabled=true` value. |
| External Secrets | chart `0.17.0`, API `external-secrets.io/v1` | 0.17+ stopped serving `v1beta1`; manifests use `v1`. Newer 1.x releases should work with the same manifests. |
| Kyverno | chart `3.6.3` (app 1.16.x) | The old 3.2.x line does not support current Kubernetes. `ClusterPolicy` is being superseded by `ValidatingPolicy`/`ImageValidatingPolicy`; migrate when you upgrade. |
| kube-prometheus-stack | `87.19.2` | Recent July 2026 release. |
| PostgreSQL | 16 | Supported on RDS and Azure Flexible Server. |

## How to refresh safely
1. Look up the newest stable release of each item (release pages, `helm search repo <chart> --versions`).
2. Change one component at a time, render with `make helm-modes`, deploy to dev, watch ArgoCD + Grafana.
3. Let Dependabot handle Go, Docker, Terraform and GitHub Actions; Helm chart versions in ArgoCD manifests are manual.
