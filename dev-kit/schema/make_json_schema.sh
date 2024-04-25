python schema_filler.py --leveling --skip-type-mismatch --off-array-list-mismatch-log \
  -n opensearch-service \
  -d ../docs/public/installation.md \
  -v ../charts/helm/opensearch-service/values.yaml \
  -o ../charts/helm/opensearch-service/values.schema.json \
  -r ../charts/helm/opensearch-service/values.rules.json