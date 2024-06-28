This section deals with troubleshooting the data files if they are corrupted on a single replica shard.

# OpenSearch Metric

The Shards Stats API enables retrieving information about all shards in the cluster.
For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-shards].

To retrieve information about all shards in the cluster, run the following command:

```sh
curl -X GET http://localhost:9200/_cat/shards/cats?v
```

Possible output:

```text
    index shard state   docs  store node
    cats  2     STARTED    4  5.5kb opensearch-2
    cats  2     STARTED    4  5.5kb opensearch-1
    cats  1     STARTED    6  5.8kb opensearch-2
    cats  1     STARTED             opensearch-0
    cats  0     STARTED    8 12.4kb opensearch-1
    cats  0     STARTED             opensearch-0
```

If a node has a problem with a disk, it will either have no shards, or shards will have no data in them.

The Nodes Stats API enables retrieving information about all nodes in the cluster. For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-nodes](https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-nodes).

To retrieve information about all nodes in the cluster, run the following command:

```sh
curl -XGET http://localhost:9200/_cat/nodes?h=name,disk.avail,segments.count
```

Possible output:

```text
opensearch-2 1.7gb 10
opensearch-0 1.7gb  9
opensearch-1 1.7gb 10
```

`disk.avail` can be set to retrieve information about the available space left on the node. If the node has problem with disk, this metric will have no data about available space.

# Grafana Dashboard

The queries to retrieve metrics of the state of disks and shards on a node are described in the sections below.

## Disk Usage Metric

Percentage of disk usage by node:

```text
SELECT 100*(mean("total_total_in_bytes") - mean("total_free_in_bytes"))/ mean("total_total_in_bytes") FROM "opensearch_fs" WHERE "node_name" =~ /^$node_name$/ AND $timeFilter GROUP BY "node_name" ,"node_host", time($interval)
```

After the disk failure, this chart will not contain data from all nodes. The chart for a node with a failed disk will be truncated at the moment of disk failure.

The following image shows a possible panel view:

![Disk failure on one node disk usage panel](/docs/public/images/disk_filure_on_one_node_disk_usage_panel.png)

On this panel, a disk has failed on `opensearch-1` node.

If one of the nodes stops sending information about disk, an alert can be sent.

## Relocating Shards Count Metric

To relocate the shards count, run the following query: `SELECT max("relocating_shards") FROM "opensearch_cluster_health" WHERE $timeFilter GROUP BY time($interval) fill(previous)`

If OpenSearch detects disk problems on some node, it starts relocating shards from this node to another working node.
This metric will contain jumps on the chart at the moment of disk failure.

# Troubleshooting Procedure

This section describes the troubleshooting procedures.

## OpenSearch High Availability

After a disk failure, some or all data can be lost.
However, OpenSearch provides tools to prevent data loss.
Each OpenSearch index consists of shards.

To save the data, the shards and the indexes must be configured properly.
It is important to remember that after an index is created, the count of shards for this index is immutable.
To edit the number of shards, the index has to be recreated.

## Three Nodes Cluster Example

Since the count of shards for an index is immutable, the index must be configured correctly.
As an example, assume that an OpenSearch cluster has three nodes.
Consider the following different index configurations.

### Count of Shards = 3, Count of Replicas = 0

For this configuration, each node holds one shard. If a disk on one node fails, all data of the shard on this node will be lost. Therefore, this configuration is not safe, but data is not duplicated.

### Count of Shards = 3, Count of Replicas = 1

For this configuration, each node holds one primary shard and one replica shard from another node.
If a disk on one node has failed, data will not be lost, because another node has a copy of all data from the failed node.
If after that a second node also fails, the data still will not be lost, because another node has replica shards from this node.
But if disks fail on two nodes simultaneously, some data will be lost. This configuration saves the data if one node fails, but it takes twice as much disk space.

### Count of Shards = 3, Count of Replicas = 2

This configuration improves data safety still further compared with the previous configuration.
If a disk fails on two nodes simultaneously, data still will not be lost, because each node has a copy of the data from the other two.
It is the safest configuration, but it takes the most disk space.

## When the Disks are Repaired

When the disks are repaired, the node does not automatically recover. To restart the node, run the following command:

```sh
oc delete pod <node name>
```

After the restart, the node will be added to the cluster.
