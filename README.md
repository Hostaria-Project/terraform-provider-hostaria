# terraform-provider-hostaria

Provider Terraform / OpenTofu pour l’API Hostaria. Un seul binaire, protocol v6.

**URLs :** `https://api.hostaria.cloud/v1/<ressource>` — jamais `/api/v1` ni `/v1/api/v1`.
L’`endpoint` est l’hôte (`https://api.hostaria.cloud` ou un mock `http://127.0.0.1:18080`). S’il finit déjà par `/v1`, le client ne re-préfixe pas.

## Dev local

```bash
cd terraform-provider-hostaria
go test ./...
go build -o terraform-provider-hostaria .
go run ./cmd/mock-api   # autre terminal : http://127.0.0.1:18080/v1

export TF_CLI_CONFIG_FILE="$PWD/examples/local/dev.tfrc"
cd examples/local
# pas de terraform init (dev_overrides)
terraform apply   # ou tofu apply
```

`dev.tfrc` pointe `hostaria/hostaria` vers le dossier du binaire compilé.

Variables : `HOSTARIA_API_KEY`, `HOSTARIA_ENDPOINT`.

## Scale in-place (Starter → Ultra)

Après deploy API : [examples/scale](examples/scale) — `./run.sh` ou les apply documentés dans le README du dossier. Même UUID, pas de recreate.

## Smoke API réelle

Prérequis : clé `host_live_…` (Compte → Développeur) + carte enregistrée (Compte → Facturation). Pas de `game_server`.

```bash
cd terraform-provider-hostaria
go build -o terraform-provider-hostaria .
export TF_CLI_CONFIG_FILE="$PWD/examples/smoke/dev.tfrc"
export TF_VAR_hostaria_api_key='host_live_…'
cd examples/smoke
terraform apply   # ou tofu apply
```

## Ressources

| Terraform | HTTP |
|-----------|------|
| `data.hostaria_offers` / `data.hostaria_offer` | `GET /v1/offers` |
| `data.hostaria_os` | `GET /v1/os` |
| `hostaria_vps` | `/v1/vps` — `offer_id` / `cpu_cores` / `ram_mb` / `storage_gb` : update in-place (disque grow-only). `os_id` et `name` : recreate. |
| `hostaria_ssh_key` | `/v1/ssh-keys` (contrat v1 ; l’API actuelle exige encore un code email) |
| `hostaria_game_server` | `/v1/game-servers` (façade v1 ; mock local) |

`api_key` : clé `host_live_…` générée dans Compte → IaC. Un JWT de session n’est plus le chemin client.
