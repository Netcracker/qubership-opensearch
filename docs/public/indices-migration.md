# Indices migration (1.x to 2.x for 3.x upgrade)

## What it does (impact and downtime)

- **Purpose:** Before upgrading OpenSearch **2.x → 3.x**, indices that were **created on 1.x** must be reindexed so their
  internal “created” version is 2.x-compatible. Otherwise they can misbehave or hit compatibility limits.
- **Actions:** Finds 1.x indices, backs up standard ones to a snapshot, reindexes each in place (write block →
  temp index → reindex → swap → restore settings), then optionally reinitializes the security index and
  restores DBaaS users, and restarts the OpenSearch operator.
- **Downtime / impact:** Plan a **maintenance window**. Reindexing is I/O-heavy and can run a long time (overall
  timeout 180 minutes). Avoid heavy writes during migration. Affected indices are write-blocked while migrated.

---

## Where and how to run it manually

- **Where:** The migration_tool is a binary in the **curator** Docker image (`qubership-opensearch-curator`), at
  `$ELASTICSEARCH_CURATOR_HOME/migration_tool` (e.g. `/opt/elasticsearch-curator/migration_tool`). **Included in the curator
  image only from Qubership OpenSearch 2.3.0.** Run it from the curator pod or from a one-off Job using the same
  image and env/RBAC as the curator.
- **How:**
  - Dry-run (no changes):  
    `kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migration_tool --dry-run`
  - Full run:  
    `kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migration_tool`
  - **`--skip-security`** — Skips security reinitialization and operator restart. Use this when the migration target is an **external OpenSearch** cluster.
  **This flag MUST be used with external OpenSearch**;
  Do not use it for in-cluster operator-managed clusters.  
    Example:  
    `kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migration_tool --skip-security`
  - **`--skip-backup`** — Skips snapshot backup before migration and restore on failure, no backup is taken.
  On migration failure only the temporary migration index is deleted (the original index is left as-is except scepial indices which have prefixes `.`).
    Example:  
    `kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migration_tool --skip-backup`  
  On failure the process exits with a non-zero code.

**Prerequisites:** Curator enabled in the OpenSearch Service Helm chart, snapshot repository registered (for backup of standard indices). 

---

## How to run it during upgrade

In your Helm values set **`migration.enabled: true`** so the hook runs the migration job. You can pass extra parameters via **`migration.args`**:

   ```yaml
   migration:
     enabled: true
     args: []   # optional: list of flags passed to the migration_tool
   ```

If `migration.enabled` is `false`, the hook runs in check-only mode (with `--dry-run`) and will fail when upgrading 2.x → 3.x with legacy 1.x indices present.

**Parameters you can set in `migration.args`:**

| Parameter         | Description |
|-------------------|-------------|
| `--dry-run`       | Run in check-only mode; no changes are applied. (When `migration.enabled` is `false`, the hook automatically uses this.) |
| `--skip-security` | Skip security reinitialization and operator restart. **MUST** be used when the migration target is external OpenSearch. |
| `--skip-backup`   | Skip snapshot backup before migration and restore on failure; **MUST** be used when the curator is disabled. |

Example for external OpenSearch (no in-cluster backup/restore or operator steps):

   ```yaml
   migration:
     enabled: true
     args:
       - "--dry-run"
       - "--skip-security"
       - "--skip-backup"
   ```

---

**Note:** - this is not recommended as may can take a lot of time.
**Note:** - Job only works during upgrade to **2.x → 3.x**, for other cases use manual migration.

## Common issues (environment-specific and what can go wrong)

- **Migration Job `opensearch-migration-1x` failed**  
  If the migration Job fails, check the Job pod logs for the exact error. For example:  
  `kubectl logs -n <namespace> job/<release-name>-migration-1x` (or the failing pod name). The logs contain the migration_tool output and point to which step failed.

- **OpenSearch client creation failed / client is nil**  
  TLS or credentials: `ES_USERNAME`/`ES_PASSWORD` must match a user that can read indices and run reindex/snapshots.
  If `TLS_HTTP_ENABLED=true`, TLS and `ROOT_CA_CERTIFICATE` must be correct. Host is set by the chart for
  in-cluster OpenSearch.

- **Disk precheck FAILED / preparation failed**  
  Cluster must have **min node available ≥ 2 × (size of largest 1.x index)**. Free disk on data nodes or reduce data.

- **Backup failed (CollectAndWaitBackup / status FAIL)**  
  Snapshot repository must be registered and in type `SUCCESS`; `SNAPSHOT_REPOSITORY_NAME` must match;
  curator/OpenSearch must have permission to create snapshots; repository path or S3 must be writable.

- **Count mismatch after reindex / Step2 failed**  
  No concurrent writes (write block is set; check for other writers). Mapping/settings must be compatible.
  Check OpenSearch logs for reindex errors. Standard indices are restored from snapshot on failure.

- **Indices with _source disabled (cannot be migrated)**  
  Migration fails at the start (dry-run or full run) if any 1.x index has `_source` disabled in its mapping.
  Reindex requires document bodies; when `_source` is disabled they were never stored and cannot be migrated.
  You need to go to the documentation of the application that owns this index and perform a reindex operation.

- **Security reinitialization failed**  
  Config secret must exist with valid `opensearch.yml`. `OPENSEARCH_STATEFULSET_NAMES` /
  `OPENSEARCH_DEPLOYMENT_NAMES` / `OPENSEARCH_CLIENT_SERVICE_NAME` must be correct; RBAC must allow
  restart and service patch. Cluster must reach green after restarts.

- **User recovery failed / timeout during user restoration**  
  DBaaS adapter must be reachable (`DBAAS_ADAPTER_ADDRESS`), credentials correct; restore API and state endpoint must respond within 240s. If you don’t use DBaaS, leave adapter env vars unset.

- **Operator restart failed**  
  `OPENSEARCH_OPERATOR_DEPLOYMENT_NAME` must match the operator Deployment; curator must have RBAC to restart that Deployment. To skip, leave the var unset.

- **kubectl / RBAC errors**  
  Migration ServiceAccount, Role, and RoleBinding must grant get/patch on secrets, patch on services, restart on statefulsets/deployments. `OPENSEARCH_NAMESPACE` must be correct.

- **Migration runs too long or times out (180 min)**  
  Large indices or slow storage. Run during low load; ensure disk and I/O; re-run after fixing failures (already-migrated indices are skipped).
