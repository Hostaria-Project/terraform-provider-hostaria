#!/usr/bin/env bash
# Scale in-place : Starter → Ultra, même UUID. Usage :
#   export TF_VAR_hostaria_api_key='host_live_…'
#   ./run.sh
# Destroy à la fin : DESTROY=1 ./run.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HERE="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
go build -o terraform-provider-hostaria .
export TF_CLI_CONFIG_FILE="$HERE/dev.tfrc"
cd "$HERE"

if [[ -z "${TF_VAR_hostaria_api_key:-}" && -z "${HOSTARIA_API_KEY:-}" ]]; then
  echo "Exporte TF_VAR_hostaria_api_key ou HOSTARIA_API_KEY" >&2
  exit 1
fi

echo "==> 1/3 apply Starter"
terraform apply -auto-approve -var='offer_name=Starter'
UUID1="$(terraform output -raw vps_uuid)"
ID1="$(terraform output -raw vps_id)"
echo "    id=$ID1 uuid=$UUID1 cpu=$(terraform output -raw vps_cpu) disk=$(terraform output -raw vps_storage_gb)"

echo "==> 2/3 plan Ultra (doit être update, PAS destroy/replace)"
PLAN=$(terraform plan -no-color -var='offer_name=Ultra')
echo "$PLAN"
if echo "$PLAN" | grep -E 'must be replaced|# hostaria_vps\.scale must be replaced|forces replacement'; then
  echo "FAIL: Terraform veut recreate. Le provider n'est pas le binaire local, ou offer_id est encore RequiresReplace." >&2
  exit 1
fi
if ! echo "$PLAN" | grep -E 'hostaria_vps.scale.*update|~ offer_id|will be updated in-place'; then
  echo "WARN: pas de ligne update visible — vérifie le plan ci-dessus." >&2
fi

echo "==> 3/3 apply Ultra"
terraform apply -auto-approve -var='offer_name=Ultra'
UUID2="$(terraform output -raw vps_uuid)"
ID2="$(terraform output -raw vps_id)"
echo "    id=$ID2 uuid=$UUID2 cpu=$(terraform output -raw vps_cpu) disk=$(terraform output -raw vps_storage_gb) offer=$(terraform output -raw offer_name)"

if [[ "$UUID1" != "$UUID2" || "$ID1" != "$ID2" ]]; then
  echo "FAIL: id/uuid ont changé (recreate). avant=$ID1/$UUID1 après=$ID2/$UUID2" >&2
  exit 1
fi
echo "OK: même VPS, scale Starter → Ultra."

if [[ "${DESTROY:-}" == "1" ]]; then
  echo "==> destroy"
  terraform destroy -auto-approve -var='offer_name=Ultra'
fi
