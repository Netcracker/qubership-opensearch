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

OpenSearch service allows you to deploy OpenSearch side services (DBaaS Adapter, Monitoring, and Curator) without deploying OpenSearch, using Amazon OpenSearch URL.

**Important**: Slow queries' functionality is not available on AWS cloud.

# Prerequisites

## Global

* External OpenSearch URL is available from the Kubernetes cluster where you are going to deploy the side services.
* OpenSearch user credentials are provided. The provided user must be master with `all_access` and `security_manager` roles.
  For more information about master user, refer to Additional master users at
  [https://docs.aws.amazon.com/opensearch-service/latest/developerguide/fgac.html#fgac-more-masters](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/fgac.html#fgac-more-masters).

## Preparations for Backup

Following are the prerequisites to collect snapshots manually (for example, by `opensearch-curator` or `dbaas-adapter`):

* AWS S3 Bucket to store snapshots.
* AWS Snapshot Role with S3 access policy.
* AWS User with AWS Authorization and ESHttpPut and Snapshot Role allowed.
* Mapped Snapshot role in OpenSearch Dashboard (Optional - if using fine-grained access control).
* Manual Snapshot repository.

1. AWS S3 Bucket configuration

   * Navigate to **Services -> Storage -> S3** in AWS Console.
   * Create a new bucket with a unique name and required region to store data. The specified `s3-bucket-name` is needed for the following steps.

2. AWS Snapshot Role configuration

   * Navigate to **Services -> Security, Identity, & Compliance -> IAM** in AWS Console. Choose "Roles" in the navigation pane.
   * Create a new role with "Another AWS account" type and your "Account ID" (You can find it in your user pane on top of the console).
   * On the **Permissions** step, click **Create policy**. You will be redirected to the Create policy page.
   * Navigate to the JSON pane and paste the following configuration (Replace `s3-bucket-name` to the bucket name you specified in the previous step):

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

   * Proceed next to the **Review** step. Specify a name for the new policy and click **Create policy**.
   * Return to the **Attach permission policies** tab, update policies list, and select the created one.
   * Proceed next to the **Review** step. Specify a role name for Snapshot Role and click **Create role**.
   * Navigate to the **Trust relationships** tab of the created snapshot role. Click **Edit trust relationship** and paste the following configuration:

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

   * Click **Update Trust Policy**.
   * `Role ARN` of the created snapshot role is will be needed in the following steps.

3. AWS User configuration

   * Navigate to **Services -> Security, Identity, & Compliance -> IAM** in AWS Console. Choose "Users" in the navigation pane.
   * Create a new user with `Access key - Programmatic access` enabled.
   * On the **Permissions** step, select "Attach existing policies directly", and click **Create policy**.
   * Navigate to the **JSON** tab, paste the following configuration
     (Replace `Role ARN`, `Domain ARN`, and `s3-bucket-name` with the snapshot role ARN, OpenSearch Service domain ARN, and the created bucket name):

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

   * Proceed next to the **Review** step, specify the name for the policy and click **Create policy**.
   * Return to the **Set permissions** tab, update the policies' list, and select the created one.
   * Proceed next to the **Review** step and click **Create user**.
   * On the result page, the created user with the generated credentials is displayed with `Access key ID` and `Secret access key`.
     Save these credentials for the following steps (`Access Key` can be found in the User security configuration, and `Secret key` is displayed only at the creation), and click **Close**.
   * `User ARN` of the created user is needed in the following steps.

4. Snapshot Role mapping (if using fine-grained access control)

   * Navigate to **Services -> Analytics -> Amazon OpenSearch Service** (successor to Amazon Elasticsearch Service). Select the required OpenSearch domain.
   * Navigate to the OpenSearch/Kibana Dashboard (Endpoint URL can be found in the **General information** field of the domain).
   * From the main menu, choose Security, Role Mappings, and select the "manage_snapshots" role.
   * Add "User ARN" to the **Users* field and "Role ARN" to **Backend roles** from the previous steps.
   * Click **Submit**.

5. Manual Snapshot repository registration

    The OpenSearch service requires AWS Authorization, so you cannot use `curl` to perform this operation.
    Instead, use Postman Desktop Agent or other method to send AWS signed request to register a snapshot.

   * Select `PUT` request and set the `domain-endpoint/_snapshot/my-snapshot-repo-name` URL,
     where `domain-endpoint` can be found in the OpenSearch domain **General information** field and `my-snapshot-repo-name` is the name of the repository.
   * In the **Authorization** tab, select "AWS Signature" type. Enter **AccessKey** and **SecretKey** with keys generated during the user creation step.
     Enter the **Region** with the OpenSearch domain region and **Service** with "es".
   * In the **Body** tab, select "raw" type and paste the following configuration (Replace `s3-bucket-name`, `region`, `Role ARN` with bucket name, region, and Snapshot Role ARN from the previous steps).

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

   * Click **Send**. If all necessary grants are provided, you get the following response:

     ```yaml
     {
         "acknowledged": true
     }
     ```

    If there are some errors in the response, check all the required prerequisites.
    For more information about repository registration, refer to [https://docs.aws.amazon.com/opensearch-service/latest/developerguide/managedomains-snapshots.html].

    After the manual snapshot repository is registered, you can perform snapshot and restore with `curl` as an OpenSearch user.

   **Note**: OpenSearch also provides service indices that are not accessible for snapshot.
   To create a snapshot, either specify the indices list ("indices": ["index1", "index2"]) or exclude service indices ("indices": "-.kibana*,-.opendistro*").

   To restore indices, make sure there are no naming conflicts between indices on the cluster and indices in the snapshot.
6. Delete indices on the existing OpenSearch Service domain, rename indices in the snapshot, or restore the snapshot to a different OpenSearch Service domain.

   For more information about restore, refer to Restoring snapshots at
   [https://docs.aws.amazon.com/opensearch-service/latest/developerguide/managedomains-snapshots.html#managedomains-snapshot-restore].

# Example of Deploy Parameters

An example of deployment parameters for external Amazon OpenSearch is as follows:

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

**Note**: This is an example, do not copy it as-is for your deployment, be sure about each parameter in your installation.

# Amazon OpenSearch Features

## Snapshots

Amazon OpenSearch does not support `fs` snapshot repositories, so you cannot create it by the operator during the installation. Only `s3` type is supported.
Amazon OpenSearch has the configured `s3` snapshot repository (example, `cs-automated-enc`) with automatically making snapshot by schedule,
but this repository cannot be used for making manual snapshots (including by DBaaS adapter or curator).
If you want to manually manage the repository, refer to Creating index snapshots in Amazon OpenSearch Service in
[https://docs.aws.amazon.com/elasticsearch-service/latest/developerguide/es-managedomains-snapshots.html#es-managedomains-snapshot-registerdirectory].
Then specify this repository name in the corresponding DBaaS Adapter and Curator parameters during the deployment.

**Note**: Only `s3` parameters are required in this Curator installation, not `backupStorage`.

If you do not need to manually take snapshots, just disable this feature with the corresponding parameters (`dbaasAdapter.opensearchRepo` is empty and `curator.enabled: false`).

To configure snapshots manually, refer to the [Preparations for Backup](#preparations-for-backup) section.

# Scaling Capabilities

This section describes how to do scaling procedures for Amazon OpenSearch.

Based on the workload, you can scale up (scale vertically) or scale out (scale horizontally) a cluster.
To scale out an OpenSearch Service domain, add additional nodes (such as data nodes, master nodes, or UltraWarm nodes) to the cluster.
To resize or scale up the domain, increase the Amazon Elastic Block Store (Amazon EBS) volume size or add more memory and vCPUs with bigger node types.

## Limitations

* Scaling Up|Down (Vertical Scaling) and Scaling Out|In (Horizontally Scaling) are manual procedures.
* Scaling Up requires downtime.
* It is necessary to upgrade the platform OpenSearch state when scaling out Amazon OpenSearch.

## Scaling In|Out

When you scale out a domain, you are adding nodes of the same configuration type as the current cluster nodes.

To add nodes to a cluster:

1. Sign in to the AWS Management console.
2. Open the OpenSearch Service console.
3. Select the domain that you want to scale.
4. Choose **Actions -> Edit Cluster Configuration**.
5. Change **Number of nodes** to the necessary value for **Data nodes** and **Dedicated master nodes** sections.
6. Click **Save changes**.
7. Run the upgrade `opensearch-service` platform job and provide the correct values for `global.externalOpenSearch.nodesCount` and `global.externalOpenSearch.dataNodesCount`.

**Note**: Amazon OpenSearch supports scaling in (reduce nodes count), but the created indices should be equal to the new data nodes count and have the corresponding replicas count.

## Scaling Up|Down

If you want to vertically scale or scale up a domain, switch to a larger instance type to add more memory or CPU resources.
It is possible to change the instance type of Amazon OpenSearch instances (data or master) for an existing cluster, but such changes require downtime.

To change the instance type:

1. Sign in to the AWS Management console.
2. Open the OpenSearch Service console.
3. Select the domain that you want to scale.
4. Choose **Actions -> Edit Cluster Configuration**.
5. Select the necessary **Instance Type** for **Data nodes** and **Dedicated master nodes** sections.
6. Click **Save changes**.

**Note**: When you scale up a domain, the EBS volume size does not automatically scale up. You must specify this setting if you want the EBS volume size to automatically scale up.

## Useful References

* [Amazon OpenSearch Scaling Guide](https://aws.amazon.com/ru/premiumsupport/knowledge-center/opensearch-scale-up/)
* [Amazon OpenSearch Sizing Guide](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/sizing-domains.html)
* [Amazon OpenSearch Pricing](https://aws.amazon.com/ru/opensearch-service/pricing/)
