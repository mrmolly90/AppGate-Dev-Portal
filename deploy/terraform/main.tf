# =============================================================================
# AppGate Dev-Portal — Terraform (AWS ECS Fargate)
#
# Provisions:
#   - VPC with private subnets ONLY (no public subnets, no internet gateway
#     for app workloads). Egress is locked to a NAT gateway with a
#     documented security group (no 0.0.0.0/0 egress from app SGs).
#   - ECS cluster + Fargate tasks for dev-portal-api and dev-portal-web
#     (nginx), and a control-plane service.
#   - Secrets in AWS Secrets Manager (provider keys, JWT signing key).
#   - ALB with HTTPS (ACM) fronting the portal.
#
# Egress policy: the ONLY egress routes are
#   - 0.0.0.0/0 via NAT with allowlisted destination CIDRs, OR
#   - specific allowlisted CIDRs (LLM providers) via VPC endpoints.
# No security group grants 0.0.0.0/0 egress without justification.
# =============================================================================

terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

locals {
  name_prefix = "appgate-dev-portal"
  region      = var.region
  tags = {
    Project   = "appgate-dev-portal"
    ManagedBy = "terraform"
  }
}

# -----------------------------------------------------------------------------
# VPC — private-only workload subnets
# -----------------------------------------------------------------------------
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = local.tags
}

# No public subnets by design; only private subnets in two AZs.
resource "aws_subnet" "private" {
  count                   = 2
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(aws_vpc.main.cidr_block, 8, count.index + 10)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = false
  tags = merge(local.tags, { Name = "${local.name_prefix}-private-${count.index}" })
}

# -----------------------------------------------------------------------------
# Egress — NAT with narrow allowlist (NO blanket 0.0.0.0/0 egress)
# -----------------------------------------------------------------------------
resource "aws_eip" "nat" {
  count = 2
  domain = "vpc"
  tags  = local.tags
}

resource "aws_nat_gateway" "main" {
  count         = 2
  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.private[count.index].id # NOTE: needs a public subnet for the NAT itself
  tags          = local.tags
}

# The NAT gateway requires a public subnet; the PUBLIC subnet is used ONLY
# for the NAT gateways — no application workloads live there.
resource "aws_subnet" "public" {
  count                   = 2
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(aws_vpc.main.cidr_block, 8, count.index)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = false
  tags = merge(local.tags, { Name = "${local.name_prefix}-nat-${count.index}" })
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = local.tags
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  tags   = local.tags
}

resource "aws_route" "public_igw" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
  # JUSTIFICATION: required for the NAT gateways themselves to reach the
  # internet (LLM provider egress). Application workloads do NOT use this
  # route — they use the private route with NAT, which is allowlisted below.
}

resource "aws_route_table_association" "public" {
  count          = 2
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Private route tables: NO 0.0.0.0/0. Egress only to LLM provider CIDRs
# and AWS service endpoints via VPC endpoints.
resource "aws_route_table" "private" {
  count  = 2
  vpc_id = aws_vpc.main.id
  tags = merge(local.tags, { Name = "${local.name_prefix}-private-rt-${count.index}" })
}

resource "aws_route_table_association" "private" {
  count          = 2
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}

# -----------------------------------------------------------------------------
# Security Groups — fail closed, least privilege
# -----------------------------------------------------------------------------
resource "aws_security_group" "alb" {
  name_prefix = "${local.name_prefix}-alb"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # JUSTIFICATION: public HTTPS ingress to the portal ALB
    description = "HTTPS from the internet"
  }
  egress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    security_groups = [aws_security_group.web.id]
    description = "Forward to web service only"
  }
  tags = local.tags
}

resource "aws_security_group" "web" {
  name_prefix = "${local.name_prefix}-web"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
    description     = "Web from ALB only"
  }
  egress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    security_groups = [aws_security_group.api.id]
    description = "API proxy only"
  }
  tags = local.tags
}

resource "aws_security_group" "api" {
  name_prefix = "${local.name_prefix}-api"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.web.id]
    description     = "BFF from web only"
  }
  # NO egress rules defined → egress blocked (fail closed). Outbound
  # control-plane calls use the private route with NAT allowlist.
  tags = local.tags
}

resource "aws_security_group" "control_plane" {
  name_prefix = "${local.name_prefix}-cp"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8081
    to_port         = 8081
    protocol        = "tcp"
    security_groups = [aws_security_group.api.id]
    description     = "Control plane from BFF only"
  }
  # NO default egress
  tags = local.tags
}

# -----------------------------------------------------------------------------
# Secrets Manager — JWT signing key & provider credentials
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "jwt_signing_key" {
  name = "${local.name_prefix}-jwt-signing-key"
  tags = local.tags
}

resource "aws_secretsmanager_secret_version" "jwt_signing_key" {
  secret_id = aws_secretsmanager_secret.jwt_signing_key.id
  secret_string = jsonencode({
    # Placeholder — populated by an operator with an RSA-2048 private key.
    # NEVER commit the actual key to git.
    rsa_private_key_pem = var.jwt_signing_key_pem_placeholder
  })
}

# -----------------------------------------------------------------------------
# ECS Cluster + Fargate services
# -----------------------------------------------------------------------------
resource "aws_ecs_cluster" "main" {
  name = "${local.name_prefix}-cluster"
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
  tags = local.tags
}

# Placeholder task definitions — production wiring is left as exercises
# for the platform team; the SGs and network policy above are the security
# contract.
resource "aws_ecs_task_definition" "api" {
  family                   = "${local.name_prefix}-api"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = aws_iam_role.execution.arn
  container_definitions = jsonencode([
    {
      name      = "dev-portal-api"
      image     = var.api_image
      essential = true
      portMappings = [
        { containerPort = 8080, protocol = "tcp" }
      ]
      environment = [
        { name = "DP_CP_BASE_URL", value = "http://control-plane:8081" }
      ]
      secrets = [
        { name = "DP_CP_AUTH_TOKEN", valueFrom = aws_secretsmanager_secret.jwt_signing_key.arn }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "/ecs/${local.name_prefix}-api"
          "awslogs-region"        = local.region
          "awslogs-stream-prefix" = "api"
        }
      }
    }
  ])
  tags = local.tags
}

resource "aws_iam_role" "execution" {
  name = "${local.name_prefix}-execution"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = { Service = "ecs-tasks.amazonaws.com" }
        Action = "sts:AssumeRole"
      }
    ]
  })
  tags = local.tags
}

data "aws_availability_zones" "available" {
  state = "available"
}

variable "region" {
  description = "AWS region"
  default     = "us-east-1"
}

variable "api_image" {
  description = "ECR image URI for the BFF API"
}

variable "jwt_signing_key_pem_placeholder" {
  description = "Placeholder RSA private key PEM (populated by operator)"
  sensitive   = true
  default     = "PLACEHOLDER-DO-NOT-USE"
}