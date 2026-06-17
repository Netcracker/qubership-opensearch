This section deals with configuring of I/O limit.

# OpenSearch Metric

The cluster Nodes Stats API enables to retrieve the statistics of all the nodes in the cluster.

To retrieve the statistics of all the nodes in the cluster, run the following command:

```sh
curl -X GET 'http://localhost:9200/_nodes/stats'
```

The `fs` flag can be set to retrieve information that concerns the file system:

* `fs.io_stats.total.operations`: The total number of read and write operations across all devices used by OpenSearch that have been completed since starting OpenSearch.
* `fs.io_stats.total.read_operations`: The total number of read operations for across all devices used by OpenSearch that have been completed since starting OpenSearch.
* `fs.io_stats.total.write_operations`: The total number of write operations across all devices used by OpenSearch that have been completed since starting OpenSearch.
* `fs.io_stats.total.read_kilobytes`: The total number of kilobytes read across all devices used by OpenSearch since starting OpenSearch.
* `fs.io_stats.total.write_kilobytes`: The total number of kilobytes written across all devices used by OpenSearch since starting OpenSearch.

# Grafana Dashboard

The queries to retrieve the metrics of disk I/O statistics are described as follows.

To retrieve the I/O operations statistics:

```text
SELECT non_negative_derivative(mean("io_stats_total_read_operations"), 1s) AS "read", non_negative_derivative(mean("io_stats_total_write_operations"), 1s) AS "write" FROM "opensearch_fs" WHERE "node_name" =~ /^$node_name$/ AND $timeFilter GROUP BY time($interval), "node_name", "node_host"
```

To retrieve the I/O usage statistics:

```text
SELECT non_negative_derivative(mean("io_stats_total_read_kilobytes"), 1s) AS "read", non_negative_derivative(mean("io_stats_total_write_kilobytes"), 1s) AS "write" FROM "opensearch_fs" WHERE "node_name" =~ /^$node_name$/ AND $timeFilter GROUP BY time($interval), "node_name", "node_host"
```

# Troubleshooting Procedure

Disks are usually the bottleneck of any modern server.
OpenSearch uses disks heavily, and the more throughput your disks can handle, the more stable your nodes will be.
For write-heavy clusters with nodes that are continually experiencing heavy I/O activity, OpenSearch recommends using SSDs to boost performance.

A search engine makes heavy use of storage devices, and watching the disk I/O ensures that this basic need gets fulfilled.
As there are so many reasons for reduced disk I/O, it is considered a key metric and a good indicator for many kinds of problems.
It is a good metric to check the effectiveness of indexing and query performance.
Distinguishing between read and write operations directly indicates what the system needs most in the specific use case.
Typically, there are many more reads from queries than writes, although a popular use case for OpenSearch is log management, which typically has high writes and low reads.
When writes are higher than reads, optimizations for indexing are more important than query optimizations.

The operating system settings for disk I/O are a base for all other tuning disk I/O can avoid potential problems.
If the disk I/O is still not sufficient, countermeasures such as optimizing the number of shards and their size,
throttling merges, replacing slow disks, moving to SSDs, or adding more nodes should be evaluated according to the circumstances causing the I/O bottlenecks.
For example, while searching, disks get trashed if the indices do not fit in the OS cache. This can be solved in a number of different ways:
by adding more RAM or data nodes, by reducing the index size (e.g. using time-based indices and aliases),
by being smarter about limiting searches to only specific shards or indices instead of searching all of them, or by caching.

## Query Optimizations

Search performance varies widely according to what type of data is being searched and how each query is structured.
Depending on the way your data is organized, you may need to experiment with a few different methods before finding one that will help speed up search performance.
Two of these methods are covered in custom routing and force merging.

Typically, when a node receives a search request, it needs to communicate that request to a copy (either primary or replica) of every shard in the index.
Custom routing allows you to store related data on the same shard, so that you only have to search a single shard to satisfy a query.

In OpenSearch, every search request has to check every segment of each shard it hits.
So once you have reduced the number of shards you have to search, you can also reduce the number of segments per shard by triggering the Force Merge API on one or more of your indices.
The Force Merge API prompts the segments in the index to continue merging until each shard’s segment count is reduced to `max_num_segments` (1, by default).
It is worth experimenting with this feature, as long as you account for the computational cost of triggering a high number of merges.

When it comes to shards with a large number of segments, the force merge process becomes much more computationally expensive.
For instance, force merging an index of 10,000 segments down to 5,000 segments does not take much time, but merging 10,000 segments all the way down to one segment can take hours.
The more merging that must occur, the more resources you take away from fulfilling search requests, which may defeat the purpose of calling a force merge in the first place.
In any case, it is usually a good idea to schedule a force merge during non-peak hours, such as overnight, when you do not expect many search or indexing requests.

## Indexing Optimizations

OpenSearch comes preconfigured with many settings that try to ensure that you retain enough resources for searching and indexing data.
However, if your usage of OpenSearch is heavily skewed towards writes, you may find that it makes sense to tweak certain settings to boost indexing performance,
even if it means losing some search performance or data replication.

The following methods can be used to optimize your use case for indexing, rather than searching, data.

* Shard Allocation

As a high-level strategy, if you are creating an index that you plan to update frequently,
make sure you designate enough primary shards so that you can spread the indexing load evenly across all of your nodes.
The general recommendation is to allocate one primary shard per node in your cluster, and possibly two or more primary shards per node,
but only if you have a lot of CPU and disk bandwidth on those nodes. However, keep in mind that shard overallocation adds overhead and may negatively impact search,
since search requests need to hit every shard in the index.
On the other hand, if you assign fewer primary shards than the number of nodes, you may create hotspots,
as the nodes that contain those shards will need to handle more indexing requests than nodes that do not contain any of the index’s shards.

* Increase the Size of the Indexing Buffer

The index buffer size setting (`indices.memory.index_buffer_size`) determines how full the buffer can get before its documents are written to a segment on disk.
The default setting limits this value to 10 percent of the total heap in order to reserve more of the heap for serving search requests,
which does not help you if you are using OpenSearch primarily for indexing.

* Adjust Translog Settings

OpenSearch flushes translog data to disk after every request, reducing the risk of data loss in the event of hardware failure.
If you want to prioritize indexing performance over potential data loss, you can change `index.translog.durability` to `async` in the index settings.
With this in place, the index will only commit writes to disk upon every `sync_interval`, rather than after each request, leaving more of its resources free to serve indexing requests.
For more information about using translog data.
