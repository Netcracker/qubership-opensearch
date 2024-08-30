This section deals with troubleshooting the data files if they are corrupted on replica shard.

Replica shards contains copies of data, so, if one or more replica shards corrupted, data files aren’t lost.

# OpenSearch Metric

The Shards Stats API allows retrieving information about all shards in the cluster.
For more information, refer to the Official OpenSearch documentation, _Shards Stats_ [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-shards].

To retrieve the metric of shards status, run the following command:

```sh
curl -XGET http://localhost:9200/_cat/shards/cats?v&h=index,shard,state,docs,store,node,unassigned.reason
```

Example response:

```text
    index shard state   node                  unassigned.reason unassigned.details
    cats  1     STARTED opensearch-0
    cats  1     STARTED opensearch-0
    cats  2     STARTED opensearch-0
    cats  2     STARTED opensearch-0
    cats  0     STARTED opensearch-0
    cats  0     STARTED opensearch-0
```

If data files are corrupted on a replica shard, the shard starts relocation.
It is usually quite quick, so this query can return a different output: the shard can be in relocating status, or it can be already relocated.

# Cluster Behavior

This section describes the different scenarios for corrupted shards in a cluster.

## One Replica Shard of Three Corrupted

If only one replica shard is corrupted, the cluster does not lose availability. All queries return and update data successfully. The corrupted shard starts reallocation. No data is lost.

## Two Replica Shards of Three Corrupted

If two replica shards are corrupted, the cluster stays healthy until the first query.
After the first query execution, that can fail or return only a part of the data, the corrupted shard is relocated, and the cluster continues working. No data is lost.

## All Replica Shards Corrupted

The result of all replica shards' corruption is similar to the previous case.
The first several queries return incomplete data, and the first several update queries end successfully.
The corrupted shards start relocation after several queries. After that, all queries start working correctly, and all data stays saved.

# Troubleshooting Procedure

OpenSearch withstands all cases with corrupted replica shards and repairs itself without any data loss.
