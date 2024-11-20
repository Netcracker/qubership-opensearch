This section provides information on the manual password changing procedures in OpenSearch cluster.

## OpenSearch Secret

The secret contains admin credentials for clients of OpenSearch, for example, OpenSearch monitoring and OpenSearch curator.
For more information, refer to the [Installation Guide](installation.md#parameters).

**Important**: The password should be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one digit, and one special character.

To update internal OpenSearch credentials:

1. Navigate to **OpenShift/Kubernetes > ${NAMESPACE} > Secrets**.
2. Select secret with name **${CLUSTER_NAME}-secret**.
3. Push **Edit resource** button.
4. Update the value of the **username** and **password** property with new credentials in BASE64 encoding.
5. Click **Save**.
6. Restart all services that use this secret or these credentials (for example, OpenSearch monitoring, OpenSearch curator) to apply the newly specified credentials.

Where:

* `${NAMESPACE}` is the name of the OpenShift/Kubernetes namespace where OpenSearch is located. For example, `opensearch-service`.
* `${CLUSTER_NAME}` is the name of the OpenSearch cluster. For example, `opensearch`.

**Note**: OpenSearch dashboards don't support password that contains only digits. Consider this when changing the password property.

# OpenSearch Curator

This section provides information on the password changing procedures in the OpenSearch Curator.

## OpenSearch Curator Secret

The OpenSearch Curator secret contains credentials for OpenSearch Curator.

Use the [OpenSearch Secret](#opensearch-secret) guide to update **username** and **password** for OpenSearch. To update Curator credentials follow the next steps:

1. Navigate to **OpenShift/Kubernetes > ${NAMESPACE} > Secrets**.
2. Select secret with name **${SERVICE_NAME}-secret**.
3. Push **Edit resource** button.
4. Update the value of the **username** and **password** property with new credentials in BASE64 encoding.
5. Click **Save**.
6. Restart OpenSearch Curator to apply the newly specified credentials.

where:

* `${NAMESPACE}` is the name of the OpenShift/Kubernetes namespace where OpenSearch Curator is located. For example, `opensearch-service`.
* `${SERVICE_NAME}` is the service name of the OpenSearch Curator. For example, `opensearch-curator`.
