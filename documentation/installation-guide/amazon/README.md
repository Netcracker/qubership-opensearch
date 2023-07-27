
This section describes the prerequisites and installation parameters for integration of Platform OpenSearch with Amazon OpenSearch.

- [Introduction](#introduction)
- [Prerequisites](#prerequisites)
    - [Global](#global)
    - [Preparations for Backup](#preparations-for-backup)
- [Example of Deploy Parameters](#example-of-deploy-parameters)
- [Amazon OpenSearch Features](#amazon-opensearch-features)
    - [Snapshots](#snapshots)
- [Scaling Capabilities](#scaling-capabilities)

# Introduction

OpenSearch Service allows you to deploy OpenSearch side services (DBaaS Adapter, Monitoring and Curator) without deploying OpenSearch, using Amazon OpenSearch URL.

*Important*: Slow queries functionality isn't available on AWS cloud.

# Prerequisites

## Global

* External OpenSearch URL is available from Kubernetes cluster where you are going to deploy side services.
* OpenSearch user credentials are provided. User has admin rights.
* There is DP Helm Deploy, App Deployer or local Helm configured to deploy to necessary Kubernetes cluster.

## Preparations for Backup

To collect snapshots manually (e.g. by `opensearch-curator` or `dbaas-adapter`) there are next prerequisites:

* AWS S3 Bucket to store snapshots;
* AWS Snapshot Role with S3 access policy;
* AWS User with AWS Authorization and ESHttpPut and Snapshot Role allowed;
* Mapped Snapshot role in OpenSearch Dashboard (Optional - if using fine-grained access control);
* Manual Snapshot repository.

1. AWS S3 Bucket configuration

   * Navigate to `Services -> Storage -> S3` in AWS Console.
   * Create new bucket with unique name and required region to store data. Specified `s3-bucket-name` will be needed on next steps.

2. AWS Snapshot Role configuration

   * Navigate to `Services -> Security, Identity, & Compliance -> IAM` in AWS Console. Choose `Roles` in the navigation pane.
   * Create new Role with `Another AWS account` type and your `Account ID` (You can find it in your user pane on top of console).
   * On `Permissions` step, press `Create policy`. You will be redirected on `Create policy` page.
   * Navigate to `JSON` pane and paste next configuration (Replace `s3-bucket-name` to bucket name you specified on previous step):

     ```yaml
     {
       "Version": "2012-10-17",
       "Statement": [
         {
           "Effect": "Allow",
           "Action": "s3:ListBucket",
           "Resource": "arn:aws:s3:::s3-bucket-name"
         },
         {
           "Effect": "Allow",
           "Action": [
               "s3:PutObject",
               "s3:GetObject",
               "s3:DeleteObject"
           ],
           "Resource": "arn:aws:s3:::s3-bucket-name/*"
         }
       ]
     }
     ```

   * Proceed next to `Review` step, specify a `Name` for new policy and press `Create policy` button.
   * Return to `Attach permission policies` tab, update policies list and select created one.
   * Proceed next to `Review` step, specify a `Role name` for Snapshot Role and press `Create role` button.
   * Navigate to `Trust relationships` tab of created Snapshot Role. Press `Edit trust relationship` button and paste next configuration:

     ```yaml
     {
       "Version": "2012-10-17",
       "Statement": [
         {
           "Sid": "",
           "Effect": "Allow",
           "Principal": {
             "Service": "es.amazonaws.com"
           },
           "Action": "sts:AssumeRole"
         }
       ]
     }
     ```

   * Press `Update Trust Policy` button.
   * `Role ARN` of created Snapshot Role will be needed on next steps.

3. AWS User configuration

   * Navigate to `Services -> Security, Identity, & Compliance -> IAM` in AWS Console. Choose `Users` in the navigation pane.
   * Create new User with `Access key - Programmatic access` enabled.
   * On `Permissions` step select `Attach existing policies directly`, press `Create policy`.
   * Navigate to `JSON` tab, paste the next configuration (Replace `Role ARN`, `Domain ARN` and `s3-bucket-name` with Snapshot role ARN, OpenSearch Service domain ARN and created bucket name):

     ```yaml
     {
         "Version": "2012-10-17",
         "Statement": [
             {
                 "Effect": "Allow",
                 "Action": "iam:PassRole",
                 "Resource": "Role ARN"
             },
             {
                 "Effect": "Allow",
                 "Action": "es:ESHttpPut",
                 "Resource": "Domain ARN"
             },
             {
                 "Effect": "Allow",
                 "Action": "s3:ListBucket",
                 "Resource": "arn:aws:s3:::s3-bucket-name"
             },
             {
                 "Effect": "Allow",
                 "Action": [
                     "s3:PutObject",
                     "s3:GetObject",
                     "s3:DeleteObject"
                 ],
                 "Resource": "arn:aws:s3:::s3-bucket-name/*"
             }
         ]
     }
     ```

   * Proceed next to `Review` step, specify `Name` for policy and press `Create policy` button.
   * Return to `Set permissions` tab, update policies list and select created one.
   * Proceed next to `Review` step and press `Create user` button.
   * On result page you get created User with generated credentials: `Access key ID` and `Secret access key`. Save these credentials for next steps (`Access Key` can be found in User security configuration, `Secret key` displayed only at creation) and press `Close` button.
   * `User ARN` of created User will be needed on next steps.

4. Snapshot Role mapping (if using fine-grained access control)

   * Navigate to `Services -> Analytics -> Amazon OpenSearch Service (successor to Amazon Elasticsearch Service)`. Select required OpenSearch domain.
   * Navigate to OpenSearch/Kibana Dashboard (Endpoint URL can be found in `General information` field of domain).
   * From the main menu choose Security, Role Mappings, and select the manage_snapshots role.
   * Add `User ARN` to `Users` field and `Role ARN` to `Backend roles` from previous steps.
   * Press `Submit` button.

5. Manual Snapshot repository registration

    OpenSearch Service requires AWS Authorization, so you can't use `curl` to perform this operation. Instead, use `Postman Desktop Agent` or other method to send AWS signed request to register snapshot.

   * Select `PUT` request and set the ```domain-endpoint/_snapshot/my-snapshot-repo-name``` URL, where `domain-endpoint` can be found in OpenSearch domain `General information` field and `my-snapshot-repo-name` is a name for repository.
   * In `Authorization` tab select `AWS Signature` type. Fill `AccessKey` and `SecretKey` with keys generated during User creation step. Fill the `Region` with the OpenSearch domain region and `Sevice` with `es`.
   * In `Body` tab select `raw` type and paste the next configuration (Replace `s3-bucket-name`, `region`, `Role ARN` with bucket name, region and Snapshot Role ARN from previous steps):

     ```yaml
     {
       "type": "s3",
       "settings": {
         "bucket": "s3-bucket-name",
         "region": "region",
         "role_arn": "Role ARN"
       }
     }
     ```

   * Press `Send` button. If all necessary grants provided, you get next response:

     ```yaml
     {
         "acknowledged": true
     }
     ```

    If there are some errors in response, check all required prerequisites. More information about repository registration in [Creating index snapshots](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/managedomains-snapshots.html).

    After manual snapshot repository registered, you can perform snapshot and restore with `curl` as OpenSearch user (See, [Manual backup](/documentation/maintenance-guide/backup/manual-backup-procedure.md) and [Manual recovery](/documentation/maintenance-guide/recovery/manual-recovery-procedure.md) guides).

   **Note:** OpenSearch also provides service indices that are not accessible for snapshot. To create snapshot either specify indices list ("indices": ["index1", "index2"]) or exclude service indices ("indices": "-.kibana*,-.opendistro*").

   To restore indices make sure there are no naming conflicts between indices on the cluster and indices in the snapshot. Delete indices on the existing OpenSearch Service domain, rename indices in snapshot or restore the snapshot to a different OpenSearch Service domain.

   For more detailed information about restore, refer to [Restoring snapshots](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/managedomains-snapshots.html#managedomains-snapshot-restore).

# Example of Deploy Parameters

Example of deployment parameters for external Amazon OpenSearch is presented below:

```yaml
dashboards:
  enabled: true
global:
  externalOpensearch:
    enabled: true
    url: "https://vpc-opensearch.us-east-1.es.amazonaws.com"
    username: "admin"
    password: "admin"
    nodesCount: 3
    dataNodesCount: 3
opensearch:
  snapshots:
    s3:
      enabled: true
      url: "https://s3.amazonaws.com"
      bucket: "opensearch-backups"
      keyId: "key"
      keySecret: "secret"
monitoring:
  enabled: true
  resources:
    requests:
      memory: 256Mi
      cpu: 50m
    limits:
      memory: 256Mi
      cpu: 200m

dbaasAdapter:
  enabled: true
  dbaasUsername: "admin"
  dbaasPassword: "admin"
  registrationAuthUsername: "admin"
  registrationAuthPassword: "admin"
  opensearchRepo: "snapshots"
  resources:
    requests:
      memory: 32Mi
      cpu: 50m
    limits:
      memory: 32Mi
      cpu: 200m
  securityContext:
    runAsUser: 1000

curator:
  enabled: true
  snapshotRepositoryName: "snapshots"
  username: "admin"
  password: "admin"
  resources:
    requests:
      memory: 256Mi
      cpu: 50m
    limits:
      memory: 256Mi
      cpu: 200m
  securityContext:
    runAsUser: 1000
    fsGroup: 1000

integrationTests:
  enabled: true
```

**NOTE:** This is an example, do not copy it as-is for your deployment, be sure about each parameter in your installation.

# Amazon OpenSearch Features

## Snapshots

Amazon OpenSearch does not support `fs` snapshot repositories, so you cannot create it by operator during installation. Only `s3` type is supported.
Amazon OpenSearch has the configured `s3` snapshot repository (e.g. `cs-automated-enc`) with automatically making snapshot by schedule, but this repository cannot be used for making manual snapshots (including by DBaaS adapter or curator).
If you want to manually manage repository you need to create it with the following guide [Creating index snapshots in Amazon OpenSearch Service](https://docs.aws.amazon.com/elasticsearch-service/latest/developerguide/es-managedomains-snapshots.html#es-managedomains-snapshot-registerdirectory).
Then specify this repository name in corresponding DBaaS Adapter and Curator parameters during deploy.

**NOTE:** Only `s3` parameters are required in this Curator installation, not `backupStorage`.

If you do not need to manually take snapshots just disable this feature with corresponding parameters (`dbaasAdapter.opensearchRepo` is empty and `curator.enabled: false`).

To configure snapshots manually, please follow the guide [Preparations for Backup](#preparations-for-backup).

# Scaling Capabilities

This section describes how to do the scaling procedures for Amazon OpenSearch.

Based on your workload, you can scale up (scale vertically) or scale out (scale horizontally) your cluster. 
To scale out your OpenSearch Service domain, add additional nodes (such as data nodes, master nodes, or UltraWarm nodes) to your cluster. 
To resize or scale up your domain, increase your Amazon Elastic Block Store (Amazon EBS) volume size or add more memory and vCPUs with bigger node types.

## Limitations

* Scaling Up|Down (Vertical Scaling) and Scaling Out|In (Horizontally Scaling) are manual procedures.
* Scaling Up requires downtime.
* It is necessary to upgrade platform OpenSearch state when scaling out Amazon OpenSearch.

## Scaling In|Out

When you scale out your domain, you are adding nodes of the same configuration type as your current cluster nodes.

To add nodes to your cluster you need:
1. Sign in to your AWS Management Console.
2. Open the OpenSearch Service console.
3. Select the domain that you want to scale.
4. Choose `Actions` -> `Edit Cluster Configuration`.
5. Change `Number of nodes` to necessary value for `Data nodes` and `Dedicated master nodes` sections.
6. Click `Save changes`.
7. Run upgrade `opensearch-service` platform job and provide correct values for `global.externalOpenSearch.nodesCount` and `global.externalOpenSearch.dataNodesCount`.

**NOTE:** Amazon OpenSearch supports scaling in (reduce nodes count), but the created indices should be able to the new data nodes count and have corresponding replicas count.

## Scaling Up|Down

If you want to vertically scale or scale up your domain, switch to a larger instance type to add more memory or CPU resources.
It is possible to change instance type of Amazon OpenSearch instances (data or master) for existing cluster, but such changes **requires downtime**.

To change the instance type you need:
1. Sign in to your AWS Management Console.
2. Open the OpenSearch Service console.
3. Select the domain that you want to scale.
4. Choose `Actions` -> `Edit Cluster Configuration`.
5. Select necessary `Instance Type` for `Data nodes` and `Dedicated master nodes` sections.
6. Click `Save changes`.

**Note:** When you scale up your domain, EBS volume size doesn't automatically scale up. You must specify this setting if you want the EBS volume size to automatically scale up.

## Useful References

* [Amazon OpenSearch Scaling Guide](https://aws.amazon.com/ru/premiumsupport/knowledge-center/opensearch-scale-up/)
* [Amazon OpenSearch Sizing Guide](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/sizing-domains.html)
* [Amazon OpenSearch Pricing](https://aws.amazon.com/ru/opensearch-service/pricing/)