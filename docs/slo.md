# Service Level Objectives

| SLI | Definition | SLO (30 days) |
|---|---|---|
| Availability | Share of API requests that do NOT return 5xx | 99.5% |
| Latency | p95 of API request duration | < 500 ms |
| Check freshness | Each target is checked at least once per 2x the interval | 99% |

**Error budget:** 0.5% of requests ≈ 3h 36m of full outage per 30 days.

## Alerting strategy (multi-window burn rate)
| Alert | Burn rate | Windows | Severity |
|---|---|---|---|
| SentinelFastBurn | 14.4x | 5m AND 1h | critical (page) |
| SentinelSlowBurn | 6x | 30m AND 6h | warning (ticket) |

Why burn-rate alerts: they page on *how fast the budget is being spent*, not on a single noisy spike,
which reduces false pages and still catches slow leaks. Rules live in `deploy/prometheus/rules.yml`
(local) and `charts/sentinel/templates/prometheusrule.yaml` (Kubernetes).
