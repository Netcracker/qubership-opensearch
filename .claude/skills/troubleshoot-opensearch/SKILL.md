---
name: troubleshoot-opensearch
description: Diagnose qubership-opensearch failures in incidents, Jira tickets, CI/CD pipelines, and project support. Use when an OpenSearch cluster or DBaaS adapter is degraded or down, pods are OOMKilled, in CrashLoopBackOff or not ready, indices are read-only or the disk is full, requests fail with CircuitBreakingException, 401/403 or TLS errors, shards or translogs are corrupted, master election or replication fails, or an upgrade or migration job fails.
---

# Troubleshoot OpenSearch

All facts and resolution steps live in `references/`; this file only routes. Use only the bundled references.

- `references/troubleshooting.md` — known failure modes, one `##` section each. Never read it in full.
- `references/scenarios/<name>.md` — detailed procedures linked directly from the table below or a section's Recommendations.
- `references/alerts.md` — Prometheus alerts (`./alerts.md#...` links).

Local links resolve inside `references/`. GitHub links point to repository docs that are not bundled; mention them to the user instead of fetching them.

Steps:

1. Match the symptom in the table below. Tickets rarely quote alert names: match on pod state, error text, or user-visible behavior first. If the table looks stale, list headers with `rg -n '^## ' references/troubleshooting.md`.
2. Read from the matched `##` header to the next `##` header (Description, Alerts, Stack trace, How to solve, Recommendations). If the row has no section header, read its linked scenario directly.
3. Open the scenario linked in the matched row or its section's Recommendations. Do not read unrelated scenarios.

## Failure scenarios

| Symptom in a ticket or pipeline | Signal in logs, API, or alerts | Section header | Scenario |
| --- | --- | --- | --- |
| Is the cluster healthy? health `yellow`/`red` | `_cluster/health`; `OpenSearchIsDownAlert` | `## Cluster Health` | - |
| Grafana status `DEGRADED`/`FAILED` | `OpenSearchIsDegradedAlert` | `## Cluster Status` | - |
| Monitoring shows `N/A`; OpenSearch unreachable | Monitoring agent down; `OpenSearchIsDownAlert` | `## Cluster Status is N/A` | - |
| Node left; unassigned shards after a pod restart | Shards not allocated; `OpenSearchIsDegradedAlert` | `## Cluster Status is Failed or Degraded` | - |
| One node stops reporting disk usage; shards missing or empty | Missing `disk.avail`, truncated disk graph, spikes in relocating shards | `## Cluster Status is Failed or Degraded` | [Disk failure on one node](references/scenarios/disk_failure_on_one_node.md) |
| Disks fail across all nodes; cluster does not recover after disk repair | Health `red`/`failed`; unassigned shards across the cluster | `## Cluster Status is Failed or Degraded` | [Disk failure on all nodes](references/scenarios/disk_failure_on_all_nodes.md) |
| Backup/restore or curator job failed | `restore.py ... non zero exit code`; `OpenSearchLastBackupHasFailedAlert` | `## Last Backup Has Failed` | - |
| Writes rejected; index read-only; PVC full; shards cannot allocate | `cluster_block_exception`, `flood-stage watermark`, `read-only-allow-delete`, low/high disk watermark exceeded; low `fs.total.available_in_bytes` or high `disk.percent` | `## Data Nodes are Out of Space` | [Disk filled on all nodes](references/scenarios/disk_filled_on_all_nodes.md) |
| `OOMKilled`, high heap/CPU, long GC, slow queries, I/O wait | `OpenSearchHeapMemoryUsageAlert`, `OpenSearchCPULoadAlert`, `OpenSearchQueryIsTooSlowAlert` | `## Lack of Resources` | - |
| Sustained CPU growth during heavy searches or indexing | High `os.cpu.percent` and `os.cpu.load_average.*`; inspect `/_nodes/hot_threads` | `## Lack of Resources` | [CPU overload](references/scenarios/cpu_overload.md) |
| Heap stays nearly full; long GC; sudden aggregate heap drop | `jvm.mem.heap_used_percent`, pools near 100%, growing `jvm.gc.collectors.*.collection_time_in_millis`; heap drops may indicate node departure or restart | `## Lack of Resources` | [Memory limit](references/scenarios/memory_limit.md) |
| Searches or indexing slow under heavy disk I/O | Rates derived from `fs.io_stats.total.read_operations`, `write_operations`, `read_kilobytes`, `write_kilobytes` show read/write activity | `## Lack of Resources` | [I/O limit](references/scenarios/io_limit.md) |
| HTTP `429`; reads and writes fail | `CircuitBreakingException`, `Data too large` | `## OpenSearch Fails Down with CircuitBreakingException` | - |
| Primary shard cannot start; updates fail with `504` without replicas; intermittent updates fail with replicas | `CorruptIndexException`, `ALLOCATION_FAILED`, primary `UNASSIGNED`; with replicas, `read past EOF: MMapIndexInput` can occur while shards remain `STARTED` | `## Data Files are Corrupted On Primary Shard` | [Primary shard corruption](references/scenarios/data_files_corrupted_on_primary_shard.md) |
| Initial queries fail or return incomplete data; replicas relocate | Replica corruption may leave health unchanged until a read; shards relocate or are already relocated | `## Data Files are Corrupted On Replica Shard` | [Replica shard corruption](references/scenarios/data_files_corrupted_on_replica_shard.md) |
| Red indices with both primary and corresponding replica copies corrupted | Matching primary/replica pairs unassigned; no healthy copy for affected shards | - | [Primary and replica corruption](references/scenarios/data_files_corrupted_on_primary_and_replica_shards.md) |
| All queries to an index fail; all its shard copies corrupted | All index shards `UNASSIGNED`; primary `ALLOCATION_FAILED` with `CorruptIndexException`, replicas `PRIMARY_FAILED` | `## Data Files are Corrupted On Entire Index` | [Entire index corruption](references/scenarios/entire_index_corrupted.md) |
| Shard recovery fails after unclean shutdown | `ALLOCATION_FAILED`, `RecoveryFailedException`, `TranslogCorruptedException` in `unassigned.details` | `## Translog Corrupted` | [Translog corruption](references/scenarios/translog_is_corrupted.md) |
| No master after several pods restarted | `master not discovered or elected yet`, `ClusterFormationFailureHelper`; missing master-eligible nodes, possibly `nodes.count.master` reduced to zero | `## New Master Cannot Be Elected` | [New master cannot be elected](references/scenarios/new_master_can_not_be_elected.md) |
| Master pod crashed or rescheduled | Master gone; other nodes communicating; reduced `nodes.count.total`/`nodes.count.master` | `## Elected Master Is Crashed` | [Elected master crashed](references/scenarios/elected_master_is_crashed.md) |
| Replica writes fail within the cluster, even in a successful response; cluster stays red with stale copies | HTTP `200` with `_shards.failed` and `primary: false`, `index_failed_engine_exception`, `no_such_file_exception`; no in-sync copy available for promotion | `## Problem During Replication` | [Shard replication](references/scenarios/problem_during_replication.md) |
| Cross-cluster replication lags or stops; standby unhealthy | `OpenSearchReplication*Alert`; inspect DR health and index replication status, not intra-cluster replica failures | `## OpenSearch Disaster Recovery Health Has Status "DEGRADED"` | - |
| Request reports failed shards while a node restarts, even with status `201` | `_shards.failed`, `node_disconnected_exception`; inspect shard failures rather than only the outer status | `## Primary Shard Is Down During User Request` | [Node down during request](references/scenarios/primary_shard_is_down_during_user_request.md) |
| Nodes leave/rejoin after a network incident | Connection loss between nodes, health `yellow`/`red`, master election after communication loss | `## Network Connection Is Lost and Restored` | [Network connection failure](references/scenarios/network_connection_failure.md) |
| Several nodes disappear in one AZ; pods stopped or nodes `NotReady` | Fewer nodes in `/_cat/nodes`; unassigned shards with `NODE_LEFT`; confirm zone outage with infrastructure evidence | `## Availability Zone Outage` | [Availability zone outage](references/scenarios/availability_zone_outage.md) |
| Planned zone maintenance; health not green after restart | Planned stop/start; check running nodes, unassigned shards, and post-start cluster health | `## Availability Zone Shutdown and Startup` | [Availability zone shutdown/startup](references/scenarios/availability_zone_shutdown.md) |
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

## Safety

Commands in the references are recommendations for an authorized operator, not permission to run them. Before deleting indices, data, PVs, secrets, or certificates, changing credentials, or modifying cluster settings, get explicit user approval and confirm the namespace, active/standby side, affected data, and recovery path.

## Assessment

Report the section header (or scenario name for a row without a section), the evidence that selected it, the impact stated there, and only the resolution steps that section or its linked reference supports. State what the available evidence could not answer. Do not invent a root cause.