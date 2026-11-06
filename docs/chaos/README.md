# Chaos experiments

Run each experiment, then **fill in the "Observed" and "Fix / lesson" fields with what really happened.**
These are intentionally blank: results must come from your own runs, because "I broke X and measured Y"
is exactly what interviewers ask about.

Template per experiment: Hypothesis -> Steady state -> Action -> Observed -> Fix/lesson.

---
## 1. Kill one API pod
- **Hypothesis:** with 2+ replicas and a PDB, no failed requests; replacement pod ready < 30s.
- **Steady state:** `sentinel_http_requests_total` 5xx ratio ~ 0; Grafana "Sentinel Overview".
- **Action:** run `k6 run loadtest/api.js` and in another terminal `kubectl -n sentinel delete pod <one-sentinel-pod>`.
- **Observed:** _(fill in)_
- **Fix / lesson:** _(fill in)_

## 2. Drain a node
- **Hypothesis:** PDB keeps >= 1 pod available; pods reschedule onto the other node.
- **Action:** `kubectl drain <node> --ignore-daemonsets --delete-emptydir-data`, later `kubectl uncordon <node>`.
- **Observed:** _(fill in)_
- **Fix / lesson:** _(fill in)_

## 3. Database outage
- **Hypothesis:** `/readyz` fails, pods leave the Service, liveness stays OK (no restart storm); recovery is automatic.
- **Action:** `kubectl -n sentinel scale statefulset <release>-postgres --replicas=0`, wait 2 min, scale back to 1.
- **Observed:** _(fill in — did SentinelFastBurn fire? how long until it fired?)_
- **Fix / lesson:** _(fill in)_

## 4. Bad deploy + rollback
- **Hypothesis:** a broken image fails readiness, the rolling update stalls with zero downtime (`maxUnavailable: 0`).
- **Action:** push a commit setting `image.tag` to a non-existent tag; watch ArgoCD; then revert the commit.
- **Observed:** _(fill in)_
- **Fix / lesson:** _(fill in)_

## 5. Restore drill
- **Action:** follow `docs/runbooks/database-restore.md` on a copy; measure restore time.
- **Observed RTO / RPO:** _(fill in)_
