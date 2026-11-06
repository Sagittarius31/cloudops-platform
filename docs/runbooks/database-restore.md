# Runbook: restore the dev Postgres from backup

Backup (manual, for learning): 
`kubectl -n sentinel exec <release>-postgres-0 -- pg_dump -U sentinel sentinel > backup.sql`

Restore into a fresh database:
`kubectl -n sentinel exec -i <release>-postgres-0 -- psql -U sentinel sentinel < backup.sql`

Targets: RPO = time since last dump; RTO = time you measured in the DR drill (record it in `docs/chaos/`).
Production path: RDS automated backups + point-in-time recovery, and Velero for cluster resources.
