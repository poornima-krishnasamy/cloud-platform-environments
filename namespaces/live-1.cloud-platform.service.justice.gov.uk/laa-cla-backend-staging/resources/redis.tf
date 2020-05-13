/*
 * When using this module through the cloud-platform-environments, the following
 * two variables are automatically supplied by the pipeline.
 *
 */

variable "cluster_name" {
}

variable "cluster_state_bucket" {
}

/*
 * Make sure that you use the latest version of the module by changing the
 * `ref=` value in the `source` attribute to the latest version listed on the
 * releases page of this repository.
 *
 */
module "redis_elasticache" {
  source                 = "github.com/ministryofjustice/cloud-platform-terraform-elasticache-cluster?ref=4.1"
  cluster_name           = var.cluster_name
  cluster_state_bucket   = var.cluster_state_bucket
  team_name              = var.team_name
  business-unit          = var.business-unit
  application            = var.application
  is-production          = var.is-production
  environment-name       = var.environment-name
  infrastructure-support = var.infrastructure-support
  engine_version         = "5.0.6"
  parameter_group_name   = "default.redis5.0"

  providers = {
    aws = aws.london
  }
}

resource "kubernetes_secret" "redis_elasticache" {
  metadata {
    name      = "ec-cluster"
    namespace = var.namespace
  }

  data = {
    primary_endpoint_address = module.redis_elasticache.primary_endpoint_address
    member_clusters          = jsonencode(module.redis_elasticache.member_clusters)
    auth_token               = module.redis_elasticache.auth_token
    url                      = "redis://${module.redis_elasticache.auth_token}@${module.redis_elasticache.primary_endpoint_address}:6379"
  }
}
