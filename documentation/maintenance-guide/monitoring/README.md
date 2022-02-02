# Overview

This documentation describes monitoring dashboards, their metrics, Zabbix alarms and Prometheus alerts.

The dashboards provide the following parameters to configure at the top of the dashboard:

* Interval time for metric display
* Node name

For all graph panels, the mean metric value is used in the given interval. For all singlestat panels, 
the last metric value is used.

## OpenSearch Monitoring

### Dashboard

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_dashboard.png)

### Metrics

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
   
   ![Oversized Heap](/documentation/maintenance-guide/monitoring/pictures/oversized-heap.png)

**Memory Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_memory_metrics.png)

* `OS memory usage` - The usage of memory by each OpenSearch node and its limit. These metric is
  useful to avoid reaching the memory limit on nodes.

**Disk Metrics**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_disk_metrics.png)

* `Disk usage in percent` - The usage of disk space in percent for an OpenSearch node.
* `Disk usage` - The usage of disk space allocated to each OpenSearch node and its limit.
* `Disk I/O operations` - The rate of change between subsequent values of input/output operations per second for each OpenSearch node.
* `Disk I/O usage` - The rate of change between subsequent values of disk usage for each OpenSearch node.

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
* `Indices time operations` - The average time to complete an operation in indices grouped by 
  operation type and OpenSearch node.
* `Indices current operations` - The number of operations performed by indices at the moment grouped 
  by operation type and OpenSearch node.
* `Indices documents count` - The number of documents stored in indices.
* `Indices data size` - The size of data stored in indices.

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
* `Flush latency` - If you see this metric increasing steadily, it could indicate a problem with slow disks.
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

* `Backup Last Version Size` - The size of the last backup.
* `Time Spent on Backup` - Time spent on last backup.
* `Storage Size/Free Space` - The space occupied by backups and the remaining amount of space.
  Not all storage supports total size, so "Total Volume Space" metrics can be zeroed.
* `Backup Activity Status` - The changes of activity status on the chart.
* `Backup Activity Status` - The current activity status of the backup daemon. The activity status 
  can be one of following:
   * Not Active - There is no running backup process.
   * In Progress/Started - The backup process is running.
* `Last Backup Status` - The state of the last backup. The backup status can be one of following:
   * SUCCESS - There is at least one successful backup, and the latest backup is successful.
   * FAILED - There are no successful backups, or the last backup failed.
   * INCOMPATIBLE - The snapshot was created with an old version of OpenSearch, and therefore is
     incompatible with the current version of the cluster.
* `Backup Versions Count` - The number of available backups.
* `Successful Backup Versions Count` - The number of successful backups.
* `Time of Last Backup` - The period of time when the last backup process was ended.
* `Time of Last Successful Backup` - The period of time when the last successful backup process was 
  ended.
* `Storage Type` - The backup storage type. The storage type can be one of following:
   * Persistent Volume - The volume is mounted to the pod of the backup daemon. The daemon does not
     recognise what is underlying in the directory, which may be NFS, Cinder, or something else.
   * AWS S3 - "Cloud" high-available storage.

**DBaaS Health**

![Dashboard](/documentation/maintenance-guide/monitoring/pictures/opensearch-monitoring_dbaas_health.png)

* `DBaaS Adapter Status` - The status of DBaaS Adapter.
* `DBaaS OpenSearch Cluster Status` - The status of OpenSearch cluster from the DBaaS Adapter side.
  
### Monitoring Alarms Description

This section describes monitoring alarms. There are two ways to monitor OpenSearch Service - InfluxDB flow and 
Prometheus flow. For first one Zabbix is used as an alarm system and all notifications are called "alarm". For 
second one Alert Manager is used. The Alert Manager is based on Prometheus rules and all its notifications are 
called "alert". Anyway in both approaches OpenSearch Service has the same alarms/alerts and single troubleshooting 
guide to resolve issues.  

|Zabbix Alarm|Alert Manager alert|Possible Reasons| Troubleshooting Guide link                                                                 |
|---|---|---|--------------------------------------------------------------------------------------------|
|OpenSearch's CPU usage is above 95%|OpenSearch_CPU_Load_Alert|CPU usage by one of the pods in the OpenSearch cluster comes close to the limit| [OpenSearch’s CPU usage](../troubleshooting-guide/README.md#opensearch-cpu-usage)          |
|OpenSearch's disk usage is above 90%|OpenSearch_Disk_Usage_Alert|Disk usage by one of the pods in the OpenSearch cluster comes close to the limit| [OpenSearch Disk Usage](../troubleshooting-guide/README.md#opensearch-disk-usage)       |
|OpenSearch's disk usage is above 98%|OpenSearch_Disk_Too_Much_Usage_Alert|Disk usage by one of the pods in the OpenSearch cluster comes close to the limit| [OpenSearch Disk Usage](../troubleshooting-guide/README.md#opensearch-disk-usage)       |
|OpenSearch's memory usage is above 95%|OpenSearch_Memory_Usage_Alert|Memory usage by one of the pods in the OpenSearch cluster comes close to the limit| [OpenSearch Memory Usage](../troubleshooting-guide/README.md#opensearch-memory-usage)   |
|OpenSearch's heap memory usage is above 95%|OpenSearch_Heap_Memory_Usage_Alert|Heap memory usage by one of the pods in the OpenSearch cluster comes close to the limit| [OpenSearch Memory Usage](../troubleshooting-guide/README.md#opensearch-memory-usage)   |
|OpenSearch is Degraded on {HOST.NAME}|OpenSearch_Is_Degraded_Alert|One or more replica shards is missing| [OpenSearch is Degraded](../troubleshooting-guide/README.md#opensearch-is-degraded)     |
|OpenSearch is Down on {HOST.NAME}|OpenSearch_Is_Down_Alert|One or more primary shards does not allocate in the cluster| [OpenSearch is Down](../troubleshooting-guide/README.md#opensearch-is-down)             |
|OpenSearch DBaaS agent is Down on {HOST.NAME}|OpenSearch_DBaaS_Is_Down_Alert|The OpenSearch DBaaS Adapter is not responding, or responding with the `Failed` state| [OpenSearch DBaaS is Down](../troubleshooting-guide/README.md#opensearch-dbaas-is-down) |
|OpenSearch Last Backup Has Failed|OpenSearch_Last_Backup_Has_Failed_Alert|The last OpenSearch backup has finished with `Failed` status| [OpenSearch Backup Failed](../troubleshooting-guide/README.md#opensearch-backup-failed) |