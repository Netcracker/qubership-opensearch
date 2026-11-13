---
name: troubleshoot-opensearch
description: Diagnose and resolve OpenSearch failures in clusters managed by the `opensearch-service` operator. Use when a cluster is degraded or down, pods are OOMKilled, in CrashLoopBackOff or not ready, indices are read-only or the disk is full, shards are unassigned or corrupted, requests fail with CircuitBreakingException, 401/403 or TLS errors, master election or cross-cluster replication fails, a backup or migration fails, or the issue touches master/data/arbiter/client roles, JVM heap, index buffers, thread pools, or any `opensearch.yml` setting.
---

# Troubleshoot OpenSearch

Use this skill when investigating issues related to:

- **opensearch-service operator** (reconciliation of the `OpenSearchService` custom resource, credential and security configuration)
- **OpenSearch cluster** health (`yellow`/`red`, `DEGRADED`/`FAILED`/`N/A` status, unassigned shards)
- **Node roles** (master, data, arbiter, client; master election, node leaving the cluster)
- **Storage** (disk watermarks, read-only indices, full PVCs, disk failures)
- **Resources** (JVM heap, OOMKilled pods, GC pauses, CPU and I/O saturation, circuit breakers, thread-pool rejections)
- **Data integrity** (corrupted primary/replica shards, entire index corruption, translog corruption)
- **Pod startup** (readiness/liveness probes, `vm.max_map_count`, `runAsNonRoot`, CRD creation)
- **Security** (TLS certificates, `securityadmin` initialization, 401/403 after a password change)
- **OpenSearch Curator** (snapshot backup/restore, indices cleaner)
- **Disaster recovery** (cross-cluster replication, switchover, standby health)
- **Upgrades** (indices migration, pre-deploy migration hooks, shard limits)

Failure-mode facts and resolution steps live in `references/`; this file provides routing and deployment-configuration guardrails. Use only the bundled references.

## Component Overview

### Operator architecture

This skill applies to any OpenSearch cluster managed by the `opensearch-service` operator deployed from `operator/charts/helm/opensearch-service`. The operator (`<fullname>-service-operator` deployment) reconciles the `OpenSearchService` custom resource (`opensearchservices`), maintains security configuration, and orchestrates disaster-recovery switchover and failover. The operator owns reconciliation; engine-level changes outside Helm values are reverted on the next reconciliation.

Key components:

- **OpenSearch** — `master` StatefulSet; optional `data` StatefulSet, `arbiter` StatefulSet, and `client` Deployment. When `opensearch.data.dedicatedPod.enabled` or `opensearch.client.dedicatedPod.enabled` is `false` (default), master pods also hold data or serve client requests.
- **OpenSearch Curator** (`<fullname>-curator`) — snapshot backup/restore through a REST API, scheduled backups, indices cleaner.
- **OpenSearch Monitoring** (`<fullname>-monitoring`) — Telegraf-based metrics, Grafana dashboards, Prometheus alerts.
- **DBaaS Adapter** — integration with the DBaaS Aggregator. Adapter-side issues belong to the separate DBaaS skill; use this skill only for OpenSearch-side symptoms.
- **OpenSearch Dashboards**, **Pod Scheduler**, **Status Provisioner**, pre-install TLS init job, and pre-deploy migration jobs.

### Critical configuration paths

Helm chart: `operator/charts/helm/opensearch-service/`.

| Helm value | Purpose |
| --- | --- |
| `opensearch.config` | `opensearch.yml` settings as flat dotted keys |
| `opensearch.master.javaOpts`, `opensearch.data.javaOpts`, `opensearch.arbiter.javaOpts`, `opensearch.client.javaOpts` | JVM heap and GC per role |
| `opensearch.data.dedicatedPod.enabled`, `opensearch.client.dedicatedPod.enabled` | Separated role topology |
| `opensearch.<role>.resources`; `opensearch.<role>.persistence` (master, data, arbiter) | Pod resources and storage |
| `opensearch.snapshots.*` | Snapshot repository (PV or S3/GCS) |
| `curator.*` | Backup schedule, eviction policy, indices cleaner |
| `global.tls.*`, `opensearch.tls.*` | TLS for REST and transport layers |
| `global.disasterRecovery.*` | DR mode and cross-cluster replication |
| `monitoring.*` | Metrics, dashboards, alerts |

#### `opensearch.yml` settings

Set `opensearch.yml` settings as flat dotted keys under `opensearch.config`. Keep dotted keys flat; do not nest their path segments:

```yaml
opensearch:
  config:
    "indices.memory.index_buffer_size": "30%"
    "search.max_buckets": 100000
    "thread_pool.write.queue_size": 1000
```

#### JVM options

JVM heap and GC options are separate from `opensearch.config` and use the `javaOpts` value for the deployed role:

```yaml
opensearch:
  master:
    javaOpts: "-Xms8192m -Xmx8192m"
  data:
    javaOpts: "-Xms16384m -Xmx16384m"
  arbiter:
    javaOpts: "-Xms1024m -Xmx1024m"
  client:
    javaOpts: "-Xms1024m -Xmx1024m"
```

The chart defines `master`, `data`, `arbiter`, and `client` Java option paths; do not assume a `coordinator` path exists. A heap setting on the wrong role is silently inert: when dedicated data or client pods are disabled, `data.javaOpts` or `client.javaOpts` does not tune the master pods that actually do that work.

### Forbidden actions

Never recommend these without explicit user approval and ticket evidence:

- `PUT /_cluster/settings` (transient or persistent) for cluster-wide settings owned by the operator; reconciliation overwrites them.
- `PUT /_all/_settings` against operator-managed indices; the operator may revert the changes.
- Manual `opensearch.yml` edits on a pod; the operator regenerates the file.
- `DELETE /<index>` as a routine fix. Index deletion must be backed by evidence that the data is deletable.
- Shard rerouting as a configuration fix; it is an operational, evidence-bound action.
- Deleting PVCs, PVs, secrets, or certificates, or changing credentials, outside the documented procedure.

## Common Failure Modes

### Shard allocation stuck or cluster red

Symptom: health `red` or `yellow` for an extended period, `unassigned shards > 0`, `cluster.routing.allocation.*` reported as the issue.

The root cause is usually operational (disk watermark, node failure, allocation-rule mismatch), not configuration. Check disk usage in the Grafana dashboard and node availability before recommending allocation tuning.

`cluster.routing.allocation.disk.watermark.low`, `.high`, and `.flood_stage` can be set through `opensearch.config`, but raising a watermark on a nearly full disk only delays the problem. Recommend storage expansion through the documented operator/platform procedure; do not assume PVC expansion is supported without a documented guide.

### Heap pressure or OOM on data nodes

Symptom: `out of memory` errors, GC pauses longer than 1 s, OOM-killed pods.

Tune the Java options of the pods that actually hold data: `opensearch.data.javaOpts` only when dedicated data pods are enabled, otherwise `opensearch.master.javaOpts`. Keep `Xms` and `Xmx` equal. Do not exceed 50% of the pod memory limit or 31 GB of heap (above that, compressed object pointers are lost).

### Index buffer or thread-pool saturation

Symptom: rising indexing latency, growing write-thread-pool rejection counters.

Consider `indices.memory.index_buffer_size` and `thread_pool.write.queue_size` under `opensearch.config`. Increasing the queue masks back-pressure; increasing the buffer trades heap for indexing throughput.

## Using the Troubleshooting Guide and Scenarios

Never read `references/troubleshooting.md` in full; load only the section that matches the symptom.

- [references/troubleshooting.md](./references/troubleshooting.md) — known failure modes, one `##` section each. Never read it in full.
- [references/alerts.md](./references/alerts.md) — Prometheus alerts linked from troubleshooting sections.

Detailed scenarios (open only the one matching the symptom):

- [references/scenarios/availability_zone_outage.md](./references/scenarios/availability_zone_outage.md)
- [references/scenarios/availability_zone_shutdown.md](./references/scenarios/availability_zone_shutdown.md)
- [references/scenarios/cpu_overload.md](./references/scenarios/cpu_overload.md)
- [references/scenarios/data_files_corrupted_on_primary_and_replica_shards.md](./references/scenarios/data_files_corrupted_on_primary_and_replica_shards.md)
- [references/scenarios/data_files_corrupted_on_primary_shard.md](./references/scenarios/data_files_corrupted_on_primary_shard.md)
- [references/scenarios/data_files_corrupted_on_replica_shard.md](./references/scenarios/data_files_corrupted_on_replica_shard.md)
- [references/scenarios/disk_failure_on_all_nodes.md](./references/scenarios/disk_failure_on_all_nodes.md)
- [references/scenarios/disk_failure_on_one_node.md](./references/scenarios/disk_failure_on_one_node.md)
- [references/scenarios/disk_filled_on_all_nodes.md](./references/scenarios/disk_filled_on_all_nodes.md)
- [references/scenarios/elected_master_is_crashed.md](./references/scenarios/elected_master_is_crashed.md)
- [references/scenarios/entire_index_corrupted.md](./references/scenarios/entire_index_corrupted.md)
- [references/scenarios/io_limit.md](./references/scenarios/io_limit.md)
- [references/scenarios/memory_limit.md](./references/scenarios/memory_limit.md)
- [references/scenarios/network_connection_failure.md](./references/scenarios/network_connection_failure.md)
- [references/scenarios/new_master_can_not_be_elected.md](./references/scenarios/new_master_can_not_be_elected.md)
- [references/scenarios/primary_shard_is_down_during_user_request.md](./references/scenarios/primary_shard_is_down_during_user_request.md)
- [references/scenarios/problem_during_replication.md](./references/scenarios/problem_during_replication.md)
- [references/scenarios/translog_is_corrupted.md](./references/scenarios/translog_is_corrupted.md)

Local links resolve relative to the file containing them. GitHub links point to repository docs that are not bundled; mention them to the user instead of fetching them.

Steps:

1. Match the symptom in the table below. Tickets rarely quote alert names: match on pod state, error text, or user-visible behavior first. If the table looks stale, list headers with `grep -n '^## ' references/troubleshooting.md`.
2. Read from the matched `##` header to the next `##` header (Description, Alerts, Stack trace, How to solve, Recommendations). If the row has no section header, read its linked scenario directly.
3. Open the scenario linked in the matched row or its section's Recommendations. Do not read unrelated scenarios.

## Failure scenarios

| Symptom in a ticket or pipeline | Signal in logs, API, or alerts | Section header | Scenario |
| --- | --- | --- | --- |
| Is the cluster healthy? health `yellow`/`red` | `_cluster/health`; `OpenSearchIsDownAlert` | `## Cluster Health` | - |
| Grafana status `DEGRADED`/`FAILED` | `OpenSearchIsDegradedAlert` | `## Cluster Status` | - |
| Monitoring shows `N/A`; OpenSearch unreachable | Monitoring agent down; `OpenSearchIsDownAlert` | `## Cluster Status is N/A` | - |
| Node left; unassigned shards after a pod restart | Shards not allocated; `OpenSearchIsDegradedAlert` | `## Cluster Status is Failed or Degraded` | - |
| One node stops reporting disk usage; shards missing or empty | Missing `disk.avail`, truncated disk graph, spikes in relocating shards | `## Cluster Status is Failed or Degraded` | [Disk failure on one node](./references/scenarios/disk_failure_on_one_node.md) |
| Disks fail across all nodes; cluster does not recover after disk repair | Health `red`/`failed`; unassigned shards across the cluster | `## Cluster Status is Failed or Degraded` | [Disk failure on all nodes](./references/scenarios/disk_failure_on_all_nodes.md) |
| Backup/restore or curator job failed | `restore.py ... non zero exit code`; `OpenSearchLastBackupHasFailedAlert` | `## Last Backup Has Failed` | - |
| Writes rejected; index read-only; PVC full; shards cannot allocate | `cluster_block_exception`, `flood-stage watermark`, `read-only-allow-delete`, low/high disk watermark exceeded; low `fs.total.available_in_bytes` or high `disk.percent` | `## Data Nodes are Out of Space` | [Disk filled on all nodes](./references/scenarios/disk_filled_on_all_nodes.md) |
| `OOMKilled`, high heap/CPU, long GC, slow queries, I/O wait | `OpenSearchHeapMemoryUsageAlert`, `OpenSearchCPULoadAlert`, `OpenSearchQueryIsTooSlowAlert` | `## Lack of Resources` | - |
| Sustained CPU growth during heavy searches or indexing | High `os.cpu.percent` and `os.cpu.load_average.*`; inspect `/_nodes/hot_threads` | `## Lack of Resources` | [CPU overload](./references/scenarios/cpu_overload.md) |
| Heap stays nearly full; long GC; sudden aggregate heap drop | `jvm.mem.heap_used_percent`, pools near 100%, growing `jvm.gc.collectors.*.collection_time_in_millis`; heap drops may indicate node departure or restart | `## Lack of Resources` | [Memory limit](./references/scenarios/memory_limit.md) |
| Searches or indexing slow under heavy disk I/O | Rates derived from `fs.io_stats.total.read_operations`, `write_operations`, `read_kilobytes`, `write_kilobytes` show read/write activity | `## Lack of Resources` | [I/O limit](./references/scenarios/io_limit.md) |
| HTTP `429`; reads and writes fail | `CircuitBreakingException`, `Data too large` | `## OpenSearch Fails Down with CircuitBreakingException` | - |
| Primary shard cannot start; updates fail with `504` without replicas; intermittent updates fail with replicas | `CorruptIndexException`, `ALLOCATION_FAILED`, primary `UNASSIGNED`; with replicas, `read past EOF: MMapIndexInput` can occur while shards remain `STARTED` | `## Data Files are Corrupted On Primary Shard` | [Primary shard corruption](./references/scenarios/data_files_corrupted_on_primary_shard.md) |
| Initial queries fail or return incomplete data; replicas relocate | Replica corruption may leave health unchanged until a read; shards relocate or are already relocated | `## Data Files are Corrupted On Replica Shard` | [Replica shard corruption](./references/scenarios/data_files_corrupted_on_replica_shard.md) |
| Red indices with both primary and corresponding replica copies corrupted | Matching primary/replica pairs unassigned; no healthy copy for affected shards | - | [Primary and replica corruption](./references/scenarios/data_files_corrupted_on_primary_and_replica_shards.md) |
| All queries to an index fail; all its shard copies corrupted | All index shards `UNASSIGNED`; primary `ALLOCATION_FAILED` with `CorruptIndexException`, replicas `PRIMARY_FAILED` | `## Data Files are Corrupted On Entire Index` | [Entire index corruption](./references/scenarios/entire_index_corrupted.md) |
| Shard recovery fails after unclean shutdown | `ALLOCATION_FAILED`, `RecoveryFailedException`, `TranslogCorruptedException` in `unassigned.details` | `## Translog Corrupted` | [Translog corruption](./references/scenarios/translog_is_corrupted.md) |
| No master after several pods restarted | `master not discovered or elected yet`, `ClusterFormationFailureHelper`; missing master-eligible nodes, possibly `nodes.count.master` reduced to zero | `## New Master Cannot Be Elected` | [New master cannot be elected](./references/scenarios/new_master_can_not_be_elected.md) |
| Master pod crashed or rescheduled | Master gone; other nodes communicating; reduced `nodes.count.total`/`nodes.count.master` | `## Elected Master Is Crashed` | [Elected master crashed](./references/scenarios/elected_master_is_crashed.md) |
| Replica writes fail within the cluster, even in a successful response; cluster stays red with stale copies | HTTP `200` with `_shards.failed` and `primary: false`, `index_failed_engine_exception`, `no_such_file_exception`; no in-sync copy available for promotion | `## Problem During Replication` | [Shard replication](./references/scenarios/problem_during_replication.md) |
| Cross-cluster replication lags or stops; standby unhealthy | `OpenSearchReplication*Alert`; inspect DR health and index replication status, not intra-cluster replica failures | `## OpenSearch Disaster Recovery Health Has Status "DEGRADED"` | - |
| Request reports failed shards while a node restarts, even with status `201` | `_shards.failed`, `node_disconnected_exception`; inspect shard failures rather than only the outer status | `## Primary Shard Is Down During User Request` | [Node down during request](./references/scenarios/primary_shard_is_down_during_user_request.md) |
| Nodes leave/rejoin after a network incident | Connection loss between nodes, health `yellow`/`red`, master election after communication loss | `## Network Connection Is Lost and Restored` | [Network connection failure](./references/scenarios/network_connection_failure.md) |
| Several nodes disappear in one AZ; pods stopped or nodes `NotReady` | Fewer nodes in `/_cat/nodes`; unassigned shards with `NODE_LEFT`; confirm zone outage with infrastructure evidence | `## Availability Zone Outage` | [Availability zone outage](./references/scenarios/availability_zone_outage.md) |
| Planned zone maintenance; health not green after restart | Planned stop/start; check running nodes, unassigned shards, and post-start cluster health | `## Availability Zone Shutdown and Startup` | [Availability zone shutdown/startup](./references/scenarios/availability_zone_shutdown.md) |
| Pod `Running` but `0/1`; `CrashLoopBackOff` | `Readiness probe failed` | `## Readiness Probe Failed` | - |
| Restart counter grows | `Liveness probe failed` | `## Liveness Probe Failed` | - |
| Pod exits at start on a new Kubernetes node | `vm.max_map_count [65530] is too low` | `## Max Virtual Memory Is Too Low` | - |
| Auxiliary pod `CreateContainerConfigError` | `container has runAsNonRoot and image will run as root` | `## Container Failed with Error: container has runAsNonRoot and image will run as root` | - |
| OpenShift 3.11 install fails on CRD | `openAPIV3Schema: Invalid value` | `## CRD Creation Failed on OpenShift 3.11` | - |
| Operator stuck after credential change | `Status code - [401]`, `opensearch-secret-old` | `## Operator Fails with Unauthorized Code on OpenSearch Readiness Check` | - |
| Every request `503` after restart | `Not yet initialized (you may need to run securityadmin)` | `## OpenSearch Does Not Start with "Not yet initialized" Error` | - |
| Nodes cannot join; TLS errors; cert expired | `SSLHandshakeException`, `PKIX path validation failed` | `## OpenSearch Starts Failing With TLS Certificate Error` | - |
| Microservice cannot log in; `500`/`401` | `Authorization to Opensearch has been failed` | `## OpenSearch Clients Fail with Authentication Error` | - |
| DBaaS adapter down; cannot create databases | Adapter `DOWN`/`WARNING`; `OpenSearchDBaaSIsDownAlert` | `## DBaaS Adapter.DBaaS Adapter Status Is Down/Warning` | - |
| DBaaS adapter up but reports a problem | Adapter `PROBLEM` | `## DBaaS Adapter.DBaaS Adapter Status Is Problem` | - |
| Standby unhealthy after DR switchover | DR health `DEGRADED` | `## OpenSearch Disaster Recovery Health Has Status "DEGRADED"` | - |
| One index never replicates to standby | `ResourceAlreadyExistsException: task with id {replication:index:...}` | `## ResourceAlreadyExistsException: task with ID {replication:index:test_index} already exist` | - |
| Standby missing data, no errors | Replication `PAUSED`/`AutoPaused` | `## Index Is Not Replicated To Standby Side Without Any Errors` | - |
| `403` for everything after password change | `no permissions for [cluster:monitor/health]` | `## No permissions after change password` | - |
| DBaaS database creation fails | `provided prefix already exists or a part of another prefix` | `## Database Cannot be Created Due to Prefix Intersection` | - |
| Upgrade stuck on migration; legacy indices | Migration job `BackoffLimitExceeded`; `OpenSearchLegacyIndicesDetectedAlert` | `## Indices migration issue` | - |
| Index creation/rollover rejected | `this action would add [N] total shards, but this cluster currently has [X]/[Y] maximum shards open` | `## Maximum Shards Open Limit Reached` | - |
| Helm/ArgoCD pre-upgrade or PreSync hook fails | `job opensearch-migration-1x failed`, `Hook: PreSync; Phase: Failed` | `## Upgrade Failed Due to Pre-Deploy Migration Hook` | - |

If several rows match, prefer the most specific startup, configuration, migration, TLS, or authorization failure, then resource and shard/replication rows, and generic health last.

If no row matches (for example, `Pending` or `Evicted`), say the reference does not cover it and ask for `kubectl describe pod` events. Do not infer a capacity problem without evidence.

## Log Analysis Tips

Use `<fullname>` from the release (default `opensearch`). Run read-only commands only.

### Operator logs

```bash
# Reconciliation, credential, and readiness-check errors
kubectl logs -n <namespace> deployment/<fullname>-service-operator | grep -iE "error|failed|unauthorized|401"

# Custom resource state
kubectl get opensearchservices -n <namespace> -o yaml
```

### OpenSearch pod logs

```bash
# Master election and cluster formation
kubectl logs -n <namespace> <opensearch-pod> | grep -E "ClusterFormationFailureHelper|master not discovered"

# Disk, memory, and circuit breakers
kubectl logs -n <namespace> <opensearch-pod> | grep -iE "flood-stage|watermark|OutOfMemoryError|CircuitBreakingException|Data too large"

# Shard and translog corruption
kubectl logs -n <namespace> <opensearch-pod> | grep -E "CorruptIndexException|TranslogCorruptedException|RecoveryFailedException"

# Startup, TLS, and security initialization
kubectl logs -n <namespace> <opensearch-pod> | grep -E "vm.max_map_count|SSLHandshakeException|PKIX path|Not yet initialized"
```

### Cluster API (read-only)

```bash
GET _cluster/health
GET _cat/nodes?v
GET _cat/shards?v&h=index,shard,prirep,state,unassigned.reason
GET _cluster/allocation/explain
GET _nodes/hot_threads
GET _plugins/_replication/<index>/_status
```

### Curator, monitoring, and migration logs

```bash
kubectl logs -n <namespace> deployment/<fullname>-curator | grep -iE "backup|restore|snapshot|error"
kubectl logs -n <namespace> deployment/<fullname>-monitoring | grep -iE "error|timeout"
kubectl logs -n <namespace> job/<fullname>-migration-1x
```

## Safety

Commands in the references are recommendations for an authorized operator, not permission to run them. Before deleting indices, data, PVs, secrets, or certificates, changing credentials, or modifying cluster settings, get explicit user approval and confirm the namespace, active/standby side, affected data, and recovery path.

## Assessment

Report the section header (or scenario name for a row without a section), the evidence that selected it, the impact stated there, and only the resolution steps that section or its linked reference supports. State what the available evidence could not answer. Do not invent a root cause.

### Recommendation discipline

- Phrase each `opensearch.yml` change as: “Set `<dotted-key>: <value>` under `opensearch.config` in the `opensearch-service` chart values and redeploy through the normal delivery channel.”
- Phrase each JVM heap change as: “Set `opensearch.<role>.javaOpts: "-Xms<N>m -Xmx<N>m"` in the `opensearch-service` chart values and redeploy.” Identify the actual deployed role and topology.
- Do not recommend index deletion or shard rerouting as a configuration fix; these actions are operational, evidence-bound, and may be destructive.

## When to Escalate

Escalate to component maintainers when:

- All copies of a shard or an entire index are corrupted and no usable snapshot exists.
- Translog recovery would require discarding operations (data loss).
- No master is elected after all master-eligible pods are running.
- The security index cannot be initialized, or `securityadmin` keeps failing.
- Cross-cluster replication cannot be resumed after following the DR section.
- The operator crashes or loops on reconciliation (attach the stack trace).
- Indices migration or a pre-deploy hook keeps failing after the documented fix.

Include in the escalation:

- Operator logs and logs of the affected OpenSearch pods.
- `kubectl get opensearchservices -n <namespace> -o yaml` and `kubectl get pods,pvc -n <namespace> -o wide`.
- Output of `_cluster/health`, `_cat/nodes`, `_cat/shards`, and `_cluster/allocation/explain`.
- Curator logs for backup/restore issues; replication status from both sides for DR issues.
