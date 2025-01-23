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

   ```sh
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

For more information, refer to [Data Nodes are Out of Space](./troubleshooting.md#data-nodes-are-out-of-space).

### Possible Causes

- Low space on the disk.
- Failed disks.
- Insufficient disk resources allocated to OpenSearch pods.

### Impact

- Prospective problems with shards allocation and the impossibility to write to the OpenSearch.

### Actions for Investigation

1. Retrieve the statistics of all the nodes in the cluster, using the following command:

   ```sh
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the disk usage trends in OpenSearch monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Increase disk space for OpenSearch if it's possible.
2. If all the data nodes are running low on disk space or some disks are failed, you need to add more data nodes to the cluster.

## OpenSearchDiskUsageAbove85%Alert

### Description

One of OpenSearch pods uses 85% of the disk.

For more information, refer to [Data Nodes are Out of Space](./troubleshooting.md#data-nodes-are-out-of-space).

### Possible Causes

- Low space on the disk.
- Failed disks.
- Insufficient disk resources allocated to OpenSearch pods.

### Impact

- Increased response time and potential slowdown of OpenSearch requests.
- Inability to allocate shards to nodes with exceeded 85% disk limit.

### Actions for Investigation

1. Retrieve the statistics of all the nodes in the cluster, using the following command:

   ```sh
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the disk usage trends in OpenSearch monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Increase disk space for OpenSearch if it's possible.
2. If all the data nodes are running low on disk space or some disks are failed, you need to add more data nodes to the cluster.

## OpenSearchDiskUsageAbove95%Alert

### Description

One of OpenSearch pods uses 95% of the disk.

For more information, refer to [Data Nodes are Out of Space](./troubleshooting.md#data-nodes-are-out-of-space).

### Possible Causes

- Low space on the disk.
- Failed disks.
- Insufficient disk resources allocated to OpenSearch pods.

### Impact

- Inability to write to OpenSearch indices that has one or more shards allocated on the problem node, and that has at least one disk exceeding the flood stage.

### Actions for Investigation

1. Retrieve the statistics of all the nodes in the cluster, using the following command:

   ```sh
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Monitor the disk usage trends in OpenSearch monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Increase disk space for OpenSearch if it's possible.
2. If all the data nodes are running low on disk space or some disks are failed, add more data nodes to the cluster.

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

   ```sh
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

For more information, refer to [Cluster Status is Failed or Degraded](./troubleshooting.md#cluster-status-is-failed-or-degraded).

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

   ```sh
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

For more information, refer to [Cluster Status is N/A](./troubleshooting.md#cluster-status-is-na) and [Cluster Status is Failed or Degraded](./troubleshooting.md#cluster-status-is-failed-or-degraded).

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

   ```sh
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

For more information, refer to [Last Backup Has Failed](./troubleshooting.md#last-backup-has-failed).

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
2. Follow [Last Backup Has Failed](https://github.com/Netcracker/qubership-opensearch/blob/main/docs/public/troubleshooting.md#last-backup-has-failed) for additional steps.

## OpenSearchQueryIsTooSlowAlert

### Description

Execution time of one of index queries in the OpenSearch exceeds the specified threshold.

This threshold can be overridden with parameter `monitoring.thresholds.slowQuerySecondsAlert` described in [OpenSearch monitoring](/docs/public/installation.md#monitoring) parameters.

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

For more information, refer to [OpenSearch Disaster Recovery Health](./troubleshooting.md#opensearch-disaster-recovery-health-has-status-degraded).

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

1. Check solutions described in [OpenSearch Disaster Recovery Health](./troubleshooting.md#opensearch-disaster-recovery-health-has-status-degraded) section.

## OpenSearchReplicationFailedAlert

### Description

Replication between two OpenSearch clusters in Disaster Recovery mode has `failed` status.

For more information, refer to [OpenSearch Disaster Recovery Health](./troubleshooting.md#opensearch-disaster-recovery-health-has-status-degraded).

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

1. Check solutions described in [OpenSearch Disaster Recovery Health](./troubleshooting.md#opensearch-disaster-recovery-health-has-status-degraded) section.

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

This limit can be overridden with parameter `monitoring.thresholds.lagAlert` described in [OpenSearch monitoring](/docs/public/installation.md#monitoring) parameters.

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
