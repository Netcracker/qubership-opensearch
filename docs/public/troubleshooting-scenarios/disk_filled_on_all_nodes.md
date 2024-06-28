This section deals with troubleshooting the disk being filled on all nodes.

# OpenSearch Metric

The cluster Nodes Stats API enables retrieving statistics of all the nodes in the cluster.

To retrieve the statistics of all the nodes in the cluster, run the following command:

```sh
curl -XGET http://localhost:9200/_nodes/stats
```

The `fs` flag can be set to retrieve information that concerns the file system.

* `fs.total.total_in_bytes`: Total size in bytes of all file stores.
* `fs.total.free_in_bytes`: Total number of unallocated bytes in all file stores.
* `fs.total.available_in_bytes`: Total number of bytes available to this Java virtual machine on all file stores.

You can use the cat allocation API to retrieve information about how many shards are allocated to each data node and how much disk space they are using.
For more information, refer to the Official OpenSearch documentation, _Cat Allocation_ [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-allocation].

To retrieve the information about how many shards are allocated to each data node and how much disk space they are using, run the following command:

```sh
curl -XGET http://localhost:9200/_cat/allocation?format=json
```

Example response:

```json
[
  {
    "shards": "3",
    "disk.indices": "45.4kb",
    "disk.used": "122.2mb",
    "disk.avail": "1.7gb",
    "disk.total": "1.8gb",
    "disk.percent": "6",
    "host": "10.128.7.6",
    "ip": "10.128.7.6",
    "node": "opensearch-0"
  }
]
```

# Grafana Dashboard

To retrieve the metric of disk usage, use the following query:

```text
SELECT 100*(mean("total_total_in_bytes") - mean("total_available_in_bytes"))/ mean("total_total_in_bytes") FROM "opensearch_fs" WHERE "node_name" =~ /^$node_name$/ AND $timeFilter GROUP BY "node_name" ,"node_host", time($interval)
```

# Troubleshooting Procedure

If all of your data nodes are running low on disk space, you will need to add more data nodes to your cluster.
You will also need to make sure that your indices have enough primary shards to be able to balance their data across all those nodes.

However, if only certain nodes are running out of disk space, this is usually a sign that you initialized an index with too few shards.
If an index is composed of a few very large shards, it is hard for OpenSearch to distribute these shards across nodes in a balanced manner.

OpenSearch takes available disk space into account when allocating shards to nodes. By default, it will not assign shards to nodes that have over 85 percent disk space in use.
You must set up a threshold alert to notify you when any individual data node’s disk space usage approaches 80 percent, which should give you enough time to take action.

There are two remedies for low disk space. One is to remove outdated data and store it off the cluster.
This may not be a viable option for all users.
If you are storing time-based data, you can store a snapshot of older indices data off-cluster for backup, and update the index settings to turn off replication for those indices.

The second approach is the only option for you if you need to continue storing all of your data on the cluster: scaling vertically or horizontally.
If you choose to scale vertically, that means upgrading your hardware.
However, to avoid having to upgrade again down the line, you should take advantage of the fact that OpenSearch was designed to scale horizontally.
To better accommodate future growth, you might be better off reindexing the data and specifying more primary shards in the newly created index
(making sure that you have enough nodes to distribute the shards evenly).

Another way to scale horizontally is to roll over the index by creating a new index, and using an alias to join the two indices together under one namespace.
Though there is technically no limit to how much data you can store on a single shard, OpenSearch recommends a soft upper limit of 50 GB per shard,
which you can use as a general guideline that signals when it is time to start a new index.

## Disk-based Shard Allocation

OpenSearch factors in the available disk space on a node before deciding whether to allocate new shards to that node or to actively relocate shards away from that node.
For more information, refer to the official OpenSearch documentation, **Disk Allocation** [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-allocation].

The settings that can be configured in the **opensearch.yml** configuration file or updated dynamically on a live cluster with the cluster-update-settings API are described as follows:

* `cluster.routing.allocation.disk.threshold_enabled` - Defaults to true. Set to false to disable the disk allocation decider.
* `cluster.routing.allocation.disk.watermark.low` - Controls the low watermark for disk usage.
  It defaults to 85%, meaning ES will not allocate new shards to nodes once they have more than 85% disk space used.
  It can also be set to an absolute byte value (like 500 MB) to prevent ES from allocating shards if less than the configured amount of space is available.
* `cluster.routing.allocation.disk.watermark.high` - Controls the high watermark for disk usage.
  It defaults to 90%, meaning ES will attempt to relocate shards to another node if the node disk usage rises above 90%.
  It can also be set to an absolute byte value (similar to the low watermark) to relocate shards once less than the configured amount of space is available on the node.

OpenSearch also logs information about disk usage, as shown in the following example.

```text
    [2017-02-06T12:23:20,713][WARN ][o.e.c.r.a.DiskThresholdMonitor] [opensearch-0] high disk watermark [90%] exceeded on [TYNktnlyQ46zMG9VX0kz9Q][opensearch-0][/usr/share/opensearch/data/nodes/0] free: 22.4mb[2.3%], shards will be relocated away from this node
    [2017-02-06T12:23:20,713][INFO ][o.e.c.r.a.DiskThresholdMonitor] [opensearch-0] rerouting shards: [high disk watermark exceeded on one or more nodes]
    [2017-02-06T12:25:50,725][INFO ][o.e.c.r.a.DiskThresholdMonitor] [opensearch-0] low disk watermark [85%] exceeded on [TYNktnlyQ46zMG9VX0kz9Q][opensearch-0][/usr/share/opensearch/data/nodes/0] free: 122.5mb[12.5%], replicas will not be assigned to this node
```
