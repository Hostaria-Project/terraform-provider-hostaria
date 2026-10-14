# Test scale in-place Starter → Ultra

Après deploy API **et** rebuild du provider local (`go build` dans `terraform-provider-hostaria`).

```bash
cd terraform-provider-hostaria
go build -o terraform-provider-hostaria .
export TF_CLI_CONFIG_FILE="$PWD/examples/scale/dev.tfrc"
export TF_VAR_hostaria_api_key='host_live_…'
cd examples/scale

# 1) créer Starter
terraform apply -auto-approve -var='offer_name=Starter'
terraform output vps_uuid   # noter

# 2) scale Ultra — le plan doit dire "updated in-place", jamais "must be replaced"
terraform plan -var='offer_name=Ultra'
terraform apply -auto-approve -var='offer_name=Ultra'
terraform output vps_uuid   # identique
terraform output vps_cpu vps_storage_gb offer_name

# 3) casser le VPS de test
terraform destroy -auto-approve -var='offer_name=Ultra'
```

Ou d’un coup : `./run.sh` puis `DESTROY=1 ./run.sh` pour tout détruire à la fin.

Si le plan montre un **replace**, tu n’utilises pas le binaire fraîchement compilé (`TF_CLI_CONFIG_FILE` / `dev_overrides`).
