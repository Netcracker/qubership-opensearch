This section deals with troubleshooting the network connection after it was lost and restored.

# OpenSearch Metric

The Cluster Health API enables retrieving statistics from a cluster-wide perspective.
For more information, refer to the official OpenSearch documentation, _Cluster Health_ [https://opensearch.org/docs/latest/opensearch/rest-api/cluster-health].

To retrieve statistics from a cluster-wide perspective, run the following command:

```sh
curl -XGET http://localhost:9200/_cluster/health
```

The cluster health status is either green, yellow, or red.
On the shard level, a red status indicates that the specific shard is not allocated in the cluster, yellow means that the primary shard is allocated but replicas are not, and green means that all
shards are allocated. The index level status is determined by the worst shard status.
The cluster status is determined by the worst index status.

A status value other than green indicates that there are some problems within the cluster.

# Troubleshooting Procedure

A troubleshooting procedure is not needed in cases when the network connection is temporarily disrupted between OpenSearch nodes, because such cases are handled by fault detection processes in
OpenSearch.

## Fault Detection

There are two fault detection processes running. The first is by the master, to ping all the other nodes in the cluster and verify that they are alive.
And on the other end, each node pings the master to verify whether it is still alive, or whether an election process needs to be initiated.

The following settings control the fault detection process using the `discovery.zen.fd` prefix:

| Setting        | Description                                                                            |
|----------------|----------------------------------------------------------------------------------------|
| ping\_interval | How often a node gets pinged. Defaults to 1s.                                          |
| ping\_timeout  | How long to wait for a ping response, defaults to 30s.                                 |
| ping\_retries  | How many ping failures / timeouts cause a node to be considered failed. Defaults to 3. |

If one of the nodes cannot communicate with the master node, then it initiates the election of a new master node from those it is still connected with.

However, split-brain can still be an issue with three or more nodes. To prevent this situation from occurring, OpenSearch provides a config setting, `discovery.zen.minimum_master_nodes`.
This sets a minimum amount of nodes that must be "alive" in the cluster before a new master can be elected.
For example, in a three node cluster, a value of two would prevent a single node that became disconnected from electing itself as master and doing its own thing.
Instead, it would simply have to wait until it rejoined the cluster. The formula for determining the value to use is `N/2 + 1`, where `N` is the total number of nodes in your cluster.

When network connection between all nodes is restored, then they rejoin the cluster and continue the operation.
