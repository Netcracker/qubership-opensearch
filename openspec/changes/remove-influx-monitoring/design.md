# Design

## Context

See proposal.md for why InfluxDB monitoring is being removed. The live path is Helm, not the operator binary.

`monitoring.monitoringType` defaults to `prometheus`. The value `influxdb` selects a second Telegraf output (`outputs.influxdb`), `SM_DB_*` env vars, a dedicated Secret, exec probes, and it suppresses port 8096, ServiceMonitor, dashboards, and Prometheus rules. Every other value, including `prometheus`, takes the Prometheus branch. The condition is a string compare against `"influxdb"` in:

- `templates/monitoring/configuration.yaml`, `deployment.yaml`, `secret.yaml`, `service.yaml`
- `templates/monitoring/grafana-dashboard.yaml`, `indices-dashboard.yaml`, `replication-dashboard.yaml`, `slow-queries-dashboard.yaml`, `prometheus-rules.yaml`, `tls_static_metrics.yaml`, `service-monitor.yaml`
- `templates/operator/cr.yaml` (`monitoring.secretName` only when the type is `influxdb`)

`values.schema.json` is produced by `nc-schema-gen` (`operator/dev-kit/make_json_schema.sh`) from `values.yaml` and `docs/public/installation.md`, with `--leveling`. That schema is shared across chart versions. `monitoring.monitoringType` and `smDbHost`, `smDbName`, `smDbUsername`, `smDbPassword` are plain strings with defaults `prometheus` and `""`. There is no enum. A comment elsewhere in the schema already records that changing a property type breaks consumers of the shared schema.

Telegraf exec inputs use `data_format = "influx"` (line protocol from the Python metric scripts). That is independent of the InfluxDB output plugin.

Grafana JSON under `monitoring/*.json` still has `"dsType": "influxdb"` on panels that also carry a PromQL `expr`. Those dashboards are installed only on the Prometheus path today.

The demo compose file runs InfluxDB 1.8 and points `elasticsearch-monitoring` at it. Prometheus in that file is commented out.

No Go type and no test in this repository mentions InfluxDB. `Monitoring.SecretName` on the CR stays; the reconciler only hashes a monitoring secret when that field is non-empty.

## Goals / Non-Goals

**Goals:**

- One monitoring render path: Prometheus on port 8096, for every `monitoringType`, including `influxdb`.
- Helm upgrade of a values file that still sets `influxdb` and `smDb*` passes schema validation and installs Prometheus monitoring.
- Regenerating the shared schema does not delete or retype those five properties.

**Non-Goals:**

- Removing `monitoringType` or the `smDb*` keys from values, the schema, or the CRD.
- Changing Telegraf `data_format = "influx"` or the exec scripts that emit line protocol.
- Removing `Monitoring.SecretName` from the Go API.
- Zabbix. Nothing in this repository implements it.
- Rewriting PromQL in the Grafana JSON.

## Decisions

### 1. Delete the InfluxDB branch; do not keep a disabled output

Drop the `eq ... "influxdb"` branches and the `ne ... "influxdb"` guards. The Prometheus block in `configuration.yaml`, the TCP probes, port 8096, and the ServiceMonitor / dashboard / rule templates become unconditional aside from the flags they already have (`monitoring.enabled`, `installDashboard`, `includeIndices`, slow queries, disaster recovery).

Delete `templates/monitoring/secret.yaml`. It exists only to store `sm-db-username` and `sm-db-password`. On upgrade Helm drops that Secret because it leaves the manifest. Stop emitting `monitoring.secretName` in `cr.yaml`. Stop mounting that secret and stop setting `SM_DB_HOST` and `SM_DB_NAME` in `deployment.yaml`.

**Alternative**: Keep the InfluxDB output behind the old flag. Rejected — the story is to remove the feature, not to hide it.

**Alternative**: Fail the render when `monitoringType` is `influxdb`. Rejected — existing values files would break at install time. The compatibility requirement is that those files still apply.

### 2. Keep the five values so the shared schema does not shrink

Leave these keys in `values.yaml` with the same defaults:

- `monitoring.monitoringType: "prometheus"`
- `monitoring.smDbHost`, `smDbName`, `smDbUsername`, `smDbPassword: ""`

Add a short comment that they are accepted for compatibility and ignored. Keep the matching rows in `docs/public/installation.md` so `nc-schema-gen` still sees the properties. Change the descriptions: the output is Prometheus; `influxdb` and the `smDb*` parameters are ignored and remain so existing values files validate. Do not add an `enum`.

If the schema is regenerated, the diff for these five properties may change description text only. Types stay `string`. Defaults stay `prometheus` and `""`. The properties stay in the schema. Do not set `additionalProperties: false` on `monitoring`.

**Alternative**: Delete the keys from `values.yaml` and rely on JSON Schema's default of allowing unknown properties. Rejected — `--leveling` builds the shared schema from the values and the doc table. Dropping the keys drops them from the schema that other chart versions consume, and a values file that still sets them fails where that schema is strict.

### 3. Dashboard JSON: change datasource type only where PromQL already exists

In `monitoring/opensearch-dashboard.json` and `monitoring/replication-dashboard.json`, set `"dsType": "prometheus"` on targets that already have an `expr`. Do not edit the PromQL. If a target has `dsType: influxdb` and no `expr`, leave that target and call it out in the implementation notes instead of inventing a query.

**Alternative**: Leave `dsType: influxdb`. Rejected — those dashboards are the Prometheus dashboards, and the leftover type binds them to an InfluxDB datasource that this change removes.

### 4. Demo stack switches to Prometheus

Remove the `influxdb` service and the InfluxDB Grafana datasource. Enable the Prometheus service that is already sketched in `demo/monitoring/docker-compose.yml`, drop `SM_DB_*` from `elasticsearch-monitoring`, and make Prometheus the default Grafana datasource. The demo is a local illustration of the chart's monitoring image, not a second supported sink.

### 5. Docs for port 8096

`docs/public/security.md` currently says port 8096 is used when `monitoringType` is not `influxdb`. State that 8096 is the monitoring port whenever monitoring is enabled.

## Risks / Trade-offs

- [Clusters that still write metrics to InfluxDB lose that sink on upgrade and start exposing Prometheus on 8096] → Called out as breaking in the proposal. Rollback is to redeploy the previous chart. There is no dual-write period.
- [Helm upgrade deletes the monitoring Secret that held `sm-db-username` / `sm-db-password`] → Intended. The secret has no remaining consumer. Credentials in values are unused.
- [An already-applied CR keeps `monitoring.secretName` until the Helm upgrade rewrites the CR] → After the upgrade the field is omitted, so the reconciler stops hashing that secret. No Go change.
- [Schema regeneration rewrites unrelated description text or drops a property] → Review the schema diff and revert any change other than the InfluxDB description updates. Do not merge a schema that removes `monitoringType` or any `smDb*` property or changes their types.
- [A Grafana panel is Influx-only] → Do not rewrite it blindly. Record it if found.
- [Comment-only edits in `values.yaml` still cause `nc-schema-gen` to touch `$id` or wording] → Accept description updates for the five properties; reject structural edits.

## Migration Plan

1. Edit templates, dashboards, demo, and docs as above. Keep the five values keys.
2. Regenerate `values.schema.json` only if the doc-table wording must flow into schema descriptions. Confirm the five properties remain, with the same types and defaults, and that `monitoringType` has no enum.
3. Render the chart twice: default values, and `monitoring.monitoringType=influxdb` plus non-empty `smDb*`. Both renders have Prometheus output and port 8096, and neither has `outputs.influxdb`, `SM_DB_HOST`, or the monitoring secret.
4. Rollback is the previous chart revision. No data migration. InfluxDB history is left where it is; this chart stops writing it.

## Open Questions

None. Compatibility is "accept and ignore", not "keep writing to InfluxDB".
