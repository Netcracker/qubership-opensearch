# Amazon Opensearch

Opensearch Service allows you to deploy Opensearch side services (DBaaS Adapter, Monitoring and Curator) without deploying Opensearch, using Amazon Opensearch URL.

## Prerequisites
* External Opensearch URL is available from Kubernetes cluster where you are going to deploy side services.
* Opensearch user credentials are provided. User has admin rights.
* There is DP Helm Deploy, App Deployer or local Helm configured to deploy to necessary Kubernetes cluster.

## Example Parameters

Example of deployment parameters for external Amazon Opensearch is presented below:

```yaml
dashboards:
  enabled: true
global:
  externalOpensearch:
    enabled: true
    url: "https://vpc-opensearch.us-east-1.es.amazonaws.com"
    username: "admin"
    password: "admin"
opensearch:
  securityConfig:
    enabled: true
    path: "/usr/share/opensearch/plugins/opensearch-security/securityconfig"
    authc:
      basic:
        username: "netcrk"
        password: "crknet"
      oidc:
        openid_connect_url: "http://identity-management-security-services-ci.dr311dev.openshift.sdntest.netcracker.com/.well-known/openid-configuration"
  master:
    enabled: true
    replicas: 3
    persistence:
      enabled: true
      storageClass: "local-path"
    resources:
      limits:
        cpu: 500m
        memory: 1536Mi
      requests:
        cpu: 50m
        memory: 1536Mi
  client:
    ingress:
      enabled: true
      path: /
      hosts:
        - opensearch-opensearch-service.ci-master.openshift.sdntest.netcracker.com
  snapshots:
    enabled: true
    repositoryName: "snapshots"
    persistentVolume: "pv-opensearch-snapshots"
    persistentVolumeClaim: "pvc-opensearch-snapshots"
    size: 2Gi

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
  dbaasUsername: "dbaas-aggregator"
  dbaasPassword: "dbaas-aggregator"
  registrationAuthUsername: "cluster-dba"
  registrationAuthPassword: "Bnmq5567_PO"
  opensearchRepo: "snapshots"
  opensearchRepoRoot: "/usr/share/opensearch"
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

## Amazon Opensearch Features

### Snapshots

Amazon Opensearch does not support `fs` snapshot repositories, so you cannot create it by operator during installation. Only `s3` type is supported.
Amazon Opensearch has the configured `s3` snapshot repository (e.g. `cs-automated-enc`) with automatically making snapshot by schedule, but this repository cannot be used for making manual snapshots (including by DBaaS adapter or curator).
If you want to manually managed repository you need to create it with the following guide [Creating index snapshots in Amazon Opensearch Service](https://docs.aws.amazon.com/elasticsearch-service/latest/developerguide/es-managedomains-snapshots.html#es-managedomains-snapshot-registerdirectory).
Then specify this repository name in corresponding DBaaS Adapter and Curator parameters during deploy.

**NOTE:** Only `s3` parameters required in this Curator installation, not `backupStorage`.

If you do not need to manual making snapshots just disable this feature with corresponding parameters (`dbaasAdapter.opensearchRepo` is empty and `curator.enabled: false`).

#### Manual Snapshots

To collect snapshots manually there are next prerequisites:

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

* Select `PUT` request and set the next URL:
```domain-endpoint/_snapshot/my-snapshot-repo-name```

where `domain-endpoint` can be found in OpenSearch domain `General information` field and `my-snapshot-repo-name` is a name for repository.
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

After manual snapshot repository registered, you can perform snapshot and restore with `curl` as OpenSearch user (See, [Manual backup](../../../../../documentation/maintenance-guide/backup/manual-backup-procedure.md) and [Manual revovery](../../../../../documentation/maintenance-guide/recovery/manual-recovery-procedure.md) guides).

**Note:** OpenSearch also provides service indices that not accessible for snapshot. To create snapshot either specify indices list ("indices": ["index1", "index2"]) or exclude service indices ("indices": "-.kibana*,-.opendistro*").

To restore indices make sure there are no naming conflicts between indices on the cluster and indices in the snapshot. Delete indices on the existing OpenSearch Service domain, rename indices in snapshot or restore the snapshot to a different OpenSearch Service domain.

More details about restore in [Restoring snapshots](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/managedomains-snapshots.html#managedomains-snapshot-restore).