provider "pingdom" {}

resource "pingdom_check" "laa-legal-adviser-api-search" {
  type                     = "http"
  name                     = "LAA Legal Adviser API Search [staging]"
  host                     = "laa-legal-adviser-api-production.cloud-platform.service.justice.gov.uk"
  resolution               = 1
  notifywhenbackup         = true
  sendnotificationwhendown = 6
  notifyagainevery         = 0
  url                      = "/ping.json"
  encryption               = true
  port                     = 443
  tags                     = "businessunit_${var.business-unit},application_${var.application},component_ping,isproduction_${var.is-production},environment_${var.environment-name},infrastructuresupport_${var.application}"
  probefilters             = "region:EU"
  integrationids           = [87631]
}