This section deals with troubleshooting the cluster if the elected master is crashed.

# OpenSearch Metric

The Cluster Stats API allows enables retrieving statistics from a cluster-wide perspective.

To retrieve the statistics from a cluster-wide perspective, run the following command:

```sh
curl -XGET http://localhost:9200/_cluster/stats
```

The `nodes` flag can be set to retrieve node-specific statistics:

* `nodes.count.total`: Total count of nodes
* `nodes.count.master`: Count of master-eligible nodes

Master-eligible node is a node that has `node.master` set to `true` (default), which makes it eligible to be elected as the master node, which controls the cluster.

# Grafana Dashboard

To retrieve the metric of master nodes count, use the following query:

```text
SELECT last("count_master") FROM "opensearch_clusterstats_nodes" WHERE $timeFilter GROUP BY time($interval) fill(0)
```

# Troubleshooting Procedure

A troubleshooting procedure is not needed in cases when the leader node has crashed and all other nodes are still capable of communicating, because OpenSearch will handle this automatically.
The remaining nodes will detect the failure of the leader and initiate leader election.

## Master Election

The problem arises when a node falls down or there is a lapse in communication between nodes for some reason.
If one of the slave nodes cannot communicate with the master node, it initiates the election of a new master node from those it is still connected with.
That new master node then will take over the duties of the previous master node.
If the older master node rejoins the cluster or communication is restored, the new master node will demote it to a slave so there is no conflict.

However, consider a scenario where you have just two nodes - one master and one slave.
If communication between the two is disrupted, the slave will be promoted to a master, but once communication is restored, you end up with two master nodes.
The original master node thinks the slave dropped and should rejoin as a slave, while the new master thinks the
original master dropped and should rejoin as a slave. Your cluster, therefore, has a "split brain."

To prevent this, you need a sort of tie-breaker, in the form of a third node.
That third node either remains with the original master and knows that the new master dropped, or sees the old master drop and participates in the election of the new master.
Therefore, there is no conflict.

Split-brain can still be an issue with three or more nodes, however. To help mitigate this situation from occurring, OpenSearch provides a config setting, `discovery.zen.minimum_master_nodes`.
This sets a minimum amount of nodes that must be "alive" in the cluster before a new master can be elected.
For example, in a three node cluster, a value of two would prevent a single node that became disconnected from electing itself as master and doing its own thing.
Instead, it would simply have to wait until it rejoined the cluster. The formula for determining the value to use is: N / 2 + 1, where N is the total number of nodes in your cluster.

For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/latest/opensearch/cluster].
