# OpenSearch Helm Chart

## Prerequisites

Setting up Kubernetes and Helm and is outside the scope
of this README. Please refer to the Kubernetes and Helm documentation.

The versions required are:

  * **Helm 3+** - This is the earliest version of Helm tested. It is possible
    it works with earlier versions but this chart is untested for those versions.
  * **Kubernetes 1.29+** - This is the earliest version of Kubernetes tested.
    It is possible that this chart works with earlier versions but it is
    untested. 

## Usage

Assuming this repository was unpacked into the directory `opensearch-service` and Kubernetes config is available with namespace `opensearch`, the chart can
then be installed directly:
```bash
helm install opensearch-service ./ -f example.yaml -n opensearch
```
Please see the many options supported in the `values.yaml`
file.

## Snapshot S3 aliases

Use these values to configure named backup-storage aliases for snapshot/restore flows:

- `opensearch.snapshots.s3Aliases` - list of S3 alias definitions. If empty, no aliases secret is rendered.

Example:

```yaml
opensearch:
  snapshots:
    s3Aliases:
      - name: default
        spec:
          default: true
          storageBucket: backup-restore-bucket
          storageProvider: aws
          storageRegion: us-east-1
          storageServerUrl: "https://s3.example.com"
          storageUsername: name
          storageSecret: storage-location
        secretContent:
          storagePassword: pass  
```