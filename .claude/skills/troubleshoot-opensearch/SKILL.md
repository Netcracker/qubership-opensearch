---
name: troubleshoot-opensearch
description: Diagnose qubership-opensearch failures in incidents, Jira tickets, CI/CD pipelines, and project support. Use when an OpenSearch cluster or DBaaS adapter is degraded or down, pods are OOMKilled, in CrashLoopBackOff or not ready, indices are read-only or the disk is full, requests fail with CircuitBreakingException, 401/403 or TLS errors, shards or translogs are corrupted, master election or replication fails, or an upgrade or migration job fails.
---

# Troubleshoot OpenSearch

All facts and resolution steps live in `references/`; this file only routes. Use only the bundled references.

- `references/troubleshooting.md` — known failure modes, one `##` section each. Never read it in full.
- `references/scenarios/<name>.md` — detailed procedures linked from a section's Recommendations.
- `references/alerts.md` — Prometheus alerts (`./alerts.md#...` links).
- `references/hwe.md` — hardware requirements.
- `references/indices-migration.md`, `references/opensearch-tls-reinit.yaml` — linked from the migration and TLS sections.

Local links resolve inside `references/`. GitHub links point to repository docs that are not bundled; mention them to the user instead of fetching them.

Steps:

1. Match the symptom in the table below. Tickets rarely quote alert names: match on pod state, error text, or user-visible behavior first. If the table looks stale, list headers with `rg -n '^## ' references/troubleshooting.md`.
2. Read from the matched `##` header to the next `##` header (Description, Alerts, Stack trace, How to solve, Recommendations).
3. Open a scenario or other reference only when that section links to it.

## Failure scenarios

| Symptom in a ticket or pipeline | Signal in logs, API, or alerts | Section header |
| --- | --- | --- |
| Is the cluster healthy? health `yellow`/`red` | `_cluster/health`; `OpenSearchIsDownAlert` | `## Cluster Health` |
| Grafana status `DEGRADED`/`FAILED` | `OpenSearchIsDegradedAlert` | `## Cluster Status` |
| Monitoring shows `N/A`; OpenSearch unreachable | Monitoring agent down; `OpenSearchIsDownAlert` | `## Cluster Status is N/A` |
| Node left; unassigned shards after a pod restart | Shards not allocated; `OpenSearchIsDegradedAlert` | `## Cluster Status is Failed or Degraded` |
| Backup/restore or curator job failed | `restore.py ... non zero exit code`; `OpenSearchLastBackupHasFailedAlert` | `## Last Backup Has Failed` |
| Writes rejected; index read-only; PVC full | `cluster_block_exception`, `flood-stage watermark`, `read-only-allow-delete`; `OpenSearchDiskUsageAbove*Alert` | `## Data Nodes are Out of Space` |
| `OOMKilled`, high heap/CPU, long GC, slow queries, I/O wait | `OpenSearchHeapMemoryUsageAlert`, `OpenSearchCPULoadAlert`, `OpenSearchQueryIsTooSlowAlert` | `## Lack of Resources` |
| HTTP `429`; reads and writes fail | `CircuitBreakingException`, `Data too large` | `## OpenSearch Fails Down with CircuitBreakingException` |
| Primary shard cannot start after a crash | `CorruptIndexException`; primary `UNASSIGNED` | `## Data Files are Corrupted On Primary Shard` |
| `yellow` with a broken replica | Corruption only on replicas | `## Data Files are Corrupted On Replica Shard` |
| Whole index lost | All shard copies corrupted | `## Data Files are Corrupted On Entire Index` |
| Shard unassigned after unclean shutdown | `TranslogCorruptedException` | `## Translog Corrupted` |
| No master after several pods restarted | `master not discovered or elected yet`, `ClusterFormationFailureHelper` | `## New Master Cannot Be Elected` |
| Master pod crashed or rescheduled | Master gone; other nodes up | `## Elected Master Is Crashed` |
| Cross-cluster replication lags or stops | `OpenSearchReplication*Alert` | `## Problem During Replication` |
| Requests fail while a node restarts | Primary shard down during request | `## Primary Shard Is Down During User Request` |
| Nodes flap after a network incident | Connection loss on port `9300` | `## Network Connection Is Lost and Restored` |
| Pods in one AZ stopped; nodes `NotReady` | Several nodes down in one zone | `## Availability Zone Outage` |
| Planned zone maintenance; restart order | Planned stop/start | `## Availability Zone Shutdown and Startup` |
| Pod `Running` but `0/1`; `CrashLoopBackOff` | `Readiness probe failed` | `## Readiness Probe Failed` |
| Restart counter grows | `Liveness probe failed` | `## Liveness Probe Failed` |
| Pod exits at start on a new Kubernetes node | `vm.max_map_count [65530] is too low` | `## Max Virtual Memory Is Too Low` |
| Auxiliary pod `CreateContainerConfigError` | `container has runAsNonRoot and image will run as root` | `## Container Failed with Error: container has runAsNonRoot and image will run as root` |
| OpenShift 3.11 install fails on CRD | `openAPIV3Schema: Invalid value` | `## CRD Creation Failed on OpenShift 3.11` |
| Operator stuck after credential change | `Status code - [401]`, `opensearch-secret-old` | `## Operator Fails with Unauthorized Code on OpenSearch Readiness Check` |
| Every request `503` after restart | `Not yet initialized (you may need to run securityadmin)` | `## OpenSearch Does Not Start with "Not yet initialized" Error` |
| Nodes cannot join; TLS errors; cert expired | `SSLHandshakeException`, `PKIX path validation failed` | `## OpenSearch Starts Failing With TLS Certificate Error` |
| Microservice cannot log in; `500`/`401` | `Authorization to Opensearch has been failed` | `## OpenSearch Clients Fail with Authentication Error` |
| DBaaS adapter down; cannot create databases | Adapter `DOWN`/`WARNING`; `OpenSearchDBaaSIsDownAlert` | `## DBaaS Adapter.DBaaS Adapter Status Is Down/Warning` |
| DBaaS adapter up but reports a problem | Adapter `PROBLEM` | `## DBaaS Adapter.DBaaS Adapter Status Is Problem` |
| Standby unhealthy after DR switchover | DR health `DEGRADED` | `## OpenSearch Disaster Recovery Health Has Status "DEGRADED"` |
| One index never replicates to standby | `ResourceAlreadyExistsException: task with id {replication:index:...}` | `## ResourceAlreadyExistsException: task with ID {replication:index:test_index} already exist` |
| Standby missing data, no errors | Replication `PAUSED`/`AutoPaused` | `## Index Is Not Replicated To Standby Side Without Any Errors` |
| `403` for everything after password change | `no permissions for [cluster:monitor/health]` | `## No permissions after change password` |
| DBaaS database creation fails | `provided prefix already exists or a part of another prefix` | `## Database Cannot be Created Due to Prefix Intersection` |
| Upgrade stuck on migration; legacy indices | Migration job `BackoffLimitExceeded`; `OpenSearchLegacyIndicesDetectedAlert` | `## Indices migration issue` |
| Index creation/rollover rejected | `this action would add [N] total shards, but this cluster currently has [X]/[Y] maximum shards open` | `## Maximum Shards Open Limit Reached` |
| Helm/ArgoCD pre-upgrade or PreSync hook fails | `job opensearch-migration-1x failed`, `Hook: PreSync; Phase: Failed` | `## Upgrade Failed Due to Pre-Deploy Migration Hook` |

If several rows match, prefer the most specific startup, configuration, migration, TLS, or authorization failure, then resource and shard/replication rows, and generic health last.

If no row matches (for example, `Pending` or `Evicted`), say the reference does not cover it, offer `references/hwe.md` capacity as a hypothesis only, and ask for `kubectl describe pod` events.

## Safety

Commands in the references are recommendations for an authorized operator, not permission to run them. Before deleting indices, data, PVs, secrets, or certificates, changing credentials, or modifying cluster settings, get explicit user approval and confirm the namespace, active/standby side, affected data, and recovery path.

## Assessment

Report the section header, the evidence that selected it, the impact stated there, and only the resolution steps that section or its linked reference supports. State what the available evidence could not answer. Do not invent a root cause.