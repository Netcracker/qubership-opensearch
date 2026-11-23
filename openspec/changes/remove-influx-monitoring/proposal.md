# Proposal

## Why

InfluxDB is no longer a supported monitoring sink for OpenSearch. The chart still branches on `monitoring.monitoringType=influxdb` (Telegraf `outputs.influxdb`, System Monitoring credentials, and a local demo stack), which keeps an obsolete path alive. The Helm values schema is shared across chart versions, so deleting those keys would make validation fail for values files that still set them.

## What Changes

- Remove the InfluxDB monitoring sink everywhere it is implemented: Helm templates, the monitoring secret, the operator CR `secretName` that exists only for InfluxDB, docs that present InfluxDB as a supported output, Grafana dashboard datasource metadata that still says `influxdb`, and the demo InfluxDB compose service.
- Monitoring with `monitoring.enabled=true` always uses the Prometheus output (`outputs.prometheus_client` on port 8096), including Service, ServiceMonitor, dashboards, and TCP probes. **BREAKING** for clusters that still send metrics to InfluxDB: those installs will scrape Prometheus instead and will no longer write to InfluxDB.
- Keep backward-compatible Helm values. `monitoring.monitoringType` (including the value `influxdb`) and `monitoring.smDbHost`, `smDbName`, `smDbUsername`, `smDbPassword` stay in `values.yaml` and `values.schema.json` with the same types and defaults. Existing values files continue to pass schema validation; the chart ignores the InfluxDB-specific values.
- Do not narrow `monitoringType` to an enum that rejects `influxdb`, and do not drop or retype the `smDb*` properties when the shared schema is regenerated.
- Leave Telegraf's input `data_format = "influx"` (Influx line protocol) in place. That is the exec-plugin wire format, not the InfluxDB output feature.
- Zabbix is out of scope: this repository has no Zabbix feature left to remove.

## Capabilities

### New Capabilities

- `opensearch-monitoring`: OpenSearch monitoring publishes Prometheus metrics. Obsolete InfluxDB settings remain valid in the shared Helm schema and are ignored.

### Modified Capabilities

<!-- No existing specs. -->

## Impact

- Helm templates under `operator/charts/helm/opensearch-service/templates/monitoring/` and `templates/operator/cr.yaml`.
- Shared schema inputs: `values.yaml`, `values.schema.json`, and the parameter table in `docs/public/installation.md` (descriptions change; property names, types, and defaults do not).
- Grafana JSON under `operator/charts/helm/opensearch-service/monitoring/` (`dsType` only).
- Demo stack: `demo/monitoring/docker-compose.yml` and `demo/monitoring/grafana/datasources/datasources.yml`.
- `docs/public/security.md` (port 8096 is no longer conditional on a non-InfluxDB type).
- No operator Go API change. `Monitoring.SecretName` stays on the CR; new renders simply stop setting it for InfluxDB.
- No dedicated InfluxDB tests exist in this repository. Any test that asserts the InfluxDB branch is removed if one is found during implementation.
