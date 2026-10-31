---
name: troubleshoot-opensearch
description: Diagnose qubership-opensearch failures in incidents, Jira tickets, CI/CD pipelines, and project support. Use when an OpenSearch cluster or DBaaS adapter is degraded or down, pods are OOMKilled, in CrashLoopBackOff or not ready, indices are read-only or the disk is full, requests fail with CircuitBreakingException, 401/403 or TLS errors, shards or translogs are corrupted, master election or replication fails, or an upgrade or migration job fails.
---

# Troubleshoot OpenSearch

Fast entrypoint for OpenSearch assessment. Failure details and resolution steps stay in the reference files; this file does not repeat them.

[references/troubleshooting.md](./references/troubleshooting.md) is the fact source for known failure modes. It is a copy of `docs/public/troubleshooting.md` in this repository. Do not read it in full for a single symptom.

Supporting references, copied from the same repository and linked from the troubleshooting sections:

- [references/scenarios/](./references/scenarios/) holds the detailed procedures that sections refer to as `scenarios/<name>.md`, for example [memory_limit.md](./references/scenarios/memory_limit.md), [cpu_overload.md](./references/scenarios/cpu_overload.md), [io_limit.md](./references/scenarios/io_limit.md), [disk_filled_on_all_nodes.md](./references/scenarios/disk_filled_on_all_nodes.md), [disk_failure_on_one_node.md](./references/scenarios/disk_failure_on_one_node.md), [disk_failure_on_all_nodes.md](./references/scenarios/disk_failure_on_all_nodes.md), [data_files_corrupted_on_primary_shard.md](./references/scenarios/data_files_corrupted_on_primary_shard.md), [data_files_corrupted_on_replica_shard.md](./references/scenarios/data_files_corrupted_on_replica_shard.md), [data_files_corrupted_on_primary_and_replica_shards.md](./references/scenarios/data_files_corrupted_on_primary_and_replica_shards.md), [entire_index_corrupted.md](./references/scenarios/entire_index_corrupted.md), [translog_is_corrupted.md](./references/scenarios/translog_is_corrupted.md), [new_master_can_not_be_elected.md](./references/scenarios/new_master_can_not_be_elected.md), [elected_master_is_crashed.md](./references/scenarios/elected_master_is_crashed.md), [problem_during_replication.md](./references/scenarios/problem_during_replication.md), [primary_shard_is_down_during_user_request.md](./references/scenarios/primary_shard_is_down_during_user_request.md), [network_connection_failure.md](./references/scenarios/network_connection_failure.md), [availability_zone_outage.md](./references/scenarios/availability_zone_outage.md), and [availability_zone_shutdown.md](./references/scenarios/availability_zone_shutdown.md).
- [references/alerts.md](./references/alerts.md) describes each Prometheus alert, its expression, and its meaning (`./alerts.md#...` links).
- [references/installation.md](./references/installation.md) holds deployment parameters and the `## HWE` hardware requirements (`./installation.md#hwe` links). It is large; read only the matched header.
- [references/indices-migration.md](./references/indices-migration.md) describes the index migration job and its dry-run mode.
- [references/opensearch-tls-reinit.yaml](./references/opensearch-tls-reinit.yaml) is the template used by the TLS certificate section (`/docs/data/opensearch-tls-reinit.yaml` link).

Links such as `/docs/public/scenarios/<name>.md` in the reference resolve to `references/scenarios/<name>.md`. The agent has no access to other repository files; use only the bundled references.

1. Find the section headers with line numbers: `rg -n '^#' references/troubleshooting.md`
2. Match the symptom against the jump table below, or the raw headers if the table is stale. Tickets rarely quote alert names: match on pod state, error text, or user-visible behavior first.
3. Read from the matched `##` header through the next `##` header. Include all child sections such as Description, Alerts, Stack trace, How to solve, and Recommendations.
4. Open a supporting reference only when the selected section links to it or the table lists it. In large files, grep the headers first and read only the matched section.

## Failure scenarios

Match one row, then read that section. The table is for navigation only; it does not contain resolution steps.

| Symptom in a ticket or pipeline | Signal in logs, API, or alerts | Section header | Supporting reference |
| --- | --- | --- | --- |
| Someone asks whether the cluster is healthy; health is `yellow` or `red` | `_cluster/health` API status; `OpenSearchIsDownAlert` | `## Cluster Health` | — |
| Grafana cluster status panel shows `DEGRADED` or `FAILED` | Custom status `UP`/`DEGRADED`/`FAILED`; `OpenSearchIsDegradedAlert` | `## Cluster Status` | — |
| Monitoring shows no data or `N/A`; OpenSearch unreachable | Status `N/A`; monitoring agent down; `OpenSearchIsDownAlert` | `## Cluster Status is N/A` | — |
| A node left the cluster; unassigned shards; health `red` or `yellow` after a pod restart | Primary or replica shards not allocated; `OpenSearchIsDegradedAlert` | `## Cluster Status is Failed or Degraded` | [disk_failure_on_one_node.md](./references/scenarios/disk_failure_on_one_node.md), [disk_failure_on_all_nodes.md](./references/scenarios/disk_failure_on_all_nodes.md) |
| Backup or restore did not complete; curator job failed | `Execution of ... restore.py ... non zero exit code`; `OpenSearchLastBackupHasFailedAlert` | `## Last Backup Has Failed` | — |
| Writes rejected; index became read-only; PVC or disk is full | `cluster_block_exception`, `flood-stage watermark`, `read-only-allow-delete`; `OpenSearchDiskUsageAbove75/85/95%Alert` | `## Data Nodes are Out of Space` | [disk_filled_on_all_nodes.md](./references/scenarios/disk_filled_on_all_nodes.md) |
| Pod `OOMKilled`, high heap or CPU, long GC pauses, slow searches or indexing, I/O wait | Memory, CPU, or disk pressure on Kubernetes or node metrics; `OpenSearchHeapMemoryUsageAlert`, `OpenSearchCPULoadAlert`, `OpenSearchQueryIsTooSlowAlert` | `## Lack of Resources` | [memory_limit.md](./references/scenarios/memory_limit.md), [cpu_overload.md](./references/scenarios/cpu_overload.md), [io_limit.md](./references/scenarios/io_limit.md) |
| Search or index requests fail with HTTP `429`; `GET` and `PUT` both fail | `CircuitBreakingException`, `Data too large` | `## OpenSearch Fails Down with CircuitBreakingException` | [memory_limit.md](./references/scenarios/memory_limit.md), [disk_filled_on_all_nodes.md](./references/scenarios/disk_filled_on_all_nodes.md) |
| Index data unavailable after a crash; primary shard cannot start | `CorruptIndexException`; primary shard `UNASSIGNED` | `## Data Files are Corrupted On Primary Shard` | [data_files_corrupted_on_primary_shard.md](./references/scenarios/data_files_corrupted_on_primary_shard.md) |
| Health `yellow` with a broken replica | Corruption reported only for replica shards | `## Data Files are Corrupted On Replica Shard` | [data_files_corrupted_on_replica_shard.md](./references/scenarios/data_files_corrupted_on_replica_shard.md) |
| Whole index lost; no usable copy of any shard | All copies of an index corrupted | `## Data Files are Corrupted On Entire Index` | [entire_index_corrupted.md](./references/scenarios/entire_index_corrupted.md), [data_files_corrupted_on_primary_and_replica_shards.md](./references/scenarios/data_files_corrupted_on_primary_and_replica_shards.md) |
| Shard stays unassigned after an unclean shutdown | `TranslogCorruptedException` | `## Translog Corrupted` | [translog_is_corrupted.md](./references/scenarios/translog_is_corrupted.md) |
| Cluster does not respond after several pods restarted; no master | `master not discovered or elected yet`, `ClusterFormationFailureHelper` | `## New Master Cannot Be Elected` | [new_master_can_not_be_elected.md](./references/scenarios/new_master_can_not_be_elected.md) |
| Master pod crashed or was rescheduled | Master node gone; other nodes stay available | `## Elected Master Is Crashed` | [elected_master_is_crashed.md](./references/scenarios/elected_master_is_crashed.md) |
| Cross-cluster replication lags or stops | Replication errors; `OpenSearchReplication*Alert` | `## Problem During Replication` | [problem_during_replication.md](./references/scenarios/problem_during_replication.md) |
| User requests fail while a node is restarting | Primary shard down during a request | `## Primary Shard Is Down During User Request` | [primary_shard_is_down_during_user_request.md](./references/scenarios/primary_shard_is_down_during_user_request.md) |
| Nodes flap after a network incident | Temporary connection loss between nodes on port `9300` | `## Network Connection Is Lost and Restored` | [network_connection_failure.md](./references/scenarios/network_connection_failure.md) |
| Pods in one zone stopped; Kubernetes nodes `NotReady` in one AZ | Several OpenSearch nodes stop in one zone | `## Availability Zone Outage` | [availability_zone_outage.md](./references/scenarios/availability_zone_outage.md) |
| Planned zone or cluster maintenance; restart order question | Nodes stop and start during planned maintenance | `## Availability Zone Shutdown and Startup` | [availability_zone_shutdown.md](./references/scenarios/availability_zone_shutdown.md) |
| Pod is `Running` but `0/1` ready; pod in `CrashLoopBackOff` | `Readiness probe failed`, `Back-off restarting failed container` | `## Readiness Probe Failed` | [installation.md](./references/installation.md) `## HWE` |
| Pod restarts repeatedly; restart counter grows | `Liveness probe failed`, `Back-off restarting failed container` | `## Liveness Probe Failed` | [installation.md](./references/installation.md) `## HWE` |
| Pod exits right after start on a new or reinstalled Kubernetes node | `bootstrap checks failed`, `vm.max_map_count [65530] is too low` | `## Max Virtual Memory Is Too Low` | — |
| Monitoring or auxiliary pod stuck in `CreateContainerConfigError` | `container has runAsNonRoot and image will run as root` | `## Container Failed with Error: container has runAsNonRoot and image will run as root` | — |
| Installation fails on OpenShift 3.11 while applying the CRD | `openAPIV3Schema: Invalid value` | `## CRD Creation Failed on OpenShift 3.11` | — |
| Operator never finishes reconcile after a credential change | `OpenSearch is not ready yet! Status code - [401]`, `opensearch-secret-old` | `## Operator Fails with Unauthorized Code on OpenSearch Readiness Check` | — |
| Every request returns `503`; security plugin not ready after restart | `Not yet initialized (you may need to run securityadmin)` | `## OpenSearch Does Not Start with "Not yet initialized" Error` | — |
| Nodes cannot join; clients get TLS errors; certificate expired | `SSLHandshakeException`, `PKIX path validation failed`, expired `transport-crt.pem` | `## OpenSearch Starts Failing With TLS Certificate Error` | [opensearch-tls-reinit.yaml](./references/opensearch-tls-reinit.yaml) |
| Microservice cannot log in to OpenSearch; HTTP `500` or `401` from the service | `Authorization to Opensearch has been failed. Check credentials` | `## OpenSearch Clients Fail with Authentication Error` | — |
| DBaaS cannot create or reach OpenSearch databases; adapter pod down | Adapter health `DOWN` or `WARNING`; `OpenSearchDBaaSIsDownAlert` | `## DBaaS Adapter.DBaaS Adapter Status Is Down/Warning` | — |
| DBaaS adapter is up but reports a problem | Adapter health `PROBLEM`; registration or OpenSearch access unhealthy | `## DBaaS Adapter.DBaaS Adapter Status Is Problem` | — |
| Standby site is unhealthy after DR switchover | DR health `DEGRADED`; unhealthy indices or failed replication | `## OpenSearch Disaster Recovery Health Has Status "DEGRADED"` | — |
| Replication of one index never starts on standby | `ResourceAlreadyExistsException: task with id {replication:index:...}` | `## ResourceAlreadyExistsException: task with ID {replication:index:test_index} already exist` | — |
| Data written on active site does not appear on standby; no errors | Replication status `PAUSED` or `AutoPaused` | `## Index Is Not Replicated To Standby Side Without Any Errors` | — |
| HTTP `403` for every request after a password change | `security_exception`, `no permissions for [cluster:monitor/health]` | `## No permissions after change password` | — |
| DBaaS database creation fails for a microservice | `provided prefix already exists or a part of another prefix` | `## Database Cannot be Created Due to Prefix Intersection` | — |
| Upgrade stuck or failed on a migration job; legacy indices remain | Migration job `BackoffLimitExceeded`; `OpenSearchLegacyIndicesDetectedAlert` | `## Indices migration issue` | [indices-migration.md](./references/indices-migration.md) |
| Index creation or rollover fails; new shards rejected | `this action would add [N] total shards, but this cluster currently has [X]/[Y] maximum shards open` | `## Maximum Shards Open Limit Reached` | — |
| Helm or ArgoCD upgrade fails on a pre-upgrade or PreSync hook | `job opensearch-migration-1x failed: BackoffLimitExceeded`, `Hook: PreSync; Phase: Failed` | `## Upgrade Failed Due to Pre-Deploy Migration Hook` | [indices-migration.md](./references/indices-migration.md) |

When several rows match, start with the most specific startup, configuration, migration, TLS, or authorization failure. Then inspect resource and shard/replication scenarios before relying on a generic cluster health status.

If no row matches (for example, a pod is `Pending` or `Evicted`), say that the troubleshooting reference does not cover the symptom. Check resource requests and node capacity against `## HWE` in [references/installation.md](./references/installation.md) only as a hypothesis, and ask for `kubectl describe pod` events.

## Safety

Treat commands in the reference as recommendations for an authorized cluster operator, not permission to execute them. Before deleting indices, data, or persistent volumes; removing secrets or certificates; changing credentials; or modifying cluster settings, obtain explicit user approval and confirm the target namespace, active/standby side, affected data, and available recovery path.

## Assessment

Report the section header, the evidence that selected it, the impact stated in that section, and only the resolution steps supported by that section or its supporting reference. State which checks the ticket, pipeline log, or cluster could not answer. Do not invent a root cause when the available evidence is insufficient.
