terraform {
  required_providers {
    hostaria = {
      source = "hostaria/hostaria"
    }
  }
}

provider "hostaria" {
  endpoint = var.hostaria_endpoint
  api_key  = var.hostaria_api_key
}

variable "hostaria_endpoint" {
  type        = string
  default     = "https://api.hostaria.cloud"
  description = "Hôte API (sans /v1). Le client ajoute /v1."
}

variable "hostaria_api_key" {
  type        = string
  default     = ""
  sensitive   = true
  description = "Clé host_live_… (Compte → IaC). Vide = HOSTARIA_API_KEY."
}

variable "offer_name" {
  type        = string
  default     = "Starter"
  description = "1er apply : Starter. 2e apply : Ultra. Doit rester la même ressource (pas de replace)."
}

variable "os_slug" {
  type    = string
  default = "ubuntu-24-04"
}

variable "vps_name" {
  type    = string
  default = "tf-scale-01"
}

variable "ssh_public_key" {
  type        = string
  default     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA tf-scale@hostaria"
  description = "Clé publique. Les clés du compte sont aussi injectées si celle-ci est dummy."
}

data "hostaria_offers" "all" {}

data "hostaria_offer" "selected" {
  name = var.offer_name
}

data "hostaria_os" "ubuntu" {
  slug = var.os_slug
}

resource "hostaria_ssh_key" "scale" {
  name       = "tf-scale"
  public_key = var.ssh_public_key
}

resource "hostaria_vps" "scale" {
  offer_id = data.hostaria_offer.selected.id
  os_id    = data.hostaria_os.ubuntu.id
  name     = var.vps_name

  bootstrap {
    username    = "deploy"
    auth_method = "ssh_only"
    ssh_key_ids = [hostaria_ssh_key.scale.id]
  }
}

output "offers" {
  value = [for o in data.hostaria_offers.all.offers : "${o.name} ${o.price}€ cpu=${o.cpu_cores} ram=${o.ram} disk=${o.storage}"]
}

output "offer_id" {
  value = data.hostaria_offer.selected.id
}

output "offer_name" {
  value = data.hostaria_offer.selected.name
}

output "vps_id" {
  value = hostaria_vps.scale.id
}

output "vps_uuid" {
  value = hostaria_vps.scale.uuid
}

output "vps_hostname" {
  value = hostaria_vps.scale.hostname
}

output "vps_ip" {
  value = hostaria_vps.scale.ip_address
}

output "vps_status" {
  value = hostaria_vps.scale.status
}

output "vps_cpu" {
  value = hostaria_vps.scale.cpu_cores
}

output "vps_ram_mb" {
  value = hostaria_vps.scale.ram_mb
}

output "vps_storage_gb" {
  value = hostaria_vps.scale.storage_gb
}
