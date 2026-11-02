## HWE

The provided values do not guarantee that these values are correct for all cases.
It is a general recommendation. The resources should be calculated and estimated for each project case with a test load on the SVT stand, especially the HDD size.

The Amazon guide suggests starting a configuration with 2 vCPU cores and 8 GiB of memory for every 100 GiB of storage requirement.
For more information, refer to [https://docs.aws.amazon.com/opensearch-service/latest/developerguide/sizing-domains.html].

### Small

Recommended for development purposes, PoC, and demos. For 40 shards and 50Gb of indexed data.

| Module                      | CPU   | RAM, Gi | Storage, Gb |
|-----------------------------|-------|---------|-------------|
| OpenSearch (x3)             | 1     | 4       | 100         |
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
| **Total (Rounded)**         | **5** | **14**  | **350**     |

<details>
<summary>Click to expand YAML</summary>

```yaml
dashboards:
  resources:
    requests:
      cpu: 100m
      memory: 512M
    limits:
      cpu: 200m
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
  config:
    cluster.routing.allocation.node_concurrent_outgoing_recoveries: 2
    cluster.routing.allocation.node_concurrent_incoming_recoveries: 2
    indices.recovery.max_bytes_per_sec: "20mb"
  indexSettings:
    - pattern: "*"
      settings:
        index.translog.durability: "async"
        index.translog.sync_interval: "30s"
        index.translog.flush_threshold_size: "1gb"
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

Recommended for deployments with average load. For 80 shards and 100Gb of indexed data.

| Module                      | CPU   | RAM, Gi | Storage, Gb |
|-----------------------------|-------|---------|-------------|
| OpenSearch (x3)             | 2     | 8       | 200         |
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
| **Total (Rounded)**         | **8** | **26**  | **600**     |

<details>
<summary>Click to expand YAML</summary>

```yaml
dashboards:
  resources:
    requests:
      cpu: 100m
      memory: 512M
    limits:
      cpu: 200m
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
  config:
    opensearch.config.indices.memory.index_buffer_size: 20%
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

Recommended for deployments with high workload and a large amount of data. For 160 shards and 200Gb of indexed data.

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
      cpu: 200m
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
  config:
    opensearch.config.indices.memory.index_buffer_size: 30%
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

**NOTE:** the following scaling may include additional OpenSearch replicas (universal or data).
