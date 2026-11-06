# Runbook: SentinelFastBurn / SentinelSlowBurn (elevated 5xx)

1. Grafana "Sentinel Overview" -> which `handler` returns 5xx? Did it start at a deploy time?
2. Logs: `kubectl -n sentinel logs deploy/<release> --since=15m | grep '"status":5'`.
3. Dependency check: `/readyz` failing means database trouble -> see database steps in `api-down.md`.
4. Recent deploy? Roll back via Git revert (do not `kubectl edit`; ArgoCD self-heal will undo manual changes).
5. Load related? Check HPA: `kubectl -n sentinel get hpa`; consider raising `maxReplicas`.
