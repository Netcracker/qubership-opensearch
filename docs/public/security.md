The document describes security hardening recommendations for OpenSearch.

## Exposed Ports

List of ports used by OpenSearch and other Services. 

| Port | Service                  | Description                                                                                                                       |
|------|--------------------------|-----------------------------------------------------------------------------------------------------------------------------------|
| 9200 | OpenSearch               | The port of OpenSearch.                                                                                                           |
| 9300 | OpenSearch               | Port used for transport communication.                                                                                            |
| 9600 | OpenSearch               | Port used for metrics collection.                                                                                                 |
| 9650 | OpenSearch               | Port used for RCA.                                                                                                                |
| 8080 | OpenSearch DBaaS adapter | The port of monitored OpenSearch DBaaS adapter.                                                                                   |
| 8443 | OpenSearch DBaaS adapter | Port is used if TLS for Opensearch DBaaS Adapter is enabled.                                                                      |
| 8443 | OpenSearch Curator       | Port used for OpenSearch Curator when TLS is enabled.                                                                             |
| 8080 | OpenSearch Curator       | Port used for OpenSearch Curator when TLS is not enabled.                                                                         |
| 5601 | OpenSearch Dashboard     | Port used for the dashboard's service.                                                                                            |
| 9200 | OpenSearch Data svc      | The port used for HTTP communication.                                                                                             |
| 9300 | OpenSearch Data svc      | Port used for transport communication.                                                                                            |
| 9600 | OpenSearch Data svc      | Port used for metrics collection.                                                                                                 |
| 9650 | OpenSearch Data svc      | Port used for RCA.                                                                                                                |
| 9300 | OpenSearch Discovery     | Port used for discovery processes within an OpenSearch cluster.                                                                   |
| 9200 | Opensearch Internal      | Port used for internal communication within the OpenSearch cluster.                                                               |
| 8125 | OpenSearch Monitoring    | Port used for StatsD monitoring in OpenSearch.                                                                                    |
| 8094 | OpenSearch Monitoring    | Port used for TCP monitoring in OpenSearch.                                                                                       |
| 8092 | OpenSearch Monitoring    | Port used for UDP monitoring in OpenSearch.                                                                                       |
| 8096 | OpenSearch Monitoring    | Port used for monitoring if `monitoring.monitoringType` is not `influxdb`.                                                        |
| 8443 | DRD                      | If TLS for Disaster Recovery is enabled the HTTPS protocol and port 8443 is used for API requests to ensure secure communication. |
| 8080 | DRD                      | Port used for SiteManager endpoints.                                                                                              |
| 8080 | Integration-tests        | Exposes the container's port to the network. It allows access to the application running in the container.                        |

## User Accounts

List of user accounts used for OpenSearch.

| Service                  | OOB accounts | Deployment parameter                           | Is Break Glass account | Can be blocked | Can be deleted | Comment                                                                                                             |
|--------------------------|--------------|------------------------------------------------|------------------------|----------------|----------------|---------------------------------------------------------------------------------------------------------------------|
| OpenSearch               | admin        | opensearch.securityConfig.authc.basic.username | yes                    | no             | no             | The default admin user. There is no default value, the name must be specified during deploy.                        |
| OpenSearch DBaaS adapter | client       | dbaasAdapter.dbaasUsername                     | no                     | yes            | yes            | The name of the OpenSearch DBaaS adapter user. There is no default value, the name must be specified during deploy. |
| OpenSearch Curator       | client       | curator.username                               | no                     | yes            | yes            | The name of the OpenSearch Curator API user. There is no default value, the name must be specified during deploy.   |

## Disabling User Accounts

OpenSearch does not support disabling user accounts.

## Password Policies

* Passwords must be at least 8 characters long. This ensures a basic level of complexity and security.
* The passwords can contain only the following symbols:
    * Alphabets: a-zA-Z
    * Numerals: 0-9
    * Punctuation marks: ., ;, !, ?
    * Mathematical symbols: -, +, *, /, %
    * Brackets: (, ), {, }, <, >
    * Additional symbols: _, |, &, @, $, ^, #, ~

**Note**: To ensure that passwords are sufficiently complex, it is recommended to include:

* A minimum length of 8 characters
* At least one uppercase letter (A-Z)
* At least one lowercase letter (a-z)
* At least one numeral (0-9)
* At least one special character from the allowed symbols list

## Changing password guide

OpenSearch Service supports the automatic password change procedure.
Any credential in the `opensearch.securityConfig.authc.basic` section can be changed and run with upgrade procedure.
Operator performs necessary logic to apply new credentials to OpenSearch pods.

The manual password changing procedures for OpenSearch Service is described in respective guide:


* [Password changing guide](/docs/public/password-changing.md)

# General Consideration

OpenSearch has its own security plugin for authentication and access control. The plugin provides numerous features to help you secure your cluster.
You can find information about security in official documentation [OpenSearch Security](https://opensearch.org/docs/latest/security-plugin/index/).

**Note:** Initial security configurations like username and password or OpenID URL cannot be changed with rolling upgrade.
Corresponding REST API or Dashboards should be used for this purpose.

# Logging

Security events and critical operations should be logged for audit purposes. You can find more details about enabling 
audit logging in [Audit Guide](/docs/public/audit.md).

Samples of audit logs:

* Index creation:

  ```text
  [2022-02-15T06:40:01,965][INFO ][sgaudit ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_transport_headers":{"_system_index_access_allowed":"false"},"audit_node_name":"opensearch-0","audit_trace_task_id":"jxL6tjiZTIiSjxmh6wTGvw:145959","audit_transport_request_type":"CreateIndexRequest","audit_category":"INDEX_EVENT","audit_request_origin":"REST","audit_request_body":"{}","audit_node_id":"jxL6tjiZTIiSjxmh6wTGvw","audit_request_layer":"TRANSPORT","@timestamp":"2022-02-15T06:40:01.964+00:00","audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_request_privilege":"indices:admin/create","audit_node_host_address":"10.129.6.154","audit_request_effective_user":"admin","audit_trace_indices":["new_index"],"audit_node_host_name":"10.129.6.154"}
  ```

* Index deletion:

  ```text
  [2022-02-15T06:41:10,814][INFO ][sgaudit ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_transport_headers":{"_system_index_access_allowed":"false"},"audit_node_name":"opensearch-0","audit_trace_task_id":"jxL6tjiZTIiSjxmh6wTGvw:146158","audit_transport_request_type":"DeleteIndexRequest","audit_category":"INDEX_EVENT","audit_request_origin":"REST","audit_node_id":"jxL6tjiZTIiSjxmh6wTGvw","audit_request_layer":"TRANSPORT","@timestamp":"2022-02-15T06:41:10.813+00:00","audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_request_privilege":"indices:admin/delete","audit_node_host_address":"10.129.6.154","audit_request_effective_user":"admin","audit_trace_indices":["new_index"],"audit_trace_resolved_indices":["new_index"],"audit_node_host_name":"10.129.6.154"}
  ```

* Failed login:

  ```text
  [2022-02-15T06:44:19,720][INFO ][sgaudit ] [opensearch-0] {"audit_cluster_name":"opensearch","audit_rest_request_params":{"v":""},"audit_node_name":"opensearch-0","audit_rest_request_method":"GET","audit_category":"FAILED_LOGIN","audit_request_origin":"REST","audit_node_id":"jxL6tjiZTIiSjxmh6wTGvw","audit_request_layer":"REST","audit_rest_request_path":"/_cat/indices","@timestamp":"2022-02-15T06:44:19.719+00:00","audit_request_effective_user_is_admin":false,"audit_format_version":4,"audit_request_remote_address":"127.0.0.1","audit_node_host_address":"10.129.6.154","audit_rest_request_headers":{"User-Agent":["curl/7.79.1"],"content-length":["0"],"Host":["localhost:9200"],"Accept":["*/*"]},"audit_request_effective_user":"admin","audit_node_host_name":"10.129.6.154"}
  ```
