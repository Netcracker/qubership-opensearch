# Spec Delta

## Purpose

OpenSearch monitoring publishes Prometheus metrics. Values left over from the InfluxDB sink stay valid in the shared Helm schema and do not change what the chart renders.

## ADDED Requirements

### Requirement: Prometheus is the only monitoring sink

When monitoring is enabled, the rendered monitoring configuration SHALL publish metrics for Prometheus on port 8096 and SHALL NOT configure an InfluxDB output.

#### Scenario: Default monitoring type

- **WHEN** monitoring is enabled and `monitoring.monitoringType` is `prometheus`
- **THEN** the Telegraf configuration contains `outputs.prometheus_client` listening on `:8096` and does not contain `outputs.influxdb`

#### Scenario: Legacy InfluxDB type still renders Prometheus

- **WHEN** monitoring is enabled and `monitoring.monitoringType` is `influxdb`
- **THEN** the Telegraf configuration contains `outputs.prometheus_client` listening on `:8096` and does not contain `outputs.influxdb`

### Requirement: Shared schema accepts obsolete InfluxDB values

The shared Helm values schema SHALL keep `monitoring.monitoringType` as a string and SHALL keep `monitoring.smDbHost`, `monitoring.smDbName`, `monitoring.smDbUsername`, and `monitoring.smDbPassword` as strings with their current defaults. Validation SHALL accept a values file that still sets those properties, including `monitoringType: influxdb`.

#### Scenario: Existing InfluxDB values pass validation

- **WHEN** a values file sets `monitoring.monitoringType` to `influxdb` and sets any of `smDbHost`, `smDbName`, `smDbUsername`, or `smDbPassword`
- **THEN** Helm values schema validation succeeds

#### Scenario: Schema does not reject the legacy type

- **WHEN** the shared schema is regenerated from the chart values and documentation
- **THEN** `monitoringType` is not restricted by an enum that excludes `influxdb`, and the `smDb*` properties are still present with type string

### Requirement: Obsolete InfluxDB settings are ignored

The chart SHALL ignore `monitoring.monitoringType` when its value is `influxdb` and SHALL ignore `smDbHost`, `smDbName`, `smDbUsername`, and `smDbPassword`. It SHALL NOT create a System Monitoring credential secret, SHALL NOT set the operator CR monitoring `secretName` from those values, and SHALL NOT inject `SM_DB_HOST` or `SM_DB_NAME` into the monitoring container.

#### Scenario: InfluxDB type does not create a monitoring secret

- **WHEN** monitoring is enabled and `monitoring.monitoringType` is `influxdb` with `smDbUsername` and `smDbPassword` set
- **THEN** the chart does not render the monitoring secret that stores `sm-db-username` and `sm-db-password`

#### Scenario: System Monitoring variables are not injected

- **WHEN** monitoring is enabled and `smDbHost` or `smDbName` is non-empty
- **THEN** the monitoring container environment does not include `SM_DB_HOST` or `SM_DB_NAME`

### Requirement: Prometheus resources are not gated on monitoring type

When monitoring is enabled, the monitoring Service SHALL expose port 8096, and the container SHALL use TCP probes on port 8096, including when `monitoring.monitoringType` is `influxdb`. ServiceMonitor, Grafana dashboards, and Prometheus rules SHALL follow their existing enablement flags and SHALL NOT be skipped because `monitoringType` is `influxdb`. Installed dashboards SHALL use a Prometheus datasource.

#### Scenario: Service and probes use port 8096 for a legacy type

- **WHEN** monitoring is enabled and `monitoring.monitoringType` is `influxdb`
- **THEN** the monitoring Service exposes port 8096 and the container liveness and readiness probes are TCP checks on port 8096

#### Scenario: Dashboards are not skipped for a legacy type

- **WHEN** monitoring is enabled, `monitoring.installDashboard` is true, and `monitoring.monitoringType` is `influxdb`
- **THEN** the Grafana dashboard resources are rendered and their datasource type is Prometheus

### Requirement: Exec metric input format stays the line protocol

Telegraf exec inputs SHALL continue to consume metrics with `data_format` set to `influx`. That setting is the line protocol of the exec scripts and SHALL NOT be removed with the InfluxDB output.

#### Scenario: Exec input format is unchanged

- **WHEN** monitoring configuration is rendered
- **THEN** each exec input that previously used `data_format = "influx"` still uses `data_format = "influx"`

### Requirement: Documentation describes a Prometheus-only sink

Installation and security documentation SHALL describe Prometheus as the monitoring output. They SHALL state that `monitoring.monitoringType=influxdb` and the `smDb*` parameters are accepted for compatibility and ignored. They SHALL NOT describe InfluxDB as a supported monitoring sink.

#### Scenario: Installation parameters no longer offer InfluxDB

- **WHEN** a user reads the monitoring parameter table
- **THEN** the table says the output is Prometheus, and the `smDb*` rows say the parameters are ignored and kept so existing values files still validate
