This section provides detailed troubleshooting procedures for the OpenSearch cluster.

# Cluster State

OpenSearch monitoring has the following two types of cluster states:

* Cluster Health
* Cluster Status

## Cluster Health

OpenSearch provides a default metric that indicates cluster state. It is called *cluster health*. To check the health of the cluster, the cluster health API can be used.

For more information on the cluster health API, refer to the official OpenSearch documentation, _Cluster Health_ [https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health](https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health).

To check the health of the cluster, the following API can be used:

```
curl -XGET http://localhost:9200/_cluster/health
```

## Cluster Status

Cluster status is a custom metric that can be found on the *Cluster status* panel in Grafana. Possible values for cluster status are:

* `UP` - Cluster health status is GREEN and all nodes working.
* `DEGRADED` - Cluster has YELLOW health status or one node is failed.
* `FAILED` - Cluster has RED health status.

# Common Problems

The following information describes the common problems you may encounter.

## Cluster Status is N/A

*N/A* status while monitoring indicates that the OpenSearch cluster is unreachable.

The main cause is any of the following:

* The cluster is down.
* The monitoring agent is not deployed.
* The monitoring agent is down.

To resolve the issue, navigate to the OpenShift console and check the service state. In the simplest scenario, starting the service solves the issue. In the event of a permanent failure, try to redeploy the cluster or recover it from the backup.

## Cluster Status is Failed or Degraded

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

## Data Nodes are Out of Space

If all data nodes are running low on disk space, more data nodes should be added to the cluster. Be sure that all indices have enough primary shards to be able to balance their data across all those nodes. However, if only some nodes are running out of disk space, this is usually a sign that an index was initialized with too few shards. If an index is composed of a few very large shards, it is hard for OpenSearch to distribute these shards across nodes in a balanced manner.

For more information, refer to [Disk Filled on All Nodes](scenarios/disk_filled_on_all_nodes.md).

## Lack of Resources

Some problems with the OpenSearch cluster can occur due to a lack of CPU, memory, and disk resources. For more information, refer to [Memory Limit](/documentation/maintenance-guide/troubleshooting-guide/scenarios/memory_limit.md), [CPU Overload](/documentation/maintenance-guide/troubleshooting-guide/scenarios/cpu_overload.md) and [I/O Limit](/documentation/maintenance-guide/troubleshooting-guide/scenarios/io_limit.md).

## OpenSearch Fails Down with CircuitBreakingException

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

## Data Files are Corrupted

This section describes the troubleshooting to be done if the data files are corrupted.

### Data Files are Corrupted On Primary Shard

Index has no replica shards, get queries return incomplete data, update queries fail, and some primary shards are in unassigned status with `CorruptIndexException`. For more details and troubleshooting procedures, refer to [Data Files Corrupted on Primary Shard](scenarios/data_files_corrupted_on_primary_shard.md).

### Data Files are Corrupted On Replica Shard

OpenSearch withstands all cases with corrupted replica shards and repairs itself without any data loss. For more information, refer to [Data Files Corrupted on Replica Shard](scenarios/data_files_corrupted_on_replica_shard.md).

### Data Files are Corrupted On Entire Index

If all data files of the index were corrupted, there is no way to get data from this index. The only solution is to restore this index from a backup, provided one exists. For more details and troubleshooting procedures, refer to [Entire Index Corrupted](scenarios/entire_index_corrupted.md).

## Translog Corrupted

To prevent data loss, each shard has a transaction log, or translog, associated with it. If a shard is failed, the most recent transactions can be replayed from the transaction log when the shard recovers.

In some cases (such as a bad drive or user error), the translog can become corrupted. When this corruption is detected by OpenSearch due to mismatching checksums, OpenSearch will fail the shard and refuse to allocate that copy of the data to the node, recovering from a replica if available.

If a translog was corrupted, the shards with a corrupted translog will have `TranslogCorruptedException` in `unassigned.details`.

For more details and troubleshooting procedures, refer to [Translog Is Corrupted](scenarios/translog_is_corrupted.md).

## Other Problems

Other problem descriptions and troubleshooting procedures can be found in the following chapters:

* [New Master Cannot Be Elected](scenarios/new_master_can_not_be_elected.md)
* [Elected Master Is Crashed](scenarios/elected_master_is_crashed.md)
* [Problem During Replication](scenarios/problem_during_replication.md)
* [Primary Shard Is Down During User Request](scenarios/primary_shard_is_down_during_user_request.md)
* [Network Connection Is Lost and Restored](scenarios/network_connection_failure.md)
* [Availability Zone Outage](scenarios/availability_zone_outage.md)
* [Availability Zone Shutdown and Startup](scenarios/availability_zone_shutdown.md)

#  OpenSearch node does not start

There are situations when the starting of the OpenSearch service fails with error after few unsuccessful attempts.

These errors can be found in `Pod` - `Events` tab of the OpenSearch deployment.

## Readiness Probe Failed

This error means the OpenSearch service did not have time to start for a given timeout. This may indicate a lack of resources
or problem with environment where OpenSearch has been deployed.

This error can be solved by increasing the initial delay to check readiness of the service.

You can find this value in:

`OpenSearch StatefulSet` - `Actions` - `Edit YAML` - `readinessProbe.initialDelaySeconds:`

Try to increase this value twice.

Retry this action for all OpenSearch resources which pods have this error.

## Liveness Probe Failed

This error means the OpenSearch service did not have time to ready for work for a given timeout. This may indicate a lack of resources
or problem with environment where OpenSearch has been deployed.

This error can be solved by increasing the initial delay to check liveness of the service.

You can find this value in:

`OpenSearch StatefulSet` - `Actions` - `Edit YAML` - `livenessProbe.initialDelaySeconds:`

Try to increase this value twice.

Retry this action for all OpenSearch resources which pods have this error.

## Max Virtual Memory Is Too Low

```
ERROR: [1] bootstrap checks failed
[1]: max virtual memory areas vm.max_map_count [65530] is too low, increase to at least [262144]
```

This error means the OpenSearch does not have enough virtual memory to start.

To resolve it you need execute the following command on all Kubernetes/OpenShift nodes, where OpenSearch is running:

```
sysctl -w vm.max_map_count=262144
```

## Container Failed with Error: container has runAsNonRoot and image will run as root

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

## CRD Creation Failed on OpenShift 3.11

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

# DBaaS Adapter Health

OpenSearch monitoring has `DBaaS Adapter Status` indicator of DBaaS Adapter health.

## Common Problems

The following information describes the common problems you may encounter.

### DBaaS Adapter Status Is Down

The `Down` state means one of the following problems:
* DBaaS Adapter is not alive. See [DBaaS Is Down alarm description](#opensearch-dbaas-is-down).
* OpenSearch has `red` state. See [OpenSearch is Down alarm description](#opensearch-is-down).

### DBaaS Adapter Status Is Warning

The `Warning` state means the following problem:
* OpenSearch has `yellow` state. See [OpenSearch is Degraded alarm description](#opensearch-is-degraded).

### DBaaS Adapter Status Is Problem

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

# Alarms and Events

Alarm notifications from Zabbix are used to notify operators about problems with OpenSearch.

For example, a message from Zabbix triggered for excessive CPU usage is as follows:

`
Trigger: OpenSearch's CPU usage is above 95%
Trigger status: PROBLEM
Trigger severity: Warning
`

This message may be followed by additional information about the trigger event.

## Common Principles

The following is the list of alarm severity levels and their description:

* The `High` alarm depicts service outage or malfunction. It requires rapid action to restore the system operability.
* The `Average` alarm depicts potential service instability or partially broken functionality. It requires attention and repair action.
* The `Warning` alarm depicts that the probability of a connected problem has increased. It requires attention and action to prevent the problem.

The description of the alarms monitored and raised by Zabbix are as follows:

### OpenSearch is Down

|Problem|Severity|Possible Reasons|
|---|---|---|
|OpenSearch is Down|High|<ul><li>OpenSearch is unresponsive.</li><li>Disk failure</li><li>Lack of memory or CPU.</li><li>Long garbage collection time.</li><li>One or more primary shards are not allocated in the cluster.</li></ul>|

**Solution:**

1. Check the health of the cluster, using the following API:

   ```
   curl -X GET 'http://localhost:9200/_cluster/health'
   ```

   The **red** status specifies that some primary shards are not allocated.
   Make sure that the **number_of_nodes** value is enough for specified replication factor. If there are some shards in the `relocating` or `initializing` status, wait until the process ends.

2. If the shards are unassigned permanently, check the service state in OpenShift:

   Navigate to **OpenShift > 'opensearch-service' project**.

3. If the OpenSearch cluster or monitoring agent failed, restart the service:

   Navigate to the OpenShift console and run the following command:

   ```
   oc delete pod <node name>
   ```

   During recovery, the cluster may be in `DEGRADED` or `FAILED` state. Wait until the shards regain the `ACTIVE` status. If there are enough OpenSearch nodes according to replication factor, shards must be allocated.

4. In case of permanent failure, redeploy the service or recover it from backup:

   Navigate to **Applications > Deployments > opensearch-n or opensearch-monitoring > Deploy**.

5. If the cluster is `running`, check the resource consumption on the monitoring dashboard:

    * Navigate to Grafana, select **OpenSearch cluster** dashboard and specify required **Cloud** parameter.
    * Check if there are enough resources to work properly using the following tabs: **Disk metrics, Memory metrics, CPU metrics, JVM heap and GC metrics**. Verify if there are any peeks of load by the metrics in the specified period.

6. In case of increased resource consumption, refer to any one of the following for resolution:

    * [OpenSearch CPU usage](#opensearch-cpu-usage)
    * [OpenSearch Memory Usage](#opensearch-memory-usage)
    * [OpenSearch Disk Usage](#opensearch-disk-usage)

   For more information, refer to [Cluster Status is N/A](#cluster-status-is-na) and [Cluster Status is Failed or Degraded](#cluster-status-is-failed-or-degraded).

   If the above steps do not help, the problem is probably not generic, and requires detailed investigation. Contact support.

### OpenSearch is Degraded

|Problem|Severity| Possible Reasons                                                                                                                                             |
|---|---|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
|OpenSearch is Degraded|High| <ul><li>One or more replica shards unassigned.</li><li>Lack of resources. For more information, refer to [OpenSearch is Down](#opensearch-is-down)</li></ul> |

**Solution**:

Execute steps from the [OpenSearch is Down](#opensearch-is-down) section.

Ensure that the `health` command returns **yellow** status. This status specifies that some replica shards are not allocated. All primary shards must be active.

For more information, refer to [Cluster Status is Failed or Degraded](#cluster-status-is-failed-or-degraded).

### OpenSearch CPU usage

|Problem|Severity|Possible Reasons|
|---|---|---|
|OpenSearch’s CPU usage|Warning|<ul><li>Heavy search request.</li><li>Indexing workload.</li></ul>|

**Solution**:

1. Get the statistics of cluster nodes using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats/'
   ```

2. Check CPU usage on the monitoring dashboard:

    * Navigate to Grafana, select the **OpenSearch cluster** dashboard and specify the required **Cloud** parameter.
    * Expand the **CPU metrics** tab and check CPU load in the specified period.

3. Check the required resources and increase the CPU limit if it is still within recommended limits:

    * Navigate to **OpenShift > opensearch-service project > Applications > Deployments > opensearch-n > Actions > Edit Resource Limits**.
    * Specify the CPU Limit and then click **Save**.

4. If the workload is still high, add additional data nodes to redistribute the load.

   Cluster update can be performed as an installation with increased `NODES_COUNT` parameter. For more information, refer to _OpenSearch Installation Procedure_.

   **Warning**: Scaling up a cluster may cause performance issues and may require additional analysis.

   For more information, refer to [CPU Overload](scenarios/cpu_overload.md).

### OpenSearch Memory Usage

|Problem|Severity|Possible Reason|
|---|---|---|
|OpenSearch Memory Usage|Warning|Heavy workload during execution.|

**Solution**:

1. Get the statistics of the cluster nodes using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats'
   ```

2. Check memory usage on the monitoring dashboard:

    * Navigate to Grafana, select the **OpenSearch cluster** dashboard and specify the required **Cloud** parameter.
    * Expand the **Memory metrics** tab and check the CPU load in the specified period.

3. Increase the memory limit if it is still within recommended limits:

    * Navigate to **OpenShift > opensearch-service project > Applications > Deployments > opensearch-n > Actions > Edit Resource Limits**.
    * Specify the memory limit and then click **Save**.

4. If the workload is still high, you can add data nodes to redistribute the load.

   Cluster update can be performed as an installation with increased `NODES_COUNT` parameter. For more information, refer to _OpenSearch Installation Procedure_.

   **Warning**: Scaling up a cluster may cause performance issues and may require additional analysis.

   For more information, refer to [Memory Limit](scenarios/memory_limit.md).

### OpenSearch Disk Usage

|Problem|Severity|Possible Reason|
|---|---|---|
|OpenSearch Disk Usage|High|Low space on the disk.|

**Solution**:

1. Retrieve the statistics of all the nodes in the cluster, using the following command:

   ```
   curl -X GET 'http://localhost:9200/_nodes/stats
   ```

2. Check disk usage on the monitoring dashboard:

    * Navigate to Grafana, select the **OpenSearch cluster** dashboard and specify the required **Cloud** parameter.
    * Expand the **Disk metrics** tab and check the CPU load in the specified period.

3. If all the data nodes are running low on disk space or some disks are failed, you need to add more data nodes to the cluster.

4. When the disks are repaired or added, restart the data node using the following command:

   ```
   oc delete pod <node name>
   ```

   For more information, refer to [Data Nodes are Out of Space](#data-nodes-are-out-of-space).

### OpenSearch DBaaS is Down

|Problem|Severity|Possible Reason|
|---|---|---|
|OpenSearch DBaaS is down|Average|Incorrect credentials provided.|

**Solution**:

1. Check the DBaaS status on the monitoring dashboard:

    * Navigate to Grafana, select the **OpenSearch cluster** dashboard and specify the required **Cloud** parameter.
    * View the DBaaS Adapter status in the **DBaaS Health** tab.

2. Ensure the provided DBaaS credentials are correct:

    * Navigate to **Resources > Secrets**.
    * Open DBaaS credentials in the **Source Secrets** tab.
    * Update the credentials and restart the affected pod using the following command:

      ```
      oc delete pod <node name>
      ```

### OpenSearch Backup Failed

|Problem|Severity|Possible Reason|
|---|---|---|
|OpenSearch backup failed|Average|The last backup execution failed.|

**Solution**:

1. Navigate to the OpenShift console and check the service state:

   Navigate to **OpenShift > opensearch-service' project**.

2. Check the free space amount on the monitoring dashboard:

    * Navigate to Grafana, select the **OpenSearch cluster** dashboard and specify the required **Cloud** parameter.
    * Expand the **Backup** tab and check the space amount in the **Storage Size/Free Space** view.

3. Manual backup can be initiated after problem resolution. For more information, refer to [Manual Backup](../backup/manual-backup-procedure.md).

# OpenSearch Disaster Recovery Health

## OpenSearch Disaster Recovery Health Has Status "DEGRADED"

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

Please, recognize list of `failed_indices`.

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

      the replication is not run on the `active` side for the specified failed `test_topic` index. Then you need to go to the `standby` side of OpenSearch cluster and check the status of replication for above index:

      ```bash
      curl -u <username>:<password> -XGET http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/<index_name>/_status?pretty
      ```

      where:
         * `<username>:<password>` are the credentials to OpenSearch.
         * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.
         * `<index_name>` is the name of failed index. For example, `test_topic`.

   3. If `status` of index replication on `standby` side is `FAILED`, you have to stop corresponding replication with the following command:

      ```bash
      curl -u <username>:<password> -XPOST  http://opensearch.<opensearch_namespace>:9200/_plugins/_replication/<index_name>/_stop -H 'Content-Type: application/json' -d'{}'
      ```

      where:
         * `<username>:<password>` are the credentials to OpenSearch.
         * `<opensearch_namespace>` is the namespace where `standby` side of OpenSearch is located. For example, `opensearch-service`.
         * `<index_name>` is the name of failed index. For example, `test_topic`.

4. For `standby` side switch OpenSearch cluster to the `active` side and return to the `standby` one. This action should restart replication properly. 
