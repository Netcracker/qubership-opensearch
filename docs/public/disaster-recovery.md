This section provides information about Disaster Recovery in OpenSearch service.

The topics covered in this section are:

- [Common Information](#common-information)
- [Configuration](#configuration)
    - [Manual Steps Before Installation](#manual-steps-before-installation)
    - [Example](#example)
    - [Google Kubernetes Engine Features](#google-kubernetes-engine-features)
- [OpenSearch Cross Cluster Replication](#opensearch-cross-cluster-replication)
- [Switchover](#switchover)
- [REST API](#rest-api)

# Common Information

The Disaster Recovery scheme implies two separate OpenSearch clusters, one of which is in the *active* mode, and the other is in the *standby* mode.

![DR scheme](/docs/public/images/opensearch_dr_with_dbaas.png)

The Disaster Recovery process for the OpenSearch service includes the following:

* Turning on/off replication of OpenSearch indices with data: when switching to the standby mode, the replication is enabled, and when switching to the active mode, the replication is disabled.
* Users' recovery when switching to the active side if the DBaaS adapter is enabled.

# Configuration

The Disaster Recovery (DR) configuration requires two separate OpenSearch clusters installed on two Kubernetes/OpenShift clusters.
First, you need to configure the parameters for OpenSearch and all the components that you need to deploy to cloud.
Then, pay attention to the following steps that are significant for the Disaster Recovery configuration:

1. The parameter that enables the DR scheme has to be set. You need to decide in advance about the mode - active,
   standby or disabled - that is used during the current OpenSearch cluster installation and set the value in the following parameter.

    ```yaml
    global:
      ...
      disasterRecovery:
        mode: "active"
    ```

2. Do not forget to specify the OpenSearch DNS name in the `opensearch.tls.subjectAlternativeName.additionalDnsNames` parameter.
   For example, for `northamerica` region, the following parameter should be specified.

   ```yaml
   opensearch:
     tls:
       subjectAlternativeName:
         additionalDnsNames:
           - opensearch-northamerica.opensearch-service.svc.clusterset.local
   ```

    **Important**: In case you use certificates that weren't created by cert-manager, you have to specify both *active* and *standby* the OpenSearch DNS names in
    the `opensearch.tls.subjectAlternativeName.additionalDnsNames` parameter.
    For example, for OpenSearch `cluster-1` and `cluster-2`, the following parameters should be specified.

   ```yaml
   opensearch:
     tls:
       subjectAlternativeName:
         additionalDnsNames:
           - opensearch.opensearch-service.svc.cluster-1.local
           - opensearch.opensearch-service.svc.cluster-2.local
   ```

3. The parameter that describes services after which the OpenSearch service switchover has to be run is as follows.

    ```yaml
    global:
      ...
      disasterRecovery:
        afterServices: ["app", "postgres-service-site-manager"]
    ```

   In common case, the OpenSearch service is a base service and does not have any `after` services.
   But if you have the DBaaS adapter enabled, then specify where the DBaaS aggregator stores its data in `Postgres` parameter.

4. In the DR scheme, OpenSearch must be deployed with indicating OpenSearch service from the opposite side.
   For example, if you deploy a standby OpenSearch cluster, you should specify the path to OpenSearch from the active side to start the replication from the active OpenSearch to standby.
   Even if the active side is deployed, specify the path to the standby OpenSearch cluster to support the switch over process.
   You can also specify some regular expression as `indicesPattern`, which is used to look up active indices to replicate them.
   Specify the following parameters in the OpenSearch configuration:

   ```yaml
     global:
      ...
      disasterRecovery:
        indicesPattern: "test-*"
        remoteCluster: "opensearch:9300"
   ```

5. The DBaaS adapter should be installed only if the DBaaS aggregator is on the cloud.

## Manual Steps Before Installation

The OpenSearch cross cluster replication is allowed only for OpenSearch services from a union cluster. This means that both OpenSearch nodes must have the same admin, transport, and rest certificates.
Install the active service and perform one of the following actions:

* If `global.tls.generateCerts.certProvider` is set to "cert-manager", copy `opensearch-rest-issuer-certs`, `opensearch-admin-issuer-certs`,
  and `opensearch-transport-issuer-certs` Kubernetes secrets to the Kubernetes namespace for the standby service *before* its installation.
* If `global.tls.generateCerts.certProvider` is set to "dev", copy `opensearch-rest-certs`, `opensearch-admin-certs`,
  and `opensearch-transport-certs` Kubernetes secrets to the Kubernetes namespace for the standby service *before* its installation.

## Example

You want to install the OpenSearch service in the DR scheme. Each OpenSearch cluster located in the `opensearch-service` namespace has secured OpenSearch with 3 nodes.

The configuration for the active OpenSearch cluster is as follows:

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

dbaasAdapter:
  enabled: true
  dbaasUsername: "dbaas-adapter"
  dbaasPassword: "dbaas-adapter"
  registrationAuthUsername: "cluster-dba"
  registrationAuthPassword: "test"
```

Before the standby cluster installation, copy `opensearch-admin-certs` and `opensearch-transport-certs` Kubernetes secrets to the Kubernetes namespace for the standby service.
The configuration for the standby OpenSearch cluster is as follows:

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

dbaasAdapter:
  enabled: true
  dbaasUsername: "dbaas-adapter"
  dbaasPassword: "dbaas-adapter"
  registrationAuthUsername: "cluster-dba"
  registrationAuthPassword: "test"
```

**Note**: Clients cannot use OpenSearch on the standby side as the corresponding service is disabled.

## Google Kubernetes Engine Features

GKE provides its own multi-cluster services (MCS) mechanism of communications between clusters.

<!-- #GFCFilterMarkerStart# -->
For more details, refer to the [GKE-DR](https://git.qubership.org/PROD.Platform.HA/kubetools/-/blob/master/documentation/public/GKE-DR.md) document.
<!-- #GFCFilterMarkerEnd# -->

To deploy OpenSearch with enabled MCS support:

* OpenSearch should be deployed to namespaces with the same names for both clusters. MCS works only if the namespace is presented for both Kubernetes clusters.
* Enter the `global.disasterRecovery.mode` parameter to the necessary mode and set `global.disasterRecovery.serviceExport.enabled` to "true".
* Enter the `global.disasterRecovery.serviceExport.region` parameter to GKE (example, `us-central`).
  This means that there are different additional replication service names to access OpenSearch for both clusters.
  The name of the replication service is built as `{OPENSEARCH_NAME}-{REGION_NAME}`. For example, `opensearch-us-central`.
* Enter the `global.disasterRecovery.remoteCluster` parameter with the remote OpenSearch replication service address in the MCS `clusterset` domain.
  For example, `opensearch-us-central.opensearch-service.svc.clusterset.local`.

**Note**: OpenSearch requires an extended virtual memory for containers on the host machine.
It may be necessary that the command `sysctl -w vm.max_map_count=262144` should be performed on Kubernetes nodes before deploying OpenSearch.
The deployment procedure can perform this command automatically if the privileged containers are available in your cluster. To enable it, use the `opensearch.sysctl.enabled: true` parameter.

An example of the configuration for an active OpenSearch cluster in the `us-central` region is as follows:

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

dbaasAdapter:
  enabled: true
  dbaasUsername: "dbaas-adapter"
  dbaasPassword: "dbaas-adapter"
  registrationAuthUsername: "cluster-dba"
  registrationAuthPassword: "test"
```

**Note**: You should install an active service and perform one of the following actions:

* If `global.tls.generateCerts.certProvider` is set to "cert-manager", copy `opensearch-rest-issuer-certs`, `opensearch-admin-issuer-certs`,
  and `opensearch-transport-issuer-certs` Kubernetes secrets to the Kubernetes namespace for the standby service before its installation.
* If `global.tls.generateCerts.certProvider` is set to "dev", copy `opensearch-rest-certs`, `opensearch-admin-certs`,
  and `opensearch-transport-certs` Kubernetes secrets to the Kubernetes namespace for the standby service before its installation.
  Moreover, you should add the following parameters to the standby side configuration:

  ```yaml
  opensearch:
    tls:
      generateCerts:
        enabled: false
  ```

An example of the configuration for a standby OpenSearch cluster in the `northamerica` region is as follows:

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

dbaasAdapter:
  enabled: true
  dbaasUsername: "dbaas-adapter"
  dbaasPassword: "dbaas-adapter"
  registrationAuthUsername: "cluster-dba"
  registrationAuthPassword: "test"
```

**Note**: The MCS feature can work unstably. Sometimes it requires redeployment if the connectivity between clusters is not established.

# OpenSearch Cross Cluster Replication

# Switchover

You can perform a switchover using the `SiteManager` functionality or OpenSearch disaster recovery REST server API.

<!-- #GFCFilterMarkerStart# -->
For more information about SiteManager, refer to [Site Manager](https://github.com/Netcracker/DRNavigator/blob/main/documentation/public/architecture.md#site-manager) article.
<!-- #GFCFilterMarkerEnd# -->

If you want to perform a switchover manually, you need to switch the standby OpenSearch cluster to the active mode and then switch the active OpenSearch cluster to the standby mode.
Run the following command from within any OpenSearch pod on the standby side:

```bash
curl -XPOST -H "Content-Type: application/json" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager -d '{"mode":"active"}'
```

Then run the following command from within any OpenSearch pod on the active side:

```bash
curl -XPOST -H "Content-Type: application/json" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager -d '{"mode":"standby"}'
```

Where:

  * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
  * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.

All OpenSearch disaster recovery REST server endpoints can be secured through Kubernetes JWT Service Account Tokens.
To enable disaster recovery REST server authentication, the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be "true".

An example for a secured `sitemanager` GET endpoint is as follows:

```bash
curl -XGET -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
```

An example for a secured `sitemanager` POST endpoint is as follows:

```bash
curl -XPOST -H "Content-Type: application/json" -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
```

Where, `TOKEN` is the Site Manager Kubernetes JWT Service Account Token. The verification service account name and namespace are specified in
the `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

**Note**: If TLS for Disaster Recovery is enabled (`global.tls.enabled` and `global.disasterRecovery.tls.enabled` parameters are set to `true`),
use `https` protocol and `8443` port in API requests rather than `http` protocol and `8080` port.

For more information about OpenSearch disaster recovery REST server API, see [REST API](#rest-api).

# REST API

The OpenSearch disaster recovery REST server provides three methods of interaction:

* The `GET` `healthz` method allows finding out the state of the current OpenSearch cluster side.
  If the current OpenSearch cluster side is `active` or `disabled`, only the OpenSearch state is checked. You can run this method from within any OpenSearch pod as follows:

  ```bash
  curl -XGET http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/healthz
  ```

  Where:

    * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
    * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.
  
  All OpenSearch disaster recovery REST server endpoints can be secured through Kubernetes JWT Service Account Tokens.
  To enable disaster recovery REST server authentication, the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be "true".

  An example for a secured `healthz` endpoint is as follows:

  ```bash
  curl -XGET -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/healthz
  ```

  Where, `TOKEN` is the Site Manager Kubernetes JWT Service Account Token.
  The verification service account name and namespace are specified in the `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

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

* The `GET` `sitemanager` method allows finding out the mode of the current OpenSearch cluster side and the actual state of the switchover procedure.
  You can run this method from within any OpenSearch pod as follows:

  ```bash
  curl -XGET http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
  ```

  Where:

    * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
    * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.
  
  All OpenSearch disaster recovery REST server endpoints can be secured through Kubernetes JWT Service Account Tokens. To enable disaster recovery REST server authentication,
  the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be "true".

  An example for a secured `sitemanager` GET endpoint is as follows:

  ```bash
  curl -XGET -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
  ```

  Where, `TOKEN` is the Site Manager Kubernetes JWT Service Account Token. The verification service account name and namespace are specified in
  the `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

  The response to such a request is as follows:

  ```json
  {"mode":"standby","status":"done"}
  ```

  Where:

    * `mode` is the mode in which the OpenSearch cluster side is deployed. The possible mode values are as follows:
        * `active` - OpenSearch accepts external requests from clients.
        * `standby` - OpenSearch does not accept external requests from clients and replication from an active OpenSearch is enabled.
        * `disabled` - OpenSearch does not accept external requests from clients and replication from an active OpenSearch is disabled.
    * `status` is the current state of switchover for the OpenSearch cluster side. The three possible status values are as follows:
        * `running` - The switchover is in progress.
        * `done` - The switchover is successful.
        * `failed` - Something went wrong during the switchover.
    * `message` is the message that contains a detailed description of the problem and is only filled out if the `status` value is "failed".

* The `POST` `sitemanager` method allows switching mode for the current side of an OpenSearch cluster. You can run this method from within any OpenSearch pod as follows:

  ```bash
  curl -XPOST -H "Content-Type: application/json" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager -d '{"mode":"<MODE>"}'
  ```

  Where:

    * `<OPENSEARCH_NAME>` is the fullname of OpenSearch. For example, `opensearch`.
    * `<NAMESPACE>` is the OpenShift/Kubernetes project/namespace of the OpenSearch cluster side. For example, `opensearch-service`.
    * `<MODE>` is the mode to be applied to the OpenSearch cluster side. The possible mode values are as follows:
        * `active` - OpenSearch accepts external requests from clients.
        * `standby` - OpenSearch does not accept external requests from clients and replication from an active OpenSearch is enabled.
        * `disabled` - OpenSearch does not accept external requests from clients and replication from an active OpenSearch is disabled.

  The response to such a request is as follows:

  All OpenSearch disaster recovery REST server endpoints can be secured through Kubernetes JWT Service Account Tokens. To enable disaster recovery REST server authentication,
  the `global.disasterRecovery.httpAuth.enabled` deployment parameter must be "true".

  An example for a secured `sitemanager` POST endpoint is as follows:

  ```bash
  curl -XPOST -H "Content-Type: application/json" -H "Authorization: Bearer <TOKEN>" http://<OPENSEARCH_NAME>-disaster-recovery.<NAMESPACE>:8080/sitemanager
  ```

  Where, `TOKEN` is the Site Manager Kubernetes JWT Service Account Token. The verification service account name and namespace are specified in
  the `global.disasterRecovery.httpAuth.smServiceAccountName` and `global.disasterRecovery.httpAuth.smNamespace` deploy parameters.

  ```json
  {"mode":"standby"}
  ```

  Where:

    * `mode` is the mode that is applied to the OpenSearch cluster side. The possible values are "active", "standby" and "disabled".
    * `status` is the state of the request on the REST server. The only possible value is "failed", when something goes wrong while processing the request.
    * `message` is the message which contains a detailed description of the problem and is only filled out if the `status` value is "failed".

**Note**: If TLS for Disaster Recovery is enabled (`global.tls.enabled` and `global.disasterRecovery.tls.enabled` parameters are set to `true`), use `https` protocol and `8443` port in API requests
rather than `http` protocol and `8080` port.
