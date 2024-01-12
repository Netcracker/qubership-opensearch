- [Introduction](#introduction)
- [Prerequisites](#prerequisites)
- [Deployment](#deployment)
    - [Configuration](#configuration)
        - [OpenSearch Service Integration Tests Parameters](#opensearch-service-integration-tests-parameters)
    - [Manual Deployment](#manual-deployment)
    - [Deployment via DP Deployer Job](#deployment-via-dp-deployer-job)
    - [Deployment via Groovy Deployer Job](#deployment-via-groovy-deployer-job)

# Introduction

This guide covers the necessary steps to install and execute an OpenSearch tests on OpenShift/Kubernetes using Helm.
The chart installs OpenSearch service integration tests deployment and secret in OpenShift/Kubernetes.

# Prerequisites

* Kubernetes 1.11+ or OpenShift 3.11+
* `kubectl` 1.11+ or `oc` 3.11+ CLI
* Helm 3.0+

# Deployment

OpenSearch service integration tests installation is based on Helm Chart. Helm Chart is placed in [integration-tests](../../charts/helm/opensearch-integration-tests)
directory.

## Environment

OpenSearch Service Integration Tests can only be performed in environment with installed OpenSearch Service.

OpenSearch Service Integration Tests service may conflict with already presented OpenSearch Service Integrations Tests resources if integration tests were enabled on deploy job.
If you want to perform integration tests as separated job on environment where integration tests pod is already presented firstly, you need
to run upgrade OpenSearch Service job with disabled execution of integration tests (`integrationTests.enabled: false`) and then run integration tests job.

## Configuration

This section provides the list of parameters required for OpenSearch Service Integration Tests installation and execution.

### OpenSearch Service Integration Tests Parameters

The `service.name` parameter specifies the name of OpenSearch integration tests service.

The `secret.opensearch.username` parameter specifies the username for OpenSearch authentication.

The `secret.opensearch.password` parameter specifies the password for OpenSearch authentication.

The `secret.idp.username` parameter specifies the name of the user for Identity Provider.
This parameter must be specified if you want to run integration tests with `authentication` tag.

The `secret.idp.password` parameter specifies the password for Identity Provider.
This parameter must be specified if you want to run integration tests with `authentication` tag.

The `secret.idp.registrationToken` parameter specifies the registration token for Identity Provider.
This parameter must be specified if you want to run integration tests with `authentication` tag.

The `secret.dbaasAdapter.username` parameter specifies the name of the DBaaS adapter user.

The `secret.dbaasAdapter.password` parameter specifies the password of the DBaaS adapter user.

The `secret.curator.username` parameter specifies the username of the OpenSearch Curator API user.
It can be empty if authentication is disabled for OpenSearch Curator.

The `secret.curator.password` parameter specifies the password of the OpenSearch Curator API user.
It can be empty if authentication is disabled for OpenSearch Curator.

The `tls.opensearch.secretName` parameter specifies the name of the secret that contains TLS certificates for OpenSearch REST layer. By default, it is empty.

The `tls.opensearch.secretCaKey` parameter specifies the key of root CA certificate in `tls.opensearch.secretName` secret. The default value is `ca.crt`.

The `tls.curator.secretName` parameter specifies the name of the secret that contains TLS certificates for OpenSearch curator. By default, it is empty.

The `tls.dbaasAdapter.secretName` parameter specifies the name of the secret that contains TLS certificates for OpenSearch DBaaS Adapter. By default, it is empty.

The `integrationTests.dockerImage` parameter specifies the docker image of OpenSearch Service integration tests.
The default value is `artifactorycn.netcracker.com:17008/product/prod.platform.elasticstack_opensearch-service:master_latest_integration-tests`.

The `integrationTests.tags` parameter specifies the tags combined with `AND`, `OR` and `NOT` operators that select test cases to run. The default value is `smoke`. You can use the following tags:

* `smoke` tag runs all tests connected to the smoke scenario:
  * `index` tag runs all tests connected to OpenSearch index scenarios:
    * `create_index` tag runs `Create Index` test.
    * `get_index` tag runs `Get Index` test.
    * `delete_index` tag runs `Delete Index` test.
  * `document` tag runs all tests connected to document scenarios:
    * `create_document` tag runs `Create Document` test.
    * `search_document` tag runs `Search Document` test.
    * `update_document` tag runs `Update Document` test.
    * `delete_document` tag tuns `Delete Document` test.
* `authentication` tag runs all tests connected to authentication scenarios:
  * `basic_authentication`  tag runs all tests connected to basic authentication scenarios.
  * `oauth` tag runs all tests connected to OAUTH scenarios.
* `regression` tag runs all tests connected to regression scenarios.
* `opensearch` tag runs all tests connected to OpenSearch scenarios:
  * `backup` tag runs all tests connected to the backup scenarios except `Full Backup And Restore` test:
    * `Full Backup And Restore` test is performed when `full_backup` tag is specified explicitly.   
    * `Full Backup And Restore On S3 Storage` test is performed when `full_backup` tag is specified explicitly.
    * `find_backup` tag runs `Find Backup By Timestamp` test.
    * `granular_backup` tag runs `Granular Backup And Restore`, `Granular Backup And Restore On S3 Storage` and `Granular Backup And Restore By Timestamp` tests.
    * `granular_backup_s3` tag runs `Granular Backup And Restore On S3 Storage` test.
    * `backup_deletion` tag runs `Delete Backup By ID` test.
    * `unauthorized_access` tag runs `Unauthorized Access` test.
  * `prometheus` tag runs all tests connected to Prometheus scenarios:
    * `opensearch_prometheus_alert` tag runs all tests connected to Prometheus alerts scenarios:
      * `opensearch_is_degraded_alert` tag runs `OpenSearch Is Degraded Alert` test.
      * `opensearch_is_down_alert` tag runs `OpenSearch Is Down Alert` test.
    * `slow_query` tag runs `Produce Slow Query Metric` test.
* `dbaas` tag runs all tests connected to DBaaS adapter scenarios:
  * `dbaas_backup` tag runs all tests connected to DBaaS adapter backup scenarios:
    * `dbaas_create_backup` tag runs `Create Backup By Dbaas Adapter` test.
    * `dbaas_delete_backup` tag runs `Delete Backup By Dbaas Adapter` test.
    * `dbaas_restore_backup` tag runs `Restore Backup By Dbaas Adapter` test.
  * `dbaas_opensearch` tag runs all tests connected to DBaaS adapter and OpenSearch scenarios:
    * `dbaas_index` tag runs all tests connected to DBaaS adapter index scenarios with specific DBaaS adapter API (`v1`):
      * `dbaas_create_index` tag runs `Create Index By Dbaas Adapter` test.
      * `dbaas_delete_index` tag runs `Delete Index By Dbaas Adapter` test.
      * `dbaas_create_index_and_write_data` tag runs `Create Index By Dbaas Adapter And Write Data` test.
      * `dbaas_create_index_with_user_and_write_data` tag runs `Create Index With User By Dbaas Adapter And Write Data` test.
    * `dbaas_resource_prefix` tag runs all tests connected to DBaaS adapter resource prefix scenarios with specific DBaaS adapter API:
      * `dbaas_create_resource_prefix` tag runs `Create Database Resource Prefix` test with DBaaS adapter `v1` API.
      * `dbaas_create_resource_prefix_with_metadata` tag runs `Create Database Resource Prefix With Metadata` test with DBaaS adapter `v1` API.
      * `dbaas_resource_prefix_authorization` tag runs `Database Resource Prefix Authorization` test with DBaaS adapter `v1` API.
      * `dbaas_delete_resource_prefix` tag runs `Delete Database Resource Prefix` test with DBaaS adapter `v1` API.
      * `dbaas_create_resource_prefix_for_multiple_users` tag runs `Create Database Resource Prefix for Multiple Users` test with DBaaS adapter `v2` API.
      * `dbaas_create_resource_prefix_with_metadata_for_multiple_users` tag runs `Create Database Resource Prefix With Metadata for Multiple Users` test with DBaaS adapter `v2` API.
      * `dbaas_create_with_custom_resource_prefix_for_multiple_users` tag runs `Create Database With Custom Resource Prefix for Multiple Users` test with DBaaS adapter `v2` API.
      * `dbaas_change_password_for_dml_user` tag runs `Change Password for DML User` test with DBaaS adapter `v2` API.
      * `dbaas_delete_resource_prefix_for_multiple_users` tag runs `Delete Database Resource Prefix for Multiple Users` test with DBaaS adapter `v2` API.
    * `dbaas_recovery` tag runs tests connected to recovery users in OpenSearch via DBaaS adapter:
      * `dbaas_recover_users` tag runs `Recover Users In OpenSearch` test with DBaaS adapter `v2` API.
    * `dbaas_v1` tag runs all tests connected to DBaaS adapter v1 scenarios.
    * `dbaas_v2` tag runs all tests connected to DBaaS adapter v2 scenarios.
* `ha` tag runs all tests connected to HA scenarios:
  * `opensearch_ha` tag runs all tests connected to OpenSearch HA scenarios:
    * `ha_elected_master_is_crashed` tag runs `Elected Master Is Crashed` test.
    * `ha_data_files_corrupted_on_primary_shard` tag runs `Data Files Corrupted On Primary Shard` test.
    * `ha_data_files_corrupted_on_replica_shard` tag runs `Data Files Corrupted On Replica Shard` test.

**Note**: It is not recommended to start `full_backup` tests on externally managed cloud with a lot of indices. To run `full_backup` tag need to specify this tag explicitly.

The `integrationTests.opensearchProtocol` parameter specifies the OpenSearch protocol. The default value is `http`.

The `integrationTests.opensearchHost` parameter specifies the host name of OpenSearch. The default value is `OpenSearch`.

The `integrationTests.opensearchPort` parameter specifies the OpenSearch port. The default value is `9200`.

The `integrationTests.opensearchMasterNodesName` parameter specifies the name of OpenSearch master nodes.
The default value is `opensearch`.

The `integrationTests.slowQueriesIntervalMinutes` parameter specifies the time in minutes of metrics collection frequency for slow queries in OpenSearch. This parameter must be specified if the `slowQueries` functionality is enabled, and you want to run integration tests with `slow_query` tag.

The `integrationTests.identityProviderUrl` parameter specifies the URL of Identity Provider.
This parameter must be specified if you want to run integration tests with `authentication` tag.

The `integrationTests.dbaasAdapterType` parameter specifies the type of DBaaS adapter that is used in OpenSearch service installation. The possible values are `opensearch` and `elasticsearch`. The default value is `opensearch`.

The `integrationTests.opensearchDbaasAdapterHost` parameter specifies the host name of DBaaS OpenSearch adapter.
The default value is `dbaas-opensearch-adapter`.

The `integrationTests.opensearchDbaasAdapterPort` parameter specifies the DBaaS OpenSearch adapter port.
The default value is `8080`. Use `8443` for TLS connection.

The `integrationTests.opensearchDbaasAdapterProtocol` parameter specifies the DBaaS OpenSearch adapter protocol.
The default value is `http`. Use `https` for TLS connection.

The `integrationTests.opensearchDbaasAdapterRepository` parameter the name of snapshot repository in OpenSearch.
The default value is `snapshots`.

The `integrationTests.opensearchDbaasAdapterApiVersion` parameter specifies the DBaaS Adapter API version which tests should use.
The default value is `v1`.

The `integrationTests.opensearchCuratorHost` parameter specifies the host name of OpenSearch Curator.
The default value is `opensearch-curator`.

The `integrationTests.prometheusUrl` parameter specifies the URL (with schema and port) to Prometheus.
For example, `http://prometheus.cloud.openshift.sdntest.example.com:80`. This parameter must be
specified if you want to run integration tests with `prometheus` tag.

The `integrationTests.s3.enabled` parameter specifies that Curator stored backups in S3 storage. OpenSearch supports the following S3 providers: AWS S3, MinIO. Google Cloud Storage is not supported. Other S3 providers may work, but are not covered by the OpenSearch test suite.

The `integrationTests.s3.url` parameter specifies the URL to the S3 storage.

The `integrationTests.s3.bucket` parameter specifies the existing bucket in the S3 storage.

The `integrationTests.statusWritingEnabled` parameter specifies whether status of Integration tests execution is to be
writen to deployment or not. The default value is `true`.

The `integrationTests.isShortStatusMessage` parameter specifies whether status message contains only first line of
`result.txt` file or not. The parameter makes no sense if `statusWritingEnabled` parameter is not set to `true`.
The default value is `true`.

The `integrationTests.resources.requests.cpu` parameter specifies the minimum number of CPUs the container should use.
The default value is `200m`.

The `integrationTests.resources.requests.memory` parameter specifies the minimum amount of memory the container should use.
The value can be specified with SI suffixes (E, P, T, G, M, K, m) or their power-of-two-equivalents (Ei, Pi, Ti, Gi, Mi, Ki).
The default value is `256Mi`.

The `integrationTests.resources.limits.cpu` parameter specifies the maximum number of CPUs the container can use.
The default value is `400m`.

The `integrationTests.resources.limits.memory` parameter specifies the maximum amount of memory the container can use.
The value can be specified with SI suffixes (E, P, T, G, M, K, m) or their power-of-two-equivalents (Ei, Pi, Ti, Gi, Mi, Ki).
The default value is `256Mi`.

The `integrationTests.affinity` parameter specifies the affinity scheduling rules. The value should be specified in json
format. The parameter can be empty.

The `integrationTests.securityContext` Parameter allows specifying pod security context for the OpenSearch integration tests pod. 
The parameter is empty by default.

## Manual Deployment

### Installation

To deploy OpenSearch service integration tests with Helm you need to customize the `values.yaml` file. For example:

```
service:
  name: opensearch-integration-tests

secret:
  opensearch:
    username: "admin"
    password: "admin"
  idp:
    username: "admin"
    password: "admin"
    registrationToken: "jyK2ztNwKMlO0fHKocPQW2glUC0Tg"
  dbaasAdapter:
    username: "admin"
    password: "admin"
  curator:
    username: "admin"
    password: "admin"

integrationTests:
  dockerImage: "artifactorycn.netcracker.com:17008/product/prod.platform.elasticstack_opensearch-service:master_latest_integration-tests"
  tags: "smoke"
  opensearchHost: "opensearch"
  opensearchPort: 9200
  opensearchMasterNodesName: "opensearch"
  identityProviderUrl: "http://identity-management.security-services-ci.svc:8080"
  opensearchDbaasAdapterHost: "dbaas-opensearch-adapter"
  opensearchDbaasAdapterPort: 8080
  opensearchDbaasAdapterProtocol: "http"
  opensearchDbaasAdapterRepository: "snapshots"
  opensearchCuratorHost: "opensearch-curator"
  opensearchCuratorPort: 8080

  prometheusUrl: "http://prometheus.cloud.openshift.sdntest.example.com:80"

  statusWritingEnabled: true
  isShortStatusMessage: true

  resources:
    requests:
      memory: 256Mi
      cpu: 100m
    limits:
      memory: 256Mi
      cpu: 400m
```

To deploy the service you need to execute the following command:

```
helm install ${RELEASE_NAME} ./opensearch-integration-tests -n ${NAMESPACE}
```

where:

* `${RELEASE_NAME}` is the release name of Helm Chart for OpenSearch service integration tests.
  For example, `opensearch-integration-tests`.
* `${NAMESPACE}` is the OpenShift/Kubernetes project/namespace to deploy OpenSearch service integration tests.
  For example, `opensearch-service`.

You can monitor the deployment process in the OpenShift/Kubernetes dashboard or using `kubectl` in the command line:

```
kubectl get pods
```

### Uninstalling

To uninstall OpenSearch service integration tests from OpenShift/Kubernetes you need to execute the following command:

```
helm delete ${RELEASE_NAME} -n ${NAMESPACE}
```

where:

* `${RELEASE_NAME}` is the release name of existing Helm Chart for OpenSearch service integration tests.
  For example, `opensearch-integration-tests`.
* `${NAMESPACE}` is the OpenShift/Kubernetes project/namespace where OpenSearch service integration tests
  is deployed. For example, `opensearch-service`.

The command uninstalls all the Kubernetes/OpenShift resources associated with the chart and deletes the release.

## Deployment via DP Deployer Job

Navigate to the Jenkins job `DP.Pub.Helm_deployer` and then click **Build with parameters**.

The job parameters are predefined and described as follows:

The `CLOUD_URL` parameter specifies the URL of the OpenShift/Kubernetes server. For example, `https://search.openshift.sdntest.example.com:8443`.

The `CLOUD_NAMESPACE` parameter specifies the name of the existing OpenShift project/Kubernetes namespace. For example,
`opensearch-service`.

The `CLOUD_USER` parameter specifies the name of the user on behalf of whom the deployment process in
OpenShift/Kubernetes starts. The parameter should be specified with `CLOUD_PASSWORD` parameter if `CLOUD_TOKEN` parameter
is not filled.

The `CLOUD_PASSWORD` parameter specifies the password for the user on behalf of whom the deployment process in
OpenShift/Kubernetes starts. The parameter should be specified with `CLOUD_USER` parameter if `CLOUD_TOKEN` parameter
is not filled.

The `CLOUD_TOKEN` parameter specifies the token for the user on behalf of whom the deployment process in
OpenShift/Kubernetes starts. The parameter should be specified if `CLOUD_USER` and `CLOUD_PASSWORD` parameters are not
filled.

The `DESCRIPTOR_URL` parameter specifies the link to the OpenSearch Service Application Manifest.

The `DEPLOYMENT_PARAMETERS` parameter specifies the yaml that contains all parameters for installation. For example,

```
service:
  name: opensearch-integration-tests

secret:
  opensearch:
    username: "admin"
    password: "admin"
  idp:
    username: "admin"
    password: "admin"
    registrationToken: "jyK2ztNwKMlO0fHKocPQW2glUC0Tg"
  dbaasAdapter:
    username: "admin"
    password: "admin"
  curator:
    username: "admin"
    password: "admin"

integrationTests:
  tags: "smoke"
  opensearchHost: "opensearch"
  opensearchPort: 9200
  opensearchMasterNodesName: "opensearch"
  identityProviderUrl: "http://identity-management.security-services-ci.svc:8080"
  opensearchDbaasAdapterHost: "dbaas-opensearch-adapter"
  opensearchDbaasAdapterPort: 8080
  opensearchDbaasAdapterProtocol: "http"
  opensearchDbaasAdapterRepository: "snapshots"
  opensearchCuratorHost: "opensearch-curator"
  opensearchCuratorPort: 8080

  prometheusUrl: "http://prometheus.cloud.openshift.sdntest.example.com:80"

  statusWritingEnabled: true
  isShortStatusMessage: true

  resources:
    requests:
      memory: 256Mi
      cpu: 100m
    limits:
      memory: 256Mi
      cpu: 400m
```

The `DEPLOYMENT_MODE` parameter specifies the mode of the deployment. The possible values are `install`, `upgrade`, `auto`,
`reinstall`, `clean`, `rollback` and `kubectl apply`.

The `ADDITIONAL_OPTIONS` parameter specifies the additional options for Helm install/upgrade commands. For example,
`--skip-crds` can be used in case of installation with restricted rights.

Click *Build*.

## Deployment via Groovy Deployer Job

Navigate to the Jenkins job `groovy.deploy.v3` and then click **Build with parameters**.

The job parameters are predefined and described as follows:

The `PROJECT` parameter specifies the name of the existing OpenShift project/Kubernetes namespace. For example,
`opensearch-service`.

The `OPENSHIFT_CREDENTIALS` parameter specifies the credentials of the user on behalf of whom the deployment process in
OpenShift/Kubernetes starts.

The `DEPLOY_MODE` parameter specifies the mode of the deployment. The possible values are `Clean Install` and
`Rolling Upgrade`. The `Clean Install` mode removes everything from the project before deployment.

The `ARTIFACT_DESCRIPTOR_VERSION` parameter specifies the version of maven artifact in the format `artifactId:artifactVersion`.
For example, `opensearch-service-integration-tests:opensearch_service_integration_tests_v01`.

The `CUSTOM_PARAMS` parameter specifies the list of parameters for OpenSearch service installation. All parameters
should be divided by `;`. For example,

```
service.name=opensearch-integration-tests;

secret.opensearch.username=admin;
secret.opensearch.password=admin;
secret.idp.username=admin;
secret.idp.password=admin;
secret.idp.registrationToken=jyK2ztNwKMlO0fHKocPQW2glUC0Tg;
secret.dbaasAdapter.username=admin;
secret.dbaasAdapter.password=admin;
secret.curator.username=admin;
secret.curator.password=admin;

integrationTests.tags=smoke;
integrationTests.opensearchHost=opensearch;
integrationTests.opensearchPort=9200;
integrationTests.opensearchMasterNodesName=opensearch;
integrationTests.identityProviderUrl=http://identity-management.security-services-ci.svc:8080;
integrationTests.opensearchDbaasAdapterHost=dbaas-opensearch-adapter;
integrationTests.opensearchDbaasAdapterPort=8080;
integrationTests.opensearchDbaasAdapterRepository=snapshots;
integrationTests.opensearchCuratorHost=opensearch-curator;
integrationTests.opensearchCuratorPort=8080;

integrationTests.prometheusUrl=http://prometheus.cloud.openshift.sdntest.example.com:80;

integrationTests.statusWritingEnabled=true;
integrationTests.isShortStatusMessage=true;

integrationTests.resources.requests.memory=256Mi;
integrationTests.resources.requests.cpu=100m;
integrationTests.resources.limits.memory=256Mi;
integrationTests.resources.limits.cpu=400m;
```

Click *Build*.
