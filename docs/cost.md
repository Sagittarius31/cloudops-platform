# Cost control (approximate; check current regional pricing)

What bills while the environment exists, even if idle:
- **EKS control plane** (fixed hourly charge) and **NAT gateway** (hourly + data) -- the two big idle costs.
- EC2 worker nodes (spot in dev), EBS volumes, ECR storage, optional RDS instance.
- AKS free tier has no control-plane fee, but nodes, disks and the load balancer bill.

Rules used in this repo:
1. Spot nodes + a single NAT gateway in dev; one-per-AZ and on-demand in prod.
2. `terraform destroy` at the end of every session (`enable_rds = false` unless you are testing Mode B).
3. AWS Budget alerts at 80% actual and 100% forecast (`aws_budgets_budget`).
4. ECR lifecycle keeps only the last 20 images; Prometheus retention 3 days.

Measure your real numbers with Cost Explorer after a week and put them in your resume bullet.
