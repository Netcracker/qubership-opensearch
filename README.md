[[_TOC_]]

# Qubership OpenSearch

## Repository structure

* `./charts` - directory with main HELM chart for OpenSearch and integration tests.
* `./config` - directory with YAML resources for operator framework.
* `./controllers` - directory with operator's Golang source code which implements controller functionality.
* `./dev-kit` - directory with scripts for working with Golang Operator Framework.
* `./disasterrecovery` - directory with operator's Golang source code which implements disaster recovery functionality.
* `./docs` - directory with actual documentation for OpenSearch service component.
* `./integration-tests` - directory with Robot Framework test cases for OpenSearch.
* `./tls-init` - directory with source code, Dockerfile and CI/CD config files for `tls-init` job.
* `./utils` - directory with operator's Golang source code which implements common functions.

## How to start

### Deploy to k8s

#### Pure helm

1. Build operator and integration tests, if you need non-main versions.
2. Prepare kubeconfig on you host machine to work with target cluster.
3. Prepare `sample.yaml` file with deployment parameters, which should contains custom docker images if it is needed.
4. Store `sample.yaml` file in `/charts/helm/opensearch-service` directory.
5. Go to `/charts/helm/opensearch-service` directory.
6. Run the following command:

  ```sh
  helm install opensearch-service ./ -f sample.yaml -n <TARGET_NAMESPACE>
  ```

### Smoke tests

There is no smoke tests.

### How to debug

#### VSCode

To debug Operator in VSCode you can use `Launch operator` configuration which is already defined in 
`.vscode/launch.json` file.

The developer should configure environment variables: 

* `KUBECONFIG` - developer should **need to define** `KUBECONFIG` environment variable
  which should contains path to the kube-config file. It can be defined on configuration level
  or on the level of user's environment variables.
* `OPENSEARCH_USERNAME` - username for REST API access.
* `OPENSEARCH_PASSWORD` - password for REST API access.
* `OPENSEARCH_HOST` - OpenSearch's Ingress.
* `OPENSEARCH_NAME`, `WATCH_NAMESPACE`.

### How to troubleshoot

There are no well-defined rules for troubleshooting, as each task is unique, but there are some tips that can do:

* Deploy parameters.
* Application manifest.
* Logs from all OpenSearch service pods: operator, OpenSearch, monitoring and others.

Also, developer can take a look on [Troubleshooting guide](/docs/public/troubleshooting.md).

## Evergreen strategy

To keep the component up to date, the following activities should be performed regularly:

* Vulnerabilities fixing.
* OpenSearch upgrade.
* Bug-fixing, improvement and feature implementation for operator and other related supplementary services.

## Useful links

* [Installation guide](/docs/public/installation.md).
* [Troubleshooting guide](/docs/public/troubleshooting.md).
* [Internal Developer Guide](/docs/internal/developing.md).
