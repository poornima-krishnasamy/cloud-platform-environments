variable "domain" {
  default = "offender-case-notes-dev.service.justice.gov.uk"
}

variable "application" {
  default = "offender-case-notes"
}

variable "namespace" {
  default = "offender-case-notes-dev"
}

variable "business-unit" {
  description = "Area of the MOJ responsible for the service."
  default     = "HMPPS"
}

variable "team_name" {
  description = "The name of your development team"
  default     = "Digital Prison Services/Tech"
}

variable "environment-name" {
  description = "The type of environment you're deploying to."
  default     = "dev"
}

variable "infrastructure-support" {
  description = "The team responsible for managing the infrastructure. Should be of the form team-email."
  default     = "matt.whittaker@digtal.justice.gov.uk"
}

variable "is-production" {
  default = "false"
}
