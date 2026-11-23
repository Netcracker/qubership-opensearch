# Tasks

## 1. Helm monitoring templates

- [x] 1.1 In `operator/charts/helm/opensearch-service/templates/monitoring/configuration.yaml`, remove the `outputs.influxdb` branch and always render `outputs.prometheus_client` on `:8096`. Leave every `data_format = "influx"` exec input unchanged. Verify with `rg "outputs.influxdb" operator/charts/helm/opensearch-service/templates` (no matches) and `rg "data_format = \"influx\"" operator/charts/helm/opensearch-service/templates/monitoring/configuration.yaml` (still present).
- [x] 1.2 In `templates/monitoring/deployment.yaml`, remove the `SM_DB_HOST` / `SM_DB_NAME` env block, the monitoring-secret volume mount, and the exec probes. Always expose container port 8096 and use TCP liveness and readiness probes on 8096. Verify with `rg "SM_DB_|sm-db-|pgrep -x telegraf" operator/charts/helm/opensearch-service/templates/monitoring/deployment.yaml` (no matches).
- [x] 1.3 Delete `templates/monitoring/secret.yaml` and stop setting `monitoring.secretName` in `templates/operator/cr.yaml`. Verify both with `rg "sm-db-username|monitoring-secret" operator/charts/helm/opensearch-service/templates` (no matches).
- [x] 1.4 Remove the `(ne .Values.monitoring.monitoringType "influxdb")` guard from `service.yaml`, `service-monitor.yaml`, `grafana-dashboard.yaml`, `indices-dashboard.yaml`, `replication-dashboard.yaml`, `slow-queries-dashboard.yaml`, `prometheus-rules.yaml`, and `tls_static_metrics.yaml`. Keep each template's other conditions (`monitoring.enabled`, `installDashboard`, `includeIndices`, slow queries, disaster recovery). Verify with `rg "influxdb" operator/charts/helm/opensearch-service/templates` (no matches).

## 2. Shared schema compatibility and docs

- [x] 2.1 Keep `monitoring.monitoringType`, `smDbHost`, `smDbName`, `smDbUsername`, and `smDbPassword` in `values.yaml` with the same defaults (`prometheus` and `""`). Add a comment that the keys are accepted for compatibility and ignored. Do not add an enum. Verify the five keys are still present with those defaults.
- [x] 2.2 Update the monitoring rows in `docs/public/installation.md` and the port 8096 note in `docs/public/security.md`: Prometheus is the only sink; `monitoringType=influxdb` and the `smDb*` parameters remain valid and are ignored. Keep the `smDb*` rows so schema generation still sees them. Verify with `rg "supported" docs/public/installation.md` that InfluxDB is not described as a supported output, and that the four `smDb*` parameter rows are still in the table.
- [x] 2.3 Update the five property descriptions in `values.schema.json` in place so they match the doc table. Do not run `nc-schema-gen`: a full regenerate would rewrite unrelated schema text. Verify the five properties remain type `string`, defaults stay `prometheus` and `""`, `monitoringType` has no `enum`, and `monitoring` does not gain `additionalProperties: false`.

## 3. Grafana dashboards

- [x] 3.1 In `monitoring/opensearch-dashboard.json` and `monitoring/replication-dashboard.json`, set `"dsType": "prometheus"` on targets that already have an `expr`. Do not change PromQL. If a target has `dsType: influxdb` and no `expr`, leave it and note the panel in the implementation summary. Verify with a count of remaining `"dsType": "influxdb"` and confirm each remainder has no sibling `expr`.

## 4. Demo stack

- [x] 4.1 Remove the `influxdb` service from `demo/monitoring/docker-compose.yml`, drop `SM_DB_*` and the `influxdb` link from `elasticsearch-monitoring`, and enable the commented Prometheus service. Point `demo/monitoring/telegraf/telegraf.conf` at `outputs.prometheus_client`. In `demo/monitoring/grafana/datasources/datasources.yml`, remove the InfluxDB datasource and make Prometheus the default. Verify with `rg "influxdb|SM_DB_" demo/monitoring` (no matches). `data_format = "influx"` stays in the demo Telegraf config: it is the exec line protocol, not the removed sink.

## 5. Render check

- [x] 5.1 Search tests for an InfluxDB monitoring assertion (`rg "influxdb|smDb|SM_DB_" --glob "*_test*"`). Delete a test only when it asserts the removed sink. Verify no remaining test expects `outputs.influxdb` or the monitoring secret.
- [x] 5.2 `helm template` the chart twice with monitoring enabled: default values, and `monitoring.monitoringType=influxdb` plus non-empty `smDb*`. Verify both renders contain `outputs.prometheus_client` and port 8096, contain `data_format = "influx"`, and contain neither `outputs.influxdb`, `SM_DB_HOST`, nor `sm-db-username`. Verify `helm template` (or `helm lint`) against the second values file exits 0 so the shared schema accepts the legacy keys.
