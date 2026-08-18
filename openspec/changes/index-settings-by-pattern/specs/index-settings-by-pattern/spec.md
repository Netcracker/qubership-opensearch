## Purpose

Enables operators to declaratively apply and maintain index-level settings across all non-system OpenSearch indices matching configurable name patterns, without requiring manual per-index API calls.

## ADDED Requirements

### Requirement: IndexSettings configuration field

The system SHALL accept an `opensearch.indexSettings` array field in the OpenSearch deployment parameters (Helm values / CR spec). Each entry SHALL contain a `pattern` (OpenSearch index name pattern string) and a `settings` map (flat key-value pairs of index setting names to values).

#### Scenario: Valid configuration accepted

- **WHEN** `opensearch.indexSettings` contains one or more entries each with a non-empty `pattern` and a non-empty `settings` map
- **THEN** the operator accepts the configuration without error

#### Scenario: Empty indexSettings disables the feature

- **WHEN** `opensearch.indexSettings` is absent or an empty array
- **THEN** the operator does NOT apply any index settings and the watcher goroutine is not started

### Requirement: Periodic application of index settings

The system SHALL periodically apply the configured settings to all non-system indices that match each pattern, using the OpenSearch `PUT <pattern>/_settings` API.

#### Scenario: Settings applied on watcher tick

- **WHEN** the watcher goroutine ticks (every 300 seconds)
- **THEN** for each entry in `indexSettings`, the operator issues a PUT `<pattern>,-.*/_settings` call with the configured settings map as the request body

#### Scenario: Entries applied in declaration order

- **WHEN** multiple `indexSettings` entries exist whose patterns match the same index
- **THEN** entries are applied in array order, so later entries' settings overwrite earlier ones on overlapping indices

### Requirement: System index exclusion

The system SHALL exclude system indices from settings application. System indices are those whose names begin with a dot (`.`).

#### Scenario: Dot-prefixed indices excluded via API pattern

- **WHEN** a settings PUT is issued for a given pattern
- **THEN** the request URL uses the form `<pattern>,-.*` so that OpenSearch natively excludes all dot-prefixed indices from the operation

### Requirement: Null value resets a setting to OpenSearch default

The system SHALL treat a `null` value for a setting key as an instruction to reset that setting to the OpenSearch default (by passing `null` in the JSON body).

#### Scenario: Setting reset by null value

- **WHEN** an entry's `settings` map contains a key with value `null`
- **THEN** the PUT body includes that key with JSON `null`, causing OpenSearch to reset the setting to its built-in default

#### Scenario: Removing a key from config does not reset it in OpenSearch

- **WHEN** a previously applied setting key is removed from the `settings` map in the deployment configuration
- **THEN** the operator does NOT issue any call to reset that setting; the setting retains its last applied value in OpenSearch until explicitly nulled

### Requirement: Watcher lifecycle tied to configuration

The system SHALL start the IndexSettingsWatcher goroutine when `opensearch.indexSettings` is non-empty, and stop it when the field is absent or becomes empty.

#### Scenario: Watcher starts on non-empty config

- **WHEN** the operator reconciles and `opensearch.indexSettings` is a non-empty array
- **THEN** the IndexSettingsWatcher goroutine is running

#### Scenario: Watcher stops on config removal

- **WHEN** the operator reconciles and `opensearch.indexSettings` is absent or empty
- **THEN** the IndexSettingsWatcher goroutine is stopped and no further PUT calls are issued

### Requirement: Documentation of behavior and null-reset pattern

The operator documentation SHALL describe the `opensearch.indexSettings` field, its schema, the periodic application behavior, and the explicit requirement to use `null` values to reset settings that were previously applied.

#### Scenario: Docs include null-reset guidance

- **WHEN** a user consults the operator configuration documentation
- **THEN** the docs explain that removing a setting from `indexSettings` does NOT remove it from OpenSearch, and that setting the value to `null` is required to reset it to the OpenSearch default
