## 1. CRD Type Changes

- [x] 1.1 Add `IndexSettingEntry` struct to `operator/api/v1/opensearchservice_types.go` with fields `Pattern string` (json: `pattern`) and `Settings map[string]interface{}` (json: `settings`)
- [x] 1.2 Add `IndexSettings []IndexSettingEntry` field (json: `indexSettings,omitempty`) to the `OpenSearch` struct in `operator/api/v1/opensearchservice_types.go`
- [x] 1.3 Regenerate or update the CRD YAML manifest to include the new `indexSettings` field (run `make generate manifests` or equivalent)

## 2. Helm Values

- [x] 2.1 Add `indexSettings: []` default entry under `opensearch:` in `operator/charts/helm/opensearch-service/values.yaml` with inline comment referencing the feature
- [x] 2.2 Verify the Helm chart schema / CRD template picks up the new field (no extra template change needed if CRD is auto-generated from Go types)

## 3. IndexSettingsWatcher Implementation

- [x] 3.1 Create `operator/controllers/index_settings_watcher.go` modeled on `slowlog_indices_watcher.go`: define `IndexSettingsHelper` struct (logger + restClient), `IndexSettingsWatcher` struct (lock `*sync.Mutex`, State `*string`), and `NewIndexSettingsWatcher(mutex *sync.Mutex)` constructor
- [x] 3.2 Implement `start(helper, entries []IndexSettingEntry)` — stop any running instance, set state to `running`, launch `go watch()`
- [x] 3.3 Implement `stop(helper)` — set state to `stopped` (no settings are reset on stop, unlike slow-log)
- [x] 3.4 Implement `watch(helper, entries)` — acquire mutex, loop: check stopped state, call `applyAllSettings()`, sleep `watchInterval` (60s)
- [x] 3.5 Implement `applyAllSettings(helper, entries)` — iterate entries in order; for each entry call `PUT <pattern>,-.*/_settings?allow_no_indices=true` with the settings map serialized as JSON; log result
- [x] 3.6 Serialize `Settings map[string]interface{}` to JSON correctly: Go `nil` values must serialize as JSON `null` (standard `encoding/json` handles this; verify with a unit test or manual check)

## 4. Reconciler Wiring

- [x] 4.1 Add `IndexSettingsWatcher IndexSettingsWatcher` field to `OpenSearchServiceReconciler` struct in `operator/controllers/opensearchservice_reconciler.go`
- [x] 4.2 Instantiate `IndexSettingsWatcher` in `operator/main.go` (allocate a new `sync.Mutex`, call `controllers.NewIndexSettingsWatcher(&mutexThree)`) and pass it to `OpenSearchServiceReconciler`
- [x] 4.3 Identify the reconcile path for `opensearch.*` fields (likely `opensearch_reconciler.go` or the main reconcile function) and add start/stop logic: if `cr.Spec.OpenSearch.IndexSettings` is non-nil and non-empty → call `IndexSettingsWatcher.start(helper, entries)`; otherwise → call `IndexSettingsWatcher.stop(helper)`

## 5. Documentation

- [x] 5.1 Add `opensearch.indexSettings` parameter table entry in `docs/public/installation.md` near the existing `opensearch.config` entry (line ~1207), describing the array structure, each entry's `pattern` and `settings` fields, and defaults
- [x] 5.2 Add a usage example in `docs/public/installation.md` showing a two-entry `indexSettings` block (one for `*`, one for `*bss*`) matching the user's requested YAML format
- [x] 5.3 Add a note explicitly stating: removing a setting key from `indexSettings` does NOT reset it in OpenSearch — set the value to `null` (e.g., `index.translog.sync_interval: null`) to reset to the OpenSearch default
