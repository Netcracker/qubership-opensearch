# OpenSearch Slow Queries

OpenSearch Slow Queries

## Tags

* `prometheus`
* `opensearch`
* `opensearch_name_and_namespace`
* `slow-queries`

## Panels

### Overview

![Dashboard](/docs/public/images/opensearch-slow-queries_overview.png)

<!-- markdownlint-disable line-length -->
| Name | Description | Thresholds | Repeat |
| ---- | ----------- | ---------- | ------ |
| Slow Queries Information | The slowest queries in processing interval with index name, shard, query, start time, number of found documents and spent time in descending order of spent time.<br/><br/>The table shows the slowest part of the query on a particular shard for each index query.<br/><br/>If the table is empty, there are no slow queries in OpenSearch. | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Slowest Query | The time of the slowest query in processing interval | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
<!-- markdownlint-enable line-length -->
