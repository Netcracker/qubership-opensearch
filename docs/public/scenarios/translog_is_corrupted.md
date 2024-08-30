This section deals with troubleshooting the translog if it is corrupted.

# Translog

Changes in Lucene only persist to disk during a Lucene commit, which is a relatively heavy operation and so cannot be performed after every index or delete operation.
Changes that happen after one commit and before another will be lost in the event of process exit or a hardware failure.
To prevent this data loss, each shard has a `transaction log` or write ahead log associated with it.

# Translog Settings

The data in the transaction log only persists to disk when the translog is synced and committed. In the event of a hardware failure, any data written since the previous translog commit will be lost.
By default, OpenSearch syncs and commits the translog every 5 seconds if `index.translog.durability` is set to `async` or `request` (default)
at the end of every index, delete, update, or bulk request.

If the translog has the default settings, a corruption will not lead to data loss because, in the event of hardware failure, all acknowledged writes will already have been committed to disk.

# OpenSearch Metric

The Shards Stats API enables retrieving information about all shards in the cluster.
For more information, refer to the official OpenSearch documentation [https://opensearch.org/docs/latest/opensearch/rest-api/cat/cat-shards].

To retrieve the shard status information, run the following command:

```sh
curl -XGET http://localhost:9200/_cat/shards?v&h=index,shard,prirep,state,docs,store,ip,node,unassigned.reason,unassigned.details
```

Example response:

```text
    index      shard prirep state      docs  store ip        node            unassigned.reason unassigned.details
    cats       0     p      STARTED       0   130b 10.1.3.3  opensearch-1
    cats       0     r      STARTED       1   130b 10.1.12.6 opensearch-0
    cats       0     r      UNASSIGNED                                       ALLOCATION_FAILED failed recovery, failure RecoveryFailedException[[cats][0]: Recovery failed from {opensearch-1}{8QnQuADIS0yvpPk74UAvig}{z4GpsVEWQCypHyzr0eNfhw}{10.1.3.3}{10.1.3.3:9300} into {opensearch-2}{du6xA4LESROG2KfnAh9OiQ}{HimMm6A3S-mblfr5PDFdeA}{10.1.13.5}{10.1.13.5:9300}]; nested: RemoteTransportException[[opensearch-1][10.1.3.3:9300][internal:index/shard/recovery/start_recovery]]; nested: RecoveryEngineException[Phase[2] phase2 failed]; nested: TranslogCorruptedException[operation size must be at least 4 but was: 0];
```

If a translog is corrupted, shards with a corrupted translog will have `TranslogCorruptedException` in `unassigned.details`.

# Troubleshooting Procedure

If `index.translog.durability` is set to `async`, fsync and commit in the background every sync\_interval.
In the event of a hardware failure, all acknowledged writes since the last automatic commit will be discarded.

When this corruption is detected by OpenSearch due to mismatching checksums, OpenSearch will fail the shard and refuse to allocate that copy of the data to the node,
recovering from a replica if available.

If there is no copy of the data from which OpenSearch can recover successfully, you may want to recover the data that is part of the shard at the cost of losing the data that is currently
contained in the translog. OpenSearch provides a command-line tool for this: `opensearch-translog`.
