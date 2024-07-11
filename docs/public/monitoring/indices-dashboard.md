# OpenSearch Indices

OpenSearch Indices

## Tags

* `prometheus`
* `opensearch`
* `opensearch_name_and_namespace`
* `indices`

## Panels

### Overview

![Dashboard](/docs/public/images/opensearch-indices_overview.png)

<!-- markdownlint-disable line-length -->
| Name | Description | Thresholds | Repeat |
| ---- | ----------- | ---------- | ------ |
| Indices | The number of documents and size in bytes for each index in descending order of size values | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
<!-- markdownlint-enable line-length -->

### Indices Information

![Dashboard](/docs/public/images/opensearch-indices_indices_information.png)

<!-- markdownlint-disable line-length -->
| Name | Description | Thresholds | Repeat |
| ---- | ----------- | ---------- | ------ |
| Incoming Documents Rate | The number of documents added to index on primary shards per second | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Store Size | The size in bytes occupied by index on primary shards and in total | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Indexing Documents Rate | The number of indexing operations in index per second | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
| Deleting Documents Rate | The number of deleted documents from index per second | Default:<br/>Mode: absolute<br/>Level 1: 80<br/><br/> |  |
<!-- markdownlint-enable line-length -->
