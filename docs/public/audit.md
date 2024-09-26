This chapter describes the security audit logging for OpenSearch.

<!-- #GFCFilterMarkerStart# -->
The following topics are covered in this chapter:

[[_TOC_]]
<!-- #GFCFilterMarkerEnd# -->

# Common Information

Audit logs let you track access to your OpenSearch cluster and are useful for compliance purposes or in the aftermath of a security breach. 
You can find more detailed information about audit logs and their configuration in official documentation [OpenSearch Audit Logs](https://opensearch.org/docs/latest/security-plugin/audit-logs/index/).

# Configuration

To enable all OpenSearch audit logs need to set `disabledRestCategories` config empty into the 
`opensearch.audit` parameter or specify necessary set of categories. For more information, see [Tracked Events](https://opensearch.org/docs/latest/security/audit-logs/index/#tracked-events).

```yaml
opensearch:
  audit:
    disabledRestCategories: []
```

By default, `opensearch.audit` parameter is empty so default `["AUTHENTICATED", "GRANTED_PRIVILEGES"]` applied. 
In this case successfully authenticated events and successful request events are ignored.

## Example of Events

The audit log format for events are described further:

### Login

A user successfully authenticated.

```text
[2024-08-19T16:34:00,259][INFO ][sgaudit                  ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_node_name":"opensearch-0","audit_request_initiating_user":"admin","audit_rest_request_method":"GET","audit_category":"AUTHENTICATED","audit_request_origin":"REST","audit_node_id":"rqYqpU02R2mh75ogmxy0BA","audit_request_layer":"REST","audit_rest_request_path":"/_all/_stats","@timestamp":"2024-08-19T16:34:00.259+00:00","audit_request_effective_user_is_admin":false,"audit_format_version":4,"audit_request_remote_address":"10.131.6.128","audit_node_host_address":"10.129.187.24","audit_rest_request_headers":{"User-Agent":["Go-http-client/1.1"],"Host":["opensearch-internal:9200"],"Accept-Encoding":["gzip"]},"audit_request_effective_user":"admin","audit_node_host_name":"10.129.187.24"}
```

### Failed Login

The credentials of a request could not be validated, most likely because the user does not exist or the password is incorrect.

```text
[2024-08-19T16:36:14,352][INFO ][sgaudit                  ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_node_name":"opensearch-0","audit_rest_request_method":"GET","audit_category":"FAILED_LOGIN","audit_request_origin":"REST","audit_node_id":"rqYqpU02R2mh75ogmxy0BA","audit_request_layer":"REST","audit_rest_request_path":"/","@timestamp":"2024-08-19T16:36:14.351+00:00","audit_request_effective_user_is_admin":false,"audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_node_host_address":"10.129.187.24","audit_rest_request_headers":{"User-Agent":["curl/8.5.0"],"Host":["localhost:9200"],"Accept":["*/*"]},"audit_request_effective_user":"admin","audit_node_host_name":"10.129.187.24"}
```

### Unauthorized Event

The user does not have the required permissions to make the request.

```text
[2024-08-19T16:40:07,482][INFO ][sgaudit                  ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_node_name":"opensearch-0","audit_trace_task_id":"rqYqpU02R2mh75ogmxy0BA:10805662","audit_transport_request_type":"MainRequest","audit_category":"MISSING_PRIVILEGES","audit_request_origin":"REST","audit_node_id":"rqYqpU02R2mh75ogmxy0BA","audit_request_layer":"TRANSPORT","@timestamp":"2024-08-19T16:40:07.482+00:00","audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_request_privilege":"cluster:monitor/main","audit_node_host_address":"10.129.187.24","audit_request_effective_user":"test","audit_node_host_name":"10.129.187.24"}
```
