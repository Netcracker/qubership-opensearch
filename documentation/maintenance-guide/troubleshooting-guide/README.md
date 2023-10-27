The following topics are covered in this chapter:

[[_TOC_]]

This section provides detailed troubleshooting procedures for the OpenSearch cluster.

# Prometheus Alerts

## OpenSearchCPULoadAlert

### Description

One of OpenSearch pods uses 95% of the CPU limit.

For more information, refer to [CPU Overload](scenarios/cpu_overload.md).

### Possible Causes

- Insufficient CPU resources allocated to OpenSearch pods.
- Heavy search request.
- Indexing workload.

### Impact

- Increased response time and potential slowdown of OpenSearch requests.
- Degraded performance of services used the OpenSearch.

### Actions for Investigation

1. Get the statistics of cluster nodes using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the CPU usage trends in OpenSearch monitoring dashboard.
3. Review OpenSearch logs for any performance related issues.

### Recommended Actions to Resolve Issue

1. Check the required resources and increase the CPU limit if it is still within recommended limits.
2. Add additional data nodes to redistribute the load.

## OpenSearchDiskUsageAbove75%Alert

### Description

One of OpenSearch pods uses 75% of the disk.

For more information, refer to [Data Nodes are Out of Space](#data-nodes-are-out-of-space).

### Possible Causes

- Low space on the disk.
- Failed disks.
- Insufficient disk resources allocated to OpenSearch pods.

### Impact

- Prospective problems with shards allocation and the impossibility to write to the OpenSearch.

### Actions for Investigation

1. Retrieve the statistics of all the nodes in the cluster, using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the disk usage trends in OpenSearch monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Increase disk space for OpenSearch if it's possible.
2. If all the data nodes are running low on disk space or some disks are failed, you need to add more data nodes to the cluster.

## OpenSearchDiskUsageAbove85%Alert

### Description

One of OpenSearch pods uses 85% of the disk.

For more information, refer to [Data Nodes are Out of Space](#data-nodes-are-out-of-space).

### Possible Causes

- Low space on the disk.
- Failed disks.
- Insufficient disk resources allocated to OpenSearch pods.

### Impact

- Increased response time and potential slowdown of OpenSearch requests.
- Inability to allocate shards to nodes with exceeded 85% disk limit.

### Actions for Investigation

1. Retrieve the statistics of all the nodes in the cluster, using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the disk usage trends in OpenSearch monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Increase disk space for OpenSearch if it's possible.
2. If all the data nodes are running low on disk space or some disks are failed, you need to add more data nodes to the cluster.

## OpenSearchDiskUsageAbove95%Alert

### Description

One of OpenSearch pods uses 95% of the disk.

For more information, refer to [Data Nodes are Out of Space](#data-nodes-are-out-of-space).

### Possible Causes

- Low space on the disk.
- Failed disks.
- Insufficient disk resources allocated to OpenSearch pods.

### Impact

- Inability to write to OpenSearch indices that has one or more shards allocated on the problem node, and that has at least one disk exceeding the flood stage.

### Actions for Investigation

1. Retrieve the statistics of all the nodes in the cluster, using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the disk usage trends in OpenSearch monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Increase disk space for OpenSearch if it's possible.
2. If all the data nodes are running low on disk space or some disks are failed, add more data nodes to the cluster.

## OpenSearchMemoryUsageAlert

### Description

One of OpenSearch pods uses 95% of the memory limit.

For more information, refer to [Memory Limit](scenarios/memory_limit.md).

### Possible Causes

- Insufficient memory resources allocated to OpenSearch pods.
- Heavy workload during execution.

### Impact

- Potential out-of-memory errors and OpenSearch cluster instability.
- Degraded performance of services used the OpenSearch.

### Actions for Investigation

1. Get the statistics of the cluster nodes using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the memory usage trends in OpenSearch monitoring dashboard.
3. Review OpenSearch logs for memory related errors.

### Recommended Actions to Resolve Issue

1. Try to increase memory request, memory limit and heap size for OpenSearch.
2. Add data nodes to redistribute the load.

## OpenSearchHeapMemoryUsageAlert

### Description

Heap memory usage by one of the pods in the OpenSearch cluster came close to the specified limit.

For more information, refer to [Memory Limit](scenarios/memory_limit.md).

### Possible Causes

- Insufficient memory resources allocated to OpenSearch pods.
- Heavy workload during execution.

### Impact

- Potentially lead to the increase of response times or crashes.
- Degraded performance of services used the OpenSearch.

### Actions for Investigation

1. Get the statistics of the cluster nodes using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the heap memory usage trends in OpenSearch monitoring dashboard.
3. Review OpenSearch logs for memory related errors.

### Recommended Actions to Resolve Issue

1. Try to increase heap size for OpenSearch.
2. Add data nodes to redistribute the load.

## OpenSearchIsDegradedAlert

### Description

OpenSearch cluster is degraded, that is, at least one of the nodes have failed, but cluster is able to work.

For more information, refer to [Cluster Status is Failed or Degraded](#cluster-status-is-failed-or-degraded).

### Possible Causes

- One or more replica shards unassigned.
- OpenSearch pod failures or unavailability.
- Resource constraints impacting OpenSearch pod performance.

### Impact

- Reduced or disrupted functionality of the OpenSearch cluster.
- Potential impact on services and processes relying on the OpenSearch.

### Actions for Investigation

1. Check the status of OpenSearch pods.
2. Check the health of the cluster, using the following API:

   ```
   curl -X GET 'http://localhost:9200/_cluster/health?pretty'
   ```

3. Review logs of OpenSearch pods for any errors or issues.
4. Verify resource utilization of OpenSearch pods (CPU, memory).

### Recommended Actions to Resolve Issue

1. Investigate issues with unassigned shards.
2. Restart or redeploy OpenSearch pods if they are in a failed state.
3. Investigate and address any resource constraints affecting the OpenSearch pod performance.

## OpenSearchIsDownAlert

### Description

OpenSearch cluster is down, and there are no available pods.

For more information, refer to [Cluster Status is N/A](#cluster-status-is-na) and [Cluster Status is Failed or Degraded](#cluster-status-is-failed-or-degraded).

### Possible Causes

- Network issues affecting the OpenSearch pod communication.
- OpenSearch's storage is corrupted.
- Lack of memory or CPU.
- Long garbage collection time.
- One or more primary shards are not allocated in the cluster.

### Impact

- Complete unavailability of the OpenSearch cluster.
- Services and processes relying on the OpenSearch will fail.

### Actions for Investigation

1. Check the status of OpenSearch pods.
2. Check the health of the cluster, using the following API:

   ```
   curl -X GET 'http://localhost:9200/_cluster/health?pretty'
   ```

3. Review logs of OpenSearch pods for any errors or issues.
4. Verify resource utilization of OpenSearch pods (CPU, memory).

### Recommended Actions to Resolve Issue

1. Check the network connectivity to the OpenSearch pods.
2. Check the OpenSearch storage for free space or data corruption.
3. Restart or redeploy all OpenSearch pods at once.

## OpenSearchDBaaSIsDownAlert

### Description

OpenSearch DBaaS adapter is not working.

### Possible Causes

- Incorrect configuration parameters, i.e. credentials.
- OpenSearch is down.

### Impact

- Complete unavailability of the OpenSearch DBaaS adapter.
- Services and processes relying on the OpenSearch DBaaS adapter will fail.

### Actions for Investigation

1. Monitor the DBaaS adapter status in OpenSearch monitoring dashboard.
2. Review logs of DBaaS adapter pod for any errors or issues.

### Recommended Actions to Resolve Issue

1. Correct the OpenSearch DBaaS adapter configuration parameters.
2. Investigate problems with the OpenSearch.

## OpenSearchLastBackupHasFailedAlert

### Description

The last OpenSearch backup has finished with `Failed` status.

For more information, refer to [Last Backup Has Failed](#last-backup-has-failed).

### Possible Causes

- Unavailable or broken backup storage (`Persistent Volume` or `S3`).
- Network issues affecting the OpenSearch and curator pod communication.

### Impact

- Unavailable backup for OpenSearch and inability to restore it in case of disaster.

### Actions for Investigation

1. Monitor the curator state on Backup Daemon Monitoring dashboard.
2. Review OpenSearch curator logs for investigation of cases the issue.
3. Check backup storage.

### Recommended Actions to Resolve Issue

1. Fix issues with backup storage if necessary.
2. Follow [Last Backup Has Failed](#last-backup-has-failed) for additional steps.

## OpenSearchQueryIsTooSlowAlert

### Description

Execution time of one of index queries in the OpenSearch exceeds the specified threshold.

This threshold can be overridden with parameter `monitoring.thresholds.slowQuerySecondsAlert` described in [OpenSearch monitoring](/documentation/installation-guide/README.md#monitoring) parameters.

### Possible Causes

- Insufficient resources allocated to OpenSearch pods.

### Impact

- The query takes too long.

### Actions for Investigation

1. Monitor queries in `OpenSearch Slow Queries` monitoring dashboard.
2. Review OpenSearch logs for investigation of cases the issue.

### Recommended Actions to Resolve Issue

1. Try to increase resources requests and limits and heap size for OpenSearch.

## OpenSearchReplicationDegradedAlert

### Description

Replication between two OpenSearch clusters in Disaster Recovery mode has `degraded` status.

For more information, refer to [OpenSearch Disaster Recovery Health](#opensearch-disaster-recovery-health).

### Possible Causes

- Replication for some indices does not work correctly.
- Replication status for some indices is `failed`.
- Some indices in OpenSearch have `red` status.

### Impact

- Some required indices are not replicated from `active` to `standby` side.

### Actions for Investigation

1. Monitor replication in `OpenSearch Replication` monitoring dashboard.
2. Review operator and OpenSearch logs for investigation of cases the issue.

### Recommended Actions to Resolve Issue

1. Check solutions described in [OpenSearch Disaster Recovery Health](#opensearch-disaster-recovery-health) section.

## OpenSearchReplicationFailedAlert

### Description

Replication between two OpenSearch clusters in Disaster Recovery mode has `failed` status.

For more information, refer to [OpenSearch Disaster Recovery Health](#opensearch-disaster-recovery-health).

### Possible Causes

- Replication for all indices does not work correctly.
- Replication rule does not exist.
- Some error during replication check occurs.

### Impact

- All required indices are not replicated from `active` to `standby` side.

### Actions for Investigation

1. Monitor replication in `OpenSearch Replication` monitoring dashboard.
2. Review operator and OpenSearch logs for investigation of cases the issue.

### Recommended Actions to Resolve Issue

1. Check solutions described in [OpenSearch Disaster Recovery Health](#opensearch-disaster-recovery-health) section.

## OpenSearchReplicationLeaderConnectionLostAlert

### Description

`follower` OpenSearch cluster has lost connection with `leader` OpenSearch cluster in Disaster Recovery mode.

### Possible Causes

- Network issues affecting the OpenSearch clusters communication.
- Dead `leader` OpenSearch cluster.

### Impact

- Replication from `active` to `standby` side doesn't work.

### Actions for Investigation

1. Monitor replication in `OpenSearch Replication` monitoring dashboard.
2. Check connectivity between Kubernetes clusters.
3. Review operator and OpenSearch logs in both OpenSearch clusters.

### Recommended Actions to Resolve Issue

1. Fix network issues between Kubernetes.
2. Restart or redeploy `leader` OpenSearch cluster.

## OpenSearchReplicationTooHighLagAlert

### Description

The documents lag of replication between two OpenSearch clusters comes close to the specified limit.

This limit can be overridden with parameter `monitoring.thresholds.lagAlert` described in [OpenSearch monitoring](/documentation/installation-guide/README.md#monitoring) parameters.

### Possible Causes

- Insufficient resources allocated to OpenSearch pods.
- Network issues affecting the OpenSearch clusters communication.

### Impact

- Some data may be lost if the `active` Kubernetes cluster fails.

### Actions for Investigation

1. Monitor resources usage trends in OpenSearch monitoring dashboard.
2. Monitor replication in `OpenSearch Replication` monitoring dashboard.
3. Review operator and OpenSearch logs in both OpenSearch clusters.

### Recommended Actions to Resolve Issue

1. Try to increase resources requests and limits and heap size for OpenSearch.

# Troubleshooting Scenarios

## Cluster State

OpenSearch monitoring has the following two types of cluster states:

* Cluster Health
* Cluster Status

### Cluster Health

OpenSearch provides a default metric that indicates cluster state. It is called *cluster health*. To check the health of the cluster, the cluster health API can be used.

For more information on the cluster health API, refer to the official OpenSearch documentation, _Cluster Health_ [https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health](https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health).

To check the health of the cluster, the following API can be used:

```
curl -XGET http://localhost:9200/_cluster/health
```

### Cluster Status

Cluster status is a custom metric that can be found on the *Cluster status* panel in Grafana. Possible values for cluster status are:

* `UP` - Cluster health status is GREEN and all nodes working.
* `DEGRADED` - Cluster has YELLOW health status or one node is failed.
* `FAILED` - Cluster has RED health status.

## Common Problems

The following information describes the common problems you may encounter.

### Cluster Status is N/A

*N/A* status while monitoring indicates that the OpenSearch cluster is unreachable.

The main cause is any of the following:

* The cluster is down.
* The monitoring agent is not deployed.
* The monitoring agent is down.

To resolve the issue, navigate to the OpenShift console and check the service state. In the simplest scenario, starting the service solves the issue. In the event of a permanent failure, try to redeploy the cluster or recover it from the backup.

### Cluster Status is Failed or Degraded

`Failed` status indicates that one or more primary shards is not allocated in the cluster. `Degraded` status means that one or more replica shards is missing.

This can happen when a node drops off the cluster for some reason. This could be due to disk failure, lack of memory or CPU, long garbage collection time, availability zone outage, and so on.

To check the health of the cluster, use the following API:

```
curl -XGET http://localhost:9200/_cluster/health
```

For more information on OpenSearch clusters, refer to the official OpenSearch documentation [https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health](https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health).

To identify the reason for the node failure, check the monitoring dashboard for any unusual changes that may have occurred around the same time the node failed. When the problem is localized, go to the appropriate problem description and follow the relevant troubleshooting procedure to fix it.

After the node is fixed, its shards remain in an initializing state before they transition back to active status. During this initialization period, the cluster state may change from `up` to `degraded` or `failed` until the shards on the recovering node regain active status. In many cases, a brief status change to `degraded` or `failed` may not require any additional actions.

If it is a permanent failure, and the node cannot be recovered, new nodes can be added, after which OpenSearch recovers data from any available replica shards. Replica shards can be promoted to primary shards and redistributed on the new nodes.

The following scenarios are examples of permanent failures:

* [Disk failure on one node](scenarios/disk_failure_on_one_node.md)
* [Disk failure on all nodes](scenarios/disk_failure_on_all_nodes.md)

If both the primary and replica copy of a shard are lost, data can be recovered from backup.

### Last Backup Has Failed

The last OpenSearch backup has finished with `Failed` status.

*Solution*

Check that OpenSearch curator pod exists and is up. If OpenSearch curator is down, restart appropriate deployment. If curator pod is up, check its state by the following command from pod's terminal:

```bash
curl -XGET http://localhost:8080/health
```

### Data Nodes are Out of Space

If all data nodes are running low on disk space, more data nodes should be added to the cluster. Be sure that all indices have enough primary shards to be able to balance their data across all those nodes. However, if only some nodes are running out of disk space, this is usually a sign that an index was initialized with too few shards. If an index is composed of a few very large shards, it is hard for OpenSearch to distribute these shards across nodes in a balanced manner.

For more information, refer to [Disk Filled on All Nodes](scenarios/disk_filled_on_all_nodes.md).

### Lack of Resources

Some problems with the OpenSearch cluster can occur due to a lack of CPU, memory, and disk resources. For more information, refer to [Memory Limit](/documentation/maintenance-guide/troubleshooting-guide/scenarios/memory_limit.md), [CPU Overload](/documentation/maintenance-guide/troubleshooting-guide/scenarios/cpu_overload.md) and [I/O Limit](/documentation/maintenance-guide/troubleshooting-guide/scenarios/io_limit.md).

### OpenSearch Fails Down with CircuitBreakingException

OpenSearch produces the following exception:

```
org.elasticsearch.common.breaker.CircuitBreakingException: [parent] Data too large, data for [<http_request>] would be larger than limit of [1453142835/1.3gb]
```

Both `GET` and `PUT` requests are failed.

OpenSearch includes a special circuit breaker that is intended to prevent `OutOfMemoryException`. The circuit breaker estimates the memory requirements of a query by inspecting the fields involved. It then checks to see whether loading the field data required would push the total field data size over the configured percentage of the heap. If the estimated query size is larger than the limit, then the circuit breaker is tripped, and the query will be aborted and return an exception. For more information on OpenSearch, refer to the official OpenSearch documentation: [https://aws.amazon.com/premiumsupport/knowledge-center/opensearch-circuit-breaker-exception](https://aws.amazon.com/premiumsupport/knowledge-center/opensearch-circuit-breaker-exception).

The main reasons for this failure are as follows:

* Query tries to load more data than memory is currently available.
* Index is bigger than available heap.
* Continuous shard relocation due to the data nodes being out of space.

For more information, refer to [Memory Limit](scenarios/memory_limit.md) and [Disk Filled on All Nodes](scenarios/disk_filled_on_all_nodes.md).

### Data Files are Corrupted

This section describes the troubleshooting to be done if the data files are corrupted.

#### Data Files are Corrupted On Primary Shard

Index has no replica shards, get queries return incomplete data, update queries fail, and some primary shards are in unassigned status with `CorruptIndexException`. For more details and troubleshooting procedures, refer to [Data Files Corrupted on Primary Shard](scenarios/data_files_corrupted_on_primary_shard.md).

#### Data Files are Corrupted On Replica Shard

OpenSearch withstands all cases with corrupted replica shards and repairs itself without any data loss. For more information, refer to [Data Files Corrupted on Replica Shard](scenarios/data_files_corrupted_on_replica_shard.md).

#### Data Files are Corrupted On Entire Index

If all data files of the index were corrupted, there is no way to get data from this index. The only solution is to restore this index from a backup, provided one exists. For more details and troubleshooting procedures, refer to [Entire Index Corrupted](scenarios/entire_index_corrupted.md).

### Translog Corrupted

To prevent data loss, each shard has a transaction log, or translog, associated with it. If a shard is failed, the most recent transactions can be replayed from the transaction log when the shard recovers.

In some cases (such as a bad drive or user error), the translog can become corrupted. When this corruption is detected by OpenSearch due to mismatching checksums, OpenSearch will fail the shard and refuse to allocate that copy of the data to the node, recovering from a replica if available.

If a translog was corrupted, the shards with a corrupted translog will have `TranslogCorruptedException` in `unassigned.details`.

For more details and troubleshooting procedures, refer to [Translog Is Corrupted](scenarios/translog_is_corrupted.md).

### Other Problems

Other problem descriptions and troubleshooting procedures can be found in the following chapters:

* [New Master Cannot Be Elected](scenarios/new_master_can_not_be_elected.md)
* [Elected Master Is Crashed](scenarios/elected_master_is_crashed.md)
* [Problem During Replication](scenarios/problem_during_replication.md)
* [Primary Shard Is Down During User Request](scenarios/primary_shard_is_down_during_user_request.md)
* [Network Connection Is Lost and Restored](scenarios/network_connection_failure.md)
* [Availability Zone Outage](scenarios/availability_zone_outage.md)
* [Availability Zone Shutdown and Startup](scenarios/availability_zone_shutdown.md)

##  OpenSearch node does not start

There are situations when the starting of the OpenSearch service fails with error after few unsuccessful attempts.

These errors can be found in `Pod` - `Events` tab of the OpenSearch deployment.

### Readiness Probe Failed

This error means the OpenSearch service did not have time to start for a given timeout. This may indicate a lack of resources
or problem with environment where OpenSearch has been deployed.

This error can be solved by increasing the initial delay to check readiness of the service.

You can find this value in:

`OpenSearch StatefulSet` - `Actions` - `Edit YAML` - `readinessProbe.initialDelaySeconds:`

Try to increase this value twice.

Retry this action for all OpenSearch resources which pods have this error.

### Liveness Probe Failed

This error means the OpenSearch service did not have time to ready for work for a given timeout. This may indicate a lack of resources
or problem with environment where OpenSearch has been deployed.

This error can be solved by increasing the initial delay to check liveness of the service.

You can find this value in:

`OpenSearch StatefulSet` - `Actions` - `Edit YAML` - `livenessProbe.initialDelaySeconds:`

Try to increase this value twice.

Retry this action for all OpenSearch resources which pods have this error.

### Max Virtual Memory Is Too Low

```
ERROR: [1] bootstrap checks failed
[1]: max virtual memory areas vm.max_map_count [65530] is too low, increase to at least [262144]
```

This error means the OpenSearch does not have enough virtual memory to start.

To resolve it you need execute the following command on all Kubernetes/OpenShift nodes, where OpenSearch is running:

```
sysctl -w vm.max_map_count=262144
```

### Container Failed with Error: container has runAsNonRoot and image will run as root

The Operator is deployed successfully and operator logs do not contain errors, but OpenSearch Monitoring, OpenSearch Curator and/or DBaaS OpenSearch adapter pods fail with the following error:

```
Error: container has runAsNonRoot and image will run as root
```

**Problem**: OpenSearch Monitoring, OpenSearch Curator and DBaaS OpenSearch adapter do not have special user to run processes, so default (`root`) user is used. If you miss the `securityContext` parameter in the pod configuration and `Pod Security Policy` is enabled, the default `securityContext` for pod is taken from `Pod Security Policy`.

If the `Pod Security Policy` is configured as follows then the error mentioned above occurs:

```
runAsUser:
  # Require the container to run without root privileges.
  rule: 'MustRunAsNonRoot'
```

**Solution**:

Specify the correct `securityContext` in the configuration of the appropriate pod during installation. For example, for OpenSearch Monitoring, OpenSearch Curator and DBaaS OpenSearch adapter you should specify the following parameter:

```
securityContext:
    runAsUser: 1000
```

### CRD Creation Failed on OpenShift 3.11

If Helm deployment or manual application of CRD failed with the following error, it depicts that the Kubernetes version is 1.11 (or less) and it is not compatible with the new format of CRD:

```
The CustomResourceDefinition "opensearchservices.netcracker.com" is invalid: spec.validation.openAPIV3Schema: Invalid value:....
: must only have "properties", "required" or "description" at the root if the status subresource is enabled
```

For more information, refer to [https://github.com/jetstack/cert-manager/issues/2200](https://github.com/jetstack/cert-manager/issues/2200).

**Solution**:

To fix the issue, you need to find the following section in the CRD (`config/crd/old/netcracker.com_opensearchservices.yaml`):

```
#Comment it if you deploy to Kubernetes 1.11 (e.g OpenShift 3.11)
type: object
``` 

Comment or delete row `type: object`, and then apply the CRD manually.

**Note**: You need to disable CRD creation during installation in case of such errors.

### Operator Fails with Unauthorized Code on OpenSearch Readiness Check

After change of OpenSearch credentials in operator logs you see the following error:

```
29T11:14:36.569Z ERROR controller.opensearchservice Reconciler error {"reconciler group": "netcracker.com", "reconciler kind": "OpenSearchService", "name": "opensearch", "namespace": "opensearch-security", "error": "OpenSearch is not ready yet! Status code - [401]."}
sigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).Start.func2.2
 /go/pkg/mod/sigs.k8s.io/controller-runtime@v0.10.0/pkg/internal/controller/controller.go:227
```

**Problem**:

During OpenSearch credentials change there was a problem to update the `opensearch-secret-old` secret in Kubernetes. It means that credentials are updated in OpenSearch, but secret used by operator is not actual.

**Solution**:

Actualize the `opensearch-secret-old` secret manually by specifying the credentials from the `opensearch-secret` secret.

## DBaaS Adapter Health

OpenSearch monitoring has `DBaaS Adapter Status` indicator of DBaaS Adapter health.

### Common Problems

The following information describes the common problems you may encounter.

#### DBaaS Adapter Status Is Down

The `Down` state means one of the following problems:
* DBaaS Adapter is not alive. See [DBaaS Is Down Alert](#opensearchdbaasisdownalert).
* OpenSearch has `red` state. See [OpenSearch is Down Alert](#opensearchisdownalert).

#### DBaaS Adapter Status Is Warning

The `Warning` state means the following problem:
* OpenSearch has `yellow` state. See [OpenSearch is Degraded Alert](#opensearchisdegradedalert).

#### DBaaS Adapter Status Is Problem

The `Problem` state means one of the following problems:

* DBaaS Adapter cannot be registered in DBaaS Aggregator.

To ensure this is the case, check the endpoint `<dbaas-opensearch-adapter-route>/health`.
The following output indicates that there is a problem with registration in DBaaS Aggregator:
```
{"status":"PROBLEM","elasticCluster":{"status":"UP"},"physicalDatabaseRegistration":{"status":"PROBLEM"}}
```
You need to check that DBaaS Aggregator is alive and correct parameters are specified in DBaaS Adapter configuration to connect to DBaaS Aggregator.
Check DBaaS Adapter logs for more information about the problem with the DBaaS Aggregator registration.

* OpenSearch is not accessible by DBaaS Adapter.

To ensure this is the case, check the endpoint `<dbaas-opensearch-adapter-route>/health`.
The following output indicates that there is a problem with access to OpenSearch:
```
{"status":"PROBLEM","elasticCluster":{"status":"PROBLEM"},"physicalDatabaseRegistration":{"status":"UP"}}
```
You need to check that OpenSearch is alive and correct address and credentials are specified in DBaaS Adapter configuration to connect to OpenSearch.
Check DBaaS Adapter logs for more information about the problem with OpenSearch.

## OpenSearch Disaster Recovery Health

### OpenSearch Disaster Recovery Health Has Status "DEGRADED"

| Problem                                    | Severity | Possible Reason                                                                                                                                              |
|--------------------------------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| OpenSearch DR health has `DEGRADED` status | Average  | Replication between `active` and `standby` sides has unhealthy indices or failed replications. The possible root cause is a locked index on the active side. |

**Solution**:

1. Navigate to the OpenSearch console on `standby` side and run the following command:

   ```bash
   curl -u <username>:<password> -XGET http://opensearch.<opensearch_namespace>:9200/_cat/indices?h=index,health&v
   ```

   where:
      * `<username>:<password>` are the credentials to OpenSearch.
      * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.

   The result can be as follows:

   ```
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

2. Navigate to the OpenSearch console on `standby` side and execute the following:

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

3. For each index from the previous step do the following:

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

      The replication is not run on the `active` side for the specified failed `test_topic` index. Then you need to go to the `standby` side of OpenSearch cluster and check the status of replication for above index:

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

4. For `standby` side switch OpenSearch cluster to the `active` side and return to the `standby` one. This action should restart replication properly. 

#### ResourceAlreadyExistsException: task with id {replication:index:test_index} already exist

| Problem                                           | Severity | Possible Reason                                                                                     |
|---------------------------------------------------|----------|-----------------------------------------------------------------------------------------------------|
| Indices are not replicated to the `standby` side. | Average  | OpenSearch data is corrupted: previous replication tasks for indices were cached in metadata files. |

**Description**:

OpenSearch disaster recovery health has `DEGRADED` status and indices are not replicated. The OpenSearch logs contain the following error:

```
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

**Solution**:

1. Scale down all pods related to OpenSearch (`master`, `data`, `ingest`, `arbiter`) on the `standby` side.
2. Clear the OpenSearch data on the `standby` side in one of the following ways:
   * Remove OpenSearch persistent volumes.
   * Clear persistent volumes manually.
3. Scale up all pods related to OpenSearch (`master`, `data`, `ingest`, `arbiter`) on the `standby` side.

**Note**: It is safe as you need to perform these steps on the `standby` side. All the data is replicated from the `active` side once the replication process has started successfully.

For more information about this issue, refer to [https://github.com/opensearch-project/cross-cluster-replication/issues/840](https://github.com/opensearch-project/cross-cluster-replication/issues/840).

### Index Is Not Replicated To Standby Side Without Any Errors

| Problem                                           | Severity | Possible Reason                                                                                                                      |
|---------------------------------------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------|
| Index changes stopped replicating to standby side | Average  | Problem index was removed and created again on active side during replication and standby OpenSearch marked replication as `paused`. |

**Solution**:

1. Navigate to the OpenSearch console on `standby` side and run the following command:

      ```bash
      curl -u <username>:<password> -XGET http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/<index_name>/_status?pretty
      ```

   Where:
   * `<username>:<password>` are the credentials to OpenSearch.
   * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.
   * `<index_name>` is the name of missed index. For example, `test_topic`.

   The following response makes it clear that index was removed in active side:

   ```
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
 
**Note**: This option cleans all index data presented on the standby side. Make sure to remove this and check whether OpenSearch on the active side has correct changes.
