Topics covered in this section:

- [Overview](#overview)
- [OpenSearch Monitoring](#opensearch-monitoring)
- [OpenSearch Indices](#opensearch-indices)
- [OpenSearch Slow Queries](#opensearch-slow-queries)
- [Table of Metrics](#table-of-metrics)
- [Monitoring Alarms Description](#monitoring-alarms-description)

# Overview

This documentation describes monitoring dashboards, their metrics, Zabbix alarms and Prometheus alerts.

The dashboards provide the following parameters to configure at the top of the dashboard:

* Interval time for metric display
* Node name

For all graph panels, the mean metric value is used in the given interval. For all singlestat panels, 
the last metric value is used.

# OpenSearch Monitoring

## Dashboard

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_dashboard.png)

## Metrics

**Cluster Overview**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_cluster_overview.png)

* `Cluster Status` - Status of OpenSearch cluster.

   If the cluster status is `degraded`, at least one replica shard is unallocated or missing. The search
   results will still be complete, but if more shards are missing, you may lose data.
   
   A `failed` cluster status indicates that:
   
    * At least one primary shard is missing.
    * You are missing data.
    * The searches will return partial results.
    * You will be blocked from indexing into that shard.
* `Nodes Status` - Status of OpenSearch nodes.
* `CPU Usage` - The maximum current CPU usage (in percent) among all OpenSearch servers.
* `JVM Heap Usage` - The maximum current JVM heap memory usage (in percent) among all OpenSearch servers.
* `Off-Heap Memory Usage` - The maximum memory usage excluding allocated JVM Heap memory.
* `Cluster Status Transitions` - The transitions of OpenSearch cluster statuses.
* `Pod Readiness Probe Transitions` - The transitions of readiness probes for each OpenSearch pod.

**OpenSearch Shards**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_opensearch_shards.png)

* `Active primary shards` - The number of active primary shards in OpenSearch cluster.
* `Active shards` - The number of active primary and replica shards in OpenSearch cluster.
* `Initializing shards` - The number of shards that are in the `initializing` state. When you first
  create an index, or when a node is rebooted, its shards are briefly in the `initializing` state
  before transitioning to `started` or `unassigned` as the master node attempts to
  assign shards to nodes in the cluster. If you see the shards remain in the `initializing` state for too
  long, it could be a warning sign that your cluster is unstable.
* `Relocating shards` - The number of shards that are relocating now and have the `relocating` state.
* `Unassigned shards` - The number of shards that are not assigned to any node and have the `unassigned`
  state. If you see the shards remain in the `unassigned` state for too long, it could be a warning sign
  that your cluster is unstable.

**OpenSearch Tasks**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_opensearch_tasks.png)

* `Pending tasks` - The number of pending tasks in OpenSearch cluster.
* `Time of most waiting tasks` - The maximum time in milliseconds that task is waiting in queue.

**Network Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_network_metrics.png)

* `Open transport connections` - The number of open transport connections for each OpenSearch node.
* `Open http connections` - The number of open HTTP connections for each OpenSearch node. If the
  total number of open HTTP connections is constantly increasing, it may indicate that your HTTP
  clients are not properly establishing persistent connections. Reestablishing connections adds extra
  milliseconds or even seconds to your request response time.
* `Transport size` - The rate of change between subsequent size values of received (rx) and transmitted
  (tx) packages for each OpenSearch node.

**JVM Heap and GC Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_jvm_heap_and_gc_metrics.png)

* `JVM heap usage` - The usage of JVM heap memory by each OpenSearch node.
* `JVM heap usage percent` - OpenSearch is set up to initiate garbage collections whenever JVM 
  heap usage hits 75 percent. As shown in the preceding image, it may be useful to monitor which nodes exhibit high 
  heap usage, and set up an alert to find out if any node is consistently using over 85 percent of 
  heap memory; this percentage indicates that the rate of garbage collection is not keeping up with the rate of 
  garbage creation. To address this problem, you can either increase your heap size (as long as it 
  remains below the guidelines previously recommended), or scale out the cluster by adding more 
  nodes.
* `JVM non heap usage` - The usage of memory outside the JVM heap by each OpenSearch node.
* `GC time` - Because garbage collection uses resources (in order to free up additional resources), you should 
  keep an eye on its frequency and duration to see if you need to adjust the heap size. Setting 
  the heap too large can result in long garbage collection times, and these excessive pauses are 
  dangerous because they can lead your cluster to mistakenly register your node as having dropped 
  off the grid. (Because the master node checks the status of every other node every 30 seconds, if 
  any node’s garbage collection time exceed 30 seconds, it will lead the master to believe that 
  the node has failed.)

OpenSearch recommends allocating less than 50% of available RAM to JVM heap, and not more than 32 GB
to JVM heap. This metric can be found in the JVM Heap usage and JVM Heap usage percent panels.

The following is a list of scenarios related to JVM Heap:

* Heap Size is Too Small

   If the heap size is too small, the application is prone to "Out of Memory" errors. While this is
   the most serious risk issuing from an undersized heap, there are additional issues that can arise
   if a heap is too small. A heap that is smaller than the application's allocation rate leads to
   *frequent small latency spikes* and reduced throughput from constant garbage collection pauses.
   Frequent short pauses impact the end user experience by shifting the latency distribution and reducing
   the number of operations the application can handle. For OpenSearch, constant short pauses reduce
   the number of indexing operations and queries handled per second. A small heap also reduces the memory
   available for indexing buffers, caches, and memory-hungry features like aggregations and suggesters. 
   The following image shows a heap that is too small. The garbage collections are barely able to free objects,
   leaving little heap space free after each collection.
   
   ![Small Heap](/documentation/maintenance-guide/monitoring/pictures/small_heap.png)

* Heap Size is Too Large

   If the heap is too large, the application is prone to *infrequent long latency spikes* from full-heap
   garbage collections. Infrequent long pauses impact the end user experience, as these pauses increase
   the tail of the latency distribution. As a result, the user requests sometimes has long response times.
   Long pauses are especially detrimental to a distributed system like OpenSearch. A long pause is
   indistinguishable from a node that is unreachable because it is frozen or otherwise isolated from the
   cluster. During a stop-the-world pause, no OpenSearch server code is executed. In case of an elected
   master, a long garbage collection pause can cause other nodes to stop following the master and elect
   a new node. In case of a data node, a long garbage collection pause can lead to the master removing
   the node from the cluster and reallocating the paused node's assigned shards. That increases network
   traffic and disk I/O across the cluster. Long garbage collection pauses are a major cause of cluster
   instability.
   
   The following image shows a heap that is too large. The heap is almost exclusively junk before each
   collection, and this memory is likely better utilized by the filesystem cache.
   
   ![Oversize Heap](/documentation/maintenance-guide/monitoring/pictures/oversized-heap.png)

**Memory Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_memory_metrics.png)

* `OS memory usage` - The usage of memory by each OpenSearch node and its limit. These metric is
  useful to avoid reaching the memory limit on nodes.
* `Off-heap memory usage` - The usage of memory excluding allocated JVM Heap memory by each OpenSearch node and its limit.
  This amount of memory is used by operating system and Apache Lucene.

**Disk Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_disk_metrics.png)

* `Disk usage in percent` - The usage of disk space in percent for an OpenSearch node.
* `Disk usage` - The usage of disk space allocated to each OpenSearch node and its limit.
* `Disk I/O operations` - The rate of change between subsequent values of input/output operations per second for each OpenSearch node.
* `Disk I/O usage` - The rate of change between subsequent values of disk usage for each OpenSearch node.
* `Open File Descriptors` - The amount of open file descriptors for each OpenSearch node.

As these metrics depend on other processes besides OpenSearch, it is useful to know the address
of the node, so these metrics are grouped by both the node name and the node host.

**CPU Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_cpu_metrics.png)

* `CPU load average (5 min)` - Five-minute CPU load average on the system.
* `CPU load in percent` - The usage of CPU in percent for each OpenSearch node.
* `CPU load by pod` - The usage of CPU by each OpenSearch node and its limit.

These metrics are useful to monitor CPU usage and avoid CPU overload. Like the disk metrics, they are
grouped by both the node name and the node host.

**Indices Statistics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_indices_statistics.png)

* `Indices total operations` - The number of operations performed by indices by the current moment 
  grouped by operation type and OpenSearch node.
* `Indices operations rate` - The number of operations performed by indices per second grouped by operation type and OpenSearch node.
* `Indices time operations` - The average time to complete an operation in indices grouped by 
  operation type and OpenSearch node.
* `Indices time operations rate` - The average time to complete an operation in indices per second grouped by operation type and OpenSearch node.
* `Indices data size` - The size of data stored in indices grouped by OpenSearch nodes.
* `Indices documents count` - The number of documents stored in indices grouped by OpenSearch nodes.
* `Indices documents rate` - The number of documents added to indices per second grouped by OpenSearch nodes.

**Thread Pool Queues and Requests Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_thread_pool_queues_and_requests_metrics.png)

* `Requests latency` - The information on the requests, indexing, flush, get-exists, search query,
  and fetch query. The fetch phase typically takes much less time than the query phase.
  If you notice the indexing latency increasing, you may be trying to index too 
  many documents at one time (OpenSearch's documentation recommends starting with a bulk indexing 
  size of 5 to 15 megabytes and increasing slowly from there). If you are planning to index many 
  documents, and you do not need the new information to be immediately available for search, you can 
  optimize for indexing performance over search performance by decreasing the refresh frequency until 
  you are done indexing.
* `Rejected requests` - The size of each thread pool's queue represents how many requests are 
  waiting to be served while the node is currently at capacity. The queue allows the node to track 
  and eventually serve these requests instead of discarding them. Thread pool rejections arise once 
  the thread pool's maximum queue size (which varies based on the type of thread pool) is reached.
* `Thread pool queues` - The size of each thread pool's queue represents how many requests are 
  waiting to be served while the node is currently at capacity. Large queues are not ideal because
  they use up resources and increase the risk of losing requests if a node goes down. If you see the
  number of queued and rejected threads increasing steadily, you might want to try slowing down the
  rate of requests (if possible), increasing the number of processors on your nodes, or increasing
  the number of nodes in the cluster. Query load spikes correlate with spikes in the search thread
  pool queue size, as the node attempts to keep up with the rate of query requests.

OpenSearch nodes use the thread pools to manage how threads consume memory and CPU. Since thread
pool settings are automatically configured based on the number of processors, you need not change them.
However, it is recommended to monitor the queues and rejections to find out if your nodes are not able 
to keep up.

**Backup**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_backup.png)

* `Backup Daemon Status` - The current activity status of the backup daemon. The activity status
  can be one of following:
    * Not Active - There is no running backup process.
    * In Progress/Started - The backup process is running.
* `Last Backup Status` - The state of the last backup. The backup status can be one of following:
    * SUCCESS - There is at least one successful backup, and the latest backup is successful.
    * FAILED - There are no successful backups, or the last backup failed.
    * INCOMPATIBLE - The snapshot was created with an old version of OpenSearch, and therefore is
      incompatible with the current version of the cluster.
* `Time of Last Backup` - The period of time when the last backup process was ended.
* `Backup Daemon Status Transitions` - The transitions of backup daemon activity status. The activity status
  can be one of following:
    * Not Active - There is no running backup process.
    * In Progress/Started - The backup process is running.
* `Storage Type` - The backup storage type. The storage type can be one of following:
    * Persistent Volume - The volume is mounted to the pod of the backup daemon. The daemon does not
      recognise what is underlying in the directory, which may be NFS, Cinder, or something else.
    * AWS S3 - "Cloud" high-available storage.
* `Successful Backup Versions Count` - The number of successful backups.
* `Time of Last Successful Backup` - The period of time when the last successful backup process was
  ended.
* `Backup Versions Count` - The number of available backups.
* `Storage Size/Free Space` - The space occupied by backups and the remaining amount of space.
  Not all storage supports total size, so "Total Volume Space" metrics can be zeroed.
* `Backup Activity Status` - The changes of activity status on the chart.
* `Time Spent on Backup` - Time spent on last backup.
* `Backup Last Version Size` - The size of the last backup.

**DBaaS Health**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_dbaas_health.png)

* `DBaaS Adapter Status` - The status of DBaaS Adapter.
* `DBaaS OpenSearch Cluster Status` - The status of OpenSearch cluster from the DBaaS Adapter side.
* `DBaaS Adapter Status Transitions` - The transitions of DBaaS Adapter status.
* `DBaaS OpenSearch Cluster Status Transitions` - The transitions of OpenSearch cluster status from the DBaaS Adapter side.

# OpenSearch Indices

This section describes the `OpenSearch Indices` dashboard and its metrics.

## Dashboard

An overview of `OpenSearch Indices` dashboard is shown below.

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-indices_dashboard.png)

## Metrics

**Overview**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-indices_overview.png)

* `Indices information` - The number of documents for each index and its size in bytes in descending order of size values presented as a table.

**Incoming Documents Rate per Index**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-indices_incoming_documents_rate.png)

* ${INDEX_NAME} - The incoming documents rate on primary shards.

Where `${INDEX_NAME}` is the name of index which metrics are presented in the widget.

**Store Size per Index**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-indices_store_size.png)

* ${INDEX_NAME} - The size in bytes occupied by index on primary shards and in total.

Where `${INDEX_NAME}` is the name of index which metrics are presented in the widget.

**Indexing Documents Rate per Index**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-indices_indexing_documents_rate.png)

* ${INDEX_NAME} - The indexing documents rate on primary shards.

Where `${INDEX_NAME}` is the name of index which metrics are presented in the widget.

**Deleting Documents Rate per Index**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-indices_deleting_documents_rate.png)

* ${INDEX_NAME} - The deleting documents rate on primary shards.

Where `${INDEX_NAME}` is the name of index which metrics are presented in the widget.

# OpenSearch Slow Queries

This section describes the `OpenSearch Slow Queries` dashboard and its metrics.

## Dashboard

An overview of `OpenSearch Slow Queries` dashboard is shown below.

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-slow-queries_dashboard.png)

## Metrics

**Overview**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-slow-queries_overview.png)

* `Slow Queries Information` - The slowest queries in processing interval with index name, shard, query, start time, number of found documents and spent time in descending order of spent time.
* `Slowest Query` - The time of the slowest query in processing interval.

# Table of Metrics

This table provides full list of Prometheus metrics being collected by OpenSearch Monitoring.

| Metric name                                                  | Description                                                                                                                                                                                                                                                                                      | Amazon        | Netcracker OpenSearch Service |
|--------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|-------------------------------|
| container_cpu_usage_seconds_total                            | The amount of CPU (in seconds) used by container                                                                                                                                                                                                                                                 | Not supported | Supported                     |
| container_memory_usage_bytes                                 | The amount of memory (in bytes) used by container                                                                                                                                                                                                                                                | Not supported | Supported                     |
| container_memory_working_set_bytes                           | The amount of working set memory (in bytes), that includes recently accessed memory, dirty memory, and kernel memory                                                                                                                                                                             | Not supported | Supported                     |
| container_spec_memory_limit_bytes                            | The amount of memory (in bytes) that the container is limited to                                                                                                                                                                                                                                 | Not supported | Supported                     |
| kube_pod_container_resource_limits_cpu_cores                 | The total CPU limits (in cores) of a pod                                                                                                                                                                                                                                                         | Not supported | Supported                     |
| kube_pod_container_resource_limits_memory_bytes              | The total memory limits (in bytes) of a pod                                                                                                                                                                                                                                                      | Not supported | Supported                     |
| kube_pod_status_ready                                        | Whether the pod is ready to serve requests                                                                                                                                                                                                                                                       | Not supported | Supported                     |
| opensearch_backups_metric_count_of_snapshots                 | The number of available snapshots in OpenSearch                                                                                                                                                                                                                                                  | Supported     | Supported                     |
| opensearch_backups_metric_count_of_successful_snapshots      | The number of successful snapshots in OpenSearch                                                                                                                                                                                                                                                 | Supported     | Supported                     |
| opensearch_backups_metric_current_status                     | The current activity status of the backup daemon. The activity status can be the following: `Not Active` (there is no running backup process), `In Progress/Started` (the backup process is running)                                                                                             | Supported     | Supported                     |
| opensearch_backups_metric_last_backup_size                   | The size (in bytes) of the last created snapshot                                                                                                                                                                                                                                                 | Supported     | Supported                     |
| opensearch_backups_metric_last_backup_status                 | The state of the last snapshot. The snapshot status can be the following: `SUCCESS` (there is at least one successful backup, and the latest backup is successful), `FAILED` (there are no successful backups, or the last backup failed)                                                        | Supported     | Supported                     |
| opensearch_backups_metric_last_backup_time_spent             | The time (in milliseconds) spent on the last snapshot creation                                                                                                                                                                                                                                   | Supported     | Supported                     |
| opensearch_backups_metric_last_snapshot_time                 | The period of time when the last snapshot process ended                                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_backups_metric_storage_type                       | The type of storage where snapshots are located. The possible values are `AWS S3` and `Persistent Volume`                                                                                                                                                                                        | Supported     | Supported                     |
| opensearch_cluster_health_active_primary_shards              | The number of active primary shards in OpenSearch                                                                                                                                                                                                                                                | Supported     | Supported                     |
| opensearch_cluster_health_active_shards                      | The number of active primary and replica shards in OpenSearch                                                                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_cluster_health_initializing_shards                | The number of shards that are in the `initializing` state. When you first create an index, or when a node is rebooted, its shards are briefly in the `initializing` state before transitioning to `started` or `unassigned` as the master node attempts to assign shards to nodes in the cluster | Supported     | Supported                     |
| opensearch_cluster_health_number_of_nodes                    | The number of alive OpenSearch nodes                                                                                                                                                                                                                                                             | Supported     | Supported                     |
| opensearch_cluster_health_number_of_pending_tasks            | The number of tasks in `pending` status                                                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_cluster_health_relocating_shards                  | The number of shards that are relocating now and have the `relocating` state                                                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_cluster_health_status_code                        | The status of OpenSearch cluster. The possible values are `UP`, `DEGRADED`, `FAILED`                                                                                                                                                                                                             | Supported     | Supported                     |
| opensearch_cluster_health_task_max_waiting_in_queue_millis   | The maximum time (in milliseconds) that task is waiting in queue                                                                                                                                                                                                                                 | Supported     | Supported                     |
| opensearch_cluster_health_unassigned_shards                  | The number of shards that are not assigned to any node and have the `unassigned` state                                                                                                                                                                                                           | Supported     | Supported                     |
| opensearch_dbaas_health_elastic_cluster_status               | The status of OpenSearch cluster from the DBaaS adapter side                                                                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_dbaas_health_status                               | The status of DBaaS adapter                                                                                                                                                                                                                                                                      | Supported     | Supported                     |
| opensearch_fs_io_stats_total_read_kilobytes                  | The amount of disk space (in kilobytes) used by `read` operations on OpenSearch node                                                                                                                                                                                                             | Supported     | Supported                     |
| opensearch_fs_io_stats_total_read_operations                 | The number of all `read` operations performed on OpenSearch node                                                                                                                                                                                                                                 | Supported     | Supported                     |
| opensearch_fs_io_stats_total_write_kilobytes                 | The amount of disk space (in kilobytes) used by `write` operations on OpenSearch node                                                                                                                                                                                                            | Supported     | Supported                     |
| opensearch_fs_io_stats_total_write_operations                | The number of all `write` operations performed on OpenSearch node                                                                                                                                                                                                                                | Supported     | Supported                     |
| opensearch_fs_total_available_in_bytes                       | The amount of disk space (in bytes) that is available to use                                                                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_fs_total_free_in_bytes                            | The amount of disk space (in bytes) that is free to use                                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_fs_total_total_in_bytes                           | The total amount of disk space (in bytes)                                                                                                                                                                                                                                                        | Supported     | Supported                     |
| opensearch_http_current_open                                 | The number of open HTTP connections on OpenSearch node                                                                                                                                                                                                                                           | Supported     | Supported                     |
| opensearch_indices_docs_count                                | The number of documents disposed on OpenSearch nodes                                                                                                                                                                                                                                             | Supported     | Supported                     |
| opensearch_indices_flush_total                               | The number of flushes performed on OpenSearch nodes                                                                                                                                                                                                                                              | Supported     | Supported                     |
| opensearch_indices_flush_total_time_in_millis                | The time (in milliseconds) spent on flushes on OpenSearch nodes                                                                                                                                                                                                                                  | Supported     | Supported                     |
| opensearch_indices_get_current                               | The number of `get` operations performed by indices at the moment                                                                                                                                                                                                                                | Supported     | Supported                     |
| opensearch_indices_get_exists_time_in_millis                 | The average time to complete a `get exists` operation in indices                                                                                                                                                                                                                                 | Supported     | Supported                     |
| opensearch_indices_get_exists_total                          | The number of `get exists` operations performed by indices during all time                                                                                                                                                                                                                       | Supported     | Supported                     |
| opensearch_indices_get_missing_time_in_millis                | The average time to complete a `get missing` operation in indices                                                                                                                                                                                                                                | Supported     | Supported                     |
| opensearch_indices_get_missing_total                         | The number of `get missing` operations performed by indices during all time                                                                                                                                                                                                                      | Supported     | Supported                     |
| opensearch_indices_get_time_in_millis                        | The average time to complete a `get` operation in indices                                                                                                                                                                                                                                        | Supported     | Supported                     |
| opensearch_indices_get_total                                 | The number of `get` operations performed by indices during all time                                                                                                                                                                                                                              | Supported     | Supported                     |
| opensearch_indices_indexing_delete_time_in_millis            | The average time to complete an `indexing delete` operation in indices                                                                                                                                                                                                                           | Supported     | Supported                     |
| opensearch_indices_indexing_delete_total                     | The number of `indexing delete` operations performed by indices during all time                                                                                                                                                                                                                  | Supported     | Supported                     |
| opensearch_indices_indexing_index_current                    | The number of `indexing index` operations performed by indices at the moment                                                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_indices_indexing_index_time_in_millis             | The average time to complete an `indexing index` operation in indices                                                                                                                                                                                                                            | Supported     | Supported                     |
| opensearch_indices_indexing_index_total                      | The number of `indexing index` operations performed by indices during all time                                                                                                                                                                                                                   | Supported     | Supported                     |
| opensearch_indices_indexing_noop_update_total                | The number of `indexing noop update` operations performed by indices during all time                                                                                                                                                                                                             | Supported     | Supported                     |
| opensearch_indices_indexing_throttle_time_in_millis          | The average time to complete an `indexing throttle` operation in indices                                                                                                                                                                                                                         | Supported     | Supported                     |
| opensearch_indices_merges_total                              | The number of `merges` operations performed by indices during all time                                                                                                                                                                                                                           | Supported     | Supported                     |
| opensearch_indices_merges_total_stopped_time_in_millis       | The average time to complete a `merges total stopped` operation in indices                                                                                                                                                                                                                       | Supported     | Supported                     |
| opensearch_indices_merges_total_throttled_time_in_millis     | The average time to complete a `merges total throttled` operation in indices                                                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_indices_merges_total_time_in_millis               | The average time to complete a `merges total` operation in indices                                                                                                                                                                                                                               | Supported     | Supported                     |
| opensearch_indices_percolate_time_in_millis                  | The average time to complete a `percolate` operation in indices                                                                                                                                                                                                                                  | Supported     | Supported                     |
| opensearch_indices_percolate_total                           | The number of `percolate` operations performed by indices during all time                                                                                                                                                                                                                        | Supported     | Supported                     |
| opensearch_indices_recovery_throttle_time_in_millis          | The average time to complete a `recovery throttle` operation in indices                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_indices_refresh_total                             | The number of `refresh` operations performed by indices during all time                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_indices_refresh_total_time_in_millis              | The average time to complete a `refresh total` operation in indices                                                                                                                                                                                                                              | Supported     | Supported                     |
| opensearch_indices_search_fetch_current                      | The number of `search fetch` operations performed by indices at the moment                                                                                                                                                                                                                       | Supported     | Supported                     |
| opensearch_indices_search_fetch_time_in_millis               | The average time to complete a `search fetch` operation in indices                                                                                                                                                                                                                               | Supported     | Supported                     |
| opensearch_indices_search_fetch_total                        | The number of `search fetch` operations performed by indices during all time                                                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_indices_search_query_current                      | The number of `search query` operations performed by indices at the moment                                                                                                                                                                                                                       | Supported     | Supported                     |
| opensearch_indices_search_query_time_in_millis               | The average time to complete a `search query` operation in indices                                                                                                                                                                                                                               | Supported     | Supported                     |
| opensearch_indices_search_query_total                        | The number of `search query` operations performed by indices during all time                                                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_indices_search_scroll_time_in_millis              | The average time to complete a `search scroll` operation in indices                                                                                                                                                                                                                              | Supported     | Supported                     |
| opensearch_indices_search_scroll_total                       | The number of `search scroll` operations performed by indices during all time                                                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_indices_stats_primaries_docs_count                | The number of documents on primary shards of OpenSearch index                                                                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_indices_stats_primaries_docs_deleted              | The number of deleted documents on primary shards of OpenSearch index                                                                                                                                                                                                                            | Supported     | Supported                     |
| opensearch_indices_stats_primaries_indexing_index_total      | The number of `indexing` operations on primary shards of OpenSearch index                                                                                                                                                                                                                        | Supported     | Supported                     |
| opensearch_indices_stats_primaries_store_size_in_bytes       | The store size (in bytes) occupied by primary shards of OpenSearch index                                                                                                                                                                                                                         | Supported     | Supported                     |
| opensearch_indices_stats_total_store_size_in_bytes           | The store size (in bytes) occupied by primary and replica shards of OpenSearch index                                                                                                                                                                                                             | Supported     | Supported                     |
| opensearch_indices_store_size_in_bytes                       | The store size (in bytes) of OpenSearch index                                                                                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_indices_store_throttle_time_in_millis             | The average time to complete a `store throttle` operation in indices                                                                                                                                                                                                                             | Supported     | Supported                     |
| opensearch_indices_suggest_time_in_millis                    | The average time to complete a `suggest` operation in indices                                                                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_indices_suggest_total                             | The number of `suggest` operations performed by indices during all time                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_indices_warmer_total                              | The number of `warmer` operations performed by indices during all time                                                                                                                                                                                                                           | Supported     | Supported                     |
| opensearch_indices_warmer_total_time_in_millis               | The average time to complete a `warmer total` operation in indices                                                                                                                                                                                                                               | Supported     | Supported                     |
| opensearch_jvm_gc_collectors_old_collection_time_in_millis   | The time (in milliseconds) spent on major GCs that collect old generation objects in JVM                                                                                                                                                                                                         | Supported     | Supported                     |
| opensearch_jvm_gc_collectors_young_collection_time_in_millis | The time (in milliseconds) spent on minor GCs that collect young generation objects in JVM                                                                                                                                                                                                       | Supported     | Supported                     |
| opensearch_jvm_mem_heap_max_in_bytes                         | The amount of JVM heap memory (in bytes) allocated for OpenSearch nodes                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_jvm_mem_heap_used_in_bytes                        | The amount of JVM heap memory (in bytes) used by OpenSearch nodes                                                                                                                                                                                                                                | Supported     | Supported                     |
| opensearch_jvm_mem_heap_used_percent                         | The usage of JVM heap memory (in percent) by OpenSearch nodes                                                                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_jvm_mem_non_heap_used_in_bytes                    | The amount of memory outside the JVM heap (in bytes) used by OpenSearch nodes                                                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_os_cpu_load_average_5m                            | The load average on the OpenSearch system for 5 minutes                                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_process_cpu_percent                               | The usage of CPU (in percent) by OpenSearch nodes                                                                                                                                                                                                                                                | Supported     | Supported                     |
| opensearch_process_open_file_descriptors                     | The number of open file descriptors by OpenSearch nodes                                                                                                                                                                                                                                          | Supported     | Supported                     |
| opensearch_thread_pool_flush_queue                           | The size of `flush` thread pool’s queue that represents how many requests are waiting to be served while the node is currently at capacity                                                                                                                                                       | Supported     | Supported                     |
| opensearch_thread_pool_get_queue                             | The size of `get` thread pool’s queue that represents how many requests are waiting to be served while the node is currently at capacity                                                                                                                                                         | Supported     | Supported                     |
| opensearch_thread_pool_get_rejected                          | The number of `get` thread pool rejections that arise once the thread pool’s maximum queue size is reached                                                                                                                                                                                       | Supported     | Supported                     |
| opensearch_thread_pool_index_queue                           | The size of `index` thread pool’s queue that represents how many requests are waiting to be served while the node is currently at capacity                                                                                                                                                       | Supported     | Supported                     |
| opensearch_thread_pool_index_rejected                        | The number of `index` thread pool rejections that arise once the thread pool’s maximum queue size is reached                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_thread_pool_refresh_queue                         | The size of `refresh` thread pool’s queue that represents how many requests are waiting to be served while the node is currently at capacity                                                                                                                                                     | Supported     | Supported                     |
| opensearch_thread_pool_search_queue                          | The size of `search` thread pool’s queue that represents how many requests are waiting to be served while the node is currently at capacity                                                                                                                                                      | Supported     | Supported                     |
| opensearch_thread_pool_search_rejected                       | The number of `search` thread pool rejections that arise once the thread pool’s maximum queue size is reached                                                                                                                                                                                    | Supported     | Supported                     |
| opensearch_thread_pool_write_queue                           | The size of `write` thread pool’s queue that represents how many requests are waiting to be served while the node is currently at capacity                                                                                                                                                       | Supported     | Supported                     |
| opensearch_thread_pool_write_rejected                        | The number of `write` thread pool rejections that arise once the thread pool’s maximum queue size is reached                                                                                                                                                                                     | Supported     | Supported                     |
| opensearch_transport_rx_size_in_bytes                        | The size of received packages (in bytes) by OpenSearch nodes                                                                                                                                                                                                                                     | Not supported | Supported                     |
| opensearch_transport_server_open                             | The number of open transport connections on OpenSearch nodes                                                                                                                                                                                                                                     | Not supported | Supported                     |
| opensearch_transport_tx_size_in_bytes                        | The size of transmitted packages (in bytes) by OpenSearch nodes                                                                                                                                                                                                                                  | Not supported | Supported                     |
| opensearch_transport_tx_size_in_bytes                        | The size of transmitted packages (in bytes) by OpenSearch nodes                                                                                                                                                                                                                                  | Not supported | Supported                     |
| opensearch_slow_query_took_millis                            | The time in milliseconds spent on a particular slow query                                                                                                                                                                                                                                        | Not supported | Supported                     |

# Monitoring Alarms Description

This section describes monitoring alarms. There are two ways to monitor OpenSearch Service - InfluxDB flow and 
Prometheus flow. For first one Zabbix is used as an alarm system and all notifications are called "alarm". For 
second one Alert Manager is used. The Alert Manager is based on Prometheus rules and all its notifications are 
called "alert". Anyway in both approaches OpenSearch Service has the same alarms/alerts and single troubleshooting 
guide to resolve issues.  

| Zabbix Alarm                                  | Alert Manager alert                | Possible Reasons                                                                                                                                                                                                                                                             | Troubleshooting Guide link                                                              |
|-----------------------------------------------|------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------|
| OpenSearch's CPU usage is above 95%           | OpenSearchCPULoadAlert             | CPU usage by one of the pods in the OpenSearch cluster comes close to the limit                                                                                                                                                                                              | [OpenSearch’s CPU usage](../troubleshooting-guide/README.md#opensearch-cpu-usage)       |
|                                               | OpenSearchDiskUsageAbove75%Alert   | Disk usage by one of the pods in the OpenSearch cluster comes close to the 75% limit                                                                                                                                                                                         | [OpenSearch Disk Usage](../troubleshooting-guide/README.md#opensearch-disk-usage)       |
|                                               | OpenSearchDiskUsageAbove85%Alert   | Disk usage by one of the pods in the OpenSearch cluster comes close to the 85% limit                                                                                                                                                                                         | [OpenSearch Disk Usage](../troubleshooting-guide/README.md#opensearch-disk-usage)       |
|                                               | OpenSearchDiskUsageAbove95%Alert   | Disk usage by one of the pods in the OpenSearch cluster comes close to the 95% limit                                                                                                                                                                                         | [OpenSearch Disk Usage](../troubleshooting-guide/README.md#opensearch-disk-usage)       |
| OpenSearch's heap memory usage is above 95%   | OpenSearchHeapMemoryUsageAlert     | Heap memory usage by one of the pods in the OpenSearch cluster comes close to the limit                                                                                                                                                                                      | [OpenSearch Memory Usage](../troubleshooting-guide/README.md#opensearch-memory-usage)   |
| OpenSearch is Degraded on {HOST.NAME}         | OpenSearchIsDegradedAlert          | One or more replica shards is missing                                                                                                                                                                                                                                        | [OpenSearch is Degraded](../troubleshooting-guide/README.md#opensearch-is-degraded)     |
| OpenSearch is Down on {HOST.NAME}             | OpenSearchIsDownAlert              | One or more primary shards does not allocate in the cluster                                                                                                                                                                                                                  | [OpenSearch is Down](../troubleshooting-guide/README.md#opensearch-is-down)             |
| OpenSearch DBaaS agent is Down on {HOST.NAME} | OpenSearchDBaaSIsDownAlert         | The OpenSearch DBaaS Adapter is not responding, or responding with the `Failed` state                                                                                                                                                                                        | [OpenSearch DBaaS is Down](../troubleshooting-guide/README.md#opensearch-dbaas-is-down) |
| OpenSearch Last Backup Has Failed             | OpenSearchLastBackupHasFailedAlert | The last OpenSearch backup has finished with `Failed` status                                                                                                                                                                                                                 | [OpenSearch Backup Failed](../troubleshooting-guide/README.md#opensearch-backup-failed) |
|                                               | OpenSearchQueryIsTooSlowAlert      | Index query in the OpenSearch cluster exceeds the specified threshold. This threshold can be overridden with parameter `monitoring.thresholds.slowQuerySecondsAlert` described in [OpenSearch monitoring](/documentation/installation-guide/README.md#monitoring) parameters |                                                                                         |