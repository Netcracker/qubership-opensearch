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

- **Where:** The migrator is a binary in the **curator** Docker image (`qubership-opensearch-curator`), at
  `$ELASTICSEARCH_CURATOR_HOME/migrator` (e.g. `/opt/elasticsearch-curator/migrator`). **Included in the curator
  image only from Qubership OpenSearch 2.2.14.** Run it from the curator pod or from a one-off Job using the same
  image and env/RBAC as the curator.
- **How:**
  - Dry-run (no changes):  
    `kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migrator -dry-run`
  - Full run:  
    `kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migrator`  
  On failure the process exits with a non-zero code.

**Prerequisites:** Curator enabled in the OpenSearch Service Helm chart, snapshot repository registered (for backup of standard indices). 

---

## How to run it during upgrade

In your Helm values set **`migration.enabled: true`** so the hook runs the migration job:

   ```yaml
   migration:
     enabled: true
   ```

If `migration.enabled` is `false`, the hook runs in check-only mode and will fail when upgrading 2.x → 3.x with legacy 1.x indices present.

---

## Common issues (environment-specific and what can go wrong)

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
