<!-- TOC -->
  * [Cluster Health](#cluster-health)
    * [Description](#description)
    * [Alerts](#alerts)
    * [Stack trace](#stack-trace)
    * [How to solve](#how-to-solve)
    * [Recommendations](#recommendations)
  * [Cluster Status](#cluster-status)
    * [Description](#description-1)
    * [Alerts](#alerts-1)
    * [Stack trace](#stack-trace-1)
    * [How to solve](#how-to-solve-1)
    * [Recommendations](#recommendations-1)
  * [Cluster Status is N/A](#cluster-status-is-na)
    * [Description](#description-2)
    * [Alerts](#alerts-2)
    * [Stack trace](#stack-trace-2)
    * [How to solve](#how-to-solve-2)
    * [Recommendations](#recommendations-2)
  * [Cluster Status is Failed or Degraded](#cluster-status-is-failed-or-degraded)
    * [Description](#description-3)
    * [Alerts](#alerts-3)
    * [Stack trace](#stack-trace-3)
    * [How to solve](#how-to-solve-3)
    * [Recommendations](#recommendations-3)
  * [Last Backup Has Failed](#last-backup-has-failed)
    * [Description](#description-4)
    * [Alerts](#alerts-4)
    * [Stack trace](#stack-trace-4)
    * [How to solve](#how-to-solve-4)
    * [Recommendations](#recommendations-4)
  * [Data Nodes are Out of Space](#data-nodes-are-out-of-space)
    * [Description](#description-5)
    * [Alerts](#alerts-5)
    * [Stack trace](#stack-trace-5)
    * [How to solve](#how-to-solve-5)
    * [Recommendations](#recommendations-5)
  * [Lack of Resources](#lack-of-resources)
    * [Description](#description-6)
    * [Alerts](#alerts-6)
    * [Stack trace](#stack-trace-6)
    * [How to solve](#how-to-solve-6)
    * [Recommendations](#recommendations-6)
  * [OpenSearch Fails Down with CircuitBreakingException](#opensearch-fails-down-with-circuitbreakingexception)
    * [Description](#description-7)
    * [Alerts](#alerts-7)
    * [Stack trace](#stack-trace-7)
    * [How to solve](#how-to-solve-7)
    * [Recommendations](#recommendations-7)
  * [Data Files are Corrupted On Primary Shard](#data-files-are-corrupted-on-primary-shard)
    * [Description](#description-8)
    * [Alerts](#alerts-8)
    * [Stack trace](#stack-trace-8)
    * [How to solve](#how-to-solve-8)
    * [Recommendations](#recommendations-8)
  * [Data Files are Corrupted On Replica Shard](#data-files-are-corrupted-on-replica-shard)
    * [Description](#description-9)
    * [Alerts](#alerts-9)
    * [Stack trace](#stack-trace-9)
    * [How to solve](#how-to-solve-9)
    * [Recommendations](#recommendations-9)
  * [Data Files are Corrupted On Entire Index](#data-files-are-corrupted-on-entire-index)
    * [Description](#description-10)
    * [Alerts](#alerts-10)
    * [Stack trace](#stack-trace-10)
    * [How to solve](#how-to-solve-10)
    * [Recommendations](#recommendations-10)
  * [Translog Corrupted](#translog-corrupted)
    * [Description](#description-11)
    * [Alerts](#alerts-11)
    * [Stack trace](#stack-trace-11)
    * [How to solve](#how-to-solve-11)
    * [Recommendations](#recommendations-11)
  * [New Master Cannot Be Elected](#new-master-cannot-be-elected)
    * [Description](#description-12)
    * [Alerts](#alerts-12)
    * [Stack trace](#stack-trace-12)
    * [How to solve](#how-to-solve-12)
    * [Recommendations](#recommendations-12)
  * [Elected Master Is Crashed](#elected-master-is-crashed)
    * [Description](#description-13)
    * [Alerts](#alerts-13)
    * [Stack trace](#stack-trace-13)
    * [How to solve](#how-to-solve-13)
    * [Recommendations](#recommendations-13)
  * [Problem During Replication](#problem-during-replication)
    * [Description](#description-14)
    * [Alerts](#alerts-14)
    * [Stack trace](#stack-trace-14)
    * [How to solve](#how-to-solve-14)
    * [Recommendations](#recommendations-14)
  * [Primary Shard Is Down During User Request](#primary-shard-is-down-during-user-request)
    * [Description](#description-15)
    * [Alerts](#alerts-15)
    * [Stack trace](#stack-trace-15)
    * [How to solve](#how-to-solve-15)
    * [Recommendations](#recommendations-15)
  * [Network Connection Is Lost and Restored](#network-connection-is-lost-and-restored)
    * [Description](#description-16)
    * [Alerts](#alerts-16)
    * [Stack trace](#stack-trace-16)
    * [How to solve](#how-to-solve-16)
    * [Recommendations](#recommendations-16)
  * [Availability Zone Outage](#availability-zone-outage)
    * [Description](#description-17)
    * [Alerts](#alerts-17)
    * [Stack](#stack-)
    * [How to solve](#how-to-solve-17)
    * [Recommendations](#recommendations-17)
  * [Availability Zone Shutdown and Startup](#availability-zone-shutdown-and-startup)
    * [Description](#description-18)
    * [Alerts](#alerts-18)
    * [Stack trace](#stack-trace-17)
    * [How to solve](#how-to-solve-18)
    * [Recommendations](#recommendations-18)
  * [Readiness Probe Failed](#readiness-probe-failed)
    * [Description](#description-19)
    * [Alerts](#alerts-19)
    * [Stack trace](#stack-trace-18)
    * [How to solve](#how-to-solve-19)
    * [Recommendations](#recommendations-19)
  * [Liveness Probe Failed](#liveness-probe-failed)
    * [Description](#description-20)
    * [Alerts](#alerts-20)
    * [Stack trace](#stack-trace-19)
    * [How to solve](#how-to-solve-20)
    * [Recommendations](#recommendations-20)
  * [Max Virtual Memory Is Too Low](#max-virtual-memory-is-too-low)
    * [Description](#description-21)
    * [Alerts](#alerts-21)
    * [Stack trace](#stack-trace-20)
    * [How to solve](#how-to-solve-21)
    * [Recommendations](#recommendations-21)
  * [Container Failed with Error: container has runAsNonRoot and image will run as root](#container-failed-with-error-container-has-runasnonroot-and-image-will-run-as-root)
    * [Description](#description-22)
    * [Alerts](#alerts-22)
    * [Stack trace](#stack-trace-21)
    * [How to solve](#how-to-solve-22)
    * [Recommendations](#recommendations-22)
  * [CRD Creation Failed on OpenShift 3.11](#crd-creation-failed-on-openshift-311)
    * [Description](#description-23)
    * [Alerts](#alerts-23)
    * [Stack trace](#stack-trace-22)
    * [How to solve](#how-to-solve-23)
    * [Recommendations](#recommendations-23)
  * [Operator Fails with Unauthorized Code on OpenSearch Readiness Check](#operator-fails-with-unauthorized-code-on-opensearch-readiness-check)
    * [Description](#description-24)
    * [Alerts](#alerts-24)
    * [Stack trace](#stack-trace-23)
    * [How to solve](#how-to-solve-24)
    * [Recommendations](#recommendations-24)
  * [OpenSearch Does Not Start with "Not yet initialized" Error](#opensearch-does-not-start-with-not-yet-initialized-error)
    * [Description](#description-25)
    * [Alerts](#alerts-25)
    * [Stack trace](#stack-trace-24)
    * [How to solve](#how-to-solve-25)
    * [Recommendations](#recommendations-25)
  * [OpenSearch Starts Failing With TLS Certificate Error](#opensearch-starts-failing-with-tls-certificate-error)
    * [Description](#description-26)
    * [Alerts](#alerts-26)
    * [Stack trace](#stack-trace-25)
    * [How to solve](#how-to-solve-26)
    * [Recommendations](#recommendations-26)
  * [OpenSearch Clients Fail with Authentication Error](#opensearch-clients-fail-with-authentication-error)
    * [Description](#description-27)
    * [Alerts](#alerts-27)
    * [Stack trace](#stack-trace-26)
    * [How to solve](#how-to-solve-27)
    * [Recommendations](#recommendations-27)
  * [DBaaS Adapter.DBaaS Adapter Status Is Down/Warning](#dbaas-adapterdbaas-adapter-status-is-downwarning)
    * [Description](#description-28)
    * [Alerts](#alerts-28)
    * [Stack trace](#stack-trace-27)
    * [How to solve](#how-to-solve-28)
    * [Recommendations](#recommendations-28)
  * [DBaaS Adapter.DBaaS Adapter Status Is Problem](#dbaas-adapterdbaas-adapter-status-is-problem)
    * [Description](#description-29)
    * [Alerts](#alerts-29)
    * [Stack trace](#stack-trace-28)
    * [How to solve](#how-to-solve-29)
    * [Recommendations](#recommendations-29)
  * [OpenSearch Disaster Recovery Health Has Status "DEGRADED"](#opensearch-disaster-recovery-health-has-status-degraded)
    * [Description](#description-30)
    * [Alerts](#alerts-30)
    * [Stack trace](#stack-trace-29)
    * [How to solve](#how-to-solve-30)
    * [Recommendations](#recommendations-30)
  * [ResourceAlreadyExistsException: task with id {replication:index:test_index} already exist](#resourcealreadyexistsexception-task-with-id-replicationindextestindex-already-exist)
    * [Description](#description-31)
    * [Alerts](#alerts-31)
    * [Stack trace](#stack-trace-30)
    * [How to solve](#how-to-solve-31)
    * [Recommendations](#recommendations-31)
  * [Index Is Not Replicated To Standby Side Without Any Errors](#index-is-not-replicated-to-standby-side-without-any-errors)
    * [Description](#description-32)
    * [Alerts](#alerts-32)
    * [Stack trace](#stack-trace-31)
    * [How to solve](#how-to-solve-32)
    * [Recommendations](#recommendations-32)
<!-- TOC -->

## Cluster Health

### Description

OpenSearch provides a default metric that indicates cluster state. It is called **cluster health**. To check the health of the cluster, the cluster health API can be used.

For more information on the cluster health API, refer to the official OpenSearch documentation, _Cluster Health_ [https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health](https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health).

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

Not applicable

### How to solve

Not applicable

### Recommendations

To check the health of the cluster, the following API can be used:

```sh
curl -XGET http://localhost:9200/_cluster/health
```

## Cluster Status

### Description

Cluster status is a custom metric that can be found on the **Cluster status** panel in Grafana. Possible values for cluster status are:

* `UP` - Cluster health status is GREEN and all nodes working.
* `DEGRADED` - Cluster has YELLOW health status or one node is failed.
* `FAILED` - Cluster has RED health status.

### Alerts

* [OpenSearchIsDegradedAlert](./alerts.md#opensearchisdegradedalert)

### Stack trace

Not applicable

### How to solve

Not applicable

### Recommendations

Not applicable

## Cluster Status is N/A

### Description

**N/A** status while monitoring indicates that the OpenSearch cluster is unreachable.

The main cause is any of the following:

* The cluster is down.
* The monitoring agent is not deployed.
* The monitoring agent is down.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

Not applicable

### How to solve

To resolve the issue, navigate to the OpenShift console and check the service state.

### Recommendations

In the simplest scenario, starting the service solves the issue. In the event of a permanent failure, try to redeploy the cluster or recover it from the backup.

## Cluster Status is Failed or Degraded

### Description

`Failed` status indicates that one or more primary shards is not allocated in the cluster. `Degraded` status means that one or more replica shards is missing.

This can happen when a node drops off the cluster for some reason. This could be due to disk failure, lack of memory or CPU, long garbage collection time, availability zone outage, and so on.

To check the health of the cluster, use the following API:

```sh
curl -XGET http://localhost:9200/_cluster/health
```

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)
* [OpenSearchIsDegradedAlert](./alerts.md#opensearchisdegradedalert)

### Stack trace

Not applicable

### How to solve

To identify the reason for the node failure, check the monitoring dashboard for any unusual changes that may have occurred around the same time the node failed.
When the problem is localized, go to the appropriate problem description and follow the relevant troubleshooting procedure to fix it.

After the node is fixed, its shards remain in an initializing state before they transition back to active status.
During this initialization period, the cluster state may change from `up` to `degraded` or `failed` until the shards on the recovering node regain active status.
In many cases, a brief status change to `degraded` or `failed` may not require any additional actions.

If it is a permanent failure, and the node cannot be recovered, new nodes can be added, after which OpenSearch recovers data from any available replica shards.
Replica shards can be promoted to primary shards and redistributed on the new nodes.

### Recommendations

For more information on OpenSearch clusters, refer to the official OpenSearch documentation [https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health].

The following scenarios are examples of permanent failures:

* [Disk failure on one node](scenarios/disk_failure_on_one_node.md)
* [Disk failure on all nodes](scenarios/disk_failure_on_all_nodes.md)

If both the primary and replica copy of a shard are lost, data can be recovered from backup.

## Last Backup Has Failed

### Description

The last OpenSearch backup has finished with `Failed` status.

### Alerts

* [OpenSearchLastBackupHasFailedAlert](./alerts.md#opensearchlastbackuphasfailedalert)

### Stack trace

```text
2024-05-09T08:47:56+0000 ERROR [Backup] [140038667422520] - Execution of '['python3', '/opt/elasticsearch-curator/restore.py', '/backup-storage/granular/20240509T084319', '-d', '["backup-test-opmabwrtuddiscy"]', '-clean', 'false']' was finished with non zero exit code: 1
```

### How to solve

Check that OpenSearch curator pod exists and is up. If OpenSearch curator is down, restart appropriate deployment. If curator pod is up, check its state by the following command from pod's terminal:

```bash
curl -XGET http://localhost:8080/health
```

### Recommendations

Not applicable

## Data Nodes are Out of Space

### Description

The disk being filled on all nodes

### Alerts

* [OpenSearchDiskUsageAbove75%Alert](./alerts.md#opensearchdiskusageabove75alert)
* [OpenSearchDiskUsageAbove85%Alert](./alerts.md#opensearchdiskusageabove85alert)
* [OpenSearchDiskUsageAbove95%Alert](./alerts.md#opensearchdiskusageabove95alert)

### Stack trace

```text
org.opensearch.OpenSearchStatusException: OpenSearch exception [type=cluster_block_exception, reason=index [test-index] blocked by: [TOO_MANY_REQUESTS/12/disk usage exceeded flood-stage watermark, index has read-only-allow-delete block];]
```

### How to solve

If all data nodes are running low on disk space, more data nodes should be added to the cluster.

### Recommendations

Be sure that all indices have enough primary shards to be able to balance their data across all those nodes.
However, if only some nodes are running out of disk space, this is usually a sign that an index was initialized with too few shards.
If an index is composed of a few very large shards, it is hard for OpenSearch to distribute these shards across nodes in a balanced manner.

For more information, refer to [Disk Filled on All Nodes](scenarios/disk_filled_on_all_nodes.md).

## Lack of Resources

### Description

Some problems with the OpenSearch cluster can occur due to a lack of CPU, memory, and disk resources.

### Alerts

* [OpenSearchDiskUsageAbove75%Alert](./alerts.md#opensearchdiskusageabove75alert)
* [OpenSearchDiskUsageAbove85%Alert](./alerts.md#opensearchdiskusageabove85alert)
* [OpenSearchDiskUsageAbove95%Alert](./alerts.md#opensearchdiskusageabove95alert)

### Stack trace

Not applicable

### How to solve

Increase OpenSearch resources.

### Recommendations

For more information, refer to [Memory Limit](/docs/public/scenarios/memory_limit.md), [CPU Overload](/docs/public/scenarios/cpu_overload.md)
and [I/O Limit](/docs/public/scenarios/io_limit.md).

## OpenSearch Fails Down with CircuitBreakingException

### Description

OpenSearch includes a special circuit breaker that is intended to prevent `OutOfMemoryException`.
The circuit breaker estimates the memory requirements of a query by inspecting the fields involved.
It then checks to see whether loading the field data required would push the total field data size over the configured percentage of the heap.
If the estimated query size is larger than the limit, then the circuit breaker is tripped, and the query will be aborted and return an exception.

Both `GET` and `PUT` requests are failed.

The main reasons for this failure are as follows:

* Query tries to load more data than memory is currently available.
* Index is bigger than available heap.
* Continuous shard relocation due to the data nodes being out of space.

### Alerts

Not applicable

### Stack trace

```text
org.elasticsearch.common.breaker.CircuitBreakingException: [parent] Data too large, data for [<http_request>] would be larger than limit of [1453142835/1.3gb]
```

### How to solve

For more information, refer to [Memory Limit](scenarios/memory_limit.md) and [Disk Filled on All Nodes](scenarios/disk_filled_on_all_nodes.md).

### Recommendations

For more information on OpenSearch, refer to the official OpenSearch documentation: [https://aws.amazon.com/premiumsupport/knowledge-center/opensearch-circuit-breaker-exception].

## Data Files are Corrupted On Primary Shard

### Description

The data files are corrupted on primary shard.
Index has no replica shards, get queries return incomplete data, update queries fail, and some primary shards are in unassigned status with `CorruptIndexException`.

### Alerts

Not applicable

### Stack trace

```sh
curl -XGET http://localhost:9200/_cat/shards/cats?v&h=index,shard,state,docs,store,node,unassigned.reason
```

Possible output:

```text
    index shard state      node                  unassigned.reason unassigned.details
    cats  1     STARTED    opensearch-0
    cats  2     STARTED    opensearch-0
    cats  0     UNASSIGNED                       ALLOCATION_FAILED shard failure, reason [search execution corruption failure], failure FetchPhaseExecutionException[Fetch Failed [Failed to fetch doc id [7]]]; nested: CorruptIndexException[Corrupted: docID=7, docBase=0, chunkDocs=0, numDocs=8 (resource=MMapIndexInput(path="/usr/share/opensearch/data/nodes/0/indices/KLrt-04kTQWB_ZUUyCK9Hg/0/index/_0.cfs") [slice=_0.fdt])];
```

### How to solve

Reindexing can help to save remaining data and make index writable.
For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/latest/opensearch/reindex-data].

### Recommendations

For more details and troubleshooting procedures, refer to [Data Files Corrupted on Primary Shard](scenarios/data_files_corrupted_on_primary_shard.md).

## Data Files are Corrupted On Replica Shard

### Description

The data files are corrupted on one or more replica shards corrupted, data files aren’t lost

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

OpenSearch withstands all cases with corrupted replica shards and repairs itself without any data loss.

### Recommendations

Refer to [Data Files Corrupted on Replica Shard](scenarios/data_files_corrupted_on_replica_shard.md).

## Data Files are Corrupted On Entire Index

### Description

All data files of the index were corrupted, there is no way to get data from this index.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

Not applicable

### How to solve

The only solution is to restore this index from a backup, provided one exists.

### Recommendations

For more details and troubleshooting procedures,refer to [Entire Index Corrupted](scenarios/entire_index_corrupted.md).

## Translog Corrupted

### Description

To prevent data loss, each shard has a transaction log, or translog, associated with it. If a shard is failed, the most recent transactions can be replayed from the transaction log when the shard recovers.
In some cases (such as a bad drive or user error), the translog can become corrupted.
When this corruption is detected by OpenSearch due to mismatching checksums, OpenSearch will fail the shard and refuse to allocate that copy of the data to the node,
recovering from a replica if available.

If a translog was corrupted, the shards with a corrupted translog will have `TranslogCorruptedException` in `unassigned.details`.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

```sh
curl -XGET http://localhost:9200/_cat/shards?v&h=index,shard,prirep,state,docs,store,ip,node,unassigned.reason,unassigned.details
```

Example response:

```text
    index      shard prirep state      docs  store ip        node            unassigned.reason unassigned.details
    cats       0     p      STARTED       0   130b 10.1.3.3  opensearch-1
    cats       0     r      STARTED       1   130b 10.1.12.6 opensearch-0
    cats       0     r      UNASSIGNED                                       ALLOCATION_FAILED failed recovery, failure RecoveryFailedException[[cats][0]: Recovery failed from {opensearch-1}{8QnQuADIS0yvpPk74UAvig}{z4GpsVEWQCypHyzr0eNfhw}{10.1.3.3}{10.1.3.3:9300} into {opensearch-2}{du6xA4LESROG2KfnAh9OiQ}{HimMm6A3S-mblfr5PDFdeA}{10.1.13.5}{10.1.13.5:9300}]; nested: RemoteTransportException[[opensearch-1][10.1.3.3:9300][internal:index/shard/recovery/start_recovery]]; nested: RecoveryEngineException[Phase[2] phase2 failed]; nested: TranslogCorruptedException[operation size must be at least 4 but was: 0];
```

### How to solve

If `index.translog.durability` is set to `async`, fsync and commit in the background every sync\_interval.
In the event of a hardware failure, all acknowledged writes since the last automatic commit will be discarded.

When this corruption is detected by OpenSearch due to mismatching checksums, OpenSearch will fail the shard and refuse to allocate that copy of the data to the node,
recovering from a replica if available.

If there is no copy of the data from which OpenSearch can recover successfully, you may want to recover the data that is part of the shard at the cost of losing the data that is currently
contained in the translog. OpenSearch provides a command-line tool for this: `opensearch-translog`.

### Recommendations

Not applicable

## New Master Cannot Be Elected

### Description

The situation when a new master cannot be elected

### Alerts

Not applicable

### Stack trace

```text
2022-02-10T10:41:12,348][WARN ][o.e.c.c.ClusterFormationFailureHelper] [es-node-1] master not discovered or elected yet, an election requires at least 2 nodes with ids from [ti0MftEaQk2lV0VMglBfTA, RampKFimRgqlgb09m-ZapA, k0-c9tKzRIeKWgxBioADnA], have only discovered non-quorum [{es-node-1}{RampKFimRgqlgb09m-ZapA}{ok6IkBWXScOUofCKJKkMpw}{10.128.162.21}{10.128.162.21:9300}{cdfhilmrstw}]; discovery will continue using [172.28.162.22:9300, 172.28.162.23:9300] from hosts providers and [{es-node-1}{RampKFimRgqlgb09m-ZapA}{ok6IkBWXScOUofCKJKkMpw}{10.128.162.21}{10.128.162.21:9300}{cdfhilmrstw}] from last-known cluster state; node term 52, last-accepted version 12542592 in term 45
```

### How to solve

If you see that the count of master-eligible nodes is reduced to zero (i.e., all master-eligible nodes are shut down), you need to run failed master-eligible nodes,
so that OpenSearch can elect a new master.

### Recommendations

For more details and troubleshooting procedures,refer to [New Master Cannot Be Elected](scenarios/new_master_can_not_be_elected.md).

## Elected Master Is Crashed

### Description

The elected master is crashed.

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

A troubleshooting procedure is not needed in cases when the leader node has crashed and all other nodes are still capable of communicating, because OpenSearch will handle this automatically.
The remaining nodes will detect the failure of the leader and initiate leader election.

### Recommendations

For more details and troubleshooting procedures,refer to [Elected Master Is Crashed](scenarios/elected_master_is_crashed.md).

## Problem During Replication

### Description

Data replication in OpenSearch is based on the primary-backup model. This model assumes a single authoritative copy of the data, called the primary.
All indexing operations first go to the primary, which is then in charge of replicating changes to active backup copies, called replica shards.
OpenSearch uses replica shards to provide failover capabilities, as well as to scale out reads.

### Alerts

* [OpenSearchReplicationDegradedAlert](./alerts.md#opensearchreplicationdegradedalert)
* [OpenSearchReplicationFailedAlert](./alerts.md#opensearchreplicationfailedalert)
* [OpenSearchReplicationTooHighLagAlert](./alerts.md#opensearchreplicationtoohighlagalert)
* [OpenSearchReplicationLeaderConnectionLostAlert](./alerts.md#opensearchreplicationleaderconnectionlostalert)

### Stack trace

```text
/go/pkg/mod/sigs.k8s.io/controller-runtime@v0.10.0/pkg/internal/controller/controller.go:227
2022-08-17T10:54:48.543Z        ERROR   controller.opensearchservice    Reconciler error        {"reconciler group": "qubership.org", "reconciler kind": "OpenSearchService", "name": "opensearch", "namespace": "platform-opensearch", "error": "some replication indicies are failed"}
```

### How to solve

When a major disaster strikes, there may be situations where only stale shard copies are available in the cluster.
OpenSearch will not automatically allocate such shard copies as primary shards, and the cluster will stay red.
In a case where all in-sync shard copies are gone for good, however, there is still a possibility for the cluster to revert to using stale copies,
but this requires manual intervention from the cluster administrator.

### Recommendations

For more details and troubleshooting procedures,refer to [Problem During Replication](scenarios/problem_during_replication.md).

## Primary Shard Is Down During User Request

### Description

Data can be lost if OpenSearch node is down during user request.
However, OpenSearch provides tools to prevent data loss, including a translog, or transaction log, which records every operation in OpenSearch as it happens.

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

OpenSearch provides the capability to subdivide your index into multiple pieces called shards. When you create an index, you can simply define the number of shards that you want.
Each shard is in itself a fully-functional and independent "index" that can be hosted on any node in the cluster.

In a network or cloud environment where failures can be expected anytime, it is very useful and highly recommended having a failover mechanism in case a shard or node somehow goes offline or
disappears for whatever reason. Therefore, OpenSearch enables you to make one or more copies of your index’s shards into what are called replica shards, replicas for short.

### Recommendations

For more details and troubleshooting procedures,refer to [Primary Shard Is Down During User Request](scenarios/primary_shard_is_down_during_user_request.md).

## Network Connection Is Lost and Restored

### Description

The network connection after it was lost and restored.

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

A troubleshooting procedure is not needed in cases when the network connection is temporarily disrupted between OpenSearch nodes, because such cases are handled by fault detection processes in
OpenSearch.

### Recommendations

For more details and troubleshooting procedures,refer to [Network Connection Is Lost and Restored](scenarios/network_connection_failure.md).

## Availability Zone Outage

### Description

The unexpected availability zone shutdown for any reason, which means the shutdown of several OpenSearch nodes.

### Alerts

Not applicable

### Stack 

Not applicable

### How to solve

If free resources are available on other availability zones, then OpenSearch should be scaled up.

### Recommendations

For more details and troubleshooting procedures,refer to [Availability Zone Outage](scenarios/availability_zone_outage.md).

## Availability Zone Shutdown and Startup

### Description

This scenario is very similar to `Availability Zone Outage` with the exception if the case when shutdown and start are planned.

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

1. If free resources are available on other availability zones, OpenSearch should be scaled up.
2. After restarting the availability zone, check that OpenSearch has the green status.

### Recommendations

For more details and troubleshooting procedures,refer to [Availability Zone Shutdown and Startup](scenarios/availability_zone_shutdown.md).

## Readiness Probe Failed

### Description

This error means the OpenSearch service did not have time to start for a given timeout. This may indicate a lack of resources
or problem with environment where OpenSearch has been deployed.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

```text
Readiness probe failed:
Back-off restarting failed container
```

### How to solve

This error can be solved by increasing the initial delay to check readiness of the service.

You can find this value in:

`OpenSearch StatefulSet` - `Actions` - `Edit YAML` - `readinessProbe.initialDelaySeconds:`

Try to increase this value twice.

Retry this action for all OpenSearch resources which pods have this error.

### Recommendations

Make sure you satisfy minimal HWE for you usage purposes, refer to [HWE](./installation.md#hwe)

## Liveness Probe Failed

### Description

This error means the OpenSearch service did not have time to ready for work for a given timeout. This may indicate a lack of resources
or problem with environment where OpenSearch has been deployed.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

```text
Liveness probe failed:
Back-off restarting failed container
```

### How to solve

This error can be solved by increasing the initial delay to check liveness of the service.

You can find this value in:

`OpenSearch StatefulSet` - `Actions` - `Edit YAML` - `livenessProbe.initialDelaySeconds:`

Try to increase this value twice.

Retry this action for all OpenSearch resources which pods have this error.

### Recommendations

Make sure you satisfy minimal HWE for you usage purposes, refer to [HWE](./installation.md#hwe)

## Max Virtual Memory Is Too Low

### Description

This error means the OpenSearch does not have enough virtual memory to start.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

```text
ERROR: [1] bootstrap checks failed
[1]: max virtual memory areas vm.max_map_count [65530] is too low, increase to at least [262144]
```

### How to solve

To resolve it you need execute the following command on all Kubernetes/OpenShift nodes, where OpenSearch is running:

```sh
sysctl -w vm.max_map_count=262144
```

### Recommendations

Not applicable

## Container Failed with Error: container has runAsNonRoot and image will run as root

### Description

The Operator is deployed successfully and operator logs do not contain errors, but OpenSearch Monitoring, OpenSearch Curator and/or DBaaS OpenSearch adapter pods fail.

OpenSearch Monitoring, OpenSearch Curator and DBaaS OpenSearch adapter do not have special user to run processes, so default (`root`) user is used.
If you miss the `securityContext` parameter in the pod configuration and `Pod Security Policy` is enabled, the default `securityContext` for pod is taken from `Pod Security Policy`.

If the `Pod Security Policy` is configured as follows then the error mentioned above occurs:

```yaml
runAsUser:
   # Require the container to run without root privileges.
   rule: 'MustRunAsNonRoot'
```

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

```text
Error: container has runAsNonRoot and image will run as root
```

### How to solve

Specify the correct `securityContext` in the configuration of the appropriate pod during installation.
For example, for OpenSearch Monitoring, OpenSearch Curator and DBaaS OpenSearch adapter you should specify the following parameter:

```yaml
securityContext:
   runAsUser: 1000
```

### Recommendations

Use appropriate security configurations.

## CRD Creation Failed on OpenShift 3.11

### Description

If Helm deployment or manual application of CRD failed with the CustomResourceDefinition is invalid error.
It depicts that the Kubernetes version is 1.11 (or less) and it is not compatible with the new format of CRD.

### Alerts

Not applicable

### Stack trace

```text
The CustomResourceDefinition "opensearchservices.qubership.org" is invalid: spec.validation.openAPIV3Schema: Invalid value:....
: must only have "properties", "required" or "description" at the root if the status subresource is enabled
```

### How to solve

To fix the issue, you need to find the following section in the CRD (`config/crd/old/qubership.org_opensearchservices.yaml`):

```yaml
#Comment it if you deploy to Kubernetes 1.11 (e.g OpenShift 3.11)
type: object
```

Comment or delete row `type: object`, and then apply the CRD manually.

### Recommendations

For more information, refer to [https://github.com/jetstack/cert-manager/issues/2200](https://github.com/jetstack/cert-manager/issues/2200).

**Note**: You need to disable CRD creation during installation in case of such errors.

## Operator Fails with Unauthorized Code on OpenSearch Readiness Check

### Description

After change of OpenSearch credentials in operator logs you see the unauthorized error.

During OpenSearch credentials change there was a problem to update the `opensearch-secret-old` secret in Kubernetes.
It means that credentials are updated in OpenSearch, but secret used by operator is not actual.

### Alerts

Not applicable

### Stack trace

```text
29T11:14:36.569Z ERROR controller.opensearchservice Reconciler error {"reconciler group": "qubership.org", "reconciler kind": "OpenSearchService", "name": "opensearch", "namespace": "opensearch-security", "error": "OpenSearch is not ready yet! Status code - [401]."}
sigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).Start.func2.2
 /go/pkg/mod/sigs.k8s.io/controller-runtime@v0.10.0/pkg/internal/controller/controller.go:227
```

### How to solve

Actualize the `opensearch-secret-old` secret manually by specifying the credentials from the `opensearch-secret` secret.

### Recommendations

Not applicable

## OpenSearch Does Not Start with "Not yet initialized" Error

### Description

This error means the OpenSearch hasn't been properly initialized or configured.

### Alerts

Not applicable

### Stack trace

```text
[ERROR][o.o.s.a.BackendRegistry  ] [opensearch-2] Not yet initialized (you may need to run securityadmin)
```

### How to solve

Restart OpenSearch pods if the error persists for more than 5 minutes after running all pods. This error is normal during the OpenSearch cluster initialization phase.

### Recommendations

Not applicable

## OpenSearch Starts Failing With TLS Certificate Error

### Description

At a certain point, OpenSearch stops working with a white screen `Not yet initialized`.
OpenSearch uses internal TLS certificates for node-to-node communications. For that connection OpenSearch uses self-signed certificates as they are not shared anywhere.
In the several versions of OpenSearch that certificate was generated with only one year duration, after that it starts failing.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

```text
[2024-02-12T14:23:07,448][WARN ][o.o.t.OutboundHandler    ] [opensearch-data-0] send message failed [channel: Netty4TcpChannel{localAddress=/10.129.21.8:57368, remoteAddress=10.129.133.71/10.129.133.71:9300}]
javax.net.ssl.SSLHandshakeException: PKIX path validation failed: java.security.cert.CertPathValidatorException: validity check failed
	at sun.security.ssl.Alert.createSSLException(Alert.java:131) ~[?:?]
     ... 30 more
...
Caused by: sun.security.validator.ValidatorException: PKIX path validation failed: java.security.cert.CertPathValidatorException: validity check failed
	at sun.security.validator.PKIXValidator.doValidate(PKIXValidator.java:369) ~[?:?]
    ... 30 more
Caused by: java.security.cert.CertPathValidatorException: validity check failed
	at sun.security.provider.certpath.PKIXMasterCertPathValidator.validate(PKIXMasterCertPathValidator.java:135) ~[?:?]
	... 30 more
Caused by: java.security.cert.CertificateExpiredException: NotAfter: Thu Jan 18 12:41:27 GMT 2024
	at sun.security.x509.CertificateValidity.valid(CertificateValidity.java:277) ~[?:?]
	at sun.security.x509.X509CertImpl.checkValidity(X509CertImpl.java:675) ~[?:?]
...
	at sun.security.ssl.CertificateMessage$T13CertificateConsumer.checkServerCerts(CertificateMessage.java:1335) ~[?:?]
```

### How to solve

The solution is to re-generate internal TLS certificates with long-lived duration.


If upgrade is not possible and manual fix is required, please follow steps:

1. Manually remove secrets "opensearch-admin-certs" and "opensearch-transport-certs"
   (and "opensearch-rest-certs" if presented) from the OpenSearch namespace.
2. Edit the template of [opensearch-tls-reinit.yaml](/docs/data/opensearch-tls-reinit.yaml) resources and specify corresponding
   namespace and OpenSearch docker image (you can take it from working pods) if required.
3. Apply result template with command `kubectl apply -f opensearch-tls-reinit.yaml` to the namespace with OpenSearch.
4. Wait until Job execution. There should be `'admin' certificates are generated` output inside.
5. Restart OpenSearch pods.
6. Remove created Job with the command `kubectl delete -f opensearch-tls-reinit.yaml` from the namespace with OpenSearch.
   **Solution**:

### Recommendations

This problem can be beforehand diagnosed with executing the following command that displays expiration time of current certificate:

```text
openssl x509 -enddate -noout -in /usr/share/opensearch/config/transport-crt.pem
```

Pay attention, this problem and provided solutions below are applicable only for disabled TLS deployment (`global.tls.enabled: false`), when only internal connections are under TLS. 
Otherwise, you have to regenerate TLS certificates with specified way (`CertManager` or manual certificates).

## OpenSearch Clients Fail with Authentication Error

### Description

DBaaS created users cannot login to OpenSearch and fails with authentication error.

DBaaS user was not correctly created on the OpenSearch side while DBaaS thought it was.
To check the real state of OpenSearch users you can reach the endpoint `{opensearch_host}/_plugins/_security/api/internalusers/`
with OpenSearch admin credentials and check the necessary DBaaS service user there.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

```text
Request action failed: Unexpected response status for RequestActionHandler.RequestDescription(method=POST, protocol=null, targetService=lead-management-core-service-v1, url=/salesManagement/v1/salesLead/start-reindex, headers=null, expectedResponseStatuses=[200], useUserToken=false, doNotAddCloudCoreTenantHeader=false, traceLogEnabled=false) request. Status:'500', but expected:'[200]'. Response = RestClientResponseEntity(responseBody={errors=[\{status=500, code=LMS-0001, reason=Authorization to Opensearch has been failed. Check credentials, message=Unexpected business error, extra={requestId=1721036300000.0.3662180000000}}]}, httpStatus=500, headers={})
```

### How to solve

  To resolve desynchronization of DBaaS database and OpenSearch users storage you can use the rollowing DBaaS restore api:

```bash
curl -u cluster-dba:{dbaas_password} -XPOST -H "Accept:application/json" -H  "Content-Type:application/json" http://dbaas-aggregator.dbaas:8080/api/v3/dbaas/internal/physical_databases/users/restore-password -d '
{
    "physicalDbId": "{OPENSEARCH-NAMESPACE}",
    "type": "opensearch",
    "settings": {}
}'
```

Then wait some time until users being synchronized.

### Recommendations

There can be a lot of causes of that desynchronization and you need to contact support with your case and provide logs from DBaaS Adapter.

## DBaaS Adapter.DBaaS Adapter Status Is Down/Warning

### Description

DBaaS Adapter is not alive.

### Alerts

* [OpenSearchDBaaS](./alerts.md#opensearchdbaasisdownalert)
* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

Not applicable

### How to solve

Make sure that OpenSearch is in `Green` state and works correctly.

### Recommendations

Not applicable

## DBaaS Adapter.DBaaS Adapter Status Is Problem

### Description

DBaaS adapter is in `problem` state.

### Alerts

* [OpenSearchIsDownAlert](./alerts.md#opensearchisdownalert)

### Stack trace

Not applicable

### How to solve

* DBaaS Adapter cannot be registered in DBaaS Aggregator.

To ensure this is the case, check the endpoint `<dbaas-opensearch-adapter-route>/health`.
The following output indicates that there is a problem with registration in DBaaS Aggregator:

```json
{"status":"PROBLEM","elasticCluster":{"status":"UP"},"physicalDatabaseRegistration":{"status":"PROBLEM"}}
```

You need to check that DBaaS Aggregator is alive and correct parameters are specified in DBaaS Adapter configuration to connect to DBaaS Aggregator.
Check DBaaS Adapter logs for more information about the problem with the DBaaS Aggregator registration.

* OpenSearch is not accessible by DBaaS Adapter.

To ensure this is the case, check the endpoint `<dbaas-opensearch-adapter-route>/health`.
The following output indicates that there is a problem with access to OpenSearch:

```json
{"status":"PROBLEM","elasticCluster":{"status":"PROBLEM"},"physicalDatabaseRegistration":{"status":"UP"}}
```

### Recommendations

You need to check that OpenSearch is alive and correct address and credentials are specified in DBaaS Adapter configuration to connect to OpenSearch.
Check DBaaS Adapter logs for more information about the problem with OpenSearch.

## OpenSearch Disaster Recovery Health Has Status "DEGRADED"

### Description

| Problem                                    | Severity | Possible Reason                                                                                                                                              |
|--------------------------------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| OpenSearch DR health has `DEGRADED` status | Average  | Replication between `active` and `standby` sides has unhealthy indices or failed replications. The possible root cause is a locked index on the active side. |

### Alerts

* [OpenSearchDBaaS](./alerts.md#opensearchdbaasisdownalert)

### Stack trace

Not applicable

### How to solve

Navigate to the OpenSearch console on `standby` side and run the following command:

   ```bash
   curl -u <username>:<password> -XGET http://opensearch.<opensearch_namespace>:9200/_cat/indices?h=index,health&v
   ```

   where:

   * `<username>:<password>` are the credentials to OpenSearch.
   * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.

   The result can be as follows:

   ```text
   health status index                  uuid                   pri rep docs.count docs.deleted store.size pri.store.size
   green  open   test_index_new         waIH2YgMRCaasksr28YkJg   5   1        198            0      1.3mb        672.3kb
   green  open   ha_test                wf2g8XAWT9SO31Q7L0DoBA   1   1    1772737           30    102.4mb         35.5mb
   green  open   test_index_1           WdE0LZzYR7e5Bl3WuoIj6A   5   1       1000            0      6.1mb            3mb
   green  open   test_index_new_one     rH0dn00iRh27hBmbC0tUog   5   1        200            0      1.3mb        693.9kb
   green  open   .opendistro_security   T6DvSm51R8eZc5IBpSGFcg   1   2          9            0    126.2kb           42kb
   green  open   .tasks                 bkWLpwKSRNe9YnVBecRRbA   1   1         19            0       38kb           19kb
   green  open   test_index_new_1       AXO1xAibTRa5f--S83B3oA   5   1        800            0      4.9mb          2.4mb
   green  open   test_index_new_one_1   MYVWZcsTT4KkJs8XBPGTvg   5   1        800            0      4.8mb          2.4mb
   green  open   test_index             tkKQhOZET6q0Fu4kMReO1Q   5   1        200            0      1.3mb          698kb
   ```

  Make sure that all indices required for replication have `green` health status.

Navigate to the OpenSearch console on `standby` side and execute the following:

   ```bash
   curl -u username:password http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/autofollow_stats
   ```

   where `opensearch.<opensearch_namespace>` are service name and namespace for OpenSearch on the `standby` side.

   The result can be as follows:

   ```json
   {
       "num_success_start_replication": 2,
       "num_failed_start_replication": 1,
       "num_failed_leader_calls": 0,
       "failed_indices": ["test_topic"],
       "autofollow_stats": [
           {
               "name": "dr-replication",
               "pattern": "*",
               "num_success_start_replication": 3,
               "num_failed_start_replication": 0,
               "num_failed_leader_calls": 0,
               "failed_indices": ["test_topic"]
           }
       ]
   }
   ```

   Recognize the list of `failed_indices`.

For each index from the previous step do the following:

   1. Navigate to the OpenSearch console on the `active` side and try to stop replication for the index:

      ```bash
      curl -u <username>:<password> -XPOST http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/<index_name>/_stop -H 'Content-Type: application/json' -d'{}'
      ```

      where:
      * `<username>:<password>` are the credentials to OpenSearch.
      * `opensearch.<opensearch_namespace>` are service name and namespace for OpenSearch on the active side.
      * `<index_name>` is the name of failed index. For example, `test_topic`.

      This is an asynchronous operation and expected response is the following:

      ```json
      {"acknowledged": true}
      ```

   2. If on the previous step you have got the following response:

      ```json
      {"error":{"root_cause":[{"type":"illegal_argument_exception","reason":"No replication in progress for index:test_topic"}],"type":"illegal_argument_exception","reason":"No replication in progress for index:test_topic"},"status":400}
      ```

      The replication is not run on the `active` side for the specified failed `test_topic` index.
      Then you need to go to the `standby` side of OpenSearch cluster and check the status of replication for above index:

      ```bash
      curl -u <username>:<password> -XGET http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/<index_name>/_status?pretty
      ```

      Where:
      * `<username>:<password>` are the credentials to OpenSearch.
      * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.
      * `<index_name>` is the name of failed index. For example, `test_topic`.

   3. If `status` of index replication on `standby` side is `FAILED`, you have to stop corresponding replication with the following command:

      ```bash
      curl -u <username>:<password> -XPOST  http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/<index_name>/_stop -H 'Content-Type: application/json' -d'{}'
      ```

      Where:
      * `<username>:<password>` are the credentials to OpenSearch.
      * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.
      * `<index_name>` is the name of failed index. For example, `test_topic`.

For `standby` side switch OpenSearch cluster to the `active` side and return to the `standby` one. This action should restart replication properly.

### Recommendations

Not applicable

## ResourceAlreadyExistsException: task with id {replication:index:test_index} already exist

### Description

| Problem                                           | Severity | Possible Reason                                                                                     |
|---------------------------------------------------|----------|-----------------------------------------------------------------------------------------------------|
| Indices are not replicated to the `standby` side. | Average  | OpenSearch data is corrupted: previous replication tasks for indices were cached in metadata files. |

OpenSearch disaster recovery health has `DEGRADED` status and indices are not replicated.

### Alerts

Not applicable

### Stack trace

```text
[2023-05-18T12:03:27,684][WARN ][o.o.r.t.a.AutoFollowTask ] [opensearch-0][leader-cluster] Failed to start replication for leader-cluster:test_index -> test_index.
org.opensearch.ResourceAlreadyExistsException: task with id {replication:index:test_index} already exist
	at org.opensearch.persistent.PersistentTasksClusterService$1.execute(PersistentTasksClusterService.java:135) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.ClusterStateUpdateTask.execute(ClusterStateUpdateTask.java:63) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.service.MasterService.executeTasks(MasterService.java:804) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.service.MasterService.calculateTaskOutputs(MasterService.java:378) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.service.MasterService.runTasks(MasterService.java:249) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.service.MasterService.access$000(MasterService.java:86) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.service.MasterService$Batcher.run(MasterService.java:173) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.service.TaskBatcher.runIfNotProcessed(TaskBatcher.java:174) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.cluster.service.TaskBatcher$BatchedTask.run(TaskBatcher.java:212) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.common.util.concurrent.ThreadContext$ContextPreservingRunnable.run(ThreadContext.java:733) [opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.common.util.concurrent.PrioritizedOpenSearchThreadPoolExecutor$TieBreakingPrioritizedRunnable.runAndClean(PrioritizedOpenSearchThreadPoolExecutor.java:275) ~[opensearch-1.3.7.jar:1.3.7]
	at org.opensearch.common.util.concurrent.PrioritizedOpenSearchThreadPoolExecutor$TieBreakingPrioritizedRunnable.run(PrioritizedOpenSearchThreadPoolExecutor.java:238) ~[opensearch-1.3.7.jar:1.3.7]
	at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1128) [?:?]
	at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:628) [?:?]
	at java.lang.Thread.run(Thread.java:829) [?:?]
```

### How to solve

1. Scale down all pods related to OpenSearch (`master`, `data`, `ingest`, `arbiter`) on the `standby` side.
2. Clear the OpenSearch data on the `standby` side in one of the following ways:
   * Remove OpenSearch persistent volumes.
   * Clear persistent volumes manually.
3. Scale up all pods related to OpenSearch (`master`, `data`, `ingest`, `arbiter`) on the `standby` side.

**Note**: It is safe as you need to perform these steps on the `standby` side. All the data is replicated from the `active` side once the replication process has started successfully.

For more information about this issue, refer to [https://github.com/opensearch-project/cross-cluster-replication/issues/840](https://github.com/opensearch-project/cross-cluster-replication/issues/840).

### Recommendations

Not applicable

## Index Is Not Replicated To Standby Side Without Any Errors

### Description

| Problem                                           | Severity | Possible Reason                                                                                                                      |
|---------------------------------------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------|
| Index changes stopped replicating to standby side | Average  | Problem index was removed and created again on active side during replication and standby OpenSearch marked replication as `paused`. |

### Alerts

* [OpenSearchReplicationFailedAlert](./alerts.md#opensearchreplicationfailedalert)

### Stack trace

Not applicable

### How to solve

1. Navigate to the OpenSearch console on `standby` side and run the following command:

      ```bash
      curl -u <username>:<password> -XGET http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/<index_name>/_status?pretty
      ```

   Where:
   * `<username>:<password>` are the credentials to OpenSearch.
   * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.
   * `<index_name>` is the name of missed index. For example, `test_topic`.

   The following response makes it clear that index was removed in active side:

   ```json
   {"status":"PAUSED","reason":"AutoPaused: [[haindex2][0] - org.opensearch.index.IndexNotFoundException - \"no such index [haindex2]\"], ","leader_alias":"leader-cluster","leader_index":"haindex2","follower_index":"haindex2"}
   ```

2. To run the replication again, you can remove presented index on standby side:

   ```bash
   curl -u <username>:<password> -XDELETE http://opensearch.<opensearch_namespace>:9200/<index_name>
   ```

   Where:
   * `<username>:<password>` are the credentials to OpenSearch.
   * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.
   * `<index_name>` is the name of missed index. For example, `test_topic`.

   Then wait some time for `autofollow` process run replication again.

### Recommendations

This option cleans all index data presented on the standby side. Make sure to remove this and check whether OpenSearch on the active side has correct changes.
