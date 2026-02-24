# Indices migration (1.x to 2.x)

This document describes the **indices migrator** application: why it exists, how it works, and how to use it when upgrading OpenSearch from 1.x/2.x to 3.x.

---

## Why we need it

When upgrading an OpenSearch cluster **from 2.x to 3.x**, indices that were **created on OpenSearch 1.x** can cause problems. OpenSearch stores an internal **index version** (“created” version) in each index’s metadata. Indices created on 1.x use an older encoding of this version. After a rolling upgrade to 3.x, such indices may not behave correctly or may hit compatibility limits.

The **indices migrator** runs **before** the 2.x → 3.x upgrade. It:

1. Finds all indices whose “created” version corresponds to **OpenSearch 1.x**.
2. **Reindexes** each of them in place so they get a **2.x-compatible** index version.
3. Optionally reinitializes the security index and restores DBaaS users.

That way, when you perform the rolling upgrade to 3.x, all indices already have a 2.x-style version and the upgrade is safe.

**When to use it:**  
Use the migrator when your cluster currently runs **OpenSearch 2.x** (or 1.x) and you plan to upgrade to **OpenSearch 3.x**, and you have (or might have) indices that were originally created on **1.x**.

---

## How it works

### High-level flow

1. **Discovery**  
   The tool connects to OpenSearch and:
   - Fetches the “created” version for every index.
   - Decodes the version (1.x uses a masked encoding) and keeps only indices with major version **1**.
   - Skips the security index (`.opendistro_security` / `.opensearch-security` / `.plugins-security`); it is handled separately (reinit).
   - Checks that the cluster has enough free disk (at least **2×** the size of the largest 1.x index) for reindexing.

2. **Backup (if there are indices to migrate)**  
   For all **standard** 1.x indices, it creates a snapshot backup. This is used only for **restore-on-failure**: if migration of a standard index fails, that index is restored from this backup and the run stops.

3. **Migration per index**  
   For each 1.x index:
   - **Write block** is set on the original index.
   - A **temporary index** (e.g. `myindex-migration`) is created with sanitized settings and the same mappings.
   - **Reindex** original → temporary (async task, waited to completion).
   - **Count check** (source vs temporary).
   - **Delete** the original index, **recreate** it with the same settings/mappings, then **reindex** temporary → original.
   - **Count check** again, **delete** the temporary index, **restore** original performance-related settings (e.g. replicas).

4. **Security and users**  
   - If the security index was detected as 1.x, the tool **reinitializes security** (disable in config → restart OpenSearch → delete security index → enable in config → restart again).
   - If the **DBaaS adapter** is configured, it triggers **user restore** (passwords) via the adapter API and waits until the restore state is “done”.
   - Finally it **restarts the OpenSearch operator** so the rest of the stack is in sync.

### System/special indices

Indices whose names start with a dot (e.g. `.opendistro_security`, `.opensearch-security`, `.plugins-security`) are treated as **special**:

- They are **not** included in the snapshot backup.
- **On migration failure:** the failed index is **deleted** and migration **continues** with the next index (no restore from backup).

### Standard indices

- They **are** included in the backup before migration.
- **On migration failure:** only that index is **restored from backup**, then the run **stops** (no further indices are migrated in that run).

### Dry run

The migrator supports a **dry-run** mode: it discovers 1.x indices and performs disk checks but **does not** create backups, reindex, or change security/operator. Use it to see what would be migrated.

```text
./migrator -dry-run
```

---

## How to use it

### Where it runs

The indices migrator is a **Go binary** (`migrator`) shipped inside the **curator** Docker image (`qubership-opensearch-curator`). The image is based on `qubership-backup-daemon` and adds the migrator under `ELASTICSEARCH_CURATOR_HOME` (e.g. `/opt/elasticsearch-curator`). It is intended to be run as part of your **upgrade / backup workflow** (e.g. from the backup daemon or a one-off job that runs before the 2.x → 3.x upgrade).

### Environment variables

The migrator reads the following environment variables (typical values come from the curator Helm deployment):

| Variable | Description | Example / default |
|----------|-------------|-------------------|
| `ES_HOST` | OpenSearch host:port | `opensearch-internal:9200` |
| `ES_USERNAME` | OpenSearch username | (from secret) |
| `ES_PASSWORD` | OpenSearch password | (from secret) |
| `TLS_HTTP_ENABLED` | Use HTTPS for OpenSearch | `false` |
| `SNAPSHOT_REPOSITORY_NAME` | Snapshot repository for backup/restore | `snapshots` |
| `OPENSEARCH_NAMESPACE` | Kubernetes namespace of OpenSearch | `default` |
| `OPENSEARCH_CONFIG_SECRET_NAME` | Secret containing `opensearch.yml` (for security reinit) | e.g. `opensearch-config` |
| `OPENSEARCH_STATEFULSET_NAMES` | OpenSearch StatefulSet names (for restart) | (comma-separated) |
| `OPENSEARCH_DEPLOYMENT_NAMES` | OpenSearch Deployment names (for restart) | (optional) |
| `OPENSEARCH_OPERATOR_DEPLOYMENT_NAME` | Operator Deployment to restart at the end | e.g. `opensearch-service-operator` |
| `OPENSEARCH_CLIENT_SERVICE_NAME` | OpenSearch client Service (for security reinit) | e.g. `opensearch` |
| `DBAAS_ADAPTER_ADDRESS` | DBaaS adapter base URL (for user restore) | (optional) |
| `DBAAS_ADAPTER_USERNAME` | DBaaS adapter auth | (optional) |
| `DBAAS_ADAPTER_PASSWORD` | DBaaS adapter auth | (optional) |

The pod must have **kubectl** and **RBAC** to:

- Read the OpenSearch config secret.
- Restart OpenSearch StatefulSets/Deployments and the operator Deployment.
- Patch the OpenSearch client Service (for security reinit).

### Running the binary

- **With dry-run (no changes):**  
  `./migrator -dry-run`

- **Full run:**  
  `./migrator`

On failure the process exits with a non-zero code (e.g. 2).

## Installation

The indices migrator is **included in the curator Docker image**. You do not install it separately.

### Prerequisites

1. **OpenSearch Service Helm chart** with the **curator** component enabled.
2. **Curator image** that contains the migrator binary (e.g. `ghcr.io/netcracker/qubership-opensearch-curator` with a tag that includes the migrator build).
3. **Snapshot repository** registered in OpenSearch (used for backup before migration). Set `curator.snapshotRepositoryName` (default: `snapshots`) and ensure the repository is created and reachable.

### Enabling the curator (Helm)

In your Helm values for OpenSearch Service:

```yaml
curator:
  enabled: true
  snapshotRepositoryName: snapshots   # must match your registered snapshot repo
  # ... other curator options
```

When `curator.enabled` is `true`, the curator Deployment is created with the correct environment variables and RBAC. The migrator binary is present in the container at `$ELASTICSEARCH_CURATOR_HOME/migrator` (e.g. `/opt/elasticsearch-curator/migrator`).

### RBAC requirements

The curator pod (and any job that runs the migrator) needs:

- **Read** the OpenSearch config Secret (`OPENSEARCH_CONFIG_SECRET_NAME`).
- **Restart** OpenSearch StatefulSets and Deployments (for security reinit).
- **Patch** the OpenSearch client Service (selector) for security reinit.
- **Restart** the OpenSearch operator Deployment.

These are already granted by the standard curator Role/RoleBinding when curator is installed via the chart.

### Running the migrator manually

To run the migrator from the curator pod:

```bash
kubectl exec -it -n <namespace> <curator-pod-name> -- /opt/elasticsearch-curator/migrator -dry-run   # no changes
kubectl exec -it -n <namespace> <curator-pod-name> -- /opt/elasticsearch-curator/migrator            # full run
```

Or run a one-off Job that uses the same curator image and the same env/RBAC as the curator Deployment.

---

## Migration guide (2.x → 3.x with 1.x indices)

Follow this sequence when upgrading to OpenSearch 3.x and you have (or may have) indices created on OpenSearch 1.x.

### 1. Prerequisites

- Cluster is on **OpenSearch 2.x** (recommended: 2.19 or later for 3.x upgrade).
- **Curator is installed** and the image includes the migrator (see [Installation](#installation)).
- **Snapshot repository** is registered and working (needed for backup of standard indices).
- **Enough free disk** on the cluster: at least **2×** the size of the largest 1.x index (the tool checks this and fails early if not met).
- **Maintenance window**: the migrator can run for a long time (overall timeout 180 minutes); reindexing is I/O-heavy.

### 2. Dry run

Run the migrator in dry-run mode to see which indices would be migrated and to verify connectivity and disk:

```bash
kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migrator -dry-run
```

Check logs for:

- `Indices need migration: [...] (count=N)` — list of 1.x indices.
- `Security needs reinit: true/false`.
- Any error (e.g. OpenSearch client creation, disk precheck). Fix those before a real run.

If **"Nothing to migrate"** or count is 0 and you have no 1.x indices, you can skip the migrator and proceed with the normal 2.x → 3.x upgrade.

### 3. Run the migration

1. **Schedule or announce** a maintenance window; avoid heavy writes to OpenSearch during migration.
2. **Run the migrator** (no `-dry-run`):
   ```bash
   kubectl exec -it -n <namespace> <curator-pod> -- /opt/elasticsearch-curator/migrator
   ```
3. **Monitor** the logs. The tool will:
   - Create a snapshot of standard 1.x indices.
   - Migrate each index (reindex → swap → restore settings).
   - If the security index was 1.x: reinit security (disable → restart → delete security index → enable → restart).
   - If DBaaS adapter is configured: trigger user restore and wait (up to 240 seconds).
   - Restart the OpenSearch operator.
4. **Confirm** exit code 0 and log line `All done. Step3 security later.`

### 4. After migration

- Verify cluster health and that indices are searchable.
- Proceed with the **OpenSearch 2.x → 3.x** upgrade (rolling upgrade, CRD upgrade, etc.) as described in the [Migration to OpenSearch 3.x](installation.md#migration-to-opensearch-3x-opensearch-service-2xx) section.

### 5. If migration fails mid-way

- **Standard index failed:** The tool restores that index from the snapshot it created and exits. Fix the underlying issue (e.g. disk, mapping, or transient error), then run the migrator again; it will create a new backup and retry from the beginning (already-migrated indices will be skipped because they are no longer 1.x).
- **System/special index failed:** That index is deleted and migration continues. If it was a system index you need, you may have to recreate or restore it manually.
- **Security reinit or user restore failed:** Check [Troubleshooting](#troubleshooting) and fix before re-running.

---

## Troubleshooting

### OpenSearch client creation failed / OpenSearch client is nil

- **Cause:** Cannot connect to OpenSearch (wrong host, TLS, or credentials).
- **Checks:**
  - `ES_HOST` is correct and reachable from the pod (e.g. `opensearch-internal:9200`).
  - `ES_USERNAME` / `ES_PASSWORD` match an OpenSearch user that can read indices and run reindex/snapshots.
  - If `TLS_HTTP_ENABLED=true`, TLS is configured correctly (certs, `ROOT_CA_CERTIFICATE` if used).

### OpenSearch 1.x index migration preparation failed / Disk precheck FAILED

- **Cause:** Discovery or disk check failed. The tool requires **min node available ≥ 2 × (size of largest 1.x index)**.
- **Checks:**
  - Free disk on the data nodes. Increase disk or free space, or reduce data so the largest 1.x index is smaller.
  - OpenSearch `_cat/indices` and `_nodes/stats/fs` are reachable with the same credentials.

### Backup failed (CollectAndWaitBackup) / Backup status FAIL

- **Cause:** Snapshot creation or completion failed (repository misconfigured, permissions, or storage).
- **Checks:**
  - Snapshot repository is registered and in type `SUCCESS`.
  - `SNAPSHOT_REPOSITORY_NAME` matches the repository name.
  - Curator/OpenSearch has permission to create snapshots; repository path or S3 bucket is writable.
  - Inspect OpenSearch snapshot API and repository logs.

### Step2 failed / count mismatch after first reindex / after second reindex

- **Cause:** Reindex did not preserve document count, or a step (create index, reindex, delete) failed.
- **Checks:**
  - No concurrent writes to the index during migration (write block is set, but check for other writers).
  - Mapping/settings are compatible (no fields that break reindex on 2.x).
  - OpenSearch logs for reindex or index errors. If the same index fails again, consider restoring from the snapshot (the tool does this for standard indices) and investigating the index contents or mapping.

### Security reinitialization failed

- **Cause:** Disabling/enabling security in the config secret, or restarting OpenSearch, or cluster not turning green within the expected time (e.g. 1200 seconds).
- **Checks:**
  - `OPENSEARCH_CONFIG_SECRET_NAME` exists and contains valid `opensearch.yml`.
  - `OPENSEARCH_STATEFULSET_NAMES` / `OPENSEARCH_DEPLOYMENT_NAMES` are correct; kubectl can restart them.
  - `OPENSEARCH_CLIENT_SERVICE_NAME` is correct; the patch (add/remove selector) is allowed by RBAC.
  - Cluster reaches green after each restart; check OpenSearch cluster health and node logs.

### User recovery failed / Timeout reached during user restoration

- **Cause:** DBaaS adapter user restore did not reach state "done" within 240 seconds, or returned "failed".
- **Checks:**
  - `DBAAS_ADAPTER_ADDRESS` is reachable from the curator pod.
  - `DBAAS_ADAPTER_USERNAME` / `DBAAS_ADAPTER_PASSWORD` are correct.
  - Adapter API `/api/v2/dbaas/adapter/opensearch/users/restore-password` and state endpoint respond; check adapter logs.
  - If you do not use DBaaS, leave adapter env vars unset; the migrator skips user restore.

### Operator restart failed

- **Cause:** `kubectl rollout restart deployment/<OPENSEARCH_OPERATOR_DEPLOYMENT_NAME>` failed (RBAC or wrong name).
- **Checks:**
  - `OPENSEARCH_OPERATOR_DEPLOYMENT_NAME` is set and matches the operator Deployment name in the same namespace.
  - Curator has RBAC to restart that Deployment. If you do not want to restart the operator, leave this env var unset; the migrator will skip the step.

### kubectl / RBAC errors

- **Cause:** The pod cannot read the config secret, patch the Service, or restart StatefulSets/Deployments.
- **Checks:**
  - Curator ServiceAccount, Role, and RoleBinding are present and grant the required verbs (get/patch on secrets, patch on services, restart on statefulsets/deployments).
  - Namespace in `OPENSEARCH_NAMESPACE` (and the one used by kubectl) is correct.

### Migration runs too long or times out

- **Cause:** Overall context timeout is 180 minutes. Large indices or slow storage can make a single reindex take a long time.
- **Mitigation:** Run during low load; ensure enough disk and I/O; if needed, migrate in multiple runs (after fixing any failure, re-run; already-migrated indices are skipped).

---

## Summary

| Topic | Description |
|--------|-------------|
| **Purpose** | Reindex OpenSearch 1.x indices to a 2.x-compatible “created” version before upgrading the cluster to 3.x. |
| **When** | Before performing the OpenSearch 2.x → 3.x upgrade, as part of your upgrade/curator workflow. |
| **How** | Discovers 1.x indices, takes a snapshot of standard ones, reindexes each in place (with temp index and count checks), then optional security reinit and DBaaS user restore. |
| **Special indices** | System/special indices are not backed up; on failure they are deleted and migration continues. |
| **Failure handling** | Standard index: restore that index from backup and stop. System/special: delete and continue. |
| **Usage** | Run `./migrator` (or `./migrator -dry-run`) from the curator image or from a job that has the same env and kubectl access. |
