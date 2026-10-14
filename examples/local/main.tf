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
  default     = "http://127.0.0.1:18080"
  description = "Hôte API (sans /api). Le client ajoute /v1."
}

variable "hostaria_api_key" {
  type      = string
  default   = "local-dev"
  sensitive = true
}

data "hostaria_offers" "all" {}

data "hostaria_offer" "starter" {
  name = "starter"
}

data "hostaria_os" "ubuntu" {
  slug = "ubuntu-24-04"
}

resource "hostaria_ssh_key" "ci" {
  name       = "ci"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA ci@hostaria"
}

resource "hostaria_vps" "web" {
  offer_id = data.hostaria_offer.starter.id
  os_id    = data.hostaria_os.ubuntu.id
  name     = "prod-web-01"

  bootstrap {
    username    = "deploy"
    auth_method = "ssh_only"
    ssh_key_ids = [hostaria_ssh_key.ci.id]
  }
}

resource "hostaria_game_server" "ark" {
  game     = "ark"
  offer_id = data.hostaria_offer.starter.id
  name     = "my-ark-server"
}

output "vps_id" {
  value = hostaria_vps.web.id
}

output "vps_ip" {
  value = hostaria_vps.web.ip_address
}

output "offers" {
  value = [for o in data.hostaria_offers.all.offers : o.name]
}
