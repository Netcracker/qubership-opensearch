The section provides information about Disaster Recovery in OpenSearch service.

The topics covered in this section are:

- [Common Information](#common-information)
- [Configuration](#configuration)
    - [Manual Steps Before Installation](#manual-steps-before-installation)
    - [Example](#example)
    - [Google Kubernetes Engine Features](#google-kubernetes-engine-features)
- [OpenSearch Cross Cluster Replication](#opensearch-cross-cluster-replication)
- [Switchover](#switchover)
- [REST API](#rest-api)
- [Troubleshooting Guide](../../maintenance-guide/troubleshooting-guide/README.md)

# Common Information

The Disaster Recovery scheme implies two separate OpenSearch clusters, one of which is in the `active` mode, and the other is in the `standby` mode.

# Configuration

The Disaster Recovery (DR) configuration requires two separate OpenSearch clusters installed on two Kubernetes/OpenShift clusters. First, you need to configure the parameters for OpenSearch and all the components that you need to deploy to cloud. Then, pay attention to the following steps that are significant for Disaster Recovery configuration:

1. The parameter that enables the DR scheme has to be set. You need to decide in advance about the mode, `active`, `standby` or `disabled` that is used during current OpenSearch cluster installation and set the value in the following parameter:

    ```
    global:
      ...
      disasterRecovery:
        mode: "active"
    ```

2. Don't forget to specify OpenSearch DNS name in `opensearch.tls.subjectAlternativeName.additionalDnsNames` parameter. For example, for `northamerica` region the following parameter should be specified:

   ```
   opensearch:
     tls:
       subjectAlternativeName:
         additionalDnsNames:
           - opensearch-northamerica.opensearch-service.svc.clusterset.local
   ```

3. The parameter that describes services after which OpenSearch service switchover has to be run.

    ```
    global:
      ...
      disasterRecovery:
        afterServices: ["app"]
    ```

   In common case OpenSearch service is a base service and does not have any `after` services.

4. In DR scheme, OpenSearch must be deployed with indicating OpenSearch service from opposite side. For example, if you deploy `standby` OpenSearch cluster you should specify path to the OpenSearch from `active` side to start replication from `active` OpenSearch to `standby`.
   Even if you deploy `active` side you need to specify path to `standby` OpenSearch cluster to support switch over process. You can also specify some regular expression as `indicesPattern` which is used for look up `active` indices to replicate them.
   You need to specify the following parameters in the OpenSearch configuration:

   ```
     global:
      ...
      disasterRecovery:
        indicesPattern: "test-*"
        remoteCluster: "opensearch:9300"
   ```

## Manual Steps Before Installation

The OpenSearch cross cluster replication is allowed only for OpenSearch services from union cluster. It means that both OpenSearch nodes must have the same `admin`, `transport` and `rest` certificates.
You should install `active` service and perform one of the following actions:
* If `global.tls.generateCerts.certProvider` is set to `cert-manager`, copy `opensearch-rest-issuer-certs`, `opensearch-admin-issuer-certs` and `opensearch-transport-issuer-certs` Kubernetes secrets to the Kubernetes namespace for `standby` service *before* its installation.
* If `global.tls.generateCerts.certProvider` is set to `dev`, copy `opensearch-rest-certs`, `opensearch-admin-certs` and `opensearch-transport-certs` Kubernetes secrets to the Kubernetes namespace for `standby` service *before* its installation.

## Example

You want to install OpenSearch service in Disaster Recovery scheme. Each OpenSearch cluster is located in `opensearch-service` namespace, has secured OpenSearch with 3 nodes.

The configuration for `active` OpenSearch cluster is as follows:

```yaml
global:
  disasterRecovery:
    mode: "active"
    remoteCluster: "opensearch.opensearch-service.svc.cluster-2.local:9300"

opensearch:
  securityConfig:
    authc:
      basic:
        username: "admin"
        password: "admin"
  securityContextCustom:
    fsGroup: 1000

  sysctl:
    enabled: true

  master:
    replicas: 3
    javaOpts: "-Xms718m -Xmx718m"
    persistence:
      storageClass: host-path
      size: 2Gi
    resources:
      limits:
        cpu: 500m
        memory: 1536Mi
      requests:
        cpu: 200m
        memory: 1536Mi

  client:
    ingress:
      enabled: true
      hosts:
        - opensearch-opensearch-service.kubernetes.docker.internal

monitoring:
  enabled: false
```

Before `standby` cluster installation you should copy `opensearch-admin-certs` and `opensearch-transport-certs` Kubernetes secrets to the Kubernetes namespace for `standby` service.
The configuration for `standby` OpenSearch cluster is as follows:

```yaml
global:
  disasterRecovery:
    mode: "standby"
    indicesPattern: "test-*"
    remoteCluster: "opensearch.opensearch-service.svc.cluster-1.local:9300"

opensearch:
  securityConfig:
    authc:
      basic:
        username: "admin"
        password: "admin"
  securityContextCustom:
    fsGroup: 1000

  sysctl:
    enabled: true

  master:
    replicas: 3
    javaOpts: "-Xms718m -Xmx718m"
    persistence:
      storageClass: host-path
      size: 2Gi
    resources:
      limits:
        cpu: 500m
        memory: 1536Mi
      requests:
        cpu: 200m
        memory: 1536Mi  

  client:
    ingress:
      enabled: true
      hosts:
        - opensearch-opensearch-service.kubernetes.docker.internal

monitoring:
  enabled: false
```

**NOTE:** Clients cannot use OpenSearch on `standby` side, corresponding service is disabled.

## Google Kubernetes Engine Features

GKE provides its own mechanism of communications between clusters - `multi-cluster services` (`MCS`), more detailed in [GKE-DR](https://git.netcracker.com/PROD.Platform.HA/kubetools/-/blob/master/documentation/public/GKE-DR.md).

To deploy OpenSearch with enabled MCS support you need to follow the points:

* OpenSearch service should be deployed to namespaces with the same names for both clusters. `MCS` works only if namespace is presented for both Kubernetes clusters.
* Fill parameters `global.disasterRecovery.mode` to necessary mode and `global.disasterRecovery.serviceExport.enabled` to `true`.
* Fill parameter `global.disasterRecovery.serviceExport.region` to GKE (e.g. `us-central`). It means you will have different additional replication service names to access OpenSearch for both clusters. The name of replication service is build as `{OPENSEARCH_NAME}-{REGION_NAME}`, e.g. `opensearch-us-central`.
* Fill parameter `global.disasterRecovery.remoteCluster` with remote OpenSearch replication service address in `MCS` domain `clusterset`, e.g. `opensearch-us-central.opensearch-service.svc.clusterset.local`.

**NOTE:** OpenSearch requires an extended virtual memory for containers on host machine. It **may** be necessary that the command `sysctl -w vm.max_map_count=262144` should be performed on Kubernetes nodes before deploy OpenSearch.
Deployment procedure can perform this command automatically if the privileged containers are available in your cluster, to enable it you need to use parameter `opensearch.sysctl.enabled: true`.

The example of configuration for `active` OpenSearch cluster in `us-central` region is as follows:

```yaml
global:
  disasterRecovery:
    mode: "active"
    remoteCluster: "opensearch-northamerica.opensearch-service.svc.clusterset.local:9300"
    serviceExport:
      enabled: true
      region: "us-central"

opensearch:
  securityConfig:
    authc:
      basic:
        username: "admin"
        password: "admin"
  securityContextCustom:
    fsGroup: 1000

  sysctl:
    enabled: true

  master:
    replicas: 3
    javaOpts: "-Xms718m -Xmx718m"
    persistence:
      storageClass: host-path
      size: 2Gi
    resources:
      limits:
        cpu: 500m
        memory: 1536Mi
      requests:
        cpu: 200m
        memory: 1536Mi        

  client:
    ingress:
      enabled: true
      hosts:
        - opensearch-opensearch-service.gke.example.us-central.com
    resources:
      limits:
        cpu: 1
        memory: 1024Mi
      requests:
        cpu: 200m
        memory: 1024Mi
```

**NOTE:** You should install `active` service and perform one of the following actions:
* If `global.tls.generateCerts.certProvider` is set to `cert-manager`, copy `opensearch-rest-issuer-certs`, `opensearch-admin-issuer-certs` and `opensearch-transport-issuer-certs` Kubernetes secrets to the Kubernetes namespace for `standby` service *before* its installation.
* If `global.tls.generateCerts.certProvider` is set to `dev`, copy `opensearch-rest-certs`, `opensearch-admin-certs` and `opensearch-transport-certs` Kubernetes secrets to the Kubernetes namespace for `standby` service *before* its installation. Moreover, you should add the following parameters to `standby` side configuration:

  ```yaml
  opensearch:
    tls:
      generateCerts:
        enabled: false
  ```

The example of configuration for `standby` OpenSearch cluster in `northamerica` region is as follows:

```yaml
global:
  disasterRecovery:
    mode: "standby"
    remoteCluster: "opensearch-us-central.opensearch-service.svc.clusterset.local:9300"
    serviceExport:
      enabled: true
      region: "northamerica"

opensearch:
  securityConfig:
    authc:
      basic:
        username: "admin"
        password: "admin"
  securityContextCustom:
    fsGroup: 1000

  sysctl:
    enabled: true

  master:
    replicas: 3
    javaOpts: "-Xms718m -Xmx718m"
    persistence:
      storageClass: host-path
      size: 2Gi
    resources:
      limits:
        cpu: 500m
        memory: 1536Mi
      requests:
        cpu: 200m
        memory: 1536Mi  

  client:
    ingress:
      enabled: true
      hosts:
        - opensearch-opensearch-service.gke.example.northamerica.com
    resources:
      limits:
        cpu: 1
        memory: 1024Mi
      requests:
        cpu: 200m
        memory: 1024Mi
```

**NOTE:** `MCS` feature can work unstable, sometimes it requires redeployment if connectivity between clusters is not established.

# OpenSearch Cross Cluster Replication

# Switchover

You can perform the switchover using the `SiteManager` functionality or OpenSearch disaster recovery REST server API.

<!-- #GFCFilterMarkerStart# -->
For more information about `SiteManager`, refer to [Site Manager](https://git.netcracker.com/PROD.Platform.HA/site-manager/-/blob/master/documentation/Architecture.md) article.
<!-- #GFCFilterMarkerEnd# -->

If you want to perform a switchover manually, you need to switch `standby` OpenSearch cluster to `active` mode and then switch `active` OpenSearch cluster to `standby` mode. You need to run the following command from within any OpenSearch pod on the `standby` side:

```
curl -XPOST -H "Content-Type: application/json" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager -d '{"mode":"active"}'
```

Then you should run the following command from within any OpenSearch pod on the `active` side:

```
curl -XPOST -H "Content-Type: application/json" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager -d '{"mode":"standby"}'
```

Where:
  * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
  * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.

All OpenSearch disaster recovery REST server endpoints can be secured via Kubernetes JWT Service Account Tokens. To enable disaster recovery REST server authentication the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be `true`.
The example for secured `sitemanager` GET endpoint is following:

```
curl -XGET -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
```

The example for secured `sitemanager` POST endpoint is following:

```
curl -XPOST -H "Content-Type: application/json, Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
```

Where `TOKEN` is Site Manager Kubernetes JWT Service Account Token. The verification service account name and namespace are specified in `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

**NOTE:** If TLS for Disaster Recovery is enabled (`global.tls.enabled` and `global.disasterRecovery.tls.enabled` parameters are set to `true`), use `https` protocol and `8443` port in API requests rather than `http` protocol and `8080` port.

For more information about OpenSearch disaster recovery REST server API, see [REST API](#rest-api).

# REST API

OpenSearch disaster recovery REST server provides three methods of interaction:

* `GET` `healthz` method allows finding out the state of the current OpenSearch cluster side. If the current OpenSearch cluster side is `active` or `disabled`, only OpenSearch state is checked. You can run this method from within any OpenSearch pod as follows:

  ```
  curl -XGET http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/healthz
  ```

  Where:
    * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
    * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.
  
  All OpenSearch disaster recovery REST server endpoints can be secured via Kubernetes JWT Service Account Tokens. To enable disaster recovery REST server authentication the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be `true`.
  The example for secured `healthz` endpoint is following:

  ```
  curl -XGET -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/healthz
  ```

  Where `TOKEN` is Site Manager Kubernetes JWT Service Account Token. The verification service account name and namespace are specified in `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

  The response to such a request is as follows:

  ```json
  {"status":"up"}
  ```

  Where:
    * `status` is the current state of the OpenSearch cluster side. The four possible status values are as follows:
        * `up` - All OpenSearch stateful sets are ready.
        * `degraded` - Some of OpenSearch stateful sets are not ready.
        * `down` - All OpenSearch stateful sets are not ready.
        * `disabled` - The OpenSearch service is switched off.

* `GET` `sitemanager` method allows finding out the mode of the current OpenSearch cluster side and the actual state of the switchover procedure. You can run this method from within any OpenSearch pod as follows:

  ```
  curl -XGET http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
  ```

  Where:
    * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
    * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.
  
  All OpenSearch disaster recovery REST server endpoints can be secured via Kubernetes JWT Service Account Tokens. To enable disaster recovery REST server authentication the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be `true`.
  The example for secured `sitemanager` GET endpoint is following:

  ```
  curl -XGET -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
  ```

  Where `TOKEN` is Site Manager Kubernetes JWT Service Account Token. The verification service account name and namespace are specified in `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

  The response to such a request is as follows:

  ```json
  {"mode":"standby","status":"done"}
  ```

  Where:
    * `mode` is the mode in which the OpenSearch cluster side is deployed. The possible mode values are as follows:
        * `active` - OpenSearch accepts external requests from clients.
        * `standby` - OpenSearch does not accept external requests from clients and replication from `active` OpenSearch is enabled.
        * `disabled` - OpenSearch does not accept external requests from clients and replication from `active` OpenSearch is disabled.
    * `status` is the current state of switchover for the OpenSearch cluster side. The three possible status values are as follows:
        * `running` - The switchover is in progress.
        * `done` - The switchover is successful.
        * `failed` - Something went wrong during the switchover.
    * `comment` is the message which contains a detailed description of the problem and is only filled out if the `status` value is `failed`.

* `POST` `sitemanager` method allows switching mode for the current side of OpenSearch cluster. You can run this method from within any OpenSearch pod as follows:

  ```
  curl -XPOST -H "Content-Type: application/json" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager -d '{"mode":"<MODE>"}'
  ```

  Where:
    * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
    * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.
    * `<MODE>` is the mode to be applied to the OpenSearch cluster side. The possible mode values are as follows:
        * `active` - OpenSearch accepts external requests from clients.
        * `standby` - OpenSearch does not accept external requests from clients and replication from `active` OpenSearch is enabled.
        * `disabled` - OpenSearch does not accept external requests from clients and replication from `active` OpenSearch is disabled.

  The response to such a request is as follows:

  All OpenSearch disaster recovery REST server endpoints can be secured via Kubernetes JWT Service Account Tokens. To enable disaster recovery REST server authentication the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be `true`.
  The example for secured `sitemanager` POST endpoint is following:

  ```
  curl -XPOST -H "Content-Type: application/json, Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
  ```

  Where `TOKEN` is Site Manager Kubernetes JWT Service Account Token. The verification service account name and namespace are specified in `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

  ```json
  {"mode":"standby"}
  ```

  Where:
    * `mode` is the mode which is applied to the OpenSearch cluster side. The possible values are `active`, `standby` and `disabled`.
    * `status` is the state of the request on the REST server. The only possible value is `failed`, when something goes wrong while processing the request.
    * `comment` is the message which contains a detailed description of the problem and is only filled out if the `status` value is `failed`.

**NOTE:** If TLS for Disaster Recovery is enabled (`global.tls.enabled` and `global.disasterRecovery.tls.enabled` parameters are set to `true`), use `https` protocol and `8443` port in API requests rather than `http` protocol and `8080` port.