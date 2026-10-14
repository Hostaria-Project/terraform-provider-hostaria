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
  description = "Hôte API (sans /api). Le client ajoute /v1."
}

variable "hostaria_api_key" {
  type        = string
  default     = ""
  sensitive   = true
  description = "Clé host_live_… (Compte → Développeur). Vide = HOSTARIA_API_KEY."
}

variable "offer_name" {
  type        = string
  default     = "Starter"
  description = "Offre publique (insensible à la casse). Prod actuelle : Starter / Booster / Ultra."
}

variable "os_slug" {
  type        = string
  default     = "ubuntu-24-04"
}

variable "vps_name" {
  type    = string
  default = "tf-smoke-01"
}

variable "ssh_public_key" {
  type        = string
  default     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA tf-smoke@hostaria"
  description = "Clé publique. Remplace-la par la tienne pour te connecter au VPS."
}

data "hostaria_offers" "all" {}

data "hostaria_offer" "selected" {
  name = var.offer_name
}

data "hostaria_os" "ubuntu" {
  slug = var.os_slug
}

resource "hostaria_ssh_key" "smoke" {
  name       = "tf-smoke"
  public_key = var.ssh_public_key
}

resource "hostaria_vps" "smoke" {
  offer_id = data.hostaria_offer.selected.id
  os_id    = data.hostaria_os.ubuntu.id
  name     = var.vps_name

  bootstrap {
    username    = "deploy"
    auth_method = "ssh_only"
    ssh_key_ids = [hostaria_ssh_key.smoke.id]
  }
}

output "offers" {
  value = [for o in data.hostaria_offers.all.offers : "${o.name} ${o.price}€"]
}

output "vps_id" {
  value = hostaria_vps.smoke.id
}

output "vps_uuid" {
  value = hostaria_vps.smoke.uuid
}

output "vps_hostname" {
  value = hostaria_vps.smoke.hostname
}

output "vps_ip" {
  value = hostaria_vps.smoke.ip_address
}

output "vps_status" {
  value = hostaria_vps.smoke.status
}
