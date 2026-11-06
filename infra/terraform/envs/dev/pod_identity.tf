# EKS Pod Identity: gives specific Kubernetes service accounts AWS permissions without static keys.

data "aws_iam_policy_document" "pod_identity_trust" {
  statement {
    actions = ["sts:AssumeRole", "sts:TagSession"]

    principals {
      type        = "Service"
      identifiers = ["pods.eks.amazonaws.com"]
    }
  }
}

# --- External Secrets Operator: read only the secrets this project needs ---
resource "aws_iam_role" "external_secrets" {
  name               = "${local.name}-external-secrets"
  assume_role_policy = data.aws_iam_policy_document.pod_identity_trust.json
}

data "aws_iam_policy_document" "external_secrets" {
  statement {
    actions = ["secretsmanager:GetSecretValue", "secretsmanager:DescribeSecret"]
    resources = concat(
      [aws_secretsmanager_secret.api_token.arn, aws_secretsmanager_secret.slack_webhook.arn],
      var.enable_rds ? [aws_db_instance.main[0].master_user_secret[0].secret_arn] : []
    )
  }
}

resource "aws_iam_role_policy" "external_secrets" {
  name   = "read-app-secrets"
  role   = aws_iam_role.external_secrets.id
  policy = data.aws_iam_policy_document.external_secrets.json
}

resource "aws_eks_pod_identity_association" "external_secrets" {
  cluster_name    = module.eks.cluster_name
  namespace       = "external-secrets"
  service_account = "external-secrets"
  role_arn        = aws_iam_role.external_secrets.arn
}

# --- Kyverno admission controller: read-only ECR access to verify image signatures ---
resource "aws_iam_role" "kyverno" {
  name               = "${local.name}-kyverno"
  assume_role_policy = data.aws_iam_policy_document.pod_identity_trust.json
}

resource "aws_iam_role_policy_attachment" "kyverno_ecr_read" {
  role       = aws_iam_role.kyverno.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_eks_pod_identity_association" "kyverno" {
  cluster_name    = module.eks.cluster_name
  namespace       = "kyverno"
  service_account = "kyverno-admission-controller"
  role_arn        = aws_iam_role.kyverno.arn
}
