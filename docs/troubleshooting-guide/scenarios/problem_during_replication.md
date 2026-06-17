This section deals with troubleshooting problems arising during replication.

OpenSearch provides failover capabilities by keeping multiple copies of your data in the cluster. In presence of network disruptions or failing nodes, changes might not make it to all the copies.

# Replication

Data replication in OpenSearch is based on the primary-backup model. This model assumes a single authoritative copy of the data, called the primary.
All indexing operations first go to the primary, which is then in charge of replicating changes to active backup copies, called replica shards.
OpenSearch uses replica shards to provide failover capabilities, as well as to scale out reads.
In cases where the current primary copy becomes either temporarily or permanently unavailable,
for example due to a server maintenance window or due to a damaged disk drive, another shard copy is selected as primary.
Because the primary is the authoritative copy, it is critical in this model that only shard copies containing the most recent data are selected as primary.
If, for example, an older shard copy was selected as primary after it was on a node that was isolated from the cluster, that old copy would become the definitive copy of the shard,
leading to the loss of all changes that were missed by this copy.

# Marking Shards as Stale

OpenSearch uses the simpler primary-backup approach.
This two-layered system allows data replication to be simple and fast, only requiring interaction with the cluster consensus layer in exceptional situations.
The basic flow for handling document write requests is as follows:

* Based on the current cluster state, the request gets routed to the node that has the primary shard.
* The operation is performed locally on the primary shard, for example by indexing, updating, or deleting a document.
* After successful execution of the operation, it is forwarded to the replica shards. If there are multiple replication targets, forwarding the operation to the replica shards is done concurrently.
* When all replicas have successfully performed the operation and responded to the primary, the primary acknowledges the successful completion of the request to the client.

For more information on the basic flow for handling document requests.

In the case of network partitions, node failures, or general shard unavailability when the node hosting the shard copy is not up, the forwarded operation might not have been successfully performed
on one or more of the replica shard copies. This means that the primary will contain changes that have not been propagated to all shard copies.

There are two solutions to this:

1. Fail the write request and undo the changes on the available copies.
2. Ensure that the divergent shard copies are not considered as in-sync anymore.

OpenSearch chooses the writing availability in this case: The primary instructs the active master to remove the IDs of the divergent shard copies from the in-sync set.
The primary then only acknowledges the write request to the client after it has received confirmation from the master that the in-sync set has been successfully updated by the
consensus layer. This ensures that only shard copies that contain all acknowledged writes can be selected as primary by the master.

For more information, refer to OpenSearch Internals, _Tracking In-Sync Shard Copies_ [https://www.elastic.co/blog/tracking-in-sync-shard-copies](https://www.elastic.co/blog/tracking-in-sync-shard-copies).

# Troubleshooting Procedure

When a major disaster strikes, there may be situations where only stale shard copies are available in the cluster.
OpenSearch will not automatically allocate such shard copies as primary shards, and the cluster will stay red.
In a case where all in-sync shard copies are gone for good, however, there is still a possibility for the cluster to revert to using stale copies,
but this requires manual intervention from the cluster administrator.

An example of the server response when OpenSearch has a problem during replication is as follows.

```json
    {
      "update": {
        "status": 200,
        "_shards": {
          "failures": [
            {
              "primary": false,
              "status": "INTERNAL_SERVER_ERROR",
              "reason": {
                "caused_by": {
                  "reason": "/usr/share/opensearch/data/nodes/0/indices/Mga16xx5SBO8Tmus2Nb7xQ/1/index/write.lock",
                  "type": "no_such_file_exception"
                },
                "index": "cats",
                "shard": "1",
                "index_uuid": "Mga16xx5SBO8Tmus2Nb7xQ",
                "reason": "Index failed for [object#1]",
                "type": "index_failed_engine_exception"
              },
              "_node": "Nznv8nmZTI2F2jMrjbF7gQ",
              "_shard": 1,
              "_index": "cats"
            }
          ],
          "failed": 1,
          "successful": 1,
          "total": 2
        },
        "result": "updated",
        "_version": 2,
        "_id": "1",
        "_type": "object",
        "_index": "cats"
      }
    }
```
