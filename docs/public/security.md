The document describes security hardening recommendations for OpenSearch.

# General Consideration

OpenSearch has its own security plugin for authentication and access control. The plugin provides numerous features to help you secure your cluster.
You can find information about security in official documentation [OpenSearch Security](https://opensearch.org/docs/latest/security-plugin/index/).

**Note:** Initial security configurations like username and password or OpenID URL cannot be changed with rolling upgrade.
Corresponding REST API or Dashboards should be used for this purpose.

# Logging

Security events and critical operations should be logged for audit purposes. You can find more detailed information
about audit logs and their configuration in official documentation [OpenSearch Audit Logs](https://opensearch.org/docs/latest/security-plugin/audit-logs/index/).

Samples of audit logs:

* Index creation:

  ```text
  [2022-02-15T06:40:01,965][INFO ][sgaudit ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_transport_headers":{"_system_index_access_allowed":"false"},"audit_node_name":"opensearch-0","audit_trace_task_id":"jxL6tjiZTIiSjxmh6wTGvw:145959","audit_transport_request_type":"CreateIndexRequest","audit_category":"INDEX_EVENT","audit_request_origin":"REST","audit_request_body":"{}","audit_node_id":"jxL6tjiZTIiSjxmh6wTGvw","audit_request_layer":"TRANSPORT","@timestamp":"2022-02-15T06:40:01.964+00:00","audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_request_privilege":"indices:admin/create","audit_node_host_address":"10.129.6.154","audit_request_effective_user":"netcrk","audit_trace_indices":["new_index"],"audit_node_host_name":"10.129.6.154"}
  ```

* Index deletion:

  ```text
  [2022-02-15T06:41:10,814][INFO ][sgaudit ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_transport_headers":{"_system_index_access_allowed":"false"},"audit_node_name":"opensearch-0","audit_trace_task_id":"jxL6tjiZTIiSjxmh6wTGvw:146158","audit_transport_request_type":"DeleteIndexRequest","audit_category":"INDEX_EVENT","audit_request_origin":"REST","audit_node_id":"jxL6tjiZTIiSjxmh6wTGvw","audit_request_layer":"TRANSPORT","@timestamp":"2022-02-15T06:41:10.813+00:00","audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_request_privilege":"indices:admin/delete","audit_node_host_address":"10.129.6.154","audit_request_effective_user":"netcrk","audit_trace_indices":["new_index"],"audit_trace_resolved_indices":["new_index"],"audit_node_host_name":"10.129.6.154"}
  ```

* Failed login:

  ```text
  [2022-02-15T06:44:19,720][INFO ][sgaudit ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_rest_request_params":{"v":""},"audit_node_name":"opensearch-0","audit_rest_request_method":"GET","audit_category":"FAILED_LOGIN","audit_request_origin":"REST","audit_node_id":"jxL6tjiZTIiSjxmh6wTGvw","audit_request_layer":"REST","audit_rest_request_path":"/_cat/indices","@timestamp":"2022-02-15T06:44:19.719+00:00","audit_request_effective_user_is_admin":false,"audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_node_host_address":"10.129.6.154","audit_rest_request_headers":{"User-Agent":["curl/7.79.1"],"content-length":["0"],"Host":["localhost:9200"],"Accept":["*/*"]},"audit_request_effective_user":"netcrk","audit_node_host_name":"10.129.6.154"}
  ```
