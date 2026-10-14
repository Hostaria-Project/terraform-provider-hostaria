# terraform-provider-hostaria

Provider Terraform et OpenTofu pour [Hostaria](https://hostaria.cloud).

## Installation

```hcl
terraform {
  required_providers {
    hostaria = {
      source  = "hostaria-project/hostaria"
      version = "0.1.1"
    }
  }
}

provider "hostaria" {}
```

La clé `host_live_…` se crée dans le [compte Hostaria](https://hostaria.cloud), section IaC. Elle peut aussi être lue depuis `HOSTARIA_API_KEY`.

## Documentation

Le détail des ressources est sur le [registry Terraform](https://registry.terraform.io/providers/hostaria-project/hostaria).

- `hostaria_vps`
- `hostaria_ssh_key`
- `hostaria_game_server`
- `hostaria_offer`
- `hostaria_offers`
- `hostaria_os`
