[[_TOC_]]

# OpenSearch Service

## Repository structure

* `./charts` - directory with main HELM chart for OpenSearch and integration tests.
* `./config` - directory with YAML resources for operator framework.
* `./controllers` - directory with operator's Golang source code which implements controller functionality.
* `./delivery` - directory with `description.yaml` and `build.sh` file for promotion process.
* `./dev-kit` - directory with scripts for working with Golang Operator Framework and Values JSON Schema.
* `./disasterrecovery` - directory with operator's Golang source code which implements disaster recovery functionality.
* `./docs` - directory with actual documentation for OpenSearch service component.
* `./documentation` - directory with legacy documentation, which is needed to redirect users to actual documentation.
* `./integration-tests` - directory with Robot Framework test cases for OpenSearch.
* `./migration-artifacts` - directory with pre-deploy script.
* `./opensearch-integration-tests` - directory with HELM Chart for integration-tests and documentation for it.
* `./tls-init` - directory with source code, Dockerfile and CI/CD config files for `tls-init` job.
* `./utils` - directory with operator's Golang source code which implements common functions.

## How to start

### Build

#### Local Build

1. Specify required version of opensource Qubership OpenSearch transfer version in `build.env`.
2. Go to the `/.gitlab-ci/build/opensearch` and run the command:

```bash
./build.sh
```

Merged Helm chart will be placed to `/.gitlab-ci/build/opensearch/helm` folder.

#### Operator Docker Image

1. Go to [DP-Builder](https://cisrvrecn.netcracker.com/job/DP.Pub.Microservice_builder_v2/)
2. Go to the "Build with parameters" tab.
3. Specify:

   * GitLab repository name - `PROD.Platform.ElasticStack/opensearch-service`.
   * Location - dev branch or `master`.
   * Prefix - leave it empty.
   
4. Click “Build” button.
5. Find your running build in the “Build History” tab in the DP-Builder page.
6. Wait for the job to finish.

#### Integration Tests Docker Image

1. Go to [DP-Builder](https://cisrvrecn.netcracker.com/job/DP.Pub.Microservice_builder_v2/)
2. Go to the "Build with parameters" tab.
3. Specify:

   * GitLab repository name - `PROD.Platform.ElasticStack/opensearch-service`.
   * Location - dev branch or `master`.
   * Prefix - `integration-tests`.
   
4. Click “Build” button.
5. Find your running build in the “Build History” tab in the DP-Builder page.
6. Wait for the job to finish.

#### Manifest

1. Actualize locations in `integration` section in `/charts/description.yaml` file.
   If one of the integrations should be taken not from master, then need to specify related branch in `location`.
   For example, if you made changes in operator code, commited them and build new Docker image, then you need to
   specify branch with your operator changes:

    ```yaml
    - type: find-latest-deployment-descriptor
      repo: PROD.Platform.ElasticStack/opensearch-service
      location: <YOUR_BRANCH_IN_REPO>
      docker-image-id: timestamp
      deploy-param: opensearchOperator
    ```

2. Go to [DP-Builder](https://cisrvrecn.netcracker.com/job/DP.Pub.Microservice_builder_v2/)
3. Go to the "Build with parameters" tab.
4. Specify:

   * GitLab repository name - `PROD.Platform.ElasticStack/opensearch-service`.
   * Location - dev branch or `master`.
   * Prefix - `/charts`.
   
5. Click “Build” button.
6. Find your running build in the “Build History” tab in the DP-Builder page.
7. Wait for the job to finish.
8. In finished build job find: `Services` section --> `independent` topic --> `operator` line.
9. Follow **the first link** in the `Artifact` column in the `operator` line.
10. Copy the `version` from the `Dependency Declaration` section.
11. Combine **the service name** and **the version** by the following way: `opensearch-service:<version>`.
12. The combination result is an **application manifest**.

#### Definition of Done

The changes might be marked as fully done if it accepts the following criteria:

1. The ticket's task done.
2. The solution is deployed to dev environment, where it can be tested.
3. Created merge request has:
   1. "Green" pipeline (linter, build, deploy & test jobs are passed).
   2. The title should follow the naming conversation: `<TMS-TICKET-ID>: <CHANGES-SHORT-DESCRIPTION>`.
   3. The description is **fully** filled.

### Deploy to k8s

#### Pure helm

1. Build operator and integration tests, if you need non-master versions. Go [Local Build](#local-build) for build Helm chart.
2. Prepare kubeconfig on you host machine to work with target cluster.
3. Prepare `sample.yaml` file with deployment parameters, which should contains custom docker images if it is needed.
4. Store `sample.yaml` file in `/.gitlab-ci/build/opensearch/helm` directory.
5. Go to `/.gitlab-ci/build/opensearch/helm` directory.
6. Run the following command:

  ```sh
  helm install opensearch-service ./ -f sample.yaml -n <TARGET_NAMESPACE>
  ```

#### Application deployer

1. Build a manifest using [Manifest building guide](#manifest).
2. Prepare Cloud Deploy Configuration:
   1. Go to the [APP-Deployer infrastructure configuration](https://cloud-deployer.netcracker.com/job/INFRA/job/groovy.deploy.v3/).
   2. INFRA --> Clouds -->  find your cloud --> Namespaces --> find your namespace.
   3. If the namespace is **not presented** then:
      1. Click `Create` button and specify: namespace and credentials. 
      2. Click `Save`.
      3. Go to your namespace configuration and specify the parameters for service installing.
   4. If the namespace is presented then: just check the parameters or change them.
3. To deploy service using APP-Deployer:
   1. Go to the [APP-Deployer groovy deploy page](https://cloud-deployer.netcracker.com/job/INFRA/job/groovy.deploy.v3/).
   2. Go to the `Build with Parameters` tab.
   3. Specify:
      1. Project: it is your cloud and namespace.
      2. The list is based on the information from the [APP-Deployer infrastructure configuration](https://cloud-deployer.netcracker.com/job/INFRA/job/groovy.deploy.v3/). 
      3. Deploy mode - `Rolling Update`. 
      4. Artifact descriptor version --> the **application manifest**.

#### ArgoCD

The information about ArgoCD deployment can be found in [Platform ArgoCD guide](https://bass.netcracker.com/display/PLAT/ArgoCD+guide).

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

#### APP-Deployer job typical errors

##### Application does not exist in the CMDB

The error like "ERROR: Application does not exist in the CMDB" means that the APP-Deployer doesn't have
configuration related to the "service-name" from application manifest.

**Solution**: check that the correct manifest is used.

##### CUSTOM_RESOURCE timeout

The error like "CUSTOM_RESOURCE timeout" means the service was deployed to the namespace, but the Custom Resource doesn't have SUCCESS status.
Usually, it is related to the service state - it might be unhealthy or repeatedly crushing.

**Solution**: there is no ready answer, need to go to namespace & check service state, operator logs to find a root cause and fix it.

## CI/CD

The main CI/CD pipeline is design to automize all basic developer routine start from code quality and finish with
deploying to stand k8s cluster.

1. `linter` - stage with jobs that run different linter to check code & documentation.
2. `build` - stage with jobs that build docker image for operator, integration-tests and tls-init job using DP-Builder.
3. `manifest` - stage with jobs that build manifest for current branch or release manifest using DP-Builder.
4. `deploy` - stage with job which deploys already build manifest to `ci-master` cluster using DP-Deployer.
5. `cloudDeploy` - optional stage with job which deploys the manifest to `miniha`, `k8s1/2` clusters using APP-Deployer.
6. `buildtests` - stage with job which builds manifest for integration tests using DP-Builder.
7. `deploytests` - stage with job which deploys integration-tests to `ci-master` cluster using DP-Deployer.
8. `manifestValidation` - optional stage with jobs that validate manifest (check is it ready to be released) and check
   vulnerabilities.

## Evergreen strategy

To keep the component up to date, the following activities should be performed regularly:

* Vulnerabilities fixing.
* OpenSearch upgrade.
* Bug-fixing, improvement and feature implementation for operator and other related supplementary services.

## Useful links

* [Installation guide](/docs/public/installation.md).
* [Troubleshooting guide](/docs/public/troubleshooting.md).
* [Internal Developer Guide](/docs/internal/developing.md).
* [Cloud INFRA Development process](https://bass.netcracker.com/display/PLAT/Cloud+Infra+Platform+services%3A+development+process).
* [ArgoCD User guide](https://bass.netcracker.com/display/PLAT/ArgoCD+guide)
