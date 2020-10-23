module "rds" {
  source                     = "github.com/ministryofjustice/cloud-platform-terraform-rds-instance?ref=5.9"
  cluster_name               = var.cluster_name
  cluster_state_bucket       = var.cluster_state_bucket
  team_name                  = var.team_name
  business-unit              = var.business_unit
  application                = var.application
  is-production              = var.is_production
  namespace                  = var.namespace
  db_engine                  = "postgres"
  db_engine_version          = "9.5"
  db_backup_retention_period = "7"
  db_name                    = "dstest_rds"
  environment-name           = var.environment
  infrastructure-support     = var.infrastructure_support

  # rds_family should be one of: postgres9.4, postgres9.5, postgres9.6, postgres10, postgres11
  # Pick the one that defines the postgres version the best
  rds_family = "postgres9.5"

  # Some engines can't apply some parameters without a reboot(ex postgres9.x cant apply force_ssl immediate).
  # You will need to specify "pending-reboot" here, as default is set to "immediate".

  # use "allow_major_version_upgrade" when upgrading the major version of an engine
  allow_major_version_upgrade = "true"

  db_parameter = [
    {
      name         = "rds.force_ssl"
      value        = "1"
      apply_method = "pending-reboot"
    }
  ]

  providers = {
    aws = aws.london
  }
}

resource "kubernetes_secret" "dstest_rds" {
  metadata {
    name      = "dstest-rds-output"
    namespace = var.namespace
  }

  data = {
    rds_instance_endpoint = module.rds.rds_instance_endpoint
    database_name         = module.rds.database_name
    database_username     = module.rds.database_username
    database_password     = module.rds.database_password
    rds_instance_address  = module.rds.rds_instance_address

    access_key_id     = module.rds.access_key_id
    secret_access_key = module.rds.secret_access_key

    url = "postgres://${module.rds.database_username}:${module.rds.database_password}@${module.rds.rds_instance_endpoint}/${module.rds.database_name}"
  }
}
