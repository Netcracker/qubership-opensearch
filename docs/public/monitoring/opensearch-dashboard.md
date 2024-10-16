# OpenSearch Monitoring

InfluxDB dashboards for telegraf metrics

## Tags

* `Prometheus`
* `OpenSearch`

## Panels

### OpenSearch cluster

![Dashboard](/docs/public/images/opensearch-monitoring_cluster_overview.png)

<!-- markdownlint-disable line-length -->
| Name                            | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | Thresholds | Repeat |
|---------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|--------|
| Cluster status                  | Status of OpenSearch cluster.<br/>If the cluster status is `degraded`, at least one replica shard is unallocated or missing. The search<br/>   results will still be complete, but if more shards are missing, you may lose data.<br/>   <br/>   A `failed` cluster status indicates that:<br/>   <br/>    * At least one primary shard is missing.<br/>    * You are missing data.<br/>    * The searches will return partial results.<br/>    * You will be blocked from indexing into that shard. |            |        |
| Nodes status                    | Status of OpenSearch nodes                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |            |        |
| CPU Usage                       | Maximum current CPU usage (in percent) among all OpenSearch servers.                                                                                                                                                                                                                                                                                                                                                                                                                                 |            |        |
| JVM Heap Usage                  | Maximum current usage JVM heap memory (in percent) among all OpenSearch servers.                                                                                                                                                                                                                                                                                                                                                                                                                     |            |        |
| Off-Heap Memory Usage           | The maximum memory usage excluding allocated JVM Heap memory (in percent) among all OpenSearch servers.<br/>This amount of memory is used by operating system and Apache Lucene.                                                                                                                                                                                                                                                                                                                     |            |        |
| OpenSearch Version              | OpenSearch version                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |            |        |
| Cluster Status Transitions      | Transitions of OpenSearch cluster status                                                                                                                                                                                                                                                                                                                                                                                                                                                             |            |        |
| Pod Readiness Probe Transitions | Transitions of readiness probes for each OpenSearch pod.                                                                                                                                                                                                                                                                                                                                                                                                                                             |            |        |
<!-- markdownlint-enable line-length -->

### OpenSearch shards

![Dashboard](/docs/public/images/opensearch-monitoring_opensearch_shards.png)

<!-- markdownlint-disable line-length -->
| Name                  | Description                                                                                                                                                                                                                                                                                                                                                                                                                        | Thresholds                                            | Repeat |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------|--------|
| Active primary shards | The number of active primary shards in OpenSearch cluster                                                                                                                                                                                                                                                                                                                                                                          | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |        |
| Active shards         | The number of active primary and replica shards in OpenSearch cluster                                                                                                                                                                                                                                                                                                                                                              | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |        |
| Initializing shards   | The number of shards that are in the `initializing` state. When you first create an index, or when a node is rebooted, its shards are briefly in the `initializing` state before transitioning to `started` or `unassigned` as the master node attempts to assign shards to nodes in the cluster. If you see the shards remain in the `initializing` state for too long, it could be a warning sign that your cluster is unstable. | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |        |
| Relocating shards     | The number of shards that are relocating now and have the `relocating` state.                                                                                                                                                                                                                                                                                                                                                      | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |        |
| Unassigned shards     | The number of shards that are not assigned to any node and have the `unassigned` state. If you see the shards remain in the `unassigned` state for too long, it could be a warning sign that your cluster is unstable.                                                                                                                                                                                                             | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |        |
<!-- markdownlint-enable line-length -->

### OpenSearch tasks

![Dashboard](/docs/public/images/opensearch-monitoring_opensearch_tasks.png)

<!-- markdownlint-disable line-length -->
| Name                      | Description                                                    | Thresholds                                                          | Repeat |
|---------------------------|----------------------------------------------------------------|---------------------------------------------------------------------|--------|
| Pending Tasks             | The number of tasks in 'pending' status in OpenSearch cluster  | Default:<br/>Mode: absolute<br/>Level 1: 1<br/>Level 2: 5<br/><br/> |        |
| Time of Most Waiting Task | The maximum time in milliseconds that task is waiting in queue |                                                                     |        |
<!-- markdownlint-enable line-length -->

### Network metrics

![Dashboard](/docs/public/images/opensearch-monitoring_network_metrics.png)

<!-- markdownlint-disable line-length -->
| Name                       | Description                                                                                                                                                                                                                                                                                                                         | Thresholds | Repeat |
|----------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|--------|
| Open transport connections | The number of open transport connections for each OpenSearch node. May be not applicable for managed external OpenSearch.                                                                                                                                                                                                           |            |        |
| Open http connections      | The number of open HTTP connections for each OpenSearch node. If the total number of open HTTP connections is constantly increasing, it may indicate that your HTTP clients are not properly establishing persistent connections. Reestablishing connections adds extra milliseconds or even seconds to your request response time. |            |        |
| Transport size             | The rate of change between subsequent size values of received (rx) and transmitted (tx) packages for each OpenSearch node. May be not applicable for managed external OpenSearch.                                                                                                                                                   |            |        |
<!-- markdownlint-enable line-length -->

### JVM heap and GC metrics

![Dashboard](/docs/public/images/opensearch-monitoring_jvm_heap_and_gc_metrics.png)

<!-- markdownlint-disable line-length -->
| Name                   | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Thresholds | Repeat |
|------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|--------|
| JVM heap usage         | The usage of JVM heap memory by each OpenSearch node                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |            |        |
| JVM heap usage percent | OpenSearch is set up to initiate garbage collections whenever JVM heap usage hits 75 percent. As shown above, it may be useful to monitor which nodes exhibit high heap usage, and set up an alert to find out if any node is consistently using over 85 percent of heap memory; this indicates that the rate of garbage collection isn't keeping up with the rate of garbage creation. To address this problem, you can either increase your heap size (as long as it remains below the recommended guidelines stated above), or scale out the cluster by adding more nodes. |            |        |
| JVM non heap usage     | The usage of memory outside the JVM heap by each OpenSearch node                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |            |        |
| GC time                | Time spent on major GCs that collect old generation objects and on minor GCs that collects young generation objects in JVM (time rate per sampling interval).                                                                                                                                                                                                                                                                                                                                                                                                                 |            |        |
<!-- markdownlint-enable line-length -->

### Memory metrics

![Dashboard](/docs/public/images/opensearch-monitoring_memory_metrics.png)

<!-- markdownlint-disable line-length -->
| Name                  | Description                                                                                                                                                                | Thresholds | Repeat |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|--------|
| OS memory usage       | The usage of memory by each OpenSearch node and its limit. These metric is useful to avoid reaching the memory limit on nodes.                                             |            |        |
| Off-heap memory usage | The usage of memory excluding allocated JVM Heap memory by each OpenSearch node and its limit. <br/>  This amount of memory is used by operating system and Apache Lucene. |            |        |
<!-- markdownlint-enable line-length -->

### Disk metrics

![Dashboard](/docs/public/images/opensearch-monitoring_disk_metrics.png)

<!-- markdownlint-disable line-length -->
| Name                  | Description                                                                                                 | Thresholds | Repeat |
|-----------------------|-------------------------------------------------------------------------------------------------------------|------------|--------|
| Disk usage in percent | The usage of disk space in percent for an OpenSearch node                                                   |            |        |
| Disk usage            | The usage of disk space allocated to each OpenSearch node and its limit                                     |            |        |
| Disk I/O operations   | The rate of change between subsequent values of input/output operations per second for each OpenSearch node |            |        |
| Disk I/O usage        | The rate of change between subsequent values of disk usage for each OpenSearch node                         |            |        |
| Open File Descriptors | The amount of open file descriptors for each OpenSearch node                                                |            |        |
<!-- markdownlint-enable line-length -->

### CPU metrics

![Dashboard](/docs/public/images/opensearch-monitoring_cpu_metrics.png)

<!-- markdownlint-disable line-length -->
| Name                     | Description                                            | Thresholds | Repeat |
|--------------------------|--------------------------------------------------------|------------|--------|
| CPU load average (5 min) | Five-minute load average on the system                 |            |        |
| CPU load in percent      | The usage of CPU in percent for each OpenSearch node   |            |        |
| CPU load by pod          | The usage of CPU by each OpenSearch node and its limit |            |        |
<!-- markdownlint-enable line-length -->

### Indices statistics

![Dashboard](/docs/public/images/opensearch-monitoring_indices_statistics.png)

<!-- markdownlint-disable line-length -->
| Name                         | Description                                                                                                   | Thresholds | Repeat |
|------------------------------|---------------------------------------------------------------------------------------------------------------|------------|--------|
| Indices total operations     | The number of operations performed by indices at the moment grouped by operation type and OpenSearch node     |            |        |
| Indices operations rate      | The operations number performed by indices per second grouped by operation type and OpenSearch node           |            |        |
| Indices time operations      | The average time to complete an operation in indices grouped by operation type and OpenSearch node            |            |        |
| Indices time operations rate | The average time to complete an operation in indices per second grouped by operation type and OpenSearch node |            |        |
| Indices data size            | The size of data stored in indices grouped by OpenSearch nodes                                                |            |        |
| Indices documents count      | The number of documents stored in indices grouped by OpenSearch nodes                                         |            |        |
| Indices documents rate       | The number of documents added to indices per second grouped by OpenSearch nodes                               |            |        |
<!-- markdownlint-enable line-length -->

### Thread pool queues and requests metrics

![Dashboard](/docs/public/images/opensearch-monitoring_thread_pool_queues_and_requests_metrics.png)

<!-- markdownlint-disable line-length -->
| Name               | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Thresholds | Repeat |
|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|--------|
| Requests latency   | If you notice the indexing latency increasing, you may be trying to index too many documents at one time (OpenSearch's documentation recommends starting with a bulk indexing size of 5 to 15 megabytes and increasing slowly from there).<br/>If you are planning to index a lot of documents and you don't need the new information to be immediately available for search, you can optimize for indexing performance over search performance by decreasing refresh frequency until you are done indexing.<br/>Flush latency:  If you see this metric increasing steadily, it could indicate a problem with slow disks |            |        |
| Rejected requests  | The size of each thread pool's queue represents how many requests are waiting to be served while the node is currently at capacity. The queue allows the node to track and eventually serve these requests instead of discarding them. Thread pool rejections arise once the thread pool's maximum queue size (which varies based on the type of thread pool) is reached.                                                                                                                                                                                                                                                |            |        |
| Thread pool queues | The size of each thread pool's queue represents how many requests are waiting to be served while the node is currently at capacity.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |            |        |
<!-- markdownlint-enable line-length -->

### Backup

![Dashboard](/docs/public/images/opensearch-monitoring_backup.png)

<!-- markdownlint-disable line-length -->
| Name                             | Description                                                                                                                                                                                                                                                   | Thresholds                                                          | Repeat |
|----------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------|--------|
| Backup Daemon Status             | Shows the current activity status of the backup daemon. The activity status can be one of following:<br/>* Not Active - There is no running backup process.<br/>* In Progress/Started - The backup process is running.                                        | Default:<br/>Mode: absolute<br/>Level 1: 1<br/>Level 2: 5<br/><br/> |        |
| Last Backup Status               | Shows the state of the last backup. The backup status can be one of following:<br/> <br/>* SUCCESS - There is at least one successful backup, and the latest backup is successful.<br/>* FAILED - There are no successful backups, or the last backup failed. | Default:<br/>Mode: absolute<br/>Level 1: 1<br/>Level 2: 5<br/><br/> |        |
| Time of Last Backup              | Shows the period of time when the last backup process was ended.                                                                                                                                                                                              | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/>               |        |
| Backup Daemon Status Transitions | Shows the current activity status of the backup daemon. The activity status can be one of following:<br/>* Not Active - There is no running backup process.<br/>* In Progress/Started - The backup process is running.                                        |                                                                     |        |
| Storage Type                     | Shows the backup storage type.                                                                                                                                                                                                                                | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/>               |        |
| Successful Backup Versions Count | Shows the amount of successful backups.                                                                                                                                                                                                                       | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/>               |        |
| Time of Last Successful Backup   | Shows the period of time when the last successful backup process was ended.                                                                                                                                                                                   | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/>               |        |
| Backup Versions Count            | Shows the amount of available backups.                                                                                                                                                                                                                        | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/>               |        |
| Storage Size/Free Space          | Shows the space occupied by backups and the remaining amount of space. <br/>Not all storage supports total size, so "Total Volume Space" metrics can be zeroed.                                                                                               |                                                                     |        |
| Backup Activity Status           | Shows the changes of activity status on the chart.                                                                                                                                                                                                            |                                                                     |        |
| Time Spent on Backup             | Shows time spent on last backup.                                                                                                                                                                                                                              |                                                                     |        |
| Backup Last Version Size         | Shows the size of the last backup.                                                                                                                                                                                                                            |                                                                     |        |
<!-- markdownlint-enable line-length -->

### DBaaS Health

![Dashboard](/docs/public/images/opensearch-monitoring_dbaas_health.png)

<!-- markdownlint-disable line-length -->
| Name                                        | Description                                                  | Thresholds | Repeat |
|---------------------------------------------|--------------------------------------------------------------|------------|--------|
| DBaaS Adapter Status                        | The status of DBaaS Adapter.                                 |            |        |
| DBaaS OpenSearch Cluster Status             | The status of OpenSearch cluster from the DBaaS Adapter side |            |        |
| DBaaS Adapter Status Transitions            | Transitions of DBaaS Adapter statuses                        |            |        |
| DBaaS OpenSearch Cluster Status Transitions | Transitions of DBaaS OpenSearch cluster status               |            |        |
<!-- markdownlint-enable line-length -->
