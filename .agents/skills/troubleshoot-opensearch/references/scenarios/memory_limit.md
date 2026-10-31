This section provides information about configuring the memory limit.

# OpenSearch Metric

The cluster Nodes StatsNodes Stats API allows enables retrieving the statistics of all the nodes in the cluster.
For more information, refer to the official OpenSearch documentation, _Cluster Nodes Stats_ [https://opensearch.org/docs/1.2/opensearch/stats-api].

To retrieve the statistics of all the nodes in the cluster, run the following command:

```sh
curl -XGET http://localhost:9200/_nodes/stats
```

The `jvm` flag can be set to retrieve statistics that concern the JVM:

* `jvm.mem.heap_used_percent` - Percentage of JVM heap currently in use.
* `jvm.mem.pools.young.used_in_bytes` - Number of used bytes in young spaces.
* `jvm.mem.pools.young.max_in_bytes` - Total number of bytes in young spaces.
* `jvm.mem.pools.survivor.used_in_bytes` - Number of used bytes in survivor spaces.
* `jvm.mem.pools.survivor.max_in_bytes` - Total number of bytes in survivor spaces.
* `jvm.mem.pools.old.used_in_bytes` - Number of used bytes in old spaces.
* `jvm.mem.pools.old.max_in_bytes` - Total number of bytes in old spaces.
* `jvm.gc.collectors.young.collection_time_in_millis` - Total time spent on young-generation garbage collections.
* `jvm.gc.collectors.old.collection_time_in_millis` - Total time spent on old-generation garbage collections.

The metrics show the percentage of each pool being used over time. When the utilization percentage of some of these memory pools' approaches 100% and stays
around this value, it is a sign that some modifications are necessary. When that happens, you might also find increased garbage collection times,
as the JVM keeps trying to free up some space in any pools that are (nearly) full.

A drastic change in memory usage or long garbage collection runs may indicate a critical situation.
For example, in a summarized view of JVM Memory over all nodes, a drop of several GB in memory might indicate that nodes left the cluster, restarted, or were reconfigured for lower heap usage.

# Heap Size

The default installation of OpenSearch is configured with a 1 GB heap. For nearly all deployments, this value is usually too small.
If you are using the default heap values, your cluster is probably configured incorrectly.

Heap is important to OpenSearch. It is used by many in-memory data structures to provide fast operation. That said, there is another major user of memory that is off heap: Lucene.

Lucene is designed to leverage the underlying OS for caching in-memory data structures. Lucene segments are stored in individual files. Because the segments are immutable, these files never change.
This makes them very cache-friendly, and the underlying OS will keep hot segments resident in memory for faster access.
These segments include both the inverted index (for fulltext search) and doc values (for aggregations).

Lucene’s performance relies on this interaction with the OS. But if you give all available memory to the OpenSearch heap, there will not be any left over for Lucene.
This can drastically impact performance.

The standard recommendation is to give 50% of the available memory to the OpenSearch heap, while leaving the other 50% free. Lucene will use up whatever is left over.
Also, if the heap is less than 32 GB, the JVM can use compressed pointers, which saves a lot of memory: 4 bytes per pointer instead of 8 bytes.

# Troubleshooting Procedure

If you see a high heap usage, you can either increase your heap size (as long as it remains below the previously stated recommended guidelines), or scale out the cluster by adding more nodes.
For data nodes, you will also need to make sure that your indices have enough primary shards to be able to balance their data across all those nodes.
If an index is composed of a few very large shards, it is hard for OpenSearch to distribute these shards across nodes in a balanced manner.
