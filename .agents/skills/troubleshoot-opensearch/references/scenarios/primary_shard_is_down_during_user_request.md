This section deals with troubleshooting the primary shard when it is down during user request.

Data can be lost if OpenSearch node is down during user request.
However, OpenSearch provides tools to prevent data loss, including a translog, or transaction log, which records every operation in OpenSearch as it happens.

# Translog

Without an `fsync` to flush data in the filesystem cache to disk, OpenSearch cannot be sure that the data will still be there after a power failure, or even after exiting the application normally.
For OpenSearch to be reliable, it needs to ensure that changes are persisted to disk.

**Note**: By default, OpenSearch performs an `fsync` call and commits the translog every 5 seconds if `index.translog.durability` is set to `async` or if set to `request` (default)
at the end of every index, delete, update, or bulk request. In fact, OpenSearch will only report success of an index, delete, update,
or bulk request to the client after the transaction log has been successfully `fsync`ed and committed on the primary and on every allocated replica.

Ultimately, that means your client will not receive a `200 OK` response until the entire request has been `fsync`'ed in the translog of the primary and all replicas.

For more information about performed operations, refer to:

* OpenSearch Reference Guide [https://opensearch.org/docs/latest/opensearch/rest-api/document-apis/index-document]
* OpenSearch Reference: Delete API [https://opensearch.org/docs/latest/opensearch/rest-api/document-apis/delete-document]
* OpenSearch Reference: Update API [https://opensearch.org/docs/latest/opensearch/rest-api/document-apis/update-document]
* OpenSearch Reference: Bulk API [https://opensearch.org/docs/latest/opensearch/rest-api/document-apis/bulk]

The translog provides a persistent record of all operations that have not yet been flushed to disk.
When starting up, OpenSearch will use the last commit point to recover known segments from disk, and will then replay all operations in the translog to add the changes that happened after the last
commit.

The translog is also used to provide real-time CRUD.
When you try to retrieve, update, or delete a document by ID, it first checks the translog for any recent changes before trying to retrieve the document from the relevant segment.
means that it always has access to the latest known version of the document, in real-time.

Executing an `fsync` after every request does come with some performance cost, although in practice it is relatively small
(especially for bulk ingestion, which amortizes the cost over many documents in the single request).
For some high-volume clusters where losing a few seconds of data is not critical, it can be advantageous to fsync asynchronously.

In this case, writes are buffered in memory and `fsync`'ed together every 5s.
If you decide to enable `async` translog behavior, you are guaranteed to lose a `sync_interval`'s worth of data if a crash happens. Please be aware of this characteristic before deciding.

# Troubleshooting Procedure

OpenSearch provides the capability to subdivide your index into multiple pieces called shards. When you create an index, you can simply define the number of shards that you want.
Each shard is in itself a fully-functional and independent "index" that can be hosted on any node in the cluster.

In a network or cloud environment where failures can be expected anytime, it is very useful and highly recommended having a failover mechanism in case a shard or node somehow goes offline or
disappears for whatever reason. Therefore, OpenSearch enables you to make one or more copies of your index’s shards into what are called replica shards, replicas for short.

**Note**: By default, each index in OpenSearch is allocated 5 primary shards and 1 replica which means that if you have at least two nodes in your cluster, your index will have 5 primary shards and
another 5 replica shards (1 complete replica) for a total of 10 shards per index.

Replicas enable you not to lose thr data of primary shards, when an OpenSearch node is down during user request, so you need to change your `number_of_replicas` setting to 1 replicas or more.
For more information about changing the number\_of\_replicas setting.

An example of the server response when an OpenSearch node is down during a user request is shown below.

```json
    {
      "index": {
        "_index": "company",
        "_type": "employees",
        "_id": "AVuG-cohEb7Fe3xYdTlM",
        "_version": 1,
        "result": "created",
        "_shards": {
          "total": 3,
          "successful": 2,
          "failed": 1,
          "failures": [
            {
              "_index": "company",
              "_shard": 1,
              "_node": "FoJoNczySUC4w0hqNKAVgA",
              "reason": {
                "type": "node_disconnected_exception",
                "reason": "[opensearch-2][10.1.0.2:9300][indices:data/write/bulk[s][r]] disconnected"
              },
              "status": "INTERNAL_SERVER_ERROR",
              "primary": false
            }
          ]
        },
        "created": true,
        "status": 201
      }
    }
```
