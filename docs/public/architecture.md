This chapter describes the architectural features of OpenSearch.
<!-- #GFCFilterMarkerStart# -->
<!-- TOC -->
* [Overview](#overview)
  * [Qubership OpenSearch Delivery and Features](#qubership-opensearch-delivery-and-features)
* [OpenSearch Components](#opensearch-components)
  * [OpenSearch Operator](#opensearch-operator)
  * [OpenSearch](#opensearch)
  * [OpenSearch Curator](#opensearch-curator)
  * [OpenSearch Monitoring](#opensearch-monitoring)
  * [DBaaS Adapter](#dbaas-adapter)
  * [OpenSearch Dashboard](#opensearch-dashboard)
  * [Pod Scheduler](#pod-scheduler)
  * [Status Provisioner](#status-provisioner)
  * [Pre-Install Jobs](#pre-install-jobs)
* [Supported Deployment Schemes](#supported-deployment-schemes)
  * [On-Prem](#on-prem)
    * [HA Joint Deployment Scheme](#ha-joint-deployment-scheme)
    * [Non-HA Deployment Scheme](#non-ha-deployment-scheme)
    * [HA Separated Deployment Scheme](#ha-separated-deployment-scheme)
    * [DR Deployment Scheme](#dr-deployment-scheme)
  * [Integration with Managed Services](#integration-with-managed-services)
    * [Google Cloud](#google-cloud)
    * [AWS OpenSearch](#aws-opensearch)
    * [Azure](#azure)
<!-- TOC -->
<!-- #GFCFilterMarkerEnd# -->
# Overview

OpenSearch is a powerful open-source and fully free search and analytics engine that serves as a viable alternative to proprietary solutions like Elasticsearch.
It is necessary because it enables businesses to implement efficient search functionality within their applications, websites, or data analysis platforms.
OpenSearch offers advanced search capabilities, including full-text search, faceted search, and real-time indexing, empowering businesses to deliver fast and accurate search results to their users.

The business value of OpenSearch lies in its ability to enhance the user experience, increase customer engagement, and drive better decision-making.
By integrating OpenSearch, products can improve the search relevance, enabling users to find the desired information quickly and easily.
This, in turn, leads to improved customer satisfaction, increased conversions, and enhanced retention rates.

In summary, OpenSearch is necessary for businesses as it empowers them to deliver an efficient search functionality, improve the user experience, make informed decisions through analytics, and
realize cost advantages compared to proprietary alternatives.

## Qubership OpenSearch Delivery and Features

The Qubership platform provides OpenSearch deployment to Kubernetes/OpenShift using helm chart based on community OpenSearch Helm chart with its own operator and additional features.
The deployment procedure and additional features include the following:

* Support of Qubership deployment jobs for HA scheme and different configurations. For more information, refer to the [Installation of OpenSearch](/docs/public/installation.md) section.
* Backup and restore data. <!-- #GFCFilterMarkerStart# -->For more information,
  refer to [OpenSearch Curator Guide](https://github.com/Netcracker/docker-elastic-curator/blob/main/README.md).<!-- #GFCFilterMarkerEnd# -->
* Monitoring integration with Grafana Dashboard and Prometheus Alerts. For more information,
  refer to the [OpenSearch Service Monitoring](/docs/public/monitoring.md) section in the _Cloud Platform Monitoring Guide_.
* User Interface (UI) provided by OpenSearch dashboards.
* DBaaS Adapter for OpenSearch integration. <!-- #GFCFilterMarkerStart# -->For more information,
  refer to [DBaaS Adapter Guide](https://github.com/Netcracker/dbaas-opensearch-adapter/blob/main/README.md).<!-- #GFCFilterMarkerEnd# -->
* Disaster Recovery scheme with data replication. For more information,
  refer to the [OpenSearch Disaster Recovery](/docs/public/disaster-recovery.md) section in the _Cloud Platform Disaster Recovery Guide_.

# OpenSearch Components

The following image illustrates the components of OpenSearch.

![Application Overview](/docs/public/images/opensearch_components_overview.drawio.png)

## OpenSearch Operator

The OpenSearch Operator is a microservice designed specifically for Kubernetes environments.
It simplifies the management of OpenSearch clusters, which are critical for distributed coordination.
With the OpenSearch Operator, administrators can easily maintain their OpenSearch security configurations, focusing on other important aspects of their applications without worrying about
the intricacies of cluster management.

In addition, the OpenSearch operator performs disaster recovery logic and orchestrates OpenSearch switchover and failover operations.

## OpenSearch

OpenSearch is custom distribution of original OpenSearch adapted for the cloud environment, offering additional features and tools for enhanced functionality and management.
It incorporates logging capabilities to capture and analyze important system events.
Additionally, it includes health check functionalities and other tools, streamlining the monitoring and maintenance of OpenSearch clusters.
With this enhanced Docker container, users can effortlessly deploy and manage robust and secure OpenSearch environments with comprehensive tooling support.

## OpenSearch Curator

The OpenSearch Curator is a microservice that offers a convenient REST API for performing backups and restores of OpenSearch indices, templates, and aliases using the OpenSearch Snapshot API.
It enables users to initiate backups and restores programmatically, making it easier to automate these processes.
Additionally, the daemon allows users to schedule regular backups, ensuring data protection and disaster recovery.
Furthermore, it offers the capability to store backups on remote S3 storage, providing a secure and scalable solution for long-term data retention.

One more feature of the OpenSearch Curator is cleaning old indices by a specified configuration pattern and schedule.
It provides significant business value by optimizing storage utilization, improving search performance, and reducing operational costs associated with managing and maintaining large volumes
of outdated data.

## OpenSearch Monitoring

The OpenSearch Monitoring microservice is built on the Telegraf framework, specializing in collecting and analyzing metrics from OpenSearch.
It seamlessly integrates with OpenSearch clusters, capturing essential data points for performance monitoring and analysis.
Additionally, the microservice provides a Grafana dashboard, offering a comprehensive visualization of OpenSearch metrics for better insights and diagnostics.
It also includes an alerting system to promptly notify administrators of any potential issues or anomalies detected within the OpenSearch environment.
With OpenSearch Monitoring, users can effectively monitor the health and performance of their OpenSearch clusters, enabling proactive management and maintenance.

## DBaaS Adapter

The DBaaS Adapter microservice implements the Database-as-a-Service (DBaaS) approach for OpenSearch. In terms of DBaaS OpenSearch, the database is a logical scope of entities
(indices, aliases and templates) with the same prefix name.
When users create a database, the DBaaS Adapter generates a prefix and creates a user with permissisons for all entities of the generated prefix name.

## OpenSearch Dashboard

The OpenSearch Dashboard is a user interface (UI) tool designed for OpenSearch by Amazon, providing a user-friendly platform to manage OpenSearch indices and perform read and write operations.
With AKHQ, users can easily navigate and monitor OpenSearch indices, manage security, and monitor performance.
It simplifies OpenSearch administration tasks, enabling efficient management of OpenSearch clusters through a comprehensive and intuitive interface.

## Pod Scheduler

The Pod Scheduler service, running as a pod in Kubernetes, is responsible for binding specific pods of the OpenSearch StatefulSet to designated Kubernetes nodes based on configuration.
This scheduler ensures that pods requiring host-path persistent volumes are assigned to the appropriate nodes, aligning with the defined configuration.
By orchestrating this allocation process, the scheduler optimizes resource utilization and enables efficient utilization of host-path persistent volumes within the Kubernetes cluster.

## Status Provisioner

The Status Provisioner service is designed to monitor the health of all OpenSearch components and relay their status information in the App | DP Deployer contract.
It checks the availability and functionality of various OpenSearch components after deployment that they are functioning properly.
By providing this status information in the Deployer contract, the Status Provisioner service enables seamless integration with other systems that rely on the health and operational status of
OpenSearch components.
After providing the status, the Status Provisioner pod is auto-removed.

## Pre-Install Jobs

The set of pre-install hooks that allow to prepare the environment for OpenSearch installation. It includes:

* TLS Init Job - generate self-signed certificates for OpenSearch modules.

# Supported Deployment Schemes

## On-Prem

### HA Joint Deployment Scheme

The following image shows the OpenSearch HA deployment scheme.

![HA Scheme](/docs/public/images/opensearch_on_prem_deploy.drawio.png)

Following the above picture, let us describe the main parts of the OpenSearch K8s deployment:

* The minimal number of replicas for HA scheme of OpenSearch is 3.
* OpenSearch pods are distributed through Kubernetes nodes and availability zones based on the affinity policy during deployment.
* Each OpenSearch pod has its own Persistent Volume storage.
* In addition to the OpenSearch main storage, the OpenSearch Backup Daemon pod has its own Persistent Volume for backups.
* The OpenSearch Monitoring pod is deployed near the OpenSearch cluster and collects the corresponding metrics.
* The OpenSearch Dashboard pod is deployed with Ingress to access the UI.
* The DBaaS Adapter pod communicates with the DBaaS Aggregator and performs the corresponding operations in OpenSearch.
* All components are deployed by Helm.

### Non-HA Deployment Scheme

For a non-HA deployment scheme, it is possible to use one pod of the OpenSearch cluster.

### HA Separated Deployment Scheme

The following image shows the OpenSearch HA deployment sheme in the separated mode.

![HA Separated Scheme](/docs/public/images/opensearch_on_prem_deploy_separated.drawio.png)

In the separated mode, it is possible to deploy OpenSearch pods with different roles and provide more load distribution:

* Master nodes perform coordination and partition assignment. They also require small storage to store meta information.
* Data nodes store indices data.
* Client nodes accept client requests and forward them to the corresponding master nodes.

### DR Deployment Scheme

The Disaster Recovery scheme of OpenSearch deployment assumes that two independent OpenSearch clusters are deployed for both sides on separate Kubernetes environments with indices replication between
them.

![DR Scheme](/docs/public/images/opensearch_dr_deploy.drawio.png)

OpenSearch provides replication of indices data between the OpenSearch clusters via the Cross Cluster Replication plugin (red arrows).

For more information about these schemes, refer to the [OpenSearch Disaster Recovery](/docs/public/disaster-recovery.md) section in the _Cloud Platform Disaster Recovery Guide_.

## Integration with Managed Services

### Google Cloud

Not Applicable; the default HA scheme is used for the deployment to Google Cloud.

### AWS OpenSearch

The OpenSearch Service allows you to deploy OpenSearch supplementary services (Monitoring, DBaaS Adapter, Curator) without deploying OpenSearch, using the Amazon OpenSearch connection and credentials.
Thus, the features and functions of these services are adapted to Amazon OpenSearch and available as for Qubership OpenSearch delivery.

![AWS Scheme](/docs/public/images/opensearch_aws_deploy.drawio.png)

For more information, refer to the [Integration With Amazon OpenSearch](/docs/public/managed/amazon.md) section.

### Azure

Not Applicable; the default HA scheme is used for the deployment to Azure.
