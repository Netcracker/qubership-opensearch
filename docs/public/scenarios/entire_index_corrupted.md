This section deals with troubleshooting the entire index corruption.

If all datafiles in an index are corrupted, any queries to the index fail with a reason as shown in the following example:

```text
"Corrupted: docID=0, docBase=0, chunkDocs=0, numDocs=18 (resource=MMapIndexInput(path=\"/usr/share/opensearch/data/nodes/0/indices/wXrcwF8fSTC_8nQQl8_IAg/0/index/_0.cfs\") [slice=_0.fdt])"
```

After the index was corrupted, all shards of this index change status to `UNASSIGNED`.
Primary shards have `ALLOCATION_FAILED` because of `unassigned reason`, and the replica shards have `PRIMARY_FAILED` reason.
`CorruptIndexException` will appear in `unassigned details` for the primary shards of the corrupted index.
All data will be lost. It can be restored only from backup.

# OpenSearch Metric

The Shards Stats API allows retrieving information about all shards in the cluster.
For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-shards].

To retrieve the shards status information, run the following command:

```sh
curl -X GET 'http://localhost:9200/_cat/shards/cats?v&h=index,shard,state,docs,store,node,unassigned.reason'
```

Example response:

```text
Shards stats:
index shard state      docs store node unassigned.reason unassigned.details
cats  0     UNASSIGNED                 ALLOCATION_FAILED shard failure, reason [search execution corruption failure], failure FetchPhaseExecutionException[Fetch Failed [Failed to fetch doc id [0]]]; nested: CorruptIndexException[Corrupted: docID=0, docBase=0, chunkDocs=0, numDocs=18 (resource=MMapIndexInput(path="/usr/share/opensearch/data/nodes/0/indices/B8pC4O6ETv6tox7yq-mrYQ/0/index/_0.cfs") [slice=_0.fdt])];
cats  0     UNASSIGNED                 ALLOCATION_FAILED shard failure, reason [search execution corruption failure], failure FetchPhaseExecutionException[Fetch Failed [Failed to fetch doc id [0]]]; nested: CorruptIndexException[Corrupted: docID=0, docBase=0, chunkDocs=0, numDocs=18 (resource=MMapIndexInput(path="/usr/share/opensearch/data/nodes/0/indices/B8pC4O6ETv6tox7yq-mrYQ/0/index/_0.cfs") [slice=_0.fdt])];
```

# Troubleshooting Procedure

If all datafiles of the index are corrupted, there is no way to retrieve data from this index. The only solution is to restore this index from backup, if it exists.
