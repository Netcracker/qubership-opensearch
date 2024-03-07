This chapter describes the installation and configuration procedures of OpenSearch.

<!-- #GFCFilterMarkerStart# -->
The following topics are covered in this chapter:

[[_TOC_]]
<!-- #GFCFilterMarkerEnd# -->

# Prerequisites

## Common

Before you start the installation and configuration of an OpenSearch cluster, ensure the following requirements are met:

* Kubernetes 1.21+ or OpenShift 4.10+
* `kubectl` 1.21+ or `oc` 4.10+ CLI
* Helm 3.0+
* All required CRDs are installed

### Custom Resource Definitions

The following Custom Resource Definitions should be installed to the cloud before the installation of OpenSearch:

* `OpenSearchService` - When you deploy with restricted rights or the CRDs' creation is disabled by the Deployer job. For more information, see [Automatic CRD Upgrade](#automatic-crd-upgrade).
* `GrafanaDashboard`, `PrometheusRule`, and `ServiceMonitor` - They should be installed when you deploy OpenSearch monitoring with `monitoring.enabled=true` and `monitoring.monitoringType=prometheus`. You need to install the Monitoring Operator service before the OpenSearch installation.
* `SiteManager` - It is installed when you deploy OpenSearch with Disaster Recovery support (`global.disasterRecovery.mode`). You have to install the SiteManager service before the OpenSearch installation.

**Important**: To create CRDs, you must have cloud rights for `CustomResourceDefinitions`. If the deployment user does not have the necessary rights, you need to perform the steps described in the [Deployment Permissions](#deployment-permissions) section before the installation.

**Note**: If you deploy OpenSearch service to Kubernetes version less than 1.16, you have to manually install CRD from `config/crd/old/netcracker.com_opensearchservices.yaml` and disable automatic CRD creation by Helm in the following way:

* Specify the `--skip-crds` in `ADDITIONAL_OPTIONS` parameter of DP Deployer Job.
* Specify `DISABLE_CRD=true;` in the `CUSTOM_PARAMS` parameter of Groovy Deployer Job.

### Deployment Permissions

To avoid using `cluster-wide` rights during the deployment, the following conditions are required:

* The cloud administrator creates the namespace/project in advance.
* The following grants should be provided for the `Role` of the deployment user:

    <details>
    <summary>Click to expand YAML</summary>

    ```yaml
    rules:
      - apiGroups:
          - netcracker.com
        resources:
          - "*"
        verbs:
          - create
          - get
          - list
          - patch
          - update
          - watch
          - delete
      - apiGroups:
          - ""
        resources:
          - pods
          - services
          - endpoints
          - persistentvolumeclaims
          - configmaps
          - secrets
          - pods/exec
          - pods/portforward
          - pods/attach
          - pods/binding
          - serviceaccounts
        verbs:
          - create
          - get
          - list
          - patch
          - update
          - watch
          - delete
      - apiGroups:
          - apps
        resources:
          - deployments
          - deployments/scale
          - deployments/status
        verbs:
          - create
          - get
          - list
          - patch
          - update
          - watch
          - delete
          - deletecollection
      - apiGroups:
          - batch
        resources:
          - jobs
          - jobs/status
        verbs:
          - create
          - get
          - list
          - patch
          - update
          - watch
          - delete
      - apiGroups:
          - ""
        resources:
          - events
        verbs:
          - create
      - apiGroups:
          - apps
        resources:
          - statefulsets
          - statefulsets/scale
          - statefulsets/status
        verbs:
          - create
          - delete
          - get
          - list
          - patch
          - update
          - watch
      - apiGroups:
          - networking.k8s.io
        resources:
          - ingresses
        verbs:
          - create
          - delete
          - get
          - list
          - patch
          - update
      - apiGroups:
          - rbac.authorization.k8s.io
        resources:
          - roles
          - rolebindings
        verbs:
          - create
          - delete
          - get
          - list
          - patch
          - update
      - apiGroups:
          - integreatly.org
        resources:
          - grafanadashboards
        verbs:
          - create
          - delete
          - get
          - list
          - patch
          - update
      - apiGroups:
          - monitoring.coreos.com
        resources:
          - servicemonitors
          - prometheusrules
        verbs:
          - create
          - delete
          - get
          - list
          - patch
          - update
      - apiGroups:
          - policy
        resources:
          - poddisruptionbudgets
        verbs:
          - create
          - get
          - patch
      - apiGroups:
          - cert-manager.io
        resources:
          - certificates
        verbs:
          - create
          - get
          - patch
    ```
    </details>

The following rules require `cluster-wide` permissions. If it is not possible to provide them to the deployment user, you have to apply the resources manually.

* If OpenSearch is installed in the disaster recovery mode and authentication on the disaster recovery server is enabled, cluster role binding for the `system:auth-delegator` role must be created.

  ```yaml
  kind: ClusterRoleBinding
  apiVersion: rbac.authorization.k8s.io/v1
  metadata:
    name: token-review-crb-NAMESPACE
  subjects:
    - kind: ServiceAccount
      name: opensearch-service-operator
      namespace: NAMESPACE
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: system:auth-delegator
  ```

* To avoid applying the CRD manually, the following grants should be provided for `ClusterRole` of the deployment user:

  ```yaml
      rules:
        - apiGroups: ["apiextensions.k8s.io"]
          resources: ["customresourcedefinitions"]
          verbs: ["get", "create", "patch"]
    ```

* The custom resource definition `OpenSearchService` should be created/applied before the installation if the corresponding rights cannot be provided to the deployment user.

    <!-- #GFCFilterMarkerStart# -->  
  The CRD for this version is stored in [crd.yaml](/charts/helm/opensearch-service/crds/crd.yaml) file and can be
  applied with the following command:

    ```
    kubectl replace -f crd.yaml
    ```
    <!-- #GFCFilterMarkerEnd# -->

* To run `privileged` containers or actions on Kubernetes before 1.25 version, apply `PodSecurityPolicy` from [psp.yaml](/charts/helm/opensearch-service/templates/psp.yaml), create `ClusterRole` with its usage, and bind it to `ServiceAccount` with `$OPENSEARCH_FULLNAME` name using `ClusterRoleBinding`.

* To run `privileged` containers or actions on Kubernetes 1.25+ version, provide the `privileged` policy to the OpenSearch namespace. It can be performed with the following command:

  ```bash
  kubectl label --overwrite ns "$OPENSEARCH_NAMESPACE" pod-security.kubernetes.io/enforce=privileged
  ```

  This command can be executed automatically with the `ENABLE_PRIVILEGED_PSS: true` property in the deployment parameters. It requires the following cluster rights for the deployment user:

  ```yaml
    - apiGroups: [""]
      resources: ["namespaces"]
      verbs: ["patch"]
      resourceNames:
      - $OPENSEARCH_NAMESPACE
  ```

* The `global.psp.create` parameter should be defined as "false".

### Multiple Availability Zone

If Kubernetes cluster has several availability zones, it is more reliable to start OpenSearch pods in different availability zones. For more information, refer to [Multiple Availability Zone Deployment](#multiple-availability-zone-deployment).

### Storage Types

The following are a few approaches of storage management used in the OpenSearch service solution deployment:

* Dynamic Persistent Volume Provisioning
* Predefined Persistent Volumes

#### Dynamic Persistent Volume Provisioning

OpenSearch Helm installation supports specifying storage class for master, arbiter, and data Persistent Volume Claims. If you are setting up the persistent volumes' resource in Kubernetes, map the OpenSearch pods to the volume using the `opensearch.master.persistence.storageClass`, `opensearch.arbiter.persistence.storageClass`, or `opensearch.data.persistence.storageClass` parameter.

#### Predefined Persistent Volumes

If you have prepared Persistent Volumes without storage class and dynamic provisioning, you can specify Persistent Volumes names using the `opensearch.master.persistence.persistentVolumes`, `opensearch.arbiter.persistence.persistentVolumes`, or `opensearch.data.persistence.persistentVolumes` parameter.

For example:

```
opensearch:
  master:
    persistence:
      persistentVolumes:
        - pv-opensearch-master-1
        - pv-opensearch-master-2
        - pv-opensearch-master-3
```

Persistent Volumes should be created on the corresponding Kubernetes nodes and should be in the `Available` state.

Set the appropriate UID and GID on hostPath directories and rule for SELinux:

```
chown -R 1000:1000 /mnt/data/<pv-name>
```

You also need to specify node names through `opensearch.master.persistence.nodes`, `opensearch.arbiter.persistence.nodes`, or `opensearch.data.persistence.nodes` parameter in the same order in which the Persistent Volumes are specified so that OpenSearch pods are assigned to these nodes.

According to the specified parameters, the `Pod Scheduler` distributes pods to the necessary Kubernetes nodes. For more information, refer to [Pod Scheduler](#pod-scheduler) section.

### Installation Modes

#### Joint

The OpenSearch `joint` installation mode implies that each node has `master`, `data`, and `client` roles.

#### Separate

The OpenSearch `separate` installation mode implies that each node either has one of the `master`, `data`, and `client` roles or a combination of the two. For example, OpenSearch installation has 3 `master` nodes, 2 `data` nodes and 2 `client` nodes, or 3 nodes with `data` and `master` roles and 2 `client` nodes.

If `data` and `master` nodes are separated, it is important to specify persistent storages not only for `data` nodes but also for `master` nodes. The size of persistent storage for a `master` node should be small. For example, `1Gi`.

#### Deployment With Arbiter Node

OpenSearch `arbiter` nodes installed to Kubernetes nodes different from `master` provide stability of the OpenSearch cluster if `master` nodes go down.

**Important**: `arbiter` nodes do not store any indices data, but still need PV to work, the size of persistent storage for `arbiter` node should be small.

## Kubernetes

* It is required to upgrade the component before upgrading Kubernetes. Follow the information in tags regarding
  Kubernetes certified versions.
* `vm.max_map_count` should be set to `262144`. To do this, use the following command on all Kubernetes nodes, where OpenSearch is running:

   ```bash
   sysctl -w vm.max_map_count=262144
   ```

  If you deploy OpenSearch to manage Kubernetes cloud, add this command as a custom command of Kubernetes node initialization. For instance, custom user scripts for Amazon EKS are [EKS Launch Templates](https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html).

  This operation can be performed automatically during the installation if `opensearch.sysctl.enabled` is set to "true", but it requires the permission to run `privileged` containers for the cluster.

  **Attention**: Running `privileged` containers is usually denied for public clouds.

  To run `privileged` containers, refer to the [Deployment Permissions](#deployment-permissions) section.

* If you install OpenSearch service on OpenDistro Elasticsearch service, execute the steps from [Migration from OpenDistro Elasticsearch](#migration-from-opendistro-elasticsearch).
* Persistent volumes should be created for `master` nodes in the `joint` mode and for `master` and `data` nodes in the `separate` mode.

## OpenShift

* It is required to upgrade the component before upgrading OpenShift. Follow the information in tags regarding OpenShift certified versions.
* `vm.max_map_count` should be set to `262144`. To do this, use the following command on all OpenShift nodes, where OpenSearch is running:

   ```bash
   sysctl -w vm.max_map_count=262144
   ```

  This operation can be performed automatically during the installation if `opensearch.sysctl.enabled` is set to "true", but it requires the permission to run `privileged` containers for the cluster.

  To run `privileged` containers, refer to the [Deployment Permissions](#deployment-permissions) section.
* The following annotations should be specified for the project:

  ```
  oc annotate --overwrite ns ${OS_PROJECT} openshift.io/sa.scc.supplemental-groups="1000/1000"
  oc annotate --overwrite ns ${OS_PROJECT} openshift.io/sa.scc.uid-range="1000/1000"
  ```

* If you install OpenSearch on OpenDistro Elasticsearch service, execute the steps from [Migration from OpenDistro Elasticsearch](#migration-from-opendistro-elasticsearch).
* Persistent volumes should be created for `master` nodes in the `joint` mode and for `master` and `data` nodes in the `separate` mode.

## Google Cloud

1. Follow the guide at [https://www.elastic.co/guide/en/elasticsearch/reference/8.4/repository-gcs.html#repository-gcs-using-service-account](https://www.elastic.co/guide/en/elasticsearch/reference/8.4/repository-gcs.html#repository-gcs-using-service-account) to create a JSON service account file.
2. Create the secret with JSON in the `opensearch` namespace. For example:

    ```yaml
    kind: Secret
    apiVersion: v1
    metadata:
      name: opensearch-gcs-secret
      namespace: opensearch
    data:
      key: >-
        ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAidGVzdC1wcm9qZWN0IiwKICAicHJpdmF0ZV9rZXlfaWQiOiAidGVzdC1rZXkiLAogICJwcml2YXRlX2tleSI6ICItLS0tLUJFR0lOIFBSSVZBVEUgS0VZLS0tLS1cbk1JSUV2Z0lCQURBTkJnay4uLi5vSnl4ZVxuLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLVxuIiwKICAiY2xpZW50X2VtYWlsIjogInNhLW9wZW5zZWFyY2gtcnctYnVja2V0QHRlc3QtcHJvamVjdC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIsCiAgImNsaWVudF9pZCI6ICIxMTExMTExMTExMTExMTExMTExMTExMTExMTEiLAogICJhdXRoX3VyaSI6ICJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20vby9vYXV0aDIvYXV0aCIsCiAgInRva2VuX3VyaSI6ICJodHRwczovL29hdXRoMi5nb29nbGVhcGlzLmNvbS90b2tlbiIsCiAgImF1dGhfcHJvdmlkZXJfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9vYXV0aDIvdjEvY2VydHMiLAogICJjbGllbnRfeDUwOV9jZXJ0X3VybCI6ICJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9yb2JvdC92MS9tZXRhZGF0YS94NTA5L3NhLW9wZW5zZWFyY2gtcnctYnVja2V0JTQwdGVzdC1wcm9qZWN0LmlhbS5nc2VydmljZWFjY291bnQuY29tIgp9
    type: Opaque
    ```

3. Set `opensearch.snapshots.s3.gcs.secretName` and `opensearch.snapshots.s3.gcs.secretKey` parameters during the installation:

    ```yaml
    opensearch:
      snapshots:
        s3:
          gcs:
            secretName: "opensearch-gcs-secret"
            secretKey: "key"
    ```

## AWS

For more information, refer to the [Integration With Amazon OpenSearch](/docs/public/managed/amazon.md#prerequisites) section.

# Best Practices and Recommendations

## OpenSearch Configurations

### Automatic Index Creation

It is recommended to disable automatic index creation for OpenSearch and create indices with corresponding request on applications side. 
The automatic index creation may lead to unexpected index with default settings and shards which could lead to incorrect behaviour. 

To disable automatic index creation you need to specify the following deployment parameter for OpenSearch and perform upgrade operation:
```yaml
opensearch:
  config:
    action.auto_create_index: false
```

To check is automatic index creation enabled or not in runtime you can execute the following request:

```bash
GET /_cluster/settings?include_defaults=true
```
And check the property: `auto_create_index`.

To disable automatic index creation in runtime you need to execute the following request:
```bash
PUT /_cluster/settings
{
   "persistent":{
      "action.auto_create_index": false
   }
}
```
## Index Configurations

### Number of Shards

The overall goal of choosing a number of shards is to distribute an index evenly across all data nodes in the cluster. However, these shards should not be too large or too numerous. A general guideline is to try to keep shard size between:

* 10–30 GiB for workloads that prioritize low search latency
* 30–50 GiB for write-heavy workloads such as log analytics

Only performance testing different numbers of shards and different shard sizes can determine the optimal number of shards for your index.

### Index Templates

OpenSearch provides API for manage [Index Templates](https://opensearch.org/docs/latest/im-plugin/index-templates/), and it still provides API for [legacy elasticsearch templates](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-templates-v1.html).

Both API, new composable index templates (`/_index_template`) and legacy templates (`/_template`) are avaialble now and can be used by applications, but composable index templates have more features and more priority than legacy.

New OpenSearch index templates offer enhanced modularity, improved prioritization for layered configurations, and a more user-friendly API, ensuring better management and future-proofing for index configurations. The new index templates are in line with OpenSearch's evolving features and capabilities. Using them ensures compatibility with current and future versions of OpenSearch, along with support for the latest functionalities.

We strongly recommend to use only actual composable index templates (`/_index_template`) API and migrate current legacy templates. 

## HWE

The provided values do not guarantee that these values are correct for all cases. It is a general recommendation. The resources should be calculated and estimated for each project case with a test load on the SVT stand, especially the HDD size.

The Amazon guide suggests starting a configuration with 2 vCPU cores and 8 GiB of memory for every 100 GiB of storage requirement. For more information, refer to [https://docs.aws.amazon.com/opensearch-service/latest/developerguide/sizing-domains.html](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/sizing-domains.html).

### Small

Recommended for development purposes, PoC, and demos.

| Module                      | CPU   | RAM, Gi | Storage, Gb |
|-----------------------------|-------|---------|-------------|
| OpenSearch (x3)             | 1     | 4       | 50          |
| Dashboards                  | 0.1   | 0.5     | 0           |
| OpenSearch Monitoring       | 0.2   | 0.2     | 0           |
| OpenSearch DBaaS adapter    | 0.2   | 0.1     | 0           |
| Elasticsearch DBaaS Adapter | 0.2   | 0.1     | 0           |
| OpenSearch Curator          | 0.2   | 0.2     | 50          |
| OpenSearch Operator         | 0.1   | 0.2     | 0           |
| Disaster Recovery           | 0.1   | 0.1     | 0           |
| Pod Scheduler               | 0.1   | 0.1     | 0           |
| Status Provisioner          | 0.1   | 0.1     | 0           |
| TLS init job                | 0.1   | 0.1     | 0           |
| **Total (Rounded)**         | **5** | **14**  | **200**     |

<details>
<summary>Click to expand YAML</summary>

```yaml
dashboards:
  resources:
    requests:
      cpu: 100m
      memory: 512M
    limits:
      cpu: 100m
      memory: 512M
global:
  disasterRecovery:
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 100m
        memory: 128Mi
operator:
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 100m
      memory: 256Mi
opensearch:
  master:
    javaOpts: "-Xms2048m -Xmx2048m"
    resources:
      requests:
        cpu: 1
        memory: 4Gi
      limits:
        cpu: 1
        memory: 4Gi
  tlsInit:
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 100m
        memory: 128Mi
podScheduler:
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 100m
      memory: 128Mi
monitoring:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 200m
      memory: 256Mi
dbaasAdapter:
  resources:
    requests:
      cpu: 200m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 128Mi
elasticsearchDbaasAdapter:
  resources:
    requests:
      cpu: 200m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 128Mi
curator:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 200m
      memory: 256Mi
statusProvisioner:
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 100m
      memory: 128Mi
```

</details>

### Medium

Recommended for deployments with average load.

| Module                      | CPU   | RAM, Gi | Storage, Gb |
|-----------------------------|-------|---------|-------------|
| OpenSearch (x3)             | 2     | 8       | 100         |
| Dashboards                  | 0.1   | 0.5     | 0           |
| OpenSearch Monitoring       | 0.2   | 0.2     | 0           |
| OpenSearch DBaaS adapter    | 0.2   | 0.1     | 0           |
| Elasticsearch DBaaS Adapter | 0.2   | 0.1     | 0           |
| OpenSearch Curator          | 0.2   | 0.2     | 100         |
| OpenSearch Operator         | 0.1   | 0.2     | 0           |
| Disaster Recovery           | 0.1   | 0.1     | 0           |
| Pod Scheduler               | 0.1   | 0.1     | 0           |
| Status Provisioner          | 0.1   | 0.1     | 0           |
| TLS init job                | 0.1   | 0.1     | 0           |
| **Total (Rounded)**         | **8** | **26**  | **400**     |

<details>
<summary>Click to expand YAML</summary>

```yaml
dashboards:
  resources:
    requests:
      cpu: 100m
      memory: 512M
    limits:
      cpu: 100m
      memory: 512M
global:
  disasterRecovery:
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 100m
        memory: 128Mi
operator:
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 100m
      memory: 256Mi
opensearch:
  master:
    javaOpts: "-Xms4096m -Xmx4096m"
    resources:
      requests:
        cpu: 2
        memory: 8Gi
      limits:
        cpu: 2
        memory: 8Gi
  tlsInit:
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 100m
        memory: 128Mi
podScheduler:
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 100m
      memory: 128Mi
monitoring:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 200m
      memory: 256Mi
dbaasAdapter:
  resources:
    requests:
      cpu: 200m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 128Mi
elasticsearchDbaasAdapter:
  resources:
    requests:
      cpu: 200m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 128Mi
curator:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 200m
      memory: 256Mi
statusProvisioner:
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 100m
      memory: 128Mi
```

</details>

### Large

Recommended for deployments with high workload and a large amount of data.

| Module                      | CPU    | RAM, Gi | Storage, Gb |
|-----------------------------|--------|---------|-------------|
| OpenSearch (x3)             | 4      | 16      | 200         |
| Dashboards                  | 0.1    | 0.5     | 0           |
| OpenSearch Monitoring       | 0.2    | 0.2     | 0           |
| OpenSearch DBaaS adapter    | 0.2    | 0.1     | 0           |
| Elasticsearch DBaaS Adapter | 0.2    | 0.1     | 0           |
| OpenSearch Curator          | 0.2    | 0.2     | 200         |
| OpenSearch Operator         | 0.1    | 0.2     | 0           |
| Disaster Recovery           | 0.1    | 0.1     | 0           |
| Pod Scheduler               | 0.1    | 0.1     | 0           |
| Status Provisioner          | 0.1    | 0.1     | 0           |
| TLS init job                | 0.1    | 0.1     | 0           |
| **Total (Rounded)**         | **14** | **50**  | **800**     |

<details>
<summary>Click to expand YAML</summary>

```yaml
dashboards:
  resources:
    requests:
      cpu: 100m
      memory: 512M
    limits:
      cpu: 100m
      memory: 512M
global:
  disasterRecovery:
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 100m
        memory: 128Mi
operator:
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 100m
      memory: 256Mi
opensearch:
  master:
    javaOpts: "-Xms8192m -Xmx8192m"
    resources:
      requests:
        cpu: 4
        memory: 16Gi
      limits:
        cpu: 4
        memory: 16Gi
  tlsInit:
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 100m
        memory: 128Mi
podScheduler:
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 100m
      memory: 128Mi
monitoring:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 200m
      memory: 256Mi
dbaasAdapter:
  resources:
    requests:
      cpu: 200m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 128Mi
elasticsearchDbaasAdapter:
  resources:
    requests:
      cpu: 200m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 128Mi
curator:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 200m
      memory: 256Mi
statusProvisioner:
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 100m
      memory: 128Mi
```

</details>

# Parameters

This section lists the configurable parameters of the OpenSearch chart and their default values.

| Parameter          | Type   | Mandatory | Default value | Description                                                                                                                                                                                                                                                              |
|--------------------|--------|-----------|---------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `nameOverride`     | string | no        | opensearch    | The name for all OpenSearch resources (Services, StatefulSets, and so on) if the `fullnameOverride` parameter is not specified.                                                                                                                                          |
| `fullnameOverride` | string | no        | opensearch    | The base name for all OpenSearch resources (Services, StatefulSets, and so on). **Important**: If you modify this parameter, you always need to add the `CUSTOM_RESOURCE_NAME` parameter with the same value when deploying through `App Deployer` or `Groovy Deployer`. |

## Global

| Parameter                                    | Type    | Mandatory | Default value | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
|----------------------------------------------|---------|-----------|---------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `global.clusterName`                         | string  | no        | opensearch    | The name of the OpenSearch cluster.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `global.psp.create`                          | boolean | no        | false         | Whether `PodSecurityPolicy` is to be created and used.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `global.velero.preHookBackupEnabled`         | boolean | no        | true          | Whether Velero backup pre-hook with the OpenSearch flush command is to be enabled. If the parameter is set to "true", OpenSearch initiates data flushing to store all `unflushed` data on the disk before the Velero backup procedure. For more information about Velero backup hooks, see [https://velero.io/docs/v1.9/backup-hooks/](https://velero.io/docs/v1.9/backup-hooks/).                                                                                                                                                                         |
| `global.customLabels`                        | object  | no        | {}            | The custom labels for all pods that are related to the OpenSearch service. These labels can be overridden by the component `customLabel` parameter.                                                                                                                                                                                                                                                                                                                                                                                                        |
| `global.securityContext`                     | object  | no        | {}            | The pod-level security attributes and common container settings for all pods that are related to the OpenSearch Service. These security contexts can be overridden by the component `securityContext` parameter.                                                                                                                                                                                                                                                                                                                                           |
| `global.tls.enabled`                         | boolean | no        | false         | Whether TLS is to be enabled for all OpenSearch services. For more information about TLS, refer to the [TLS Encryption](/docs/public/tls.md) section.                                                                                                                                                                                                                                                                                                                                                                                                      |
| `global.tls.cipherSuites`                    | list    | no        | []            | The list of cipher suites that are used to negotiate the security settings for a network connection using a TLS or SSL network protocol. By default, all the available cipher suites are supported.                                                                                                                                                                                                                                                                                                                                                        |
| `global.tls.generateCerts.enabled`           | boolean | no        | true          | Whether TLS certificates are to be generated. This parameter is taken into account only if the `global.tls.enabled` parameter is set to "true".                                                                                                                                                                                                                                                                                                                                                                                                            |
| `global.tls.generateCerts.certProvider`      | string  | no        | cert-manager  | The provider used for TLS certificates' generation. The possible values are "cert-manager" and "dev".                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `global.tls.generateCerts.durationDays`      | integer | no        | 365           | The TLS certificates' validity period in days.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `global.tls.generateCerts.clusterIssuerName` | string  | no        | ""            | The name of the `ClusterIssuer` resource. If the parameter is not set or empty, the `Issuer` resource is created in the current Kubernetes namespace. It is used when the `global.tls.generateCerts.certProvider` parameter is set to "cert-manager".                                                                                                                                                                                                                                                                                                      |
| `global.tls.renewCerts`                      | boolean | no        | true          | Whether to renew development certificates if they expire in less than 10 years.                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `global.externalOpensearch.enabled`          | boolean | no        | false         | Whether external OpenSearch is to be used.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `global.externalOpensearch.url`              | string  | no        | ""            | The URL (with protocol) of external OpenSearch. For example, `https://external-opensearch.eks.amazon.com`.                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `global.externalOpensearch.username`         | string  | no        | ""            | The username of the external OpenSearch user to connect. The user must have full permissions to the cluster and manage roles and role mappings.                                                                                                                                                                                                                                                                                                                                                                                                            |
| `global.externalOpensearch.password`         | string  | no        | ""            | The password of the external OpenSearch user to connect. The user must have full permissions to the cluster and manage roles and role mappings.                                                                                                                                                                                                                                                                                                                                                                                                            |
| `global.externalOpensearch.nodesCount`       | integer | no        | 3             | The total number of external OpenSearch nodes (data and master).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `global.externalOpensearch.dataNodesCount`   | integer | no        | 3             | The number of external OpenSearch data nodes. If master and data nodes are the same, the same value should be used.                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `global.externalOpensearch.tlsSecretName`    | string  | no        | ""            | The secret which contains REST TLS certificates. If you set an ingress url in `global.externalOpensearch.url`, then you need to create the secret with an ingress certificate. **Important**: the specified secret should exist before deployment. If the secret key names differ from the default of the `opensearch.tls.rest.existingCertSecretCertSubPath`, `opensearch.tls.rest.existingCertSecretKeySubPath`, `opensearch.tls.rest.existingCertSecretRootCASubPath` parameters, then it's also necessary to specify actual value for that parameters. |

### Disaster Recovery

| Parameter                                                                  | Type    | Mandatory | Default value            | Description                                                                                                                                                                                                                                              |
|----------------------------------------------------------------------------|---------|-----------|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `global.disasterRecovery.image`                                            | string  | no        | Calculates automatically | The image of the OpenSearch Disaster Recovery container.                                                                                                                                                                                                 |
| `global.disasterRecovery.tls.enabled`                                      | boolean | no        | true                     | Whether TLS is to be enabled for Disaster Recovery Daemon. This parameter is taken into account only if the `global.tls.enabled` parameter is set to "true". For more information about TLS, refer to the [TLS Encryption](/docs/public/tls.md) section. |
| `global.disasterRecovery.tls.secretName`                                   | string  | no        | ""                       | The secret that contains TLS certificates. It is required if TLS for Disaster Recovery Daemon is enabled and certificates' generation is disabled.                                                                                                       |
| `global.disasterRecovery.tls.cipherSuites`                                 | list    | no        | []                       | The list of cipher suites that are used to negotiate the security settings for a network connection using a TLS or SSL network protocol. If this parameter is not specified, cipher suites are taken from the `global.tls.cipherSuites` parameter.       |
| `global.disasterRecovery.tls.subjectAlternativeName.additionalDnsNames`    | list    | no        | []                       | The list of additional DNS names to be added to the `Subject Alternative Name` field of a TLS certificate.                                                                                                                                               |
| `global.disasterRecovery.tls.subjectAlternativeName.additionalIpAddresses` | list    | no        | []                       | The list of additional IP addresses to be added to the `Subject Alternative Name` field of a TLS certificate.                                                                                                                                            |
| `global.disasterRecovery.httpAuth.enabled`                                 | boolean | no        | false                    | Whether site manager authentication is to be enabled.                                                                                                                                                                                                    |
| `global.disasterRecovery.httpAuth.smNamespace`                             | string  | no        | site-manager             | The name of the Kubernetes namespace where the site manager is located.                                                                                                                                                                                  |
| `global.disasterRecovery.httpAuth.smServiceAccountName`                    | string  | no        | sm-auth-sa               | The name of the Kubernetes service account where the site manager is used.                                                                                                                                                                               |
| `global.disasterRecovery.httpAuth.restrictedEnvironment`                   | boolean | no        | false                    | Whether the `system:auth-delegator` cluster role is to be bound to the OpenSearch operator service account.                                                                                                                                              |
| `global.disasterRecovery.mode`                                             | string  | no        | ""                       | The mode of OpenSearch Disaster Recovery installation. If you do not specify this parameter, the service is deployed in the regular mode, not the Disaster Recovery mode. The possible values are "active", "standby", and "disable".                    |
| `global.disasterRecovery.indicesPattern`                                   | string  | no        | *                        | The regular expression used to find OpenSearch indices for cross cluster replication.                                                                                                                                                                    |
| `global.disasterRecovery.remoteCluster`                                    | string  | no        | ""                       | The URL of the `active` OpenSearch service. For example, `opensearch.opensearch-service.svc.cluster-2.local:9300`.                                                                                                                                       |
| `global.disasterRecovery.siteManagerEnabled`                               | boolean | no        | true                     | Whether creation of a Kubernetes Custom Resource for `SiteManager` is to be enabled. This property is used for inner developers' purposes.                                                                                                               |
| `global.disasterRecovery.timeout`                                          | integer | no        | 600                      | The timeout for a switchover.                                                                                                                                                                                                                            |
| `global.disasterRecovery.afterServices`                                    | list    | no        | []                       | The list of `SiteManager` names for services after which the OpenSearch service switchover is to be run.                                                                                                                                                 |
| `global.disasterRecovery.replicationWatcherEnabled`                        | boolean | no        | false                    | Whether the Replication Watcher feature is to be enabled. It periodically checks that replication on the `standby` side is running correctly and restarts the replication if something goes wrong.                                                       |
| `global.disasterRecovery.replicationWatcherIntervalSeconds`                | integer | no        | 30                       | The interval in seconds to check the replication status by Replication Watcher.                                                                                                                                                                          |
| `global.disasterRecovery.serviceExport.enabled`                            | boolean | no        | false                    | Whether the `net.gke.io/v1 ServiceExport` resource is to be created. It should be set to "true" only on the GKE cluster with configured MCS. If it is enabled, the `global.disasterRecovery.serviceExport.region` parameter should also be specified.    |
| `global.disasterRecovery.serviceExport.region`                             | string  | no        | ""                       | The region of the cloud where the current instance of OpenSearch service is installed. For example, `us-central`. It should be specified if `global.disasterRecovery.serviceExport.enabled` is set to "true".                                            |
| `global.disasterRecovery.resources.requests.cpu`                           | string  | no        | 25m                      | The minimum number of CPUs the disaster recovery daemon container should use.                                                                                                                                                                            |
| `global.disasterRecovery.resources.requests.memory`                        | string  | no        | 32Mi                     | The minimum amount of memory the disaster recovery daemon container should use.                                                                                                                                                                          |
| `global.disasterRecovery.resources.limits.cpu`                             | string  | no        | 100m                     | The maximum number of CPUs the disaster recovery daemon container should use.                                                                                                                                                                            |
| `global.disasterRecovery.resources.limits.memory`                          | string  | no        | 128Mi                    | The maximum amount of memory the disaster recovery daemon container should use.                                                                                                                                                                          |

## Dashboards

| Parameter                                            | Type    | Mandatory | Default value                                                                | Description                                                                                                                                                                                                                                                                                                        |
|------------------------------------------------------|---------|-----------|------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `dashboards.enabled`                                 | boolean | no        | false                                                                        | Whether the installation of Dashboards is to be enabled.                                                                                                                                                                                                                                                           |
| `dashboards.dockerImage`                             | string  | no        | Calculates automatically                                                     | The docker image of dashboards.                                                                                                                                                                                                                                                                                    |
| `dashboards.imagePullPolicy`                         | string  | no        | Always                                                                       | The image pull policy for the dashboards' container.                                                                                                                                                                                                                                                               |
| `dashboards.imagePullSecrets`                        | list    | no        | []                                                                           | The list of references to secrets in the same namespace to use for pulling any of the images used by the dashboards' container.                                                                                                                                                                                    |
| `dashboards.updateStrategy`                          | string  | no        | Recreate                                                                     | The strategy used to replace old pods by new ones. The possible values are "Recreate" and "RollingUpdate".                                                                                                                                                                                                         |
| `dashboards.replicas`                                | integer | no        | 1                                                                            | The number of dashboards' instances.                                                                                                                                                                                                                                                                               |
| `dashboards.resources.requests.cpu`                  | string  | no        | 100m                                                                         | The minimum number of CPUs the dashboards' container should use.                                                                                                                                                                                                                                                   |
| `dashboards.resources.requests.memory`               | string  | no        | 512M                                                                         | The minimum amount of memory the dashboards' container should use.                                                                                                                                                                                                                                                 |
| `dashboards.resources.limits.cpu`                    | string  | no        | 100m                                                                         | The maximum number of CPUs the dashboards' container should use.                                                                                                                                                                                                                                                   |
| `dashboards.resources.limits.memory`                 | string  | no        | 512M                                                                         | The maximum amount of memory the dashboards' container should use.                                                                                                                                                                                                                                                 |
| `dashboards.opensearchHosts`                         | string  | no        | `<protocol>://<name>-internal:9200`                                          | The OpenSearch hosts for dashboards to connect.                                                                                                                                                                                                                                                                    |
| `dashboards.secretMounts`                            | list    | no        | []                                                                           | The list of secrets and their paths to mount inside the dashboards' pod.                                                                                                                                                                                                                                           |
| `dashboards.extraEnvs`                               | list    | no        | []                                                                           | The list of extra environment variables to add inside the dashboards' pod.                                                                                                                                                                                                                                         |
| `dashboards.envFrom`                                 | list    | no        | []                                                                           | The list of sources to populate environment variables in the dashboards' container.                                                                                                                                                                                                                                |
| `dashboards.extraVolumes`                            | list    | no        | []                                                                           | The list of extra volumes to add inside the dashboards' pod.                                                                                                                                                                                                                                                       |
| `dashboards.extraVolumeMounts`                       | list    | no        | []                                                                           | The list of extra volume mounts to add inside the dashboards' pod.                                                                                                                                                                                                                                                 |
| `dashboards.extraInitContainers`                     | list    | no        | []                                                                           | The list of extra init containers to add inside the dashboards' pod.                                                                                                                                                                                                                                               |
| `dashboards.extraContainers`                         | list    | no        | []                                                                           | The list of extra containers to add inside the dashboards' pod.                                                                                                                                                                                                                                                    |
| `dashboards.ingress.enabled`                         | boolean | no        | false                                                                        | Whether the dashboards' ingress is to be enabled.                                                                                                                                                                                                                                                                  |
| `dashboards.ingress.annotations`                     | object  | no        | {}                                                                           | The annotations for the dashboards' ingress.                                                                                                                                                                                                                                                                       |
| `dashboards.ingress.className`                       | string  | no        | ""                                                                           | The class name of the dashboards' ingress.                                                                                                                                                                                                                                                                         |
| `dashboards.ingress.hosts`                           | list    | no        | [{"host": "chart-example.local", "paths": [{"path": "/"}]}]                  | The list of host names for the dashboards' ingress.                                                                                                                                                                                                                                                                |
| `dashboards.ingress.tls`                             | list    | no        | []                                                                           | The list of TLS configurations for the dashboards' ingress.                                                                                                                                                                                                                                                        |
| `dashboards.service.type`                            | string  | no        | ClusterIP                                                                    | The type of the dashboards' service.                                                                                                                                                                                                                                                                               |
| `dashboards.service.port`                            | string  | no        | 5601                                                                         | The port that is used for the dashboards' service.                                                                                                                                                                                                                                                                 |
| `dashboards.service.loadBalancerIP`                  | string  | no        | ""                                                                           | The load balancer IP that is used for the dashboards' service.                                                                                                                                                                                                                                                     |
| `dashboards.service.nodePort`                        | string  | no        | ""                                                                           | The node port that is used for the dashboards' service.                                                                                                                                                                                                                                                            |
| `dashboards.service.labels`                          | object  | no        | {}                                                                           | The labels that are to be specified on the dashboards' service.                                                                                                                                                                                                                                                    |
| `dashboards.service.annotations`                     | object  | no        | {}                                                                           | The annotations that are to be specified on the dashboards' service.                                                                                                                                                                                                                                               |
| `dashboards.service.loadBalancerSourceRanges`        | list    | no        | []                                                                           | The client IPs for which traffic is not restricted through cloud-provider load-balancer.                                                                                                                                                                                                                           |
| `dashboards.service.httpPortName`                    | string  | no        | http                                                                         | The name of the HTTP port for the dashboards' service.                                                                                                                                                                                                                                                             |
| `dashboards.config`                                  | object  | no        | {}                                                                           | The configuration of dashboards (`dashboards.yml`).                                                                                                                                                                                                                                                                |
| `dashboards.nodeSelector`                            | object  | no        | {}                                                                           | The selector that defines the nodes where dashboards' pods are scheduled on.                                                                                                                                                                                                                                       |
| `dashboards.tolerations`                             | list    | no        | []                                                                           | The list of toleration policies for the dashboards' pod in the `JSON` format.                                                                                                                                                                                                                                      |
| `dashboards.affinity`                                | object  | no        | {}                                                                           | The affinity scheduling rules in the `JSON` format.                                                                                                                                                                                                                                                                |
| `dashboards.priorityClassName`                       | string  | no        | ""                                                                           | The priority class to be used by the dashboards' pod. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/). |
| `dashboards.opensearchAccount.secret`                | string  | no        | ""                                                                           | The name of the secret with the dashboards' server user as configured in `dashboards.yml`.                                                                                                                                                                                                                         |
| `dashboards.opensearchAccount.keyPassphrase.enabled` | boolean | no        | false                                                                        | Whether mounting in key passphrase for `opensearchAccount` is to be enabled.                                                                                                                                                                                                                                       |
| `dashboards.labels`                                  | object  | no        | {}                                                                           | The labels that are to be specified on dashboards' pod.                                                                                                                                                                                                                                                            |
| `dashboards.hostAliases`                             | list    | no        | []                                                                           | The list of hosts and IPs that are injected into the dashboards' pod's hosts file.                                                                                                                                                                                                                                 |
| `dashboards.serverHost`                              | string  | no        | 0.0.0.0                                                                      | The host of the dashboards' server.                                                                                                                                                                                                                                                                                |
| `dashboards.serviceAccount.create`                   | boolean | no        | true                                                                         | Whether the default service account for dashboards is to be created.                                                                                                                                                                                                                                               |
| `dashboards.serviceAccount.name`                     | string  | no        | ""                                                                           | The name for the dashboards' service account. If it is empty and the `dashboards.serviceAccount.create` parameter is set to "true", a name is generated using the fullname template.                                                                                                                               |
| `dashboards.podAnnotations`                          | object  | no        | {}                                                                           | The annotations for the dashboards' pod.                                                                                                                                                                                                                                                                           |
| `dashboards.podSecurityContext`                      | object  | no        | {}                                                                           | The pod-level security attributes and common container settings for the dashboards' pod.                                                                                                                                                                                                                           |
| `dashboards.securityContext`                         | object  | no        | {"capabilities": {"drop": ["ALL"]}, "runAsNonRoot": true, "runAsUser": 1000} | The container-level security attributes and common container settings for the dashboards' container.                                                                                                                                                                                                               |

Where:

* `<protocol>` is the OpenSearch protocol.
* `<name>` is the value of the `fullnameOverride` parameter.

## Operator

| Parameter                            | Type    | Mandatory | Default value            | Description                                                                                                                                                                                                                                                                                                     |
|--------------------------------------|---------|-----------|--------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `operator.dockerImage`               | string  | no        | Calculates automatically | The docker image of OpenSearch Service Operator.                                                                                                                                                                                                                                                                |
| `operator.replicas`                  | integer | no        | 1                        | The number of OpenSearch Service Operator pods.                                                                                                                                                                                                                                                                 |
| `operator.reconcilePeriod`           | integer | no        | 60                       | The maximum delay in seconds before the next reconciliation call.                                                                                                                                                                                                                                               |
| `operator.tolerations`               | list    | no        | []                       | The list of toleration policies for OpenSearch Service Operator pods.                                                                                                                                                                                                                                           |
| `operator.affinity`                  | object  | no        | {}                       | The affinity scheduling rules in the `JSON` format.                                                                                                                                                                                                                                                             |
| `operator.customLabels`              | object  | no        | {}                       | The custom labels for the OpenSearch Service Operator pod.                                                                                                                                                                                                                                                      |
| `operator.securityContext`           | object  | no        | {}                       | The pod-level security attributes and common container settings for the OpenSearch Service operator pod.                                                                                                                                                                                                        |
| `operator.resources.requests.cpu`    | string  | no        | 25m                      | The minimum number of CPUs the operator container should use.                                                                                                                                                                                                                                                   |
| `operator.resources.requests.memory` | string  | no        | 128Mi                    | The minimum amount of memory the operator container should use.                                                                                                                                                                                                                                                 |
| `operator.resources.limits.cpu`      | string  | no        | 100m                     | The maximum number of CPUs the operator container should use.                                                                                                                                                                                                                                                   |
| `operator.resources.limits.memory`   | string  | no        | 128Mi                    | The maximum amount of memory the operator container should use.                                                                                                                                                                                                                                                 |
| `operator.priorityClassName`         | string  | no        | ""                       | The priority class to be used by the operator pod. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/). |

## OpenSearch

| Parameter                                                     | Type    | Mandatory | Default value                                                     | Description                                                                                                                                                                                                                                                                                                            |
|---------------------------------------------------------------|---------|-----------|-------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opensearch.discoveryOverride`                                | string  | no        | ""                                                                | The possibility to override the `discovery.seed_hosts` environment variable that allows a second aliased deployment to find the cluster.                                                                                                                                                                               |
| `opensearch.compatibilityModeEnabled`                         | boolean | no        | true                                                              | Whether the compatibility mode is to be enabled.                                                                                                                                                                                                                                                                       |
| `opensearch.gcLoggingEnabled`                                 | boolean | no        | false                                                             | Whether garbage collection logging is to be enabled for OpenSearch.                                                                                                                                                                                                                                                    |
| `opensearch.performanceAnalyzerEnabled`                       | boolean | no        | true                                                              | Whether the OpenSearch Performance Analyzer plugin is to be running.                                                                                                                                                                                                                                                   |
| `opensearch.rollingUpdate`                                    | boolean | no        | false                                                             | Whether operator performs rolling update on its own in accordance with [guide](#operator-rolling-upgrade-feature). Otherwise Kubernetes performs rolling upgrade in accordance with default StatefulSet policy.                                                                                                        |
| `opensearch.readinessTimeout`                                 | string  | no        | 800s                                                              | The timeout for OpenSearch readiness check in operator. The value is a sequence of decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".                                                                 |
| `opensearch.securityConfig.enabled`                           | boolean | no        | true                                                              | Whether custom [security configs](https://opensearch.org/docs/latest/security/configuration/index/) are to be used.                                                                                                                                                                                                    |
| `opensearch.securityConfig.path`                              | string  | no        | /usr/share/opensearch/config/opensearch-security                  | The path to the files of security configuration.                                                                                                                                                                                                                                                                       |
| `opensearch.securityConfig.authc.basic.username`              | string  | yes       | ""                                                                | The username of the OpenSearch user with `admin` privileges.                                                                                                                                                                                                                                                           |
| `opensearch.securityConfig.authc.basic.password`              | string  | yes       | ""                                                                | The password of the OpenSearch user with `admin` privileges. The value should be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one digit, and one special character.                                                                                                     |
| `opensearch.securityConfig.authc.oidc.openid_connect_url`     | string  | no        | ""                                                                | The URL of OpenID Connect Identity Provider's well-known endpoint. If it is specified, the OpenID Connect authentication is enabled. For example, `"http://idp:8080/.well-known/openid-configuration"`.                                                                                                                |
| `opensearch.securityConfig.authc.oidc.subject_key`            | string  | no        | preferred_username                                                | The key in the JSON payload that stores the user's name. If it is not defined, the subject registered claim is used.                                                                                                                                                                                                   |
| `opensearch.securityConfig.authc.oidc.roles_key`              | string  | no        | preferred_username                                                | The key in the JSON payload that stores the user's roles. The value of this key must be a comma-separated list of roles. It is required only if you want to use roles in the JWT.                                                                                                                                      |
| `opensearch.securityConfig.actionGroupsSecret`                | string  | no        | ""                                                                | The name of the secret with [action_groups.yml](https://opensearch.org/docs/latest/security/configuration/yaml/#action_groupsyml) configuration.                                                                                                                                                                       |
| `opensearch.securityConfig.configSecret`                      | string  | no        | ""                                                                | The name of the secret with [config.yml](https://opensearch.org/docs/latest/security/configuration/configuration/) configuration.                                                                                                                                                                                      |
| `opensearch.securityConfig.internalUsersSecret`               | string  | no        | ""                                                                | The name of the secret with [internal_users.yml](https://opensearch.org/docs/latest/security/configuration/yaml/#internal_usersyml) configuration.                                                                                                                                                                     |
| `opensearch.securityConfig.rolesSecret`                       | string  | no        | ""                                                                | The name of the secret with [roles.yml](https://opensearch.org/docs/latest/security/configuration/yaml/#rolesyml) configuration.                                                                                                                                                                                       |
| `opensearch.securityConfig.rolesMappingSecret`                | string  | no        | ""                                                                | The name of the secret with [roles_mapping.yml](https://opensearch.org/docs/latest/security/configuration/yaml/#roles_mappingyml) configuration.                                                                                                                                                                       |
| `opensearch.securityConfig.tenantsSecret`                     | string  | no        | ""                                                                | The name of the secret with [tenants.yml](https://opensearch.org/docs/latest/security/configuration/yaml/#tenantsyml) configuration.                                                                                                                                                                                   |
| `opensearch.securityConfig.config.securityConfigSecret`       | string  | no        | ""                                                                | The name of the secret for security configuration.                                                                                                                                                                                                                                                                     |
| `opensearch.securityConfig.config.data`                       | object  | no        | {}                                                                | The security configuration for OpenSearch in the JSON format.                                                                                                                                                                                                                                                          |
| `opensearch.securityContextCustom`                            | object  | no        | {}                                                                | The pod-level security attributes and common container settings for OpenSearch pods.                                                                                                                                                                                                                                   |
| `opensearch.extraEnvs`                                        | list    | no        | []                                                                | The list of extra environment variables to be passed to OpenSearch pods.                                                                                                                                                                                                                                               |
| `opensearch.extraInitContainers`                              | list    | no        | []                                                                | The list of extra init containers to be passed to OpenSearch pods.                                                                                                                                                                                                                                                     |
| `opensearch.extraVolumes`                                     | list    | no        | []                                                                | The list of extra volumes to be passed to OpenSearch pods.                                                                                                                                                                                                                                                             |
| `opensearch.extraVolumeMounts`                                | list    | no        | []                                                                | The list of extra volume mounts to be passed to OpenSearch pods.                                                                                                                                                                                                                                                       |
| `opensearch.initContainer.dockerImage`                        | string  | no        | Calculates automatically                                          | The docker image of OpenSearch init containers.                                                                                                                                                                                                                                                                        |
| `opensearch.sysctl.enabled`                                   | boolean | no        | false                                                             | Whether the `initContainer` parameter to set `sysctl` to `vm.max_map_count` is to be enabled. This operation requires permissions to run `privileged` containers. The information about such permissions you can find in [Deployment Permissions](#deployment-permissions) section.                                    |
| `opensearch.fixmount.enabled`                                 | boolean | no        | false                                                             | Whether `initContainer` to fix mount permissions is to be enabled. It is not required if you set `fsGroup` via `securityContext`. This operation requires permissions to run `privileged` actions. For information about such permissions, refer to the [Deployment Permissions](#deployment-permissions) section.     |
| `opensearch.fixmount.securityContext`                         | object  | no        | {}                                                                | The pod-level security attributes and common container settings for `fixmount` init container in OpenSearch pods.                                                                                                                                                                                                      |
| `opensearch.tls.enabled`                                      | boolean | no        | true                                                              | Whether TLS is to be enabled for REST layer of OpenSearch. It is recommended to keep this parameter to `true`, because OpenSearch does not support some types of security configurations on REST layer without encryption. For more information about TLS, refer to the [TLS Encryption](/docs/public/tls.md) section. |
| `opensearch.tls.cipherSuites`                                 | list    | no        | []                                                                | The list of cipher suites that are used to negotiate the security settings for a network connection using a TLS or SSL network protocol. If this parameter is not specified, cipher suites are taken from the `global.tls.cipherSuites` parameter.                                                                     |
| `opensearch.tls.generateCerts.enabled`                        | boolean | no        | true                                                              | Whether OpenSearch certificates are to be generated. This parameter is taken into account only if the `global.tls.generateCerts.enabled` parameter is set to "true".                                                                                                                                                   |
| `opensearch.tls.subjectAlternativeName.additionalDnsNames`    | list    | no        | []                                                                | The list of additional DNS names to be added to the `Subject Alternative Name` field of the REST TLS certificate for OpenSearch.                                                                                                                                                                                       |
| `opensearch.tls.subjectAlternativeName.additionalIpAddresses` | list    | no        | []                                                                | The list of additional IP addresses to be added to the `Subject Alternative Name` field of the REST TLS certificate for OpenSearch.                                                                                                                                                                                    |
| `opensearch.tls.transport.existingCertSecret`                 | string  | no        | ""                                                                | The name of the secret that contains the transport certificates. If the value is not specified, the secret with transport certificates named `<fullname>-transport-certs` is created, where `<fullname>` is the OpenSearch full name.                                                                                  |
| `opensearch.tls.transport.existingCertSecretCertSubPath`      | string  | no        | tls.crt                                                           | The sub-path of `certificate` in the OpenSearch transport certificates' secret. This parameter is not taken into account if the `opensearch.tls.transport.existingCertSecret` parameter is empty.                                                                                                                      |
| `opensearch.tls.transport.existingCertSecretKeySubPath`       | string  | no        | tls.key                                                           | The sub-path of `key` in the OpenSearch transport certificates' secret. This parameter is not taken into account if the `opensearch.tls.transport.existingCertSecret` parameter is empty.                                                                                                                              |
| `opensearch.tls.transport.existingCertSecretRootCASubPath`    | string  | no        | ca.crt                                                            | The sub-path of `root ca` in the OpenSearch transport certificates' secret. This parameter is not taken into account if the `opensearch.tls.transport.existingCertSecret` parameter is empty.                                                                                                                          |
| `opensearch.tls.rest.existingCertSecret`                      | string  | no        | ""                                                                | The name of the secret that contains the OpenSearch REST certificates. If the value is not specified, the secret with admin certificates named `<fullname>-rest-certs` is created, where `<fullname>` is the OpenSearch full name.                                                                                     |
| `opensearch.tls.rest.existingCertSecretCertSubPath`           | string  | no        | tls.crt                                                           | The sub-path of `certificate` in the OpenSearch REST certificates' secret.                                                                                                                                                                                                                                             |
| `opensearch.tls.rest.existingCertSecretKeySubPath`            | string  | no        | tls.key                                                           | The sub-path of `key` in the OpenSearch REST certificates' secret.                                                                                                                                                                                                                                                     |
| `opensearch.tls.rest.existingCertSecretRootCASubPath`         | string  | no        | ca.crt                                                            | The sub-path of `root ca` in the OpenSearch REST certificates' secret.                                                                                                                                                                                                                                                 |
| `opensearch.tls.admin.existingCertSecret`                     | string  | no        | ""                                                                | The name of the secret that contains admin users' OpenSearch certificates. If the value is not specified, the secret with admin certificates named `<fullname>-admin-certs` is created, where `<fullname>` is the OpenSearch full name.                                                                                |
| `opensearch.tls.admin.existingCertSecretCertSubPath`          | string  | no        | tls.crt                                                           | The sub-path of `certificate` in the OpenSearch admin certificates' secret.                                                                                                                                                                                                                                            |
| `opensearch.tls.admin.existingCertSecretKeySubPath`           | string  | no        | tls.key                                                           | The sub-path of `key` in the OpenSearch admin certificates' secret.                                                                                                                                                                                                                                                    |
| `opensearch.tls.admin.existingCertSecretRootCASubPath`        | string  | no        | ca.crt                                                            | The sub-path of `root ca` in the OpenSearch admin certificates' secret.                                                                                                                                                                                                                                                |
| `opensearch.tlsInit.resources.requests.cpu`                   | string  | no        | 25m                                                               | The minimum number of CPUs the job for TLS initialization should use.                                                                                                                                                                                                                                                  |
| `opensearch.tlsInit.resources.requests.memory`                | string  | no        | 128Mi                                                             | The minimum amount of memory the job for TLS initialization should use.                                                                                                                                                                                                                                                |
| `opensearch.tlsInit.resources.limits.cpu`                     | string  | no        | 100m                                                              | The maximum number of CPUs the job for TLS initialization should use.                                                                                                                                                                                                                                                  |
| `opensearch.tlsInit.resources.limits.memory`                  | string  | no        | 128Mi                                                             | The maximum amount of memory the job for TLS initialization should use.                                                                                                                                                                                                                                                |
| `opensearch.config`                                           | object  | no        | See in [values.yaml](/charts/helm/opensearch-service/values.yaml) | The configuration of common properties for OpenSearch (`opensearch.yml`). For more information, see [Modifying the YAML files](https://opensearch.org/docs/latest/security/configuration/yaml/#opensearchyml).                                                                                                         |
| `opensearch.log4jConfig`                                      | object  | no        | {}                                                                | The configuration of `log4j` properties for OpenSearch (`log4j2.properties`).                                                                                                                                                                                                                                          |
| `opensearch.loggingConfig`                                    | object  | no        | See in [values.yaml](/charts/helm/opensearch-service/values.yaml) | The configuration of logging properties for OpenSearch (`logging.yml`).                                                                                                                                                                                                                                                |
| `opensearch.transportKeyPassphrase.enabled`                   | boolean | no        | false                                                             | Whether OpenSearch transport key passphrase is required.                                                                                                                                                                                                                                                               |
| `opensearch.transportKeyPassphrase.passPhrase`                | string  | no        | ""                                                                | The transport key passphrase for OpenSearch.                                                                                                                                                                                                                                                                           |
| `opensearch.sslKeyPassphrase.enabled`                         | boolean | no        | false                                                             | Whether OpenSearch SSL key passphrase is required.                                                                                                                                                                                                                                                                     |
| `opensearch.sslKeyPassphrase.passPhrase`                      | string  | no        | ""                                                                | The SSL key passphrase for OpenSearch.                                                                                                                                                                                                                                                                                 |
| `opensearch.maxMapCount`                                      | integer | no        | 262144                                                            | The value of `max_map_count` for OpenSearch.                                                                                                                                                                                                                                                                           |
| `opensearch.dockerImage`                                      | string  | no        | Calculates automatically                                          | The docker image of OpenSearch.                                                                                                                                                                                                                                                                                        |
| `opensearch.dockerTlsInitImage`                               | string  | no        | Calculates automatically                                          | The docker image of OpenSearch TLS init container.                                                                                                                                                                                                                                                                     |
| `opensearch.imagePullPolicy`                                  | string  | no        | Always                                                            | The image pull policy for OpenSearch containers. The possible values are "Always", "IfNotPresent", or "Never".                                                                                                                                                                                                         |
| `opensearch.configDirectory`                                  | string  | no        | /usr/share/opensearch/config                                      | The location of OpenSearch configuration.                                                                                                                                                                                                                                                                              |
| `opensearch.serviceAccount.create`                            | boolean | no        | true                                                              | Whether the default service account for OpenSearch is to be created and used.                                                                                                                                                                                                                                          |
| `opensearch.serviceAccount.name`                              | string  | no        | ""                                                                | The name of the OpenSearch service account.                                                                                                                                                                                                                                                                            |

### Master Nodes

| Parameter                                              | Type    | Mandatory | Default value                                                                                               | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
|--------------------------------------------------------|---------|-----------|-------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opensearch.master.enabled`                            | boolean | no        | true                                                                                                        | Whether the OpenSearch master nodes are to be enabled.                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.master.replicas`                           | integer | no        | 3                                                                                                           | The number of OpenSearch master nodes.                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.master.updateStrategy`                     | string  | no        | "RollingUpdate"                                                                                             | The strategy used to replace old pods by new ones. The possible values are `OnDelete` and `RollingUpdate`. If OpenSearch is deployed in `joint` mode with enabled `opensearch.rollingUpdate` parameter, then the default value is `OnDelete` and operator will perform rolling upgrade in accordance with [guide](#operator-rolling-upgrade-feature). Otherwise `RollingUpgrade` is used and Kubernetes performs rolling upgrade in accordance with StatefulSet default flow. |
| `opensearch.master.persistence.enabled`                | boolean | no        | true                                                                                                        | Whether persistent storage is to be enabled on OpenSearch master nodes.                                                                                                                                                                                                                                                                                                                                                                                                       |
| `opensearch.master.persistence.existingClaim`          | string  | no        | ""                                                                                                          | The name of the existing persistent volume claim for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                                                                                                                 |
| `opensearch.master.persistence.subPath`                | string  | no        | ""                                                                                                          | The subdirectory of the volume to mount to.                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.persistence.storageClass`           | string  | yes       | -                                                                                                           | The storage class name that is used for OpenSearch master nodes. If it is set to `-`, dynamic provisioning is disabled. If it is empty or set to `null`, the default provisioner is chosen.                                                                                                                                                                                                                                                                                   |
| `opensearch.master.persistence.persistentVolumes`      | list    | no        | []                                                                                                          | The list of predefined persistent volumes for OpenSearch master nodes. The number of persistent volumes should be equal to `opensearch.master.replicas` parameter. If `hostPath` PVs are used, the `nodes` parameters is also should be specified.                                                                                                                                                                                                                            |
| `opensearch.master.persistence.nodes`                  | list    | no        | []                                                                                                          | The list of Kubernetes node names to assign OpenSearch master nodes. The number of nodes should be equal to `opensearch.master.replicas` parameter. It should not be used with `storageClass` pod assignment.                                                                                                                                                                                                                                                                 |
| `opensearch.master.persistence.accessModes`            | list    | no        | ["ReadWriteOnce"]                                                                                           | The list of access modes of persistent volumes for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.persistence.size`                   | string  | no        | 5Gi                                                                                                         | The size of persistent volumes for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.persistence.annotations`            | object  | no        | {}                                                                                                          | The annotations of persistent volumes for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                                                                                                                            |
| `opensearch.master.resources.requests.cpu`             | string  | no        | 200m                                                                                                        | The minimum number of CPUs the OpenSearch master node container should use.                                                                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.resources.requests.memory`          | string  | no        | 1536Mi                                                                                                      | The minimum number of memory the OpenSearch master node container should use.                                                                                                                                                                                                                                                                                                                                                                                                 |
| `opensearch.master.resources.limits.cpu`               | string  | no        | 500m                                                                                                        | The maximum number of CPUs the OpenSearch master node container should use.                                                                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.resources.limits.memory`            | string  | no        | 1536Mi                                                                                                      | The maximum number of memory the OpenSearch master node container should use.                                                                                                                                                                                                                                                                                                                                                                                                 |
| `opensearch.master.javaOpts`                           | string  | no        | -Xms718m -Xmx718m                                                                                           | The Java options that are used for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.podDisruptionBudget.enabled`        | boolean | no        | false                                                                                                       | Whether the disruption budget for OpenSearch master nodes is to be created.                                                                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.podDisruptionBudget.minAvailable`   | integer | no        | 1                                                                                                           | The minimum number or percentage of pods that [should remain scheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                                                                                                                                                                              |
| `opensearch.master.podDisruptionBudget.maxUnavailable` | integer | no        |                                                                                                             | The maximum number or percentage of pods that [may be unscheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                                                                                                                                                                                   |
| `opensearch.master.readinessProbe`                     | object  | no        | {}                                                                                                          | The configuration of the [readinessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                         |
| `opensearch.master.livenessProbe`                      | object  | no        | {"tcpSocket": {"port": "transport"}, "initialDelaySeconds": 90, "periodSeconds": 20, "failureThreshold": 5} | The configuration of the [livenessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                          |
| `opensearch.master.startupProbe`                       | object  | no        | {}                                                                                                          | The configuration of the [startupProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) for OpenSearch master nodes.                                                                                                                                                                                                                                                                                                   |
| `opensearch.master.imagePullSecrets`                   | list    | no        | []                                                                                                          | The list of references to secrets in the same namespace to use for pulling any of the images used by OpenSearch master containers.                                                                                                                                                                                                                                                                                                                                            |
| `opensearch.master.nodeSelector`                       | object  | no        | {}                                                                                                          | The selector that defines the nodes where the OpenSearch master nodes are scheduled on.                                                                                                                                                                                                                                                                                                                                                                                       |
| `opensearch.master.tolerations`                        | list    | no        | []                                                                                                          | The list of toleration policies for OpenSearch master nodes in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                                                 |
| `opensearch.master.affinity`                           | object  | no        | <anti_affinity_rule>                                                                                        | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `opensearch.master.podAnnotations`                     | object  | no        | {}                                                                                                          | The annotations for OpenSearch master pod.                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `opensearch.master.extraInitContainers`                | list    | no        | []                                                                                                          | The list of extra init containers to add inside the OpenSearch master pod.                                                                                                                                                                                                                                                                                                                                                                                                    |
| `opensearch.master.extraContainers`                    | list    | no        | []                                                                                                          | The list of extra containers to add inside the OpenSearch master pod.                                                                                                                                                                                                                                                                                                                                                                                                         |
| `opensearch.master.customLabels`                       | object  | no        | {}                                                                                                          | The custom labels for the OpenSearch master pods.                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `opensearch.master.priorityClassName`                  | string  | no        | ""                                                                                                          | The priority class to be used by the OpenSearch master nodes. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                                                                                                                                                    |

Where:
* `<anti_affinity_rule>` is as follows:

  ```
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - topologyKey: "kubernetes.io/hostname"
        labelSelector:
          matchLabels:
            role: master
  ```

### Arbiter Nodes

| Parameter                                               | Type    | Mandatory | Default value                                                                                               | Description                                                                                                                                                                                                                                                                                                                 |
|---------------------------------------------------------|---------|-----------|-------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opensearch.arbiter.enabled`                            | boolean | no        | false                                                                                                       | Whether the OpenSearch arbiter nodes are to be enabled. OpenSearch `arbiter` nodes installed to different Kubernetes nodes than `master` provide cluster stability if `master` nodes goes down. **Important**: `arbiter` pods do not store any indices data.                                                                |
| `opensearch.arbiter.replicas`                           | integer | no        | 1                                                                                                           | The number of OpenSearch arbiter nodes.                                                                                                                                                                                                                                                                                     |
| `opensearch.arbiter.updateStrategy`                     | string  | no        | RollingUpdate                                                                                               | The strategy used to replace old pods by new ones. The possible values are "OnDelete" and "RollingUpdate".                                                                                                                                                                                                                  |
| `opensearch.arbiter.persistence.enabled`                | boolean | no        | true                                                                                                        | Whether persistent storage is to be enabled on OpenSearch arbiter nodes.                                                                                                                                                                                                                                                    |
| `opensearch.arbiter.persistence.existingClaim`          | string  | no        | ""                                                                                                          | The name of the existing persistent volume claim for OpenSearch arbiter nodes.                                                                                                                                                                                                                                              |
| `opensearch.arbiter.persistence.subPath`                | string  | no        | ""                                                                                                          | The subdirectory of the volume to mount to.                                                                                                                                                                                                                                                                                 |
| `opensearch.arbiter.persistence.storageClass`           | string  | no        | -                                                                                                           | The storage class name that is used for OpenSearch arbiter nodes. If it is set to `-`, dynamic provisioning is disabled. If it is empty or set to `null`, the default provisioner is chosen.                                                                                                                                |
| `opensearch.arbiter.persistence.persistentVolumes`      | list    | no        | []                                                                                                          | The list of predefined persistent volumes for OpenSearch arbiter nodes. The number of persistent volumes should be equal to `opensearch.arbiter.replicas` parameter. If `hostPath` PVs are used, the `nodes` parameters is also should be specified.                                                                        |
| `opensearch.arbiter.persistence.nodes`                  | list    | no        | []                                                                                                          | The list of Kubernetes node names to assign OpenSearch arbiter nodes. The number of nodes should be equal to `opensearch.arbiter.replicas` parameter. It should not be used with `storageClass` pod assignment.                                                                                                             |
| `opensearch.arbiter.persistence.accessModes`            | list    | no        | ["ReadWriteOnce"]                                                                                           | The list of access modes of persistent volumes for OpenSearch arbiter nodes.                                                                                                                                                                                                                                                |
| `opensearch.arbiter.persistence.size`                   | string  | no        | 5Gi                                                                                                         | The size of persistent volumes for OpenSearch arbiter nodes.                                                                                                                                                                                                                                                                |
| `opensearch.arbiter.persistence.annotations`            | object  | no        | {}                                                                                                          | The annotations of persistent volumes for OpenSearch arbiter nodes.                                                                                                                                                                                                                                                         |
| `opensearch.arbiter.resources.requests.cpu`             | string  | no        | 200m                                                                                                        | The minimum number of CPUs the OpenSearch arbiter node container should use.                                                                                                                                                                                                                                                |
| `opensearch.arbiter.resources.requests.memory`          | string  | no        | 1536Mi                                                                                                      | The minimum number of memory the OpenSearch arbiter node container should use.                                                                                                                                                                                                                                              |
| `opensearch.arbiter.resources.limits.cpu`               | string  | no        | 500m                                                                                                        | The maximum number of CPUs the OpenSearch arbiter node container should use.                                                                                                                                                                                                                                                |
| `opensearch.arbiter.resources.limits.memory`            | string  | no        | 1536Mi                                                                                                      | The maximum number of memory the OpenSearch arbiter node container should use.                                                                                                                                                                                                                                              |
| `opensearch.arbiter.javaOpts`                           | string  | no        | -Xms718m -Xmx718m                                                                                           | The Java options that are used for OpenSearch arbiter nodes.                                                                                                                                                                                                                                                                |
| `opensearch.arbiter.podDisruptionBudget.enabled`        | boolean | no        | false                                                                                                       | Whether the disruption budget for OpenSearch arbiter nodes is to be created.                                                                                                                                                                                                                                                |
| `opensearch.arbiter.podDisruptionBudget.minAvailable`   | integer | no        | 1                                                                                                           | The minimum number or percentage of pods that [should remain scheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                            |
| `opensearch.arbiter.podDisruptionBudget.maxUnavailable` | integer | no        |                                                                                                             | The maximum number or percentage of pods that [may be unscheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                                 |
| `opensearch.arbiter.readinessProbe`                     | object  | no        | {}                                                                                                          | The configuration of the [readinessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch arbiter nodes.                                                                                                                                                      |
| `opensearch.arbiter.livenessProbe`                      | object  | no        | {"tcpSocket": {"port": "transport"}, "initialDelaySeconds": 90, "periodSeconds": 20, "failureThreshold": 5} | The configuration of the [livenessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch arbiter nodes.                                                                                                                                                       |
| `opensearch.arbiter.startupProbe`                       | object  | no        | {}                                                                                                          | The configuration of the [startupProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) for OpenSearch arbiter nodes.                                                                                                                                                |
| `opensearch.arbiter.imagePullSecrets`                   | list    | no        | []                                                                                                          | The list of references to secrets in the same namespace to use for pulling any of the images used by OpenSearch arbiter containers.                                                                                                                                                                                         |
| `opensearch.arbiter.nodeSelector`                       | object  | no        | {}                                                                                                          | The selector that defines the nodes where the OpenSearch arbiter nodes are scheduled on.                                                                                                                                                                                                                                    |
| `opensearch.arbiter.tolerations`                        | list    | no        | []                                                                                                          | The list of toleration policies for OpenSearch arbiter nodes in `JSON` format.                                                                                                                                                                                                                                              |
| `opensearch.arbiter.affinity`                           | object  | no        | <anti_affinity_rule>                                                                                        | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                             |
| `opensearch.arbiter.podAnnotations`                     | object  | no        | {}                                                                                                          | The annotations for OpenSearch arbiter pod.                                                                                                                                                                                                                                                                                 |
| `opensearch.arbiter.extraInitContainers`                | list    | no        | []                                                                                                          | The list of extra init containers to add inside the OpenSearch arbiter pod.                                                                                                                                                                                                                                                 |
| `opensearch.arbiter.extraContainers`                    | list    | no        | []                                                                                                          | The list of extra containers to add inside the OpenSearch arbiter pod.                                                                                                                                                                                                                                                      |
| `opensearch.arbiter.customLabels`                       | object  | no        | {}                                                                                                          | The custom labels for the OpenSearch arbiter pods.                                                                                                                                                                                                                                                                          |
| `opensearch.arbiter.priorityClassName`                  | string  | no        | ""                                                                                                          | The priority class to be used by the OpenSearch arbiter nodes. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/). |

Where:
* `<anti_affinity_rule>` is as follows:

  ```
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - topologyKey: "kubernetes.io/hostname"
        labelSelector:
          matchLabels:
            role: master
  ```

### Data Nodes

| Parameter                                            | Type    | Mandatory | Default value                                                                                               | Description                                                                                                                                                                                                                                                                                                                                                                                                        |
|------------------------------------------------------|---------|-----------|-------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opensearch.data.enabled`                            | boolean | no        | true                                                                                                        | Whether the OpenSearch data nodes are to be enabled.                                                                                                                                                                                                                                                                                                                                                               |
| `opensearch.data.dedicatedPod.enabled`               | boolean | no        | false                                                                                                       | Whether dedicated `StatefulSet` for data is to be enabled. Otherwise `master` nodes are used as data storage.                                                                                                                                                                                                                                                                                                      |
| `opensearch.data.replicas`                           | integer | no        | 3                                                                                                           | The number of OpenSearch data nodes.                                                                                                                                                                                                                                                                                                                                                                               |
| `opensearch.data.updateStrategy`                     | string  | no        | RollingUpdate                                                                                               | The strategy used to replace old pods by new ones. The possible values are `OnDelete` and `RollingUpdate`. If `opensearch.rollingUpdate` is `true` then `OnDelete` is used by default and operator performs rolling upgrade in accordance with [guide](#operator-rolling-upgrade-feature). Otherwise `RollingUpgrade` is used and Kubernetes performs rolling upgrade in accordance with StatefulSet default flow. |
| `opensearch.data.persistence.enabled`                | boolean | no        | true                                                                                                        | Whether persistent storage is to be enabled on OpenSearch data nodes.                                                                                                                                                                                                                                                                                                                                              |
| `opensearch.data.persistence.existingClaim`          | string  | no        | ""                                                                                                          | The name of the existing persistent volume claim for OpenSearch data nodes.                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.data.persistence.subPath`                | string  | no        | ""                                                                                                          | The subdirectory of the volume to mount to.                                                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.data.persistence.storageClass`           | string  | no        | -                                                                                                           | The storage class name that is used for OpenSearch data nodes. If it is set to `-`, dynamic provisioning is disabled. If it is empty or set to `null`, the default provisioner is chosen.                                                                                                                                                                                                                          |
| `opensearch.data.persistence.persistentVolumes`      | list    | no        | []                                                                                                          | The list of predefined persistent volumes for OpenSearch data nodes. The number of persistent volumes should be equal to `opensearch.data.replicas` parameter. If `hostPath` PVs are used, the `nodes` parameters is also should be specified.                                                                                                                                                                     |
| `opensearch.data.persistence.nodes`                  | list    | no        | []                                                                                                          | The list of Kubernetes node names to assign OpenSearch data nodes. The number of nodes should be equal to `opensearch.data.replicas` parameter. It should not be used with `storageClass` pod assignment.                                                                                                                                                                                                          |
| `opensearch.data.persistence.accessModes`            | list    | no        | ["ReadWriteOnce"]                                                                                           | The list of access modes of persistent volumes for OpenSearch data nodes.                                                                                                                                                                                                                                                                                                                                          |
| `opensearch.data.persistence.size`                   | string  | no        | 5Gi                                                                                                         | The size of persistent volumes for OpenSearch data nodes.                                                                                                                                                                                                                                                                                                                                                          |
| `opensearch.data.persistence.annotations`            | object  | no        | {}                                                                                                          | The annotations of persistent volumes for OpenSearch data nodes.                                                                                                                                                                                                                                                                                                                                                   |
| `opensearch.data.resources.requests.cpu`             | string  | no        | 200m                                                                                                        | The minimum number of CPUs the OpenSearch data node container should use.                                                                                                                                                                                                                                                                                                                                          |
| `opensearch.data.resources.requests.memory`          | string  | no        | 1536Mi                                                                                                      | The minimum number of memory the OpenSearch data node container should use.                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.data.resources.limits.cpu`               | string  | no        | 500m                                                                                                        | The maximum number of CPUs the OpenSearch data node container should use.                                                                                                                                                                                                                                                                                                                                          |
| `opensearch.data.resources.limits.memory`            | string  | no        | 1536Mi                                                                                                      | The maximum number of memory the OpenSearch data node container should use.                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.data.javaOpts`                           | string  | no        | -Xms718m -Xmx718m                                                                                           | The Java options that are used for OpenSearch data nodes.                                                                                                                                                                                                                                                                                                                                                          |
| `opensearch.data.podDisruptionBudget.enabled`        | boolean | no        | false                                                                                                       | Whether the disruption budget for OpenSearch data nodes is to be created.                                                                                                                                                                                                                                                                                                                                          |
| `opensearch.data.podDisruptionBudget.minAvailable`   | integer | no        | 1                                                                                                           | The minimum number or percentage of pods that [should remain scheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                                                                                                                   |
| `opensearch.data.podDisruptionBudget.maxUnavailable` | integer | no        |                                                                                                             | The maximum number or percentage of pods that [may be unscheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                                                                                                                        |
| `opensearch.data.readinessProbe`                     | object  | no        | {}                                                                                                          | The configuration of the [readinessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch data nodes.                                                                                                                                                                                                                                                |
| `opensearch.data.livenessProbe`                      | object  | no        | {"tcpSocket": {"port": "transport"}, "initialDelaySeconds": 60, "periodSeconds": 20, "failureThreshold": 5} | The configuration of the [livenessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch data nodes.                                                                                                                                                                                                                                                 |
| `opensearch.data.startupProbe`                       | object  | no        | {}                                                                                                          | The configuration of the [startupProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) for OpenSearch data nodes.                                                                                                                                                                                                                                          |
| `opensearch.data.imagePullSecrets`                   | list    | no        | []                                                                                                          | The list of references to secrets in the same namespace to use for pulling any of the images used by OpenSearch data containers.                                                                                                                                                                                                                                                                                   |
| `opensearch.data.nodeSelector`                       | object  | no        | {}                                                                                                          | The selector that defines the nodes where the OpenSearch data nodes are scheduled on.                                                                                                                                                                                                                                                                                                                              |
| `opensearch.data.tolerations`                        | list    | no        | []                                                                                                          | The list of toleration policies for OpenSearch data nodes in `JSON` format.                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.data.affinity`                           | object  | no        | <anti_affinity_rule>                                                                                        | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                    |
| `opensearch.data.podAnnotations`                     | object  | no        | {}                                                                                                          | The annotations for OpenSearch data pod.                                                                                                                                                                                                                                                                                                                                                                           |
| `opensearch.data.customLabels`                       | object  | no        | {}                                                                                                          | The custom labels for the OpenSearch data pods.                                                                                                                                                                                                                                                                                                                                                                    |
| `opensearch.data.priorityClassName`                  | string  | no        | ""                                                                                                          | The priority class to be used by the OpenSearch data nodes. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                                                                                           |

Where:
* `<anti_affinity_rule>` is as follows:

  ```
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 1
        podAffinityTerm:
          topologyKey: "kubernetes.io/hostname"
          labelSelector:
            matchLabels:
              role: data
  ```

### Client Nodes

| Parameter                                              | Type    | Mandatory | Default value                                                                                               | Description                                                                                                                                                                                                                                                                                                                |
|--------------------------------------------------------|---------|-----------|-------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opensearch.client.enabled`                            | boolean | no        | true                                                                                                        | Whether the OpenSearch `client`/`ingester` nodes are to be enabled.                                                                                                                                                                                                                                                        |
| `opensearch.client.dedicatedPod.enabled`               | boolean | no        | false                                                                                                       | Whether dedicated `Deployment` for data is to be enabled. Otherwise `master` nodes are used as client/ingest.                                                                                                                                                                                                              |
| `opensearch.client.service.type`                       | string  | no        | ClusterIP                                                                                                   | The type of OpenSearch client service.                                                                                                                                                                                                                                                                                     |
| `opensearch.client.service.annotations`                | object  | no        | {}                                                                                                          | The annotations for OpenSearch client service.                                                                                                                                                                                                                                                                             |
| `opensearch.client.replicas`                           | integer | no        | 3                                                                                                           | The number of OpenSearch client nodes.                                                                                                                                                                                                                                                                                     |
| `opensearch.client.javaOpts`                           | string  | no        | -Xms512m -Xmx512m                                                                                           | The Java options that are used for OpenSearch client nodes.                                                                                                                                                                                                                                                                |
| `opensearch.client.ingress.enabled`                    | boolean | no        | false                                                                                                       | Whether OpenSearch client ingress is to be enabled.                                                                                                                                                                                                                                                                        |
| `opensearch.client.ingress.annotations`                | object  | no        | {}                                                                                                          | The annotations for OpenSearch client ingress.                                                                                                                                                                                                                                                                             |
| `opensearch.client.ingress.className`                  | string  | no        | ""                                                                                                          | The class name for OpenSearch client ingress.                                                                                                                                                                                                                                                                              |
| `opensearch.client.ingress.labels`                     | object  | no        | {}                                                                                                          | The labels for OpenSearch client ingress.                                                                                                                                                                                                                                                                                  |
| `opensearch.client.ingress.path`                       | string  | no        | /                                                                                                           | The path for OpenSearch client ingress.                                                                                                                                                                                                                                                                                    |
| `opensearch.client.ingress.hosts`                      | list    | no        | []                                                                                                          | The list of hosts for OpenSearch client ingress.                                                                                                                                                                                                                                                                           |
| `opensearch.client.ingress.tls`                        | list    | no        | []                                                                                                          | The list of TLS configurations for OpenSearch client ingress.                                                                                                                                                                                                                                                              |
| `opensearch.client.resources.requests.cpu`             | string  | no        | 200m                                                                                                        | The minimum number of CPUs the OpenSearch client node container should use.                                                                                                                                                                                                                                                |
| `opensearch.client.resources.requests.memory`          | string  | no        | 1024Mi                                                                                                      | The minimum number of memory the OpenSearch client node container should use.                                                                                                                                                                                                                                              |
| `opensearch.client.resources.limits.cpu`               | string  | no        | 1                                                                                                           | The maximum number of CPUs the OpenSearch client node container should use.                                                                                                                                                                                                                                                |
| `opensearch.client.resources.limits.memory`            | string  | no        | 1024Mi                                                                                                      | The maximum number of memory the OpenSearch client node container should use.                                                                                                                                                                                                                                              |
| `opensearch.client.podDisruptionBudget.enabled`        | boolean | no        | false                                                                                                       | Whether the disruption budget for OpenSearch client nodes is to be created.                                                                                                                                                                                                                                                |
| `opensearch.client.podDisruptionBudget.minAvailable`   | integer | no        | 1                                                                                                           | The minimum number or percentage of pods that [should remain scheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                           |
| `opensearch.client.podDisruptionBudget.maxUnavailable` | integer | no        |                                                                                                             | The maximum number or percentage of pods that [may be unscheduled](https://kubernetes.io/docs/tasks/run-application/configure-pdb/#think-about-how-your-application-reacts-to-disruptions).                                                                                                                                |
| `opensearch.client.readinessProbe`                     | object  | no        | {}                                                                                                          | The configuration of the [readinessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch client nodes.                                                                                                                                                      |
| `opensearch.client.livenessProbe`                      | object  | no        | {"tcpSocket": {"port": "transport"}, "initialDelaySeconds": 60, "periodSeconds": 20, "failureThreshold": 5} | The configuration of the [livenessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) for OpenSearch client nodes.                                                                                                                                                       |
| `opensearch.client.startupProbe`                       | object  | no        | {}                                                                                                          | The configuration of the [startupProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) for OpenSearch client nodes.                                                                                                                                                |
| `opensearch.client.imagePullSecrets`                   | list    | no        | []                                                                                                          | The list of references to secrets in the same namespace to use for pulling any of the images used by OpenSearch client containers.                                                                                                                                                                                         |
| `opensearch.client.nodeSelector`                       | object  | no        | {}                                                                                                          | The selector that defines the nodes where the OpenSearch data nodes are scheduled on.                                                                                                                                                                                                                                      |
| `opensearch.client.tolerations`                        | list    | no        | []                                                                                                          | The list of toleration policies for OpenSearch client nodes in `JSON` format.                                                                                                                                                                                                                                              |
| `opensearch.client.affinity`                           | object  | no        | <anti_affinity_rule>                                                                                        | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                            |
| `opensearch.client.podAnnotations`                     | object  | no        | {}                                                                                                          | The annotations for OpenSearch client pod.                                                                                                                                                                                                                                                                                 |
| `opensearch.client.customLabels`                       | object  | no        | {}                                                                                                          | The custom labels for the OpenSearch client pods.                                                                                                                                                                                                                                                                          |
| `opensearch.client.priorityClassName`                  | string  | no        | ""                                                                                                          | The priority class to be used by the OpenSearch client nodes. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/). |

Where:
* `<anti_affinity_rule>` is as follows:

  ```
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 1
        podAffinityTerm:
          topologyKey: "kubernetes.io/hostname"
          labelSelector:
            matchLabels:
              role: client
  ```

### Snapshots

| Parameter                                    | Type    | Mandatory | Default value | Description                                                                                                                                                                                                                                                                                                                                                             |
|----------------------------------------------|---------|-----------|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opensearch.snapshots.enabled`               | boolean | no        | false         | Whether persistent volume claim for snapshot repository is to be created and mounted.                                                                                                                                                                                                                                                                                   |
| `opensearch.snapshots.repositoryName`        | string  | no        | snapshots     | The name of snapshot repository in OpenSearch.                                                                                                                                                                                                                                                                                                                          |
| `opensearch.snapshots.persistentVolume`      | string  | no        | ""            | The name of `RWX` persistent volume to store snapshots.                                                                                                                                                                                                                                                                                                                 |
| `opensearch.snapshots.persistentVolumeClaim` | string  | no        | ""            | The name of `RWX` persistent volume claim to store snapshots. If it is specified, OpenSearch pods use specified persistent volume claim.                                                                                                                                                                                                                                |
| `opensearch.snapshots.storageClass`          | string  | no        | ""            | The name of shared storage class to store snapshots.                                                                                                                                                                                                                                                                                                                    |
| `opensearch.snapshots.size`                  | string  | no        | 5Gi           | The size of persistent volume to store snapshots.                                                                                                                                                                                                                                                                                                                       |
| `opensearch.snapshots.s3.enabled`            | boolean | no        | false         | Whether OpenSearch backups are to be stored in S3 storage. OpenSearch supports the following S3 providers: AWS S3, MinIO. Google Cloud Storage is not supported. Other S3 providers may work, but are not covered by the OpenSearch test suite. **Note**: Parameters `snapshots.persistentVolume` and `snapshots.storageClass` are not needed if S3 storage is enabled. |
| `opensearch.snapshots.s3.pathStyleAccess`    | boolean | no        | false         | Whether path style access to S3 storage is to be enabled. **Note**: For Minio, this parameter value must be set to `true`.                                                                                                                                                                                                                                              |
| `opensearch.snapshots.s3.url`                | string  | no        | ""            | The URL to the S3 storage.                                                                                                                                                                                                                                                                                                                                              |
| `opensearch.snapshots.s3.bucket`             | string  | no        | ""            | The existing bucket in the S3 storage.                                                                                                                                                                                                                                                                                                                                  |
| `opensearch.snapshots.s3.basePath`           | string  | no        | ""            | The base path in the S3 storage.                                                                                                                                                                                                                                                                                                                                        |
| `opensearch.snapshots.s3.region`             | string  | no        | default       | The region in the S3 storage.                                                                                                                                                                                                                                                                                                                                           |
| `opensearch.snapshots.s3.keyId`              | string  | no        | ""            | The key ID for the S3 storage.                                                                                                                                                                                                                                                                                                                                          |
| `opensearch.snapshots.s3.keySecret`          | string  | no        | ""            | The key secret for the S3 storage.                                                                                                                                                                                                                                                                                                                                      |
| `opensearch.snapshots.s3.gcs.secretName`     | string  | no        | ""            | The name of pre-created secret with JSON key to GCS bucket. The key must be created according to the [Google Cloud Prerequisites](#google-cloud) guide.                                                                                                                                                                                                                 |
| `opensearch.snapshots.s3.gcs.secretKey`      | string  | no        | ""            | The key of value with GCS JSON key inside secret.                                                                                                                                                                                                                                                                                                                       |

## Pod Scheduler

| Parameter                                | Type    | Mandatory | Default value            | Description                                                                                                                                                                                                                                                                                                                 |
|------------------------------------------|---------|-----------|--------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `podScheduler.enabled`                   | boolean | no        | true                     | Whether custom Kubernetes pod scheduler pod is to be deployed to assign OpenSearch pods to nodes with `hostPath` persistent volumes. It must be enabled if `persistentVolumes` and `nodes` are specified for `master` or `data` persistence.                                                                                |
| `podScheduler.dockerImage`               | string  | no        | Calculates automatically | The docker image for Pod Scheduler.                                                                                                                                                                                                                                                                                         |
| `podScheduler.affinity`                  | object  | no        | {}                       | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                             |
| `podScheduler.nodeSelector`              | object  | no        | {}                       | The selector that defines the nodes where the Pod Scheduler pods are to be scheduled on.                                                                                                                                                                                                                                    |
| `podScheduler.resources.requests.cpu`    | string  | no        | 15m                      | The minimum number of CPUs the Pod Scheduler container should use.                                                                                                                                                                                                                                                          |
| `podScheduler.resources.requests.memory` | string  | no        | 128Mi                    | The minimum number of memory the Pod Scheduler container should use.                                                                                                                                                                                                                                                        |
| `podScheduler.resources.limits.cpu`      | string  | no        | 50m                      | The maximum number of CPUs the Pod Scheduler container should use.                                                                                                                                                                                                                                                          |
| `podScheduler.resources.limits.memory`   | string  | no        | 128Mi                    | The maximum number of memory the Pod Scheduler container should use.                                                                                                                                                                                                                                                        |
| `podScheduler.securityContext`           | object  | no        | {}                       | The pod-level security attributes and common container settings for the Pod Scheduler pods. It should be filled as `runAsUser: 1000` for non-root privileges environments.                                                                                                                                                  |
| `podScheduler.customLabels`              | object  | no        | {}                       | The custom labels for the OpenSearch scheduler pod.                                                                                                                                                                                                                                                                         |
| `podScheduler.priorityClassName`         | string  | no        | ""                       | The priority class to be used by the OpenSearch pod scheduler. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/). |

## Monitoring

| Parameter                                          | Type    | Mandatory | Default value            | Description                                                                                                                                                                                                                                                                                                                                                                                                                            |
|----------------------------------------------------|---------|-----------|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `monitoring.enabled`                               | boolean | no        | true                     | Whether the installation of OpenSearch monitoring is to be enabled.                                                                                                                                                                                                                                                                                                                                                                    |
| `monitoring.dockerImage`                           | string  | no        | Calculates automatically | The docker image of OpenSearch monitoring.                                                                                                                                                                                                                                                                                                                                                                                             |
| `monitoring.imagePullPolicy`                       | string  | no        | Always                   | The image pull policy for OpenSearch monitoring container. The possible values are "Always", "IfNotPresent", or "Never".                                                                                                                                                                                                                                                                                                               |
| `monitoring.monitoringType`                        | string  | no        | prometheus               | The type of output plugin that is used for OpenSearch monitoring. The possible values are "influxdb" and "prometheus". If the value of this parameter is `influxdb`, you need to check and specify the parameters necessary for `InfluxDB` plugin (`smDbHost`, `smDbName`, `smDbUsername`, `smDbPassword`). Prometheus plugin does not require additional parameters for the configuration.                                            |
| `monitoring.installDashboard`                      | boolean | no        | true                     | Whether the installation of OpenSearch Grafana dashboard is to be enabled.                                                                                                                                                                                                                                                                                                                                                             |
| `monitoring.smDbHost`                              | string  | no        | ""                       | The host of the System Monitoring database. You must specify the parameter only if `monitoringType` parameter is equal to `influxdb`.                                                                                                                                                                                                                                                                                                  |
| `monitoring.smDbName`                              | string  | no        | ""                       | The name of the database in System Monitoring. You must specify the parameter only if `monitoringType` parameter is equal to `influxdb`.                                                                                                                                                                                                                                                                                               |
| `monitoring.smDbUsername`                          | string  | no        | ""                       | The name of the database user in System Monitoring. The parameter should be specified if `monitoringType` parameter is equal to `influxdb` and authentication is enabled in System Monitoring.                                                                                                                                                                                                                                         |
| `monitoring.smDbPassword`                          | string  | no        | ""                       | The password of the database user in System Monitoring. The parameter should be specified if `monitoringType` parameter is equal to `influxdb` and authentication is enabled in System Monitoring.                                                                                                                                                                                                                                     |
| `monitoring.includeIndices`                        | boolean | no        | false                    | Whether the collection of indices metrics is to be included in the Telegraf plugin.                                                                                                                                                                                                                                                                                                                                                    |
| `monitoring.slowQueries.enabled`                   | boolean | no        | false                    | Whether the slow queries metric is to be enabled. **Important**: Slow queries functionality doesn't work on AWS cloud.                                                                                                                                                                                                                                                                                                                 |
| `monitoring.slowQueries.topNumber`                 | integer | no        | 10                       | The number of slow queries that should be calculated.                                                                                                                                                                                                                                                                                                                                                                                  |
| `monitoring.slowQueries.processingIntervalMinutes` | integer | no        | 5                        | The duration in minutes of the interval that is used to process records from `slow-log` file. If the value is set to `5` minutes and the `slow_queries_metric.py` script is performed at `2023-07-27T08:03:43` then the processing interval is `2023-07-27T07:58:43`-`2023-07-27T08:03:43` and all log records from the slow-log file that are associated with that period are to be selected to calculate the rating of slow queries. |
| `monitoring.slowQueries.minSeconds`                | integer | no        | 5                        | The time in seconds from which a query is considered slow and is written to `slow-log` file by OpenSearch.                                                                                                                                                                                                                                                                                                                             |
| `monitoring.slowQueries.indicesPattern`            | string  | no        | *                        | The pattern with wildcards used to define OpenSearch indices to track slow queries.                                                                                                                                                                                                                                                                                                                                                    |
| `monitoring.thresholds.lagAlert`                   | integer | no        |                          | The maximum value of replication lag before a replication alert occurs. If it is not specified, the alert is not added.                                                                                                                                                                                                                                                                                                                |
| `monitoring.thresholds.slowQuerySecondsAlert`      | integer | no        | 10                       | The threshold in seconds that is used for slow query (`OpenSearchQueryIsTooSlowAlert`) alert.                                                                                                                                                                                                                                                                                                                                          |
| `monitoring.opensearchHost`                        | string  | no        | ""                       | The host address of OpenSearch. If it is not specified, the `<name>-internal` value is used, where `<name>` is the value of the `fullnameOverride` parameter.                                                                                                                                                                                                                                                                          |
| `monitoring.opensearchPort`                        | string  | no        | 9200                     | The port of OpenSearch.                                                                                                                                                                                                                                                                                                                                                                                                                |
| `monitoring.opensearchExecPluginTimeout`           | string  | no        | 15s                      | The timeout for OpenSearch exec Telegraf plugin.                                                                                                                                                                                                                                                                                                                                                                                       |
| `monitoring.opensearchDbaasAdapterHost`            | string  | no        | ""                       | The host of monitored OpenSearch DBaaS adapter. If it is not specified, the `dbaas-<name>-adapter` value is used, where `<name>` is the value of the `fullnameOverride` parameter.                                                                                                                                                                                                                                                     |
| `monitoring.opensearchDbaasAdapterPort`            | string  | no        | 8080                     | The port of monitored OpenSearch DBaaS adapter.                                                                                                                                                                                                                                                                                                                                                                                        |
| `monitoring.resources.requests.cpu`                | string  | no        | 200m                     | The minimum number of CPUs the monitoring container should use.                                                                                                                                                                                                                                                                                                                                                                        |
| `monitoring.resources.requests.memory`             | string  | no        | 256Mi                    | The minimum amount of memory the monitoring container should use.                                                                                                                                                                                                                                                                                                                                                                      |
| `monitoring.resources.limits.cpu`                  | string  | no        | 200m                     | The maximum number of CPUs the monitoring container should use.                                                                                                                                                                                                                                                                                                                                                                        |
| `monitoring.resources.limits.memory`               | string  | no        | 256Mi                    | The maximum amount of memory the monitoring container should use.                                                                                                                                                                                                                                                                                                                                                                      |
| `monitoring.nodeSelector`                          | object  | no        | {}                       | The selector that defines the nodes where the OpenSearch monitoring pods are to be scheduled on.                                                                                                                                                                                                                                                                                                                                       |
| `monitoring.tolerations`                           | list    | no        | []                       | The list of toleration policies for OpenSearch monitoring pod in `JSON` format.                                                                                                                                                                                                                                                                                                                                                        |
| `monitoring.affinity`                              | object  | no        | {}                       | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                                        |
| `monitoring.securityContext`                       | object  | no        | {"runAsUser": 1000}      | The pod-level security attributes and common container settings for the OpenSearch monitoring pods.                                                                                                                                                                                                                                                                                                                                    |
| `monitoring.monitoringCoreosGroup`                 | boolean | no        | false                    | Whether the `monitoringCoreosGroup` verbs are to be added to the OpenSearch service operator role.                                                                                                                                                                                                                                                                                                                                     |
| `monitoring.customLabels`                          | object  | no        | {}                       | The custom labels for the OpenSearch monitoring pod.                                                                                                                                                                                                                                                                                                                                                                                   |
| `monitoring.priorityClassName`                     | string  | no        | ""                       | The priority class to be used by the OpenSearch monitoring pods. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                                                                                                          |

## OpenSearch DBaaS Adapter

| Parameter                                                       | Type    | Mandatory | Default value                                          | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
|-----------------------------------------------------------------|---------|-----------|--------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `dbaasAdapter.enabled`                                          | boolean | no        | false                                                  | Whether the installation of OpenSearch DBaaS adapter is to be enabled. It provides connection with credentials with necessary grants only for indices meant for a particular microservice. The migration procedure between Elasticsearch and OpenSearch DBaaS adapter is described in [DBaaS Adapter Migration](#dbaas-adapter-migration) section.                                                                                                                                                                                                                                                              |
| `dbaasAdapter.dockerImage`                                      | string  | no        | Calculates automatically                               | The docker image of OpenSearch DBaaS adapter.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `dbaasAdapter.imagePullPolicy`                                  | string  | no        | Always                                                 | The image pull policy for OpenSearch DBaaS adapter container. The possible values are "Always", "IfNotPresent", or "Never".                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `dbaasAdapter.dbaasAdapterAddress`                              | string  | no        | `<protocol>://dbaas-<name>-adapter.<namespace>:<port>` | The address of OpenSearch DBaaS adapter, where aggregator should send requests.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `dbaasAdapter.dbaasUsername`                                    | string  | no        | ""                                                     | The name of the OpenSearch DBaaS adapter user, either a new user or an existing one.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `dbaasAdapter.dbaasPassword`                                    | string  | no        | ""                                                     | The password of the OpenSearch DBaaS adapter user, either a new user or an existing one.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `dbaasAdapter.apiVersion`                                       | string  | no        | v2                                                     | The version of OpenSearch DBaaS Adapter API. Selected version changes strategy of DBaaS Aggregator registration, response format and work. It would be used in case of DBaaS unavailability. Otherwise, `apiVersion` is resolved by DBaaS Aggregator `api-version` request. The possible values are `v1`, `v2`. The `v1` version allows to create users only with `admin` permissions, but the `v2` version creates 3 users with different roles (`admin`, `dml`, `readonly`) on each corresponding request. If you are upgraded OpenSearch DBaaS adapter from `v1` to `v2` version, you must not downgrade it. |
| `dbaasAdapter.dbaasAggregatorRegistrationAddress`               | string  | no        | `<protocol>://dbaas-aggregator.dbaas:<port>`           | The address of DBaaS aggregator, which should register physical database. You need to specify this only if there are more than one aggregators installed in cloud and you need to choose one, or if the adapter is not in the same cloud, where aggregator is, or if default aggregator is not installed in the default `dbaas` project.                                                                                                                                                                                                                                                                        |
| `dbaasAdapter.dbaasAggregatorPhysicalDatabaseIdentifier`        | string  | no        | <namespace>                                            | The unique ID of physical database, which OpenSearch DBaaS adapter connects to.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `dbaasAdapter.registrationAuthUsername`                         | string  | no        | ""                                                     | The name of user for DBaaS aggregator's registration API.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `dbaasAdapter.registrationAuthPassword`                         | string  | no        | ""                                                     | The password of user for DBaaS aggregator's registration API.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `dbaasAdapter.opensearchHost`                                   | string  | no        | `<name>.<namespace>`                                   | The host address of OpenSearch.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `dbaasAdapter.opensearchPort`                                   | string  | no        | 9200                                                   | The port of OpenSearch. If the OpenSearch URL does not contain port for example `https://opensearch`, the default protocol port should be specified: `80` for `http` and `443` for `https`.                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `dbaasAdapter.opensearchProtocol`                               | string  | no        | https                                                  | The protocol of communication with the OpenSearch. The allowed values are `http`, `https`. To access to `https` OpenSearch you need to install trusted TLS certificates for DBaaS Adapter.                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `dbaasAdapter.opensearchRepo`                                   | string  | no        | snapshots                                              | The name of snapshot repository in OpenSearch. The default behavior is to create a new repository with file storage location for each backup.                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `dbaasAdapter.opensearchRepoRoot`                               | string  | no        | /usr/share/opensearch                                  | The absolute path in OpenSearch file system where snapshot repositories for backups are created.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `dbaasAdapter.opensearchClusterVersion`                         | string  | no        | ""                                                     | The one of labels to set on the project, would be needed for clients to choose appropriate cluster. If not specified, empty string would be included in labels, client could not know which version is OpenSearch before requesting DBaaS to create index in it.                                                                                                                                                                                                                                                                                                                                                |
| `dbaasAdapter.netcrackerOpensearchClusterVersion`               | string  | no        | ""                                                     | The one of labels to set on the project, would be needed for clients to choose appropriate cluster. If not specified, an empty string would be included in labels, and it could not be known which version is the Netcracker wrapper around OpenSearch before requesting DBaaS to create index in it.                                                                                                                                                                                                                                                                                                           |
| `dbaasAdapter.tls.enabled`                                      | boolean | no        | true                                                   | Whether TLS is to be enabled for OpenSearch DBaaS adapter. This parameter is taken into account only if the `global.tls.enabled` parameter is set to `true`.                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `dbaasAdapter.tls.secretName`                                   | string  | no        | ""                                                     | The name of the secret that contains TLS certificates. It is required if TLS for OpenSearch DBaaS adapter is enabled and certificates generation is disabled.                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `dbaasAdapter.tls.subjectAlternativeName.additionalDnsNames`    | list    | no        | []                                                     | The list of additional DNS names to be added to the `Subject Alternative Name` field of TLS certificate for OpenSearch DBaaS adapter.                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `dbaasAdapter.tls.subjectAlternativeName.additionalIpAddresses` | list    | no        | []                                                     | The list of additional IP addresses to be added to the `Subject Alternative Name` field of TLS certificate for OpenSearch DBaaS adapter.                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `dbaasAdapter.resources.requests.cpu`                           | string  | no        | 200m                                                   | The minimum number of CPUs the DBaaS adapter container should use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `dbaasAdapter.resources.requests.memory`                        | string  | no        | 32Mi                                                   | The minimum amount of memory the DBaaS adapter container should use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `dbaasAdapter.resources.limits.cpu`                             | string  | no        | 200m                                                   | The maximum number of CPUs the DBaaS adapter container should use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `dbaasAdapter.resources.limits.memory`                          | string  | no        | 32Mi                                                   | The maximum amount of memory the DBaaS adapter container can use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `dbaasAdapter.nodeSelector`                                     | object  | no        | {}                                                     | The selector that defines the nodes where the OpenSearch DBaaS adapter pods are scheduled on.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `dbaasAdapter.tolerations`                                      | list    | no        | []                                                     | The list of toleration policies for OpenSearch DBaaS adapter pod in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `dbaasAdapter.affinity`                                         | object  | no        | {}                                                     | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `dbaasAdapter.securityContext`                                  | object  | no        | {"runAsUser": 1000}                                    | The pod-level security attributes and common container settings for OpenSearch DBaaS adapter pod.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `dbaasAdapter.customLabels`                                     | object  | no        | {}                                                     | The custom labels for the OpenSearch DBaaS adapter pod.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `dbaasAdapter.priorityClassName`                                | string  | no        | ""                                                     | The priority class to be used by the OpenSearch DBaaS adapter pods. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                                                                                                                                                                                                                                                                                |

Where:

* `<protocol>` is the `http` or `https` protocol depending on `dbaasAdapter.tls.enabled` parameter.
* `<name>` is the value of the `fullnameOverride` parameter.
* `<namespace>` is the current namespace.
* `<port>` is the `8080` or `8443` port depending on `dbaasAdapter.tls.enabled` parameter.

## Elasticsearch DBaaS Adapter

| Parameter                                                             | Type    | Mandatory | Default value                                    | Description                                                                                                                                                                                                                                                                                                                                           |
|-----------------------------------------------------------------------|---------|-----------|--------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `elasticsearchDbaasAdapter.enabled`                                   | boolean | no        | false                                            | Whether the installation of Elasticsearch DBaaS adapter is to be enabled. It provides connection with credentials with necessary grants only for indices meant for a particular microservice. The migration procedure between Elasticsearch and OpenSearch DBaaS adapter is described in [DBaaS Adapter Migration](#dbaas-adapter-migration) section. |
| `elasticsearchDbaasAdapter.name`                                      | string  | no        | dbaas-elasticsearch-adapter                      | The base name of all Elasticsearch DBaaS adapter resources: deployment, service, secret, configuration.                                                                                                                                                                                                                                               |
| `elasticsearchDbaasAdapter.dockerImage`                               | string  | no        | Calculates automatically                         | The docker image of Elasticsearch DBaaS adapter.                                                                                                                                                                                                                                                                                                      |
| `elasticsearchDbaasAdapter.imagePullPolicy`                           | string  | no        | Always                                           | The image pull policy for Elasticsearch DBaaS adapter container. The possible values are "Always", "IfNotPresent", or "Never".                                                                                                                                                                                                                        |
| `elasticsearchDbaasAdapter.dbaasAdapterAddress`                       | string  | no        | `<protocol>://<adapter_name>.<namespace>:<port>` | The address of Elasticsearch DBaaS adapter, where aggregator should send requests.                                                                                                                                                                                                                                                                    |
| `elasticsearchDbaasAdapter.dbaasUsername`                             | string  | no        | ""                                               | The name of the Elasticsearch DBaaS adapter user, either a new user or an existing one.                                                                                                                                                                                                                                                               |
| `elasticsearchDbaasAdapter.dbaasPassword`                             | string  | no        | ""                                               | The password of the Elasticsearch DBaaS adapter user, either a new user or an existing one.                                                                                                                                                                                                                                                           |
| `elasticsearchDbaasAdapter.dbaasAggregatorRegistrationAddress`        | string  | no        | `<protocol>://dbaas-aggregator.dbaas:<port>`     | The address of DBaaS aggregator, which should register physical database. You need to specify this only if there are more than one aggregators installed in cloud and you need to choose one, or if the adapter is not in the same cloud, where aggregator is, or if default aggregator is not installed in the default `dbaas` project.              |
| `elasticsearchDbaasAdapter.dbaasAggregatorPhysicalDatabaseIdentifier` | string  | no        | <namespace>                                      | The unique ID of physical database, which Elasticsearch DBaaS adapter connects to.                                                                                                                                                                                                                                                                    |
| `elasticsearchDbaasAdapter.registrationAuthUsername`                  | string  | no        | ""                                               | The name of user for DBaaS aggregator's registration API.                                                                                                                                                                                                                                                                                             |
| `elasticsearchDbaasAdapter.registrationAuthPassword`                  | string  | no        | ""                                               | The password of user for DBaaS aggregator's registration API.                                                                                                                                                                                                                                                                                         |
| `elasticsearchDbaasAdapter.opensearchHost`                            | string  | no        | `<name>.<namespace>`                             | The host address of OpenSearch.                                                                                                                                                                                                                                                                                                                       |
| `elasticsearchDbaasAdapter.opensearchPort`                            | string  | no        | 9200                                             | The port of OpenSearch. If the OpenSearch URL does not contain port for example `https://opensearch`, the default protocol port should be specified: `80` for `http` and `443` for `https`.                                                                                                                                                           |
| `elasticsearchDbaasAdapter.opensearchProtocol`                        | string  | no        | https                                            | The protocol of communication with the OpenSearch. The allowed values are `http`, `https`. To access to `https` OpenSearch you need to install trusted TLS certificates for DBaaS Adapter.                                                                                                                                                            |
| `elasticsearchDbaasAdapter.opensearchRepo`                            | string  | no        | snapshots                                        | The name of snapshot repository in OpenSearch. The default behavior is to create a new repository with file storage location for each backup.                                                                                                                                                                                                         |
| `elasticsearchDbaasAdapter.opensearchRepoRoot`                        | string  | no        | /usr/share/opensearch                            | The absolute path in OpenSearch file system where snapshot repositories for backups are created.                                                                                                                                                                                                                                                      |
| `elasticsearchDbaasAdapter.opensearchClusterVersion`                  | string  | no        | ""                                               | The one of labels to set on the project, would be needed for clients to choose appropriate cluster. If not specified, empty string would be included in labels, client could not know which version is OpenSearch before requesting DBaaS to create index in it.                                                                                      |
| `elasticsearchDbaasAdapter.netcrackerOpensearchClusterVersion`        | string  | no        | ""                                               | The one of labels to set on the project, would be needed for clients to choose appropriate cluster. If not specified, an empty string would be included in labels, and it could not be known which version is the Netcracker wrapper around OpenSearch before requesting DBaaS to create index in it.                                                 |
| `elasticsearchDbaasAdapter.resources.requests.cpu`                    | string  | no        | 200m                                             | The minimum number of CPUs the DBaaS adapter container should use.                                                                                                                                                                                                                                                                                    |
| `elasticsearchDbaasAdapter.resources.requests.memory`                 | string  | no        | 32Mi                                             | The minimum amount of memory the DBaaS adapter container should use.                                                                                                                                                                                                                                                                                  |
| `elasticsearchDbaasAdapter.resources.limits.cpu`                      | string  | no        | 200m                                             | The maximum number of CPUs the DBaaS adapter container should use.                                                                                                                                                                                                                                                                                    |
| `elasticsearchDbaasAdapter.resources.limits.memory`                   | string  | no        | 32Mi                                             | The maximum amount of memory the DBaaS adapter container can use.                                                                                                                                                                                                                                                                                     |
| `elasticsearchDbaasAdapter.nodeSelector`                              | object  | no        | {}                                               | The selector that defines the nodes where the Elasticsearch DBaaS adapter pods are scheduled on.                                                                                                                                                                                                                                                      |
| `elasticsearchDbaasAdapter.tolerations`                               | list    | no        | []                                               | The list of toleration policies for Elasticsearch DBaaS adapter pod in `JSON` format.                                                                                                                                                                                                                                                                 |
| `elasticsearchDbaasAdapter.affinity`                                  | object  | no        | {}                                               | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                                                       |
| `elasticsearchDbaasAdapter.securityContext`                           | object  | no        | {}                                               | The pod-level security attributes and common container settings for Elasticsearch DBaaS adapter pod. For example, `{"runAsUser": 1000}`.                                                                                                                                                                                                              |
| `elasticsearchDbaasAdapter.customLabels`                              | object  | no        | {}                                               | The custom labels for the Elasticsearch DBaaS adapter pod.                                                                                                                                                                                                                                                                                            |
| `elasticsearchDbaasAdapter.priorityClassName`                         | string  | no        | ""                                               | The priority class to be used by the Elasticsearch DBaaS adapter pods. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                   |

Where:

* `<protocol>` is the `http` or `https` protocol depending on `dbaasAdapter.tls.enabled` parameter.
* `<adapter_name>` is the value of `elasticsearchDbaasAdapter.name` parameter.
* `<name>` is the value of the `fullnameOverride` parameter.
* `<namespace>` is the current namespace.
* `<port>` is the `8080` or `8443` port depending on `dbaasAdapter.tls.enabled` parameter.

## Curator

| Parameter                                                  | Type    | Mandatory | Default value            | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
|------------------------------------------------------------|---------|-----------|--------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `curator.enabled`                                          | boolean | no        | false                    | Whether the installation of OpenSearch curator is to be enabled.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `curator.dockerImage`                                      | string  | no        | Calculates automatically | The docker image of OpenSearch curator.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `curator.dockerIndicesCleanerImage`                        | string  | no        | Calculates automatically | The docker image of OpenSearch indices cleaner.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `curator.imagePullPolicy`                                  | string  | no        | Always                   | The image pull policy for OpenSearch curator container. The possible values are "Always", "IfNotPresent", or "Never".                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `curator.opensearchHost`                                   | string  | no        | `<name>-internal:9200`   | The host address of OpenSearch, including the port.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `curator.snapshotRepositoryName`                           | string  | no        | snapshots                | The name of snapshot repository in the OpenSearch. This parameter determines **logical** name of folder where OpenSearch stores snapshots. This parameter is necessary for OpenSearch internal process to make snapshots. Do not use slash symbol `/` in this name. Note that the value of this parameter must be the same as the value of the similar parameter for OpenSearch.                                                                                                                                                                                                |
| `curator.backupSchedule`                                   | string  | no        | ""                       | The schedule time in cron format (value must be within quotes). If this parameter is empty, the default schedule (`"0 0 * * *"`), defined in OpenSearch Curator configuration, is used. The value `0 0 * * *` means that snapshots are created everyday at 0:00.                                                                                                                                                                                                                                                                                                                |
| `curator.evictionPolicy`                                   | string  | no        | ""                       | The eviction policy for snapshots. It is a comma-separated string of policies written as `$start_time/$interval`. This policy splits all backups older than `$start_time` to numerous time intervals `$interval` time long. Then it deletes all backups in every interval except the newest one. For example, `1d/7d` policy means "take all backups older then one day, split them in groups by 7-days interval, and leave only the newest". If this parameter is empty, the default eviction policy (`"0/1d,7d/delete"`) defined in OpenSearch Curator configuration is used. |
| `curator.username`                                         | string  | no        | ""                       | The name of the OpenSearch Curator API user. This parameter enables OpenSearch Curator authentication. If the parameter is empty, OpenSearch Curator is deployed with disabled authentication.                                                                                                                                                                                                                                                                                                                                                                                  |
| `curator.password`                                         | string  | no        | ""                       | The password of the OpenSearch Curator API user. This parameter enables OpenSearch Curator authentication. If the parameter is empty, OpenSearch Curator is deployed with disabled authentication.                                                                                                                                                                                                                                                                                                                                                                              |
| `curator.tls.enabled`                                      | boolean | no        | true                     | Whether TLS is to be enabled for OpenSearch Curator. This parameter is taken into account only if `global.tls.enabled` parameter is set to `true`. For more information about TLS, refer to the [TLS Encryption](/docs/public/tls.md) section.                                                                                                                                                                                                                                                                                                                                  |
| `curator.tls.secretName`                                   | string  | no        | ""                       | The name of the secret that contains TLS certificates of OpenSearch Curator. It is required if TLS for OpenSearch Curator is enabled and certificates generation is disabled.                                                                                                                                                                                                                                                                                                                                                                                                   |
| `curator.tls.subjectAlternativeName.additionalDnsNames`    | list    | no        | []                       | The list of additional DNS names to be added to the `Subject Alternative Name` field of OpenSearch Curator TLS certificate.                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `curator.tls.subjectAlternativeName.additionalIpAddresses` | list    | no        | []                       | The list of additional IP addresses to be added to the `Subject Alternative Name` field of OpenSearch Curator TLS certificate.                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `curator.indicesCleanerSchedulerUnit`                      | string  | no        | days                     | The scheduler unit which specifies measure for value in `curator.indicesCleanerSchedulerUnitCount` parameter. The possible values are `day`, `days`, `friday`, `hour`, `hours`, `minute`, `minutes`, `monday`, `saturday`, `second`, `seconds`, `should_run`, `sunday`, `thursday`, `tuesday`, `wednesday`, `week`, `weeks`.                                                                                                                                                                                                                                                    |
| `curator.indicesCleanerSchedulerUnitCount`                 | string  | no        | 1                        | The count value for `curator.indicesCleanerSchedulerUnit` parameter. It can be number (`2`) of units specified in `curator.indicesCleanerSchedulerUnit` parameter or particular time (`18:57`) to execute script. For example, `18:51` and `day` in `curator.indicesCleanerSchedulerUnit` parameter mean that cleaner script will be executed every day at `18:51` for current time zone.                                                                                                                                                                                       |
| `curator.indicesCleanerConfigurationKey`                   | string  | no        | patterns_to_delete       | The key for YAML key-value pair in `opensearch-indices-cleaner-configuration` config map. The value for this pair should be list of `configuration items`. If you change this key in the config map, you should change it in the OpenSearch Curator deployment config.                                                                                                                                                                                                                                                                                                          |
| `curator.indicesCleanerConfiguration`                      | list    | no        | []                       | The list of YAML key-value pair configurations in `opensearch-indices-cleaner-configuration` config map. The parameter should represent a list of `configuration items`.                                                                                                                                                                                                                                                                                                                                                                                                        |
| `curator.resources.requests.cpu`                           | string  | no        | 200m                     | The minimum number of CPUs the curator and indices cleaner containers should use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `curator.resources.requests.memory`                        | string  | no        | 256Mi                    | The minimum amount of memory the curator and indices cleaner containers should use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `curator.resources.limits.cpu`                             | string  | no        | 200m                     | The maximum number of CPUs the curator and indices cleaner containers should use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `curator.resources.limits.memory`                          | string  | no        | 256Mi                    | The maximum amount of memory the curator and indices cleaner containers should use.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `curator.nodeSelector`                                     | object  | no        | {}                       | The selector that defines the nodes where the OpenSearch curator pods are scheduled on.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `curator.tolerations`                                      | list    | no        | []                       | The list of toleration policies for OpenSearch curator pod  in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `curator.affinity`                                         | object  | no        | {}                       | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `curator.securityContext`                                  | object  | no        | {}                       | The pod-level security attributes and common container settings for OpenSearch curator pod. For example, `fsGroup: 1000`.                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `curator.customLabels`                                     | object  | no        | {}                       | The custom labels for the OpenSearch curator pod.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `curator.priorityClassName`                                | string  | no        | ""                       | The priority class to be used by the OpenSearch Curator pods. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                                                                                                                                                                                                                                                      |

Where:
* `<name>` is the value of the `fullnameOverride` parameter.

## Status Provisioner

| Parameter                                     | Type    | Mandatory | Default value            | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
|-----------------------------------------------|---------|-----------|--------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `statusProvisioner.dockerImage`               | string  | no        | Calculates automatically | The image for Deployment Status Provisioner pod.                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `statusProvisioner.cleanupEnabled`            | boolean | no        |                          | Whether forced cleanup of previous Status Provisioner job is enabled. If the parameter is set to `false` and Kubernetes version is less than `1.21`, then the previous Status Provisioner job must be manually removed before deployment. If the parameter is not defined, then its value is calculated automatically according to the following rules: `false` if Kubernetes version is greater than or equal to `1.21`, `true` if Kubernetes version is less than `1.21`. |
| `statusProvisioner.lifetimeAfterCompletion`   | integer | no        | 600                      | The number of seconds that the job remains alive after its completion. This functionality works only since `1.21` Kubernetes version.                                                                                                                                                                                                                                                                                                                                       |
| `statusProvisioner.podReadinessTimeout`       | integer | no        | 800                      | The timeout in seconds that the job waits for the monitored resources to be ready or completed.                                                                                                                                                                                                                                                                                                                                                                             |
| `statusProvisioner.crProcessingTimeout`       | integer | no        | 600                      | The timeout in seconds that the job waits for each of the monitored custom resources to have `successful` or `failed` status.                                                                                                                                                                                                                                                                                                                                               |
| `statusProvisioner.integrationTestsTimeout`   | integer | no        | 300                      | The timeout in seconds that the job waits for the integration tests to complete.                                                                                                                                                                                                                                                                                                                                                                                            |
| `statusProvisioner.resources.requests.cpu`    | string  | no        | 50m                      | The minimum number of CPUs the container should use.                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `statusProvisioner.resources.requests.memory` | string  | no        | 50Mi                     | The minimum amount of memory the container should use.                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `statusProvisioner.resources.limits.cpu`      | string  | no        | 100m                     | The maximum number of CPUs the container should use.                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `statusProvisioner.resources.limits.memory`   | string  | no        | 100Mi                    | The maximum amount of memory the container should use.                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `statusProvisioner.customLabels`              | object  | no        | {}                       | The custom labels for the Status Provisioner pod.                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `statusProvisioner.securityContext`           | object  | no        | {}                       | The pod-level security attributes and common container settings for the Status Provisioner pod. The parameter is empty by default.                                                                                                                                                                                                                                                                                                                                          |

## Integration Tests

| Parameter                                        | Type    | Mandatory | Default value            | Description                                                                                                                                                                                                                                                                                                                                                |
|--------------------------------------------------|---------|-----------|--------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `integrationTests.enabled`                       | boolean | no        | false                    | Whether the installation of OpenSearch integration tests is to be enabled.                                                                                                                                                                                                                                                                                 |
| `integrationTests.dockerImage`                   | string  | no        | Calculates automatically | The docker image of OpenSearch integration tests.                                                                                                                                                                                                                                                                                                          |
| `integrationTests.statusWritingEnabled`          | boolean | no        | true                     | Whether the status of OpenSearch integration tests execution is to be written to deployment.                                                                                                                                                                                                                                                               |
| `integrationTests.isShortStatusMessage`          | boolean | no        | true                     | Whether the status message is to contain only first line of `result.txt` file. The parameter makes sense only if `statusWritingEnabled` parameter is set to `true`.                                                                                                                                                                                        |
| `integrationTests.secrets.idp.username`          | string  | no        | ""                       | The name of the user for Identity Provider. This parameter must be specified if you want to run integration tests with `authentication` tag.                                                                                                                                                                                                               |
| `integrationTests.secrets.idp.password`          | string  | no        | ""                       | The password of the user for Identity Provider. This parameter must be specified if you want to run integration tests with `authentication` tag.                                                                                                                                                                                                           |
| `integrationTests.secrets.idp.registrationToken` | string  | no        | ""                       | The registration token for Identity Provider. This parameter must be specified if you want to run integration tests with `authentication` tag.                                                                                                                                                                                                             |
| `integrationTests.secrets.prometheus.user`       | string  | no        | ""                       | The username for authentication on Prometheus/VictoriaMetrics secured endpoints.                                                                                                                                                                                                                                                                           |
| `integrationTests.secrets.prometheus.password`   | string  | no        | ""                       | The password for authentication on Prometheus/VictoriaMetrics secured endpoints.                                                                                                                                                                                                                                                                           |
| `integrationTests.tags`                          | string  | no        | smoke                    | The tags combined with `AND`, `OR` and `NOT` operators that select test cases to run. For more information about the available tags, refer to the [Tags Description](#tags-description) section.                                                                                                                                                           |
| `integrationTests.opensearchPort`                | string  | no        | 9200                     | The port of the OpenSearch.                                                                                                                                                                                                                                                                                                                                |
| `integrationTests.opensearchDbaasAdapterPort`    | string  | no        | 8080                     | The port of the DBaaS OpenSearch adapter. The allowed values are `8080` and `8443`. Use `8443` if TLS for Opensearch DBaaS Adapter is enabled.                                                                                                                                                                                                             |
| `integrationTests.identityProviderUrl`           | string  | no        | ""                       | The URL of Identity Provider. This parameter must be specified if you want to run integration tests with `authentication` tag.                                                                                                                                                                                                                             |
| `integrationTests.prometheusUrl`                 | string  | no        | ""                       | The URL (with schema and port) to Prometheus. For example, `http://prometheus.cloud.openshift.sdntest.example.com:80`. This parameter must be specified if you want to run integration tests with `prometheus` tag. **Note:** This parameter could be used as VictoriaMetrics URL instead of Prometheus. For example, `http://vmauth-k8s.monitoring:8427`. |
| `integrationTests.resources.requests.cpu`        | string  | no        | 200m                     | The minimum number of CPUs the container should use.                                                                                                                                                                                                                                                                                                       |
| `integrationTests.resources.requests.memory`     | string  | no        | 256Mi                    | The minimum amount of memory the container should use.                                                                                                                                                                                                                                                                                                     |
| `integrationTests.resources.limits.cpu`          | string  | no        | 400m                     | The maximum number of CPUs the container should use.                                                                                                                                                                                                                                                                                                       |
| `integrationTests.resources.limits.memory`       | string  | no        | 256Mi                    | The maximum amount of memory the container should use.                                                                                                                                                                                                                                                                                                     |
| `integrationTests.affinity`                      | object  | no        | {}                       | The affinity scheduling rules in `JSON` format.                                                                                                                                                                                                                                                                                                            |
| `integrationTests.customLabels`                  | object  | no        | {}                       | The custom labels for the OpenSearch integration tests pod.                                                                                                                                                                                                                                                                                                |
| `integrationTests.securityContext`               | object  | no        | {}                       | The pod-level security attributes and common container settings for the OpenSearch integration tests pod.                                                                                                                                                                                                                                                  |
| `integrationTests.priorityClassName`             | string  | no        | ""                       | The priority class to be used by the OpenSearch integration tests pods. You should create the priority class beforehand. For more information about this feature, refer to [https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/).                       |

### Tags Description

This section contains information about integration test tags that can be used in order to test OpenSearch service. You can use the following tags:

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
    * `find_backup` tag runs `Find Backup By Timestamp` test.
    * `granular_backup` tag runs `Granular Backup And Restore` and `Granular Backup And Restore By Timestamp` tests.
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

# Installation

## Before You Begin

* Make sure the environment corresponds the requirements in the [Prerequisites](#prerequisites) section.
* Make sure you review the [Upgrade](#upgrade) section.
* Before doing major upgrade, it is recommended to make a backup.
* Check if the application is already installed and find its previous deployments' parameters to make changes.

### App Deployer Preparation

1. Navigate to `CMDB` of your tenant's cloud and create namespace for the application.
2. Fill the application deployment parameters in the `YAML` format in `CMDB`.
   Example of deployment parameters is provided in the [On-Prem Examples](#on-prem-examples) section.
   **Important**: You should always specify `DEPLOY_W_HELM: true` and `ESCAPE_SEQUENCE: true` to correctly deploy the Helm release.
3. Navigate to the Application Deployer or Groovy Deployer job and specify the following data for build:
   * `PROJECT` is your cloud name and namespace name in format of {cloud}-{namespace}.
   * `ARTIFACT_DESCRIPTOR_VERSION` is the version of opensearch-service. It should be provided in the format, `opensearch-service:x.x.x_delivery_x.x.x_timestamp`. <!-- #GFCFilterMarkerStart# -->The all versions are available on [Release Page](https://git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/-/releases)<!-- #GFCFilterMarkerEnd# -->.
   * `DEPLOY_MODE` is the mode of the deployment procedure. It can be `Rolling Update` or `Clean Install`. The `Clean Install` mode removes everything from the namespace before the deployment, including Persistent Volumes Claims. Never use it for upgrades on production.
4. Run the installation.

### Ops Portal Preparation

Make sure all YAML values are escaped according to the Ops portal syntax.

### Helm

Pure Helm deployment does not support the automatic CRD upgrade procedure. In that case, you need to perform it manually.

## On-Prem Examples

### HA Scheme

The minimal template for HA scheme is as follows:

```yaml
dashboards:
  enabled: true
  ingress:
    enabled: true
    hosts:
      - host: dashboards-{namespace}.{url_to_kubernetes}
        paths:
          - path: /
opensearch:
  securityConfig:
    enabled: true
    authc:
      basic:
        username: admin
        password: admin
  securityContextCustom:
    fsGroup: 1000
  master:
    enabled: true
    replicas: 3
    persistence:
      enabled: true
      storageClass: {applicable_to_env_storage_class}
      size: 5Gi
    resources:
      limits:
        cpu: 500m
        memory: 2Gi
      requests:
        cpu: 200m
        memory: 2Gi
    javaOpts: "-Xms1024m -Xmx1024m"
  client:
    enabled: true
    ingress:
      enabled: true
      hosts:
        - opensearch-{namespace}.{url_to_kubernetes}
  snapshots:
    enabled: true
    repositoryName: "snapshots"
    storageClass: {applicable_to_env_rwx_storage_class}
    size: 5Gi
monitoring:
  enabled: true
dbaasAdapter:
  enabled: true
  dbaasUsername: dbaas-adapter
  dbaasPassword: dbaas-adapter
  registrationAuthUsername: {dbaas_aggregator_username}
  registrationAuthPassword: {dbaas_aggregator_password}
curator:
  enabled: true
  username: "admin"
  password: "admin"
  securityContext:
    runAsUser: 1000
integrationTests:
  enabled: false
DEPLOY_W_HELM: true
ESCAPE_SEQUENCE: true
```

### DR Scheme

Refer to the [OpenSearch Disaster Recovery](/docs/public/disaster-recovery.md) section in the _Cloud Platform Disaster Recovery Guide_.

## Google Cloud Examples

### HA Scheme

<details>
<summary>Click to expand YAML</summary>

```yaml
dashboards:
  enabled: true
  ingress:
    enabled: true
    hosts:
      - host: dashboards-{namespace}.{url_to_kubernetes}
        paths:
          - path: /
opensearch:
  securityConfig:
    enabled: true
    authc:
      basic:
        username: admin
        password: admin
  securityContextCustom:
    fsGroup: 1000
  master:
    enabled: true
    replicas: 3
    persistence:
      enabled: true
      storageClass: {applicable_to_env_storage_class}
      size: 5Gi
    resources:
      limits:
        cpu: 500m
        memory: 2Gi
      requests:
        cpu: 200m
        memory: 2Gi
    javaOpts: "-Xms1024m -Xmx1024m"
  client:
    enabled: true
    ingress:
      enabled: true
      hosts:
        - opensearch-{namespace}.{url_to_kubernetes}
  snapshots:
    enabled: true
    repositoryName: "snapshots"
    s3:
      enabled: true
      url: "https://storage.googleapis.com"
      bucket: {google_cloud_storage_bucket}
      gcs:
        secretName: {google_cloud_storage_secret_name}
        secretKey: {google_cloud_storage_secret_key}
monitoring:
  enabled: true
dbaasAdapter:
  enabled: true
  dbaasUsername: dbaas-adapter
  dbaasPassword: dbaas-adapter
  registrationAuthUsername: {dbaas_aggregator_username}
  registrationAuthPassword: {dbaas_aggregator_password}
curator:
  enabled: true
  username: "admin"
  password: "admin"
  securityContext:
    runAsUser: 1000
integrationTests:
  enabled: false
DEPLOY_W_HELM: true
ESCAPE_SEQUENCE: true
```

</details>

### DR Scheme

Refer to Google Kubernetes Engine in the [OpenSearch Disaster Recovery](/docs/public/disaster-recovery.md#google-kubernetes-engine-features) section in the _Cloud Platform Disaster Recovery Guide_.

## AWS Examples

### HA Scheme

Refer to the [Integration With Amazon OpenSearch](/docs/public/managed/amazon.md) section.

### DR Scheme

Not applicable

## Azure Examples

### HA Scheme

The same as [On-Prem Examples HA Scheme](#on-prem-examples).

### DR Scheme

The same as [On-Prem Examples DR Scheme](#on-prem-examples).

# Upgrade

## Common

In the common way, the upgrade procedure is the same as the initial deployment. You need to follow `Release Notes` and `Breaking Changes` in the version you install to find details. If you upgrade to a version which has several major diff changes from the installed version (e.g. `0.2.8` over `0.0.3`), you need to check `Release Notes` and `Breaking Changes` sections for `0.1.0` and `0.2.0` versions.

## Scale-In Cluster

OpenSearch does not support reducing the number of nodes without additional manipulations to move data replicas from nodes being removed, or understanding that there are enough data replicas on the remaining nodes, or data replicas can be moved to other nodes automatically without data loss.

## Rolling Upgrade

OpenSearch supports rolling upgrade feature with near-zero downtime.

### Operator rolling upgrade feature
According to [Rolling Upgrade](https://opensearch.org/docs/latest/install-and-configure/upgrade-opensearch/rolling-upgrade/) article in the OpenSearch documentation the cluster should be prepared before performing the rolling upgrade procedure.
The operator can perform the rolling upgrade on its own following the recommendations.

In order to enable that functionality set `opensearch.rollingUpdate` parameter to `true` and use default update strategies in master and data stateful sets.

If operator crashes or restarts while performing rolling upgrade procedure, upgrade will be continued after the operator is restored.

#### Algorithm description

##### Preparation
1. Operator checks that OpenSearch stateful set has `OnDelete` update strategy. Which OpenSearch stateful set will be used depends on installation mode.
2. Operator checks for at least one non-updated replica.
3. Operator checks that OpenSearch is healthy.

If all criteria are accepted then operator starts rolling upgrade procedure. Otherwise, the rolling upgrade procedure is skipped.

##### Rolling Upgrade procedure
1. Operator disables OpenSearch shard replication.
2. Operator sends request to OpenSearch to perform flush procedure.
3. Operator deletes non-updated OpenSearch pods one by one waiting for OpenSearch to become ready.
4. Operator enables OpenSearch shard replication.

## CRD Upgrade

Custom resource definition `OpenSearchService` should be upgraded before the installation if the new version has major changes.
<!-- #GFCFilterMarkerStart# -->
The CRD for this version is stored in [crd.yaml](/charts/helm/opensearch-service/crds/crd.yaml) and can be applied with the following command:

```
kubectl replace -f crd.yaml
```
<!-- #GFCFilterMarkerEnd# -->
It can be done automatically during the upgrade with [Automatic CRD Upgrade](#automatic-crd-upgrade) feature.

### Automatic CRD Upgrade

It is possible to upgrade CRD automatically on the environment to the latest one which is presented with the installing version. This feature is enabled by default if the `DISABLE_CRD` parameter is not `true`.

Automatic CRD upgrade requires the following cluster rights for the deployment user:

```yaml
  - apiGroups: [ "apiextensions.k8s.io" ]
    resources: [ "customresourcedefinitions" ]
    verbs: [ "get", "create", "patch" ]
```

## Migration

### Migration to OpenSearch 2.x (OpenSearch Service 1.x.x)

There are the following breaking changes:

1. The `type` parameter has been removed from all OpenSearch API endpoints. Instead, indices can be categorized by document type. For more details, see [Remove mapping types](https://github.com/opensearch-project/opensearch/issues/1940) issue.
2. The OpenSearch recommends TLS for REST layer if security is enabled. So, by default all layers (`transport`, `admin`, `rest`) are encrypted since 2.x version of OpenSearch. For more details, see [Security Admin](https://opensearch.org/docs/2.4/security/configuration/security-admin/#basic-usage) article in OpenSearch documentation and [Disabled TLS for REST is an unsupported configuration](https://github.com/opensearch-project/documentation-website/issues/913) issue.

**Important**: By default, TLS certificates for all layers (`transport`, `admin`, `rest`) are self-signed, so you will not be able to communicate with the OpenSearch without specifying corresponding certificate. For more details, refer to the [TLS Encryption](/docs/public/tls.md) section.

If you need migrate to OpenSearch Service `1.x.x` (with OpenSearch 2.x) from previous version there are the following rules:

**If TLS enabled:**
* No additional steps required, upgrade as is from any OpenSearch Service version.

**If TLS disabled:**
* Disable OpenSearch TLS on REST layer with property (`opensearch.tls.enabled: false`).
* Depending on the installed OpenSearch Service version:
  * if [0.2.4](https://git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/-/tags/0.2.4) (or newest) version installed just proceed with upgrade.
  * if version before `0.2.4` installed, you need previously upgrade to version `0.2.4` to migrate security configuration to new format and then install required `1.x.x` version.

**Important**: OpenSearch Service `1.x.x` requires `DEPLOY_W_HELM: true` for installation. If your current deployment was installed without Helm please follow [How to Deploy With DEPLOY_W_HELM True Over False](#how-to-deploy-with-deploy_w_helm-true-over-false).

### Migration From OpenDistro Elasticsearch

OpenSearch Service allows migration from OpenDistro Elasticsearch deployments.

There are 3 ways for migration:

1. Automatic via Deployer Job (DP|App)
2. Manual Steps
3. Backup and Restore

**Note**: If you need to migrate from Elasticsearch 6.8 cluster to OpenSearch, refer to [Migrate From Elasticsearch Service](#migrate-from-elasticsearch-68-service).

OpenSearch also can be deployed with the same name as OpenDistro Elasticsearch installation:

```
nameOverride: "elasticsearch"
fullnameOverride: "elasticsearch"
```

In this case no necessary to perform steps for the persistent volume migration because names of entities are the same. But this is not recommended way, because OpenSearch is the different solution.

**Note**: Refer to the general [Prerequisites](#prerequisites) and perform the necessary steps before the deployment.

#### Automatic Migration With Deployer Job

OpenSearch Deployer job can perform migration steps automatically if this feature is enabled in parameters. It is possible only for Helm-based installations (`DEPLOY_W_HELM` is `true` from previous and current deployments). To enable it you need to add the following properties to deployment params of your job (DP|App):

| Parameter                       | Description                                                                                           | Default           |
|---------------------------------|-------------------------------------------------------------------------------------------------------|-------------------|
| `ENABLE_MIGRATION`              | Whether to perform automatic migration during deployment. Disable it when migration finished.         | `false`           |
| `ENABLE_HELM_DELETE`            | Whether to delete previous Helm releases of OpenDistro Elasticsearch.                                 | `true`            |
| `ENABLE_PV_PATCH`               | Whether to perform patching for existing persistent volumes of OpenDistro Elasticsearch to save data. | `true`            |
| `PREVIOUS_CUSTOM_RESOURCE_NAME` | The previous name of OpenDistro Elasticsearch cluster.                                                | `"elasticsearch"` |
| `PREVIOUS_DEDICATED_ARBITER`    | Whether existing OpenDistro Elasticsearch cluster contains dedicated arbiter nodes.                   | `false`           |
| `PREVIOUS_DEDICATED_DATA`       | Whether existing OpenDistro Elasticsearch cluster contains dedicated data nodes.                      | `false`           |

**Important**: The option `ENABLE_PV_PATCH` requires cluster rights for editing persistent volumes by deployment user.

If you want to enable automatic migration, your user for deployment should have the following cluster role rights:

```yaml
  - apiGroups:
      - ""
    resources:
      - persistentvolumes
    verbs:
      - create
      - get
      - list
      - patch
      - watch
```

Basically it is necessary to add only `ENABLE_MIGRATION: true` to deploy parameters to enable migration feature.

You also need to specify your previous persistent volumes and storage class for parameters of OpenSearch `persistence` sections, and previous persistent volume claim for `snapshots` if you want to save previous data. For example:

```yaml
ENABLE_MIGRATION: true

opensearch:
  master:
    replicas: 3
    persistence:
      enabled: true
      storageClass: local-storage
      persistentVolumes:
        - pv-elasticsearch-1
        - pv-elasticsearch-2
        - pv-elasticsearch-3
  snapshots:
    enabled: true
    persistentVolumeClaim: pvc-elasticsearch-snapshots
```

**Important**: Do not use `Clean Install` mode for App Deployer Job. Use the `Rolling Update` mode for App Deployer and `auto` or `install` mode for DP Deployer.

**Note**: If something went wrong with automatic migration, the process will be interrupted, you need to perform migration procedure manually with the guide below.

#### Manual Migration Steps

The following steps should be performed from the host with installed `kubectl`, `helm` and `cluster-wide` rights to the cluster.

1. Uninstall existing OpenDistro Elasticsearch Helm release:

   For App Deployer installations:

    ```
    helm uninstall elasticsearch-service
    ```

   For DP Deployer installations:

    ```
    helm uninstall elasticsearch-service-{NAMESPACE}
    ```

   Or all resources if previous installation was not Helm-based:

    ```
    kubectl delete all --all -n {NAMESPACE}
    kubectl delete secret --all -n {NAMESPACE}
    kubectl delete configmap --all -n {NAMESPACE}
    ```

2. Patch existing persistent volumes of Elasticsearch data (not snapshot) to `Retain` policy:

   ```
   kubectl patch pv <your-pv-name> -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
   ```

   You can get the persistent volume names from the persistent volume claims via ```kubectl get pvc```.
3. Delete existing persistent volume claims of Elasticsearch data (not snapshot):

   ```
   kubectl delete pvc <your-pvc-name>
   ```

4. Patch existing persistent volumes of Elasticsearch data (not snapshot) to available status:

   ```
   kubectl patch pv <your-pv-name> -p '{"spec":{"claimRef": null}}'
   ```

5. Deploy OpenSearch service specifying the previous persistent volumes and storage class for OpenSearch persistent parameters and previous persistent volume claim for snapshot:

    ```yaml
    opensearch:
      master:
        replicas: 3
        persistence:
          enabled: true
          storageClass: local-storage
          persistentVolumes:
            - pv-elasticsearch-1
            - pv-elasticsearch-2
            - pv-elasticsearch-3
      snapshots:
        enabled: true
        persistentVolumeClaim: pvc-elasticsearch-snapshots
    ```

   Any deployment mode can be used.

#### Backup and Restore

With this approach snapshots collected on Elasticsearch side and restored on OpenSearch side. This migration also requires manual steps.

**Prerequisites**: Elasticsearch and OpenSearch clusters should be installed with enabled independent snapshots storages (Separate snapshot PVC with access from all Elasticsearch/OpenSearch pods)

1. Perform manual backup on Elasticsearch side:
   * From any Elasticsearch pod, run the backup procedure (all pods have access to snapshots, so it doesn't matter, what pod to choose here and below):

     ```bash
     curl -XPUT -u username:password "http://localhost:9200/_snapshot/${SNAPSHOTS_REPOSITORY_NAME:-snapshots}/${SNAPSHOT_NAME:-elasticsearch_snapshot}?wait_for_completion=true"
     ```

     With Elasticsearch `username:password` specified.

   * Check the snapshot status and indices from response.
     <!-- #GFCFilterMarkerStart# -->Additional information about manual backup described in [Manual backup guide](https://git.netcracker.com/PROD.Platform.ElasticStack/elasticsearch-service/-/blob/master/documentation/maintenance-guide/backup/manual-backup-procedure.md)<!-- #GFCFilterMarkerEnd# -->

2. Copy `snapshots` directory from Elasticsearch side:
   * Use `kubectl` with config on Elasticsearch cluster:

     ```bash
     kubectl cp elasticsearch-pod-name:/usr/share/elasticsearch/snapshots ./snapshots/ -n elasticsearch-namespace
     ```

     With `elasticsearch-pod-name` and `elasticsearch-namespace`. Snapshots will be copied to local path `./snapshots`.

3. Copy `snapshots` directory from local environment to OpenSearch cluster:
   * Use `kubectl` with config on OpenSearch cluster

     ```
     kubectl cp ./snapshots/ opensearch-pod-name:/usr/share/opensearch -n opensearch-namespace
     ```

     With `opensearch-pod-name` and `opensearch-namespace`. Snapshots will be copied from local path `./snapshots`.

4. Perform manual restore on OpenSearch side:
   * From any OpenSearch pod run restore procedure

     ```bash
     curl -XPOST -u username:password "http://localhost:9200/_snapshot/${SNAPSHOTS_REPOSITORY_NAME:-snapshots}/${SNAPSHOT_NAME:-elasticsearch_snapshot}/_restore"
     ```

     With OpenSearch `username:password` specified and `SNAPSHOT_NAME` the same, that was defined for backup on Elasticsearch side.

   * Check that response is `"accepted":true`. Otherwise, some problem occurred and described in response, such as already existing open index in new cluster, but if OpenSearch cluster has clean installation, no conflicts expected. If such problem reproduced, close or delete indices that already exists or use renaming pattern.
     <!-- #GFCFilterMarkerStart# -->Additional information about manual snapshot recovery described in [Manual recovery guide](https://git.netcracker.com/PROD.Platform.ElasticStack/elasticsearch-service/-/blob/master/documentation/maintenance-guide/recovery/manual-recovery-procedure.md)<!-- #GFCFilterMarkerEnd# -->

#### Migrate From Elasticsearch 6.8 Service

It is also possible to migrate from Elasticsearch 6.8 installations, but only with manual steps.

##### Migrate Elasticsearch 6.8 Persistent Volumes

1. Folder rights
   
   If previously Persistent Volumes `hostPath` folders were created with rights `100:101` it is necessary to change folders owner to `1000:1000`.

   If running as `root` user is allowed, you need to add the following deploy parameters:

    ```
    opensearch:
      securityContextCustom:
        runAsUser: 1000
        fsGroup: 1000
      fixMount:
        enabled: true
        securityContext:
          runAsUser: 0
    ```

   If running as `root` user is not allowed, you need to change folders owner manually:

    ```
    chown 1000:1000 -R /data/pv1
    ```

   The same for the snapshot persistent volume.

2. Naming and Persistent Volume Claims

   If persistent volume claims exist and have naming `pvc-opensearch-0(1,2)` then you can use them without specifying storage class or persistent volume names.

   If you use `hostPath` predefined persistent volumes, you need to specify `nodes` to assign pods:

    ```
    opensearch:
      master:
        persistence:
          nodes:
            - node-1
            - node-2
            - node-3
    ```

   If persistent volume claims have another naming you have to specify both `persistentVolumes` and `nodes` during deployment.

   If persistent volumes were created with `storageClass` you need to specify it without specifying `nodes`.

   Creating new persistent volume claims for existing utilized persistent volumes requires you to unbind persistent volumes from previous claims. You can do it with the following command:

    ```
    kubectl patch pv pv1 -p '{"spec":{"claimRef": null}}'
    ```

   Old persistent volume claims should be removed with removing deployments of previous installation.

##### Manual Migration From Elasticsearch 6.8

1. Perform steps from [Migrate Elasticsearch 6.8 Persistent Volumes](#migrate-elasticsearch-68-persistent-volumes).
2. Delete previous deployments

   Delete previous Helm release. For example:

    ```
    helm uninstall elasticsearch-service -n elasticsearch-cluster
    ```

   Delete previous non-Helm resources. For example:

    ```
    kubectl delete secret --all -n elasticsearch-cluster
   
    kubectl delete configmap --all -n elasticsearch-cluster
    ```

3. Install OpenSearch release.

### DBaaS Adapter Migration

There is no migration between the Elasticsearch DBaaS adapter and the OpenSearch DBaaS adapter, because they use different approaches for managing resources and different microservice clients.

If you want to upgrade OpenSearch service from `0.0.2` or lower version to `0.0.3` or higher version, you need to decide which DBaaS adapter you want to use. If you want to continue using the Elasticsearch DBaaS adapter, you have to update your installation parameters and replace the following part:

```
dbaasAdapter:
  enabled: true
  ...
```

with the following:

```
elasticsearchDbaasAdapter:
  enabled: true
  ...
```

If you want to use the OpenSearch DBaaS adapter, you have to manually adapt service with one of the following options:

* Edit `physical_database` table in Postgres that is used by DBaaS aggregator:

  * Find the record in the `physical_database` table that matches your `physical_database_identifier` with the following command:

    ```
    select * from physical_database where physical_database_identifier='<physical_database_identifier>';
    ```

  * Change the type of the found physical database from `elasticsearch` to `opensearch` with the following command:

    ```
    update physical_database set type = 'opensearch' where physical_database_identifier='<physical_database_identifier>';
    ```

  * Restart DBaaS aggregator.

* Add or update `dbaasAdapter.dbaasAggregatorPhysicalDatabaseIdentifier` parameter in OpenSearch service parameters during its installation. For example,

  ```
  dbaasAdapter:
    ...
    dbaasAggregatorPhysicalDatabaseIdentifier: "opensearch-dbaas-adapter"
  ```

  **Pay attention**, the value of this parameter should differ from the default value (the name of namespace where OpenSearch service is located).

## Rollback

OpenSearch does not support rollback with downgrade of a version. In this case, you need to do the following steps:

1. Deploy the previous version using the `Clean Install` mode of [App Deployer](#app-deployer-preparation).
2. Restore the data from backup.

# Additional Features

## Multiple Availability Zone Deployment

When deploying to a cluster with several availability zones, it is important that OpenSearch pods start in different availability zones.

### Affinity

You can manage pods' distribution using `affinity` rules to prevent Kubernetes from running OpenSearch pods on nodes of the same availability zone.

**Note**: This section describes deployment only for `storage class` persistent volumes (PV) type because with predefined PV, the OpenSearch pods are started on the nodes that are specified explicitly with persistent volumes. In that way, it is necessary to take care of creating PVs on nodes belonging to different availability zones in advance.

#### Replicas Fewer Than Availability Zones

For cases when the number of OpenSearch pods (value of the `opensearch.master.replicas` parameter) is equal to or less than the number of availability zones, you need to restrict the start of pods to one pod per availability zone. You can also specify additional node affinity rule to start pods on allowed Kubernetes nodes.

For this, you can use the following affinity rules:

<details>
<summary>Click to expand YAML</summary>

```yaml
opensearch:
  master:
    affinity: {
      "podAntiAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": [
          {
            "labelSelector": {
              "matchLabels": [
                "role": "master"
              ]
            },
            "topologyKey": "topology.kubernetes.io/zone"
          }
        ]
      },
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [
            {
              "matchExpressions": [
                {
                  "key": "role",
                  "operator": "In",
                  "values": [
                    "compute"
                  ]
                }
              ]
            }
          ]
        }
      }
    }
```

</details>

Where:

* `topology.kubernetes.io/zone` is the name of the label that defines the availability zone. This is the default name for Kubernetes 1.17+. Earlier, `failure-domain.beta.kubernetes.io/zone` was used.
* `role` and `compute` are the sample name and value of label that defines the region to run OpenSearch pods.

#### Replicas More Than Availability Zones

For cases when the number of OpenSearch pods (value of the `opensearch.master.replicas` parameter) is greater than the number of availability zones, you need to restrict the start of pods to one pod per node and specify the preferred rule to start on different availability zones. You can also specify an additional node affinity rule to start the pods on allowed Kubernetes nodes.

For this, you can use the following affinity rules:

<details>
<summary>Click to expand YAML</summary>

```yaml
opensearch:
  master:
    affinity: {
      "podAntiAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": [
          {
            "labelSelector": {
              "matchLabels": [
                "role": "master"
              ]
            },
            "topologyKey": "kubernetes.io/hostname"
          }
        ],
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 100,
            "podAffinityTerm": {
              "labelSelector": {
                "matchLabels": [
                  "role": "master"
                ]
              },
              "topologyKey": "topology.kubernetes.io/zone"
            }
          }
        ]
      },
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [
            {
              "matchExpressions": [
                {
                  "key": "role",
                  "operator": "In",
                  "values": [
                    "compute"
                  ]
                }
              ]
            }
          ]
        }
      }
    }
```

</details>

Where:

* `kubernetes.io/hostname` is the name of the label that defines the Kubernetes node. This is a standard name for Kubernetes.
* `topology.kubernetes.io/zone` is the name of the label that defines the availability zone. This is a standard name for Kubernetes 1.17+. Earlier, `failure-domain.beta.kubernetes.io/zone` was used.
* `role` and `compute` are the sample name and value of the label that defines the region to run OpenSearch pods.

# Frequently Asked Questions

## How to Deploy Using App Deployer Over an Installed DP Helm Deployer Version?

App Deployer does not support migration from DP Helm Deployer.

If you need it, you have to delete the current release with Helm command.

For example:

```bash
helm list --namespace=<namespace_name>

helm delete opensearch-service-<namespace_name> --namespace=<namespace_name>
```

Then update `release` annotations for config maps, secrets and persistent volume claims:

```bash
kubectl get configmap -n <namespace_name> -o json | jq '.items[].metadata|select(.annotations."meta.helm.sh/release-name")|.name' | awk '{print "kubectl annotate --overwrite configmap", $1, "meta.helm.sh/release-name=opensearch-service -n <namespace_name>"}' | bash -x
kubectl get secret -n <namespace_name> -o json | jq '.items[].metadata|select(.annotations."meta.helm.sh/release-name")|.name' | awk '{print "kubectl annotate --overwrite secret", $1, "meta.helm.sh/release-name=opensearch-service -n <namespace_name>"}' | bash -x
kubectl get pvc -n <namespace_name> -o json | jq '.items[].metadata|select(.annotations."meta.helm.sh/release-name")|.name' | awk '{print "kubectl annotate --overwrite pvc", $1, "meta.helm.sh/release-name=opensearch-service -n <namespace_name>"}' | bash -x
```

And finally install OpenSearch service using App Deployer.

## How to Deploy With DEPLOY_W_HELM True Over False?

App Deployer does not support migration from `DEPLOY_W_HELM: false` to `DEPLOY_W_HELM: true`.

If you need it, you have to delete all resources that belong to the current installation. For example,

```bash
kubectl delete all,secrets,configmaps,ingresses,serviceaccounts,roles,rolebindings,opensearchservice,grafanadashboard,prometheusrule,servicemonitor,certificates,sitemanager --all --namespace=<namespace_name>
```

If you use storage classes for OpenSearch persistent volume claims, you need to actualize annotations and labels only for `pvc-<name>-snapshots` persistent volume claim:

```bash
kubectl annotate --overwrite pvc pvc-<name>-snapshots meta.helm.sh/release-name=opensearch-service meta.helm.sh/release-namespace=<namespace_name> -n <namespace_name>
kubectl label --overwrite pvc pvc-<name>-snapshots app.kubernetes.io/managed-by=Helm -n <namespace_name>
```

If you use predefined persistent volumes for OpenSearch, you need to actualize annotations and labels for all persistent volume claims:

```bash
kubectl annotate --overwrite pvc --all meta.helm.sh/release-name=opensearch-service meta.helm.sh/release-namespace=<namespace_name> -n <namespace_name>
kubectl label --overwrite pvc --all app.kubernetes.io/managed-by=Helm -n <namespace_name>
```

Then install OpenSearch service using App Deployer and `DEPLOY_W_HELM: true`.

## What to Do if a Kubernetes Version Is Upgraded Before Application?

It is important to upgrade the application to a certified version until a Kubernetes upgrade.

If you already face the issue, you have to delete all Helm specific secrets (for example, `sh.helm.release.v1.opensearch-service-opensearch-service.v1`) from the namespace.

For example:

```bash
kubectl get secret -l "owner=helm"

kubectl delete secret -l "owner=helm"
```

Then install a new version with App Deployer and `DEPLOY_W_HELM: true`.

## Deployer Job Failed With Status Check but Application Works Fine

It can be an issue with timeouts or long start of OpenSearch pods. You need to get statuses from the `opensearch-status-provisioner` job resource and analyze them. You can do it in two ways:

* Look through `opensearch-status-provisioner` pod logs:

  ```
  Status Provisioner have started calculating the state of the cluster
  Processing [Deployment opensearch-service-operator] resource
  Processing [Deployment opensearch-dashboards] resource
  Processing [Deployment opensearch-curator] resource
  Processing [Deployment dbaas-opensearch-adapter] resource
  Processing [Deployment opensearch-monitoring] resource
  Processing [StatefulSet opensearch] resource
  Failed components statuses are []
  ```

* Get content of the `opensearch-status-provisioner` job resource:

  ```bash
  kubect get job opensearch-status-provisioner -o yaml
  ```

**Important**: By default, `opensearch-status-provisioner` job resource remains alive `600` seconds after its completion. You can increase this timeout by specifying `statusProvisioner.lifetimeAfterCompletion` parameter value.

You can also increase the pod readiness timeout `statusProvisioner.podReadinessTimeout: 1000` and try to run the Deployer job again.

## Deployer Job Failed With Unknown Fields in Opensearchservices.netcracker.com

It can be an issue with CRD changes. Refer to [CRD Upgrade](#crd-upgrade) for details.

## Deployer Job Failed With an Error in Templates

Make sure you performed the necessary [Prerequisites](#prerequisites). Fill the [Parameters](#parameters) correctly and compare with [Examples](#on-prem-examples).

## Deployer Job Fails With "Forbidden: updates to statefulset spec for fields..." Error

The following error in Deployer job means that you have changed parameters that can't be updated in `StatefulSet`.

```
Error: UPGRADE FAILED: cannot patch "opensearch" with kind StatefulSet: StatefulSet.apps "opensearch" is invalid: spec: Forbidden: updates to statefulset spec for fields other than 'replicas', 'template', 'updateStrategy', 'persistentVolumeClaimRetentionPolicy' and 'minReadySeconds' are forbidden
```

Most often it is associated with persistent volumes configuration. To determine the problem area, in OpenShift/Kubernetes find `StatefulSet` configuration using the following command:

```
kubectl describe statefulset <fullnameOverride> -n <namespace_name>
```

Pay attention to the `Volume Claims` section and compare its values to the persistence parameters (`opensearch.master.persistence.size`, `opensearch.master.persistence.storageClass`) specified in Deployer job.

```
Volume Claims:
  Name:          pvc
  StorageClass:  local-path
  Labels:        <none>
  Annotations:   <none>
  Capacity:      5Gi
  Access Modes:  [ReadWriteOnce]
```

There are two ways to solve the problem:

1. In Deployer job, use the same values specified in the `StatefulSet` configuration.
2. Remove the `StatefulSet` resource without deleting the pods:

   ```
   kubectl delete sts <fullnameOverride> -n <namespace_name> --cascade=orphan
   ```

   And run Deployer job with required parameters and `upgrade` mode.
