# OpenSearch Replication

OpenSearch Replication

## Tags

* `prometheus`
* `opensearch`
* `opensearch_name_and_namespace`
* `replication`

## Panels

### OpenSearch Replication

<!-- markdownlint-disable line-length -->
| Name | Description | Thresholds | Repeat |
| ---- | ----------- | ---------- | ------ |
| Replication status | Status of OpenSearch cross-cluster replication.<br/><br/>If the cluster status is `Degraded`, at least one replicated indices is in `Failed` state. <br/>   <br/>A `Failed` replication status indicates that all indices are in `Failed` state.<br/><br/>A `Not in progress` status means that replication from remote cluster disabled. |  |  |
| Syncing indices | The number of indices that are in the `Syncing` state. | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Bootstrapping indices | The number of indices that are in the `Bootstrapping` state. | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Paused indices | The number of indices that are in the `Paused` state. | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Failed indices | The number of indices that are in the `Failed` state. | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Replication status transitions | Transitions of OpenSearch replication status |  |  |
| Indices status | Status of OpenSearch replicated indices | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Indices status transitions | Transitions of status for OpenSearch replicated indices |  |  |
| Syncing indices lag | The replication lag between leader and follower sides. Evaluated as difference between `leader_checkpoint` and `follower_checkpoint`. | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Syncing indices rate | Cross-cluster replication rate per second | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
<!-- markdownlint-enable line-length -->
