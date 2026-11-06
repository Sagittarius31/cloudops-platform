# Runbook: SentinelDown (API unreachable)

**Impact:** API and checks stopped; no new results recorded.

1. `kubectl -n sentinel get pods -l app.kubernetes.io/name=sentinel`
2. Pods `CrashLoopBackOff`? `kubectl -n sentinel logs <pod> --previous` (look for `database not reachable`).
3. Pods `Pending`? `kubectl -n sentinel describe pod <pod>` (node capacity, PVC, image pull).
4. Database down? `kubectl -n sentinel get pods -l app.kubernetes.io/component=postgres`.
5. Bad release? Roll back: revert the image tag commit in Git; ArgoCD syncs the previous version.
6. After recovery, record timeline + root cause in a postmortem (`docs/postmortems/`).
