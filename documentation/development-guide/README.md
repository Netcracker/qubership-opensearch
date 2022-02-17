- [Introduction](#introduction)
- [Custom Resource Definition Versioning](#custom-resource-definition-versioning)
- [Build Custom Docker Image for Operator SDK](#build-custom-docker-image-for-operator-sdk)
- [Useful Operator Commands](#useful-operator-commands)
- [Useful Helm commands](#useful-helm-commands)

# Introduction

This guide provides the necessary information for the OpenSearch service development.

# Custom Resource Definition Versioning

Custom resource definition versioning allows to have different incompatible CRD versions of the OpenSearch
service in several namespaces of OpenShift/Kubernetes. Each OpenSearch service is placed in separate
namespace and controlled by corresponding operator. Each operator reconciles the only version of the application which
it is compatible with.

## Build Custom Docker Image for Operator SDK

To make it easier to prepare Operator SDK environment, you can create `docker` image that extends
`quay.io/operator-framework/operator-sdk:v1.16` image by installing `gcc` and `git` utilities:

```
FROM quay.io/operator-framework/operator-sdk:v1.16

RUN microdnf install --nodocs \
    gcc \
    git \
    && microdnf clean all

ENTRYPOINT ["/bin/bash"]
```

You can use this script to build the extended image:

```
#!/usr/bin/env bash

set -e

DOCKER_FILE=Dockerfile
IMAGE_NAME=operator-sdk-ext:v1.16

docker build \
  --pull \
  --file=${DOCKER_FILE} \
  -t ${IMAGE_NAME} \
  --no-cache \
  .

docker inspect ${IMAGE_NAME}
```

To run the environment you can use the following command:

```
docker run -it -v <path_to_operator>:/opensearch-service-operator operator-sdk-ext:v1.16
```

where `<path_to_operator>` is the full path to your `opensearch-service` directory.

## Create New Operator Version with API Changes

There are times when incompatible changes do not affect deployment, but most often such changes cause problems. The changes
which can be incompatible and should be monitored are described in [Backward compatibility gotchas](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#backward-compatibility-gotchas).
The most important of these are as following:

* Adding a new representation, since clients that only understood the old representation would not be aware of the new
  representation nor its semantics.
* Adding a new option to the set of fields, since it may not follow the appropriate conventions of the original object.
* Changing any validation rules, since it changes the assumptions about part of the API.

If API model has incompatible changes, new application version should be released. There are several steps to make it:

1. Add a new API by running the following command in the `root` directory:

    ```
    operator-sdk create api --version <new_version> --kind OpenSearchService --resource
    ```

   where `<new_version>` is the name of version that should appear. For example, `v2`.

   After the operation is completed, the package `api/<new_version>` has to be created.

2. Copy types structure from last active API version to
   `api/<new_version>/opensearchservice_types.go` file.

3. Add the marker `//+kubebuilder:storageversion` to the `opensearchservice_types.go` of new version and remove it for previous.

4. Change dependencies from the previous version to the new one in code of operator: `controllers` folder.
   So, in the new version reconciliation of old model is not supported.

5. Make necessary changes in API.

6. Update generated code using the following commands:

    ```
    make generate
    make manifests
    ```

   Make sure generated CRD `config/crd/bases/netcracker.com_opensearchservices.yaml` contains new version.

7. Copy new generated CRD to `charts/helm/opensearch-service/crds/crd.yaml` file.

8. Change `apiVersion` from old to new one in `charts/helm/opensearch-service/templates/operator/cr.yaml` file.

# Useful Operator Commands

_operator-sdk_:

* `operator-sdk create api --group netcracker.com --version <new_version> --kind=OpenSearchService --resource` is
  the command to add a new custom resource definition API called OpenSearchService, with APIVersion
  `netcracker.com/<new_version>` and Kind `OpenSearchService`.
* `operator-sdk create api --group netcracker.com --version <new_version> --kind=OpenSearchService --controller`
  is the command to add a new controller to the project that will watch and reconcile the OpenSearchService resource.
* `make generate` is the command to update the generated code for the OpenSearchService resource.
  You should run this command every time when you change `opensearchservice_types.go`.
* `make manifests` is the command to update the OpenAPI validation section in the custom resource definition.
  You should run this command every time when you change `opensearchservice_types.go`.

_minikube_:
* `minikube start --kubernetes-version=v1.11.10` is the command to start minikube with specific Kubernetes version.
* `minikube dashboard` is the command to run Kubernetes dashboard.

_kubectl_:

* `kubectl config set-context $(kubectl config current-context) --namespace=${NAMESPACE}` is the command to permanently save
  the namespace for all subsequent kubectl commands in that context.
* `kubectl get crd` is the command to get custom resource definitions.

# Useful Helm Commands

* `helm template charts/helm/opensearch-service/` is the command that renders templates.
* `helm install --dry-run <your-release-name> charts/helm/opensearch-service/` is the command that runs template rendering with connecting
  to Kubernetes/OpenShift, but does not deploy anything.
* `helm install --debug <your-release-name> charts/helm/opensearch-service/` is the command that prints additional logs which can help
  investigate issues with charts.
* `helm test charts/helm/opensearch-service/` is the command to run test suite which is stored in directory `charts/helm/opensearch-service/tests`.