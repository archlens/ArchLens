terraform {
  cloud {
    organization = "Perlt"
    workspaces {
      name = "mt-diagrams-plantuml"
    }
  }
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.45.0"
    }
  }
}

provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
  client_id       = var.client_id
  client_secret   = var.client_secret
  tenant_id       = var.tenant_id
}

variable "subscription_id" { type = string }
variable "client_id" { type = string }
variable "client_secret" { type = string }
variable "tenant_id" { type = string }


variable "plantuml_size_limit" {
  type    = string
  default = "8192"
}

data "azurerm_service_plan" "main" {
  name                = "mt-diagram-service-plan"
  resource_group_name = "mt-diagrams"
}

resource "azurerm_linux_web_app" "main" {
  name                = "mt-plantuml-app-service"
  resource_group_name = data.azurerm_service_plan.main.resource_group_name
  location            = data.azurerm_service_plan.main.location
  service_plan_id     = data.azurerm_service_plan.main.id

  https_only = true

  app_settings = {
    WEBSITES_PORT                   = 8080
    PLANTUML_LIMIT_SIZE             = var.plantuml_size_limit
    DOCKER_REGISTRY_SERVER_USERNAME = ""
    DOCKER_REGISTRY_SERVER_PASSWORD = ""
    DOCKER_ENABLE_CI                = true
    DOCKER_REGISTRY_SERVER_URL      = "https://index.docker.io/v1"
  }
  site_config {
    always_on = false

    application_stack {
      docker_image     = "plantuml/plantuml-server"
      docker_image_tag = "tomcat"
    }
  }
}
