## Why

OpenSearch does not support global index settings templates for existing indices; only newly-created indices can be configured via index templates. Operators need a way to declaratively apply and maintain index-level settings (such as translog durability, sync interval, and flush threshold) across all matching indices — including those that already exist — without manual per-index API calls.

## What Changes

- Add `opensearch.indexSettings` array field to the Helm `values.yaml` and the operator CRD (`OpenSearchServiceSpec`), accepting a list of `{pattern, settings}` entries.
- Each entry's `pattern` is an OpenSearch index name pattern (e.g., `*`, `*bss*`); `settings` is a flat map of index setting key-value pairs.
- Implement an `IndexSettingsWatcher` goroutine (modeled after `SlowLogIndicesWatcher`) that periodically issues `PUT <pattern>,-.*/_settings` calls for each entry, skipping system indices (dot-prefixed and any other known system indices).
- The watcher is started/stopped by the reconciler when `opensearch.indexSettings` is present/absent.
- Settings are applied in array order; later entries for overlapping patterns override earlier ones on matched indices.
- Removing a setting from config does NOT remove it from OpenSearch automatically; operators must set the value to `null` to reset it to the OpenSearch default.
- Update Helm chart documentation and operator CRD docs to describe the new field, its behavior, and the `null`-reset pattern.

## Capabilities

### New Capabilities

- `index-settings-by-pattern`: Periodic application of index-level settings to all non-system indices matching a given pattern, configured via `opensearch.indexSettings` in deployment parameters.

### Modified Capabilities

<!-- No existing spec-level capabilities are being modified. -->

## Impact

- **CRD types**: `operator/api/v1/opensearchservice_types.go` — add `IndexSettings []IndexSettingEntry` to `OpenSearch` struct.
- **Helm values**: `operator/charts/helm/opensearch-service/values.yaml` — add `opensearch.indexSettings` array.
- **New controller file**: `operator/controllers/index_settings_watcher.go` — goroutine watcher following `SlowLogIndicesWatcher` pattern.
- **Reconciler**: `operator/controllers/opensearch_reconciler.go` (or equivalent) — wire start/stop of the new watcher.
- **main.go**: instantiate and register `IndexSettingsWatcher`.
- **Docs**: README / operator configuration docs updated with field description and null-reset note.
