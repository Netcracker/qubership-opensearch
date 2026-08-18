## Context

The operator uses goroutine-based watchers (not Kubernetes CronJobs) to apply settings to OpenSearch indices on a recurring interval. `SlowLogIndicesWatcher` (`operator/controllers/slowlog_indices_watcher.go`) is the established pattern: a struct holding a `*sync.Mutex` and a `*string` state (`running`/`stopped`), a `start()` that stops any running instance and launches `go watch()`, and a `watch()` loop that acquires the mutex, applies settings on every tick, and sleeps for `watchInterval` (60s).

The `OpenSearch` struct in `operator/api/v1/opensearchservice_types.go` does not currently carry index-settings configuration. The `opensearch.config` key in Helm `values.yaml` is rendered into `opensearch.yml` (node-level config), which is unrelated to per-index settings.

System indices are excluded in API calls by appending `,-.*` to the index pattern (native OpenSearch syntax), consistent with how `SlowLogIndicesWatcher` and replication code already handle it.

## Goals / Non-Goals

**Goals:**
- Add `opensearch.indexSettings` to the Helm values and to the `OpenSearch` CR struct.
- Implement `IndexSettingsWatcher` goroutine that iterates over entries in declaration order and calls `PUT <pattern>,-.*/_settings?allow_no_indices=true` for each.
- Wire the watcher into `OpenSearchServiceReconciler` and start/stop it during reconciliation when the CR's `opensearch.indexSettings` changes.
- Document the field and the null-reset pattern.

**Non-Goals:**
- Resetting settings when they are removed from config (requires explicit `null` value).
- Applying settings at index-creation time (index templates cover that use case).
- Validating that setting keys or values are recognized by OpenSearch.

## Decisions

### 1. Reuse the existing watcher goroutine pattern

Follow `SlowLogIndicesWatcher` exactly: same mutex/state lifecycle, same `watchInterval` (60s), same `allow_no_indices=true` flag to avoid errors when a pattern matches no indices.

**Alternative**: A Kubernetes CronJob. Rejected — the project has no CronJob precedent; goroutine watchers are already the project standard, keeping all logic inside the operator binary.

### 2. Index settings as `[]IndexSettingEntry` on the `OpenSearch` struct

Add a new field `IndexSettings []IndexSettingEntry` with `omitempty`. `IndexSettingEntry` holds `Pattern string` and `Settings map[string]interface{}`.

Using `map[string]interface{}` (not `map[string]string`) allows `null` values to be represented natively in Go and serialized correctly as JSON `null` in PUT bodies.

**Alternative**: `map[string]string` with a sentinel string `"null"`. Rejected — requires special-case serialization; `interface{}` cleanly handles both string values and Go `nil` → JSON `null`.

### 3. Settings applied in array order per tick

On each watcher tick, iterate entries in declaration order and issue one PUT per entry. Because OpenSearch applies settings field-by-field, later entries naturally override earlier ones on overlapping indices.

**Alternative**: Merge all settings for an index upfront and issue one PUT per index. Rejected — requires fetching the index list, which adds an extra API round-trip and complexity; order-based override is simpler and sufficient for the stated use case.

### 4. System-index exclusion via URL pattern only

Append `,-.*` to every pattern in the URL (matching `SlowLogIndicesWatcher`). No additional Go-side filtering.

The existing list of non-dot system index exclusions (used in replication) is not needed here because `PUT /_settings` with `allow_no_indices=true` does not need an explicit list — unmatched patterns are silently ignored.

### 5. Watcher wired from opensearch_reconciler, not monitoring_reconciler

`IndexSettings` belongs to `opensearch.indexSettings` in the CR (under the `OpenSearch` field), so start/stop logic belongs in the OpenSearch reconciliation path, not the monitoring reconciler. A new `IndexSettingsWatcher` field is added to `OpenSearchServiceReconciler`.

## Risks / Trade-offs

- **Settings drift between ticks**: A setting changed directly in OpenSearch will be overwritten at the next watcher tick (up to 60s later). This is intentional for declarative management but operators must be aware.
- **`null` reset semantics**: Users removing a key from config will NOT see the setting reset. This is documented but may surprise users — the null-reset pattern must be clearly explained in docs.
- **Large index count**: If a cluster has thousands of indices, each PUT with a wildcard pattern is handled by OpenSearch server-side, so the operator issues only one PUT per entry regardless of index count. No scalability concern.
- **OpenSearch 404 on empty pattern**: `allow_no_indices=true` prevents errors when no indices match the pattern.

## Migration Plan

- No data migration required; the feature is additive.
- Deploying the new operator version without setting `opensearch.indexSettings` is a no-op.
- To enable: add `opensearch.indexSettings` entries to the Helm values; the reconciler starts the watcher on the next reconcile cycle.
- To roll back: remove `opensearch.indexSettings`; the watcher stops on the next reconcile. Previously applied settings remain in OpenSearch until reset manually or via null values.
