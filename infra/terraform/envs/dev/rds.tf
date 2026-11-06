# Optional managed PostgreSQL. The master password is generated and stored by RDS in
# AWS Secrets Manager (manage_master_user_password), so it never appears in Terraform state or Git.

resource "aws_db_subnet_group" "main" {
  count      = var.enable_rds ? 1 : 0
  name       = local.name
  subnet_ids = module.vpc.private_subnets
}

resource "aws_security_group" "rds" {
  count       = var.enable_rds ? 1 : 0
  name        = "${local.name}-rds"
  description = "PostgreSQL access from EKS nodes only"
  vpc_id      = module.vpc.vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "rds_from_nodes" {
  count                        = var.enable_rds ? 1 : 0
  security_group_id            = aws_security_group.rds[0].id
  referenced_security_group_id = module.eks.node_security_group_id
  from_port                    = 5432
  to_port                      = 5432
  ip_protocol                  = "tcp"
  description                  = "PostgreSQL from EKS nodes"
}

resource "aws_db_instance" "main" {
  count      = var.enable_rds ? 1 : 0
  identifier = local.name

  engine         = "postgres"
  engine_version = "16"
  instance_class = var.db_instance_class

  allocated_storage     = 20
  max_allocated_storage = 50
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = "sentinel"
  username = "sentinel"

  manage_master_user_password = true

  db_subnet_group_name   = aws_db_subnet_group.main[0].name
  vpc_security_group_ids = [aws_security_group.rds[0].id]
  publicly_accessible    = false
  multi_az               = var.db_multi_az

  backup_retention_period = 7
  copy_tags_to_snapshot   = true

  # Dev convenience so `terraform destroy` works. Flip both for production.
  deletion_protection = false
  skip_final_snapshot = true

  lifecycle {
    ignore_changes = [engine_version]
  }
}
