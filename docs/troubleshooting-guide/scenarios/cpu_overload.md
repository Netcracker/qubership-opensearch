This section describes the troubleshooting of the CPU Overload scenario.

# OpenSearch Metric

The cluster Nodes Stats API enables retrieving statistics of all the nodes in the cluster.
For more information, refer to the _Official OpenSearch Documentation_ [https://opensearch.org/docs/1.2/opensearch/stats-api](https://opensearch.org/docs/1.2/opensearch/stats-api).

To retrieve the statistics of all the nodes in the cluster, run the following command:

```sh
curl -XGET http://localhost:9200/_nodes/stats
```

The `os` flag can be set to retrieve statistics that concern the operating system:

* `os.cpu.percent`: Recent CPU usage for the whole system, or -1 if not supported.
* `os.cpu.load_average.1m`: One-minute load average on the system. The field is not present if one-minute load average is not available.
* `os.cpu.load_average.5m`: Five-minute load average on the system. The field is not present if five-minute load average is not available.
* `os.cpu.load_average.15m`: Fifteen-minute load average on the system. The field is not present if fifteen-minute load average is not available.

# Troubleshooting Procedure

If you see an increase in CPU usage, this is usually caused by a heavy search or indexing workload.
Set up a notification to find out if your nodes' CPU usage is consistently increasing, and add more nodes to redistribute the load if needed.
You also need to make sure that your indices have enough primary shards to be able to balance their data across all those nodes.

The Nodes hot threads API enables getting the current hot threads on each node in the cluster. The endpoints are `/_nodes/hot_threads`, and `/_nodes/{nodesIds}/hot_threads`.

* `threads`: The number of hot threads to provide. Default value is `3`.
* `interval`: The interval to do the second sampling of threads. Default value is `500ms`.
* `type`: The type to sample. Defaults to cpu, but supports wait and block to see hot threads that are in wait or block state.
* `ignore_idle_threads`: If true, known idle threads such as waiting in a socket select, or to get a task from an empty queue, are filtered out. Default value is `true`.

You can use this API to analyze CPU usage by thread.
