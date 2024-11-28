This chapter describes the installation and configuration procedures of OpenSearch.

<!-- #GFCFilterMarkerStart# -->
The following topics are covered in this chapter:

[[_TOC_]]
<!-- #GFCFilterMarkerEnd# -->

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
   * `ARTIFACT_DESCRIPTOR_VERSION` is the version of opensearch-service.
     It should be provided in the format, `opensearch-service:x.x.x_delivery_x.x.x_timestamp`.
     <!-- #GFCFilterMarkerStart# -->The all versions are available on [Release Page](https://git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/-/releases)<!-- #GFCFilterMarkerEnd# -->.
   * `DEPLOY_MODE` is the mode of the deployment procedure.
     It can be `Rolling Update` or `Clean Install`. The `Clean Install` mode removes everything from the namespace before the deployment, including Persistent Volumes Claims.
     Never use it for upgrades on production.
4. Run the installation.

### Ops Portal Preparation

Make sure all YAML values are escaped according to the Ops portal syntax.