This section covers the unexpected availability zone shutdown for any reason, which means the shutdown of several OpenSearch nodes.

The OpenShift administrator cannot detect an availability zone outage but can only assume the occurrence of this scenario.

To avoid the appearance of this scenario, implement the following recommendation: The OpenSearch should be evenly distributed across your availability zones, more than 1.
If you have more than one availability zone, then you do not need to worry about the shutdown of one of them.

# OpenSearch Metric

To see the number of active nodes, use the following command:

```sh
curl -XGET http://localhost:9200/_cat/nodes/?v
```

Response example:

```text
ip           heap.percent ram.percent cpu load_1m load_5m load_15m node.role master name
10.129.6.154           28          78   3    0.49    0.45     0.43 dimr      -      opensearch-0
10.130.5.131           48          78   3    0.64    0.69     0.56 dimr      -      opensearch-2
10.128.7.6             37          78   3    1.17    1.09     0.98 dimr      *      opensearch-1
```

To see the number of the unassigned shards with `NODE_LEFT` as the value of the `unassigned.reason` column, use the following command:

```sh
curl -XGET http://localhost:9200/_cat/shards?v&h=index,shard,prirep,state,docs,store,ip,node,unassigned.reason,unassigned.details
```

Response example:

```text
index                shard prirep state   docs  store ip           node
dbaas_metadata       0     p      STARTED    1    3kb 10.128.7.6   opensearch-1
dbaas_metadata       0     r      STARTED    1    3kb 10.130.5.131 opensearch-2
.kibana_1            0     r      STARTED    0   208b 10.128.7.6   opensearch-1
.kibana_1            0     p      STARTED    0   208b 10.129.6.154 opensearch-0
.opendistro_security 0     p      STARTED    9 42.1kb 10.128.7.6   opensearch-1
.opendistro_security 0     r      STARTED    9 42.1kb 10.129.6.154 opensearch-0
.opendistro_security 0     r      STARTED    9 42.1kb 10.130.5.131 opensearch-2
```

# Grafana Dashboard

The following graphs of the dashboard provide information:

1. The **Nodes status** graph provides the number of running nodes and downed nodes.
2. The **Unassigned shards** graph provides the number of unassigned shards, but does not provide the reason. To detect the reason, see the OpenSearch endpoint.

# Troubleshooting Procedure

If free resources are available on other availability zones, then OpenSearch should be scaled up.
