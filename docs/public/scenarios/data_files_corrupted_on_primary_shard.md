This section deals with troubleshooting the data files if they are corrupted on primary shard.

If an index has no replicas, data file corruption leads to data loss, because OpenSearch has no copies of this data. OpenSearch provides replica shards approach to prevent the situation.

# OpenSearch Metric

This section describes how to retrieve information about the shards.

## Without Replication

The Shards Stats API allows retrieving information about all shards in the cluster.
For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-shards].

To retrieve information about all shards in the cluster, run the following command:

```sh
curl -XGET http://localhost:9200/_cat/shards/cats?v&h=index,shard,state,docs,store,node,unassigned.reason
```

Possible output:

```text
    index shard state      node                  unassigned.reason unassigned.details
    cats  1     STARTED    opensearch-0
    cats  2     STARTED    opensearch-0
    cats  0     UNASSIGNED                       ALLOCATION_FAILED shard failure, reason [search execution corruption failure], failure FetchPhaseExecutionException[Fetch Failed [Failed to fetch doc id [7]]]; nested: CorruptIndexException[Corrupted: docID=7, docBase=0, chunkDocs=0, numDocs=8 (resource=MMapIndexInput(path="/usr/share/opensearch/data/nodes/0/indices/KLrt-04kTQWB_ZUUyCK9Hg/0/index/_0.cfs") [slice=_0.fdt])];
```

If data files of any shards were corrupted, the shard will be in an unassigned status. In this case, the output contains `CorruptIndexException` and a path to corrupted file.

## With Replication

To retrieve the information about all shards in the cluster, run the following command:

```sh
curl -XGET http://localhost:9200/_cat/shards/cats?v&h=index,shard,state,docs,store,node,unassigned.reason
```

Possible output:

```text
    index shard state   node                  unassigned.reason unassigned.details
    cats  1     STARTED opensearch-0
    cats  1     STARTED opensearch-0
    cats  2     STARTED opensearch-0
    cats  2     STARTED opensearch-0
    cats  0     STARTED opensearch-0
    cats  0     STARTED opensearch-0
```

If data files are corrupted on a primary shard of an index with replicas, there are no unassigned shards, and all data will be saved. Updates continue to work,
but some of them fail with `read past EOF: MMapIndexInput(path=\"/usr/share/opensearch/data/nodes/0/indices/RncVMMtyQIeIMPz-0Dhjpw/0/index/_0.cfs\") [slice=_0.fdt]`,
because OpenSearch does not detect problems with data files before read. After several requests, OpenSearch reassigns the shard with corrupted files,
and processing of all requests should finish successfully.

# Troubleshooting Procedure

This section describes the troubleshooting procedures without replication of shards.

## Without Replication

If an OpenSearch cluster consists of one node, or an index has no replica shards, data will be lost after file corruption.
Update queries fail with 504 HTTP-status. The cluster sets status to `failed`, because the corrupted primary shard was in an unassigned status.
Reindexing can help to save remaining data and make index writable.
For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/latest/opensearch/reindex-data].
