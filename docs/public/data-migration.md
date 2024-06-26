This guide describes the data migration procedures for OpenSearch.

<!-- #GFCFilterMarkerStart# -->
The following topics are covered in this chapter:

[[_TOC_]]
<!-- #GFCFilterMarkerEnd# -->

# Prerequisites

* `kubectl` tool is installed: [https://kubernetes.io/ru/docs/tasks/tools/install-kubectl/]
* `tar` tool is installed
* `mc` tool is installed: [https://github.com/minio/mc/blob/master/README.md]

# Transfer Data Schemes

## Data Migration with Database Replacement

This scheme helps to restore database `B` from `source` environment as a full replacement of database `A` on `target` environment.

To achieve this, follow the guide:

1. Make a backup of the database `B` on `source` environment:

   ```bash
   curl -XPOST -u ${USERNAME}:${PASSWORD} -v -H "Content-Type: application/json" -d '{"dbs":["B"]}'  http://opensearch-curator:8080/backup
   ```

   As a response you will receive `${BACKUP_NAME}` to check its status. For example,

   ```bash
   20240409T113357
   ```

2. Track backup status with the following command:

   ```bash
   curl -u ${USERNAME}:${PASSWORD} -XGET http://opensearch-curator:8080/jobstatus/${BACKUP_NAME}
   ```

   Wait for the backup to complete successfully:

   ```bash
   {"task_id": "20240409T113357", "type": "backup", "status": "Successful", "vault": "20240409T113357", "err": ""}
   ```

3. [Transfer Snapshots Data Between Storages](#transfer-snapshots-data-between-storages) depending on used storages on the `source` and `target` environments.
4. Restore database `B` on `target` environment with removing `A` database and renaming `B` database to `A`:

   ```bash
   curl -XPOST -u ${USERNAME}:${PASSWORD} -v -H "Content-Type: application/json" -d  '{"vault":"${BACKUP_NAME}", "dbs":["B"], "changeDbNames":{"B":"A"}, "clean":"true"}' http://opensearch-curator:8080/restore
   ```

   `clean` option removes indices, templates and aliases of database that should be restored. In the above scenario, database `A` will be removed before restoring data.

   As a response you will receive `${TASK_ID}` to check its status. For example,

   ```bash
   ecaef6f0-2e2f-4361-ab96-f956d89e30c4
   ```

5. Track recovery status with the following command:

   ```bash
   curl -u ${USERNAME}:${PASSWORD} -XGET http://opensearch-curator:8080/jobstatus/${TASK_ID}
   ```

   Wait for the recovery to complete successfully:

   ```bash
   {"task_id": "ecaef6f0-2e2f-4361-ab96-f956d89e30c4", "type": "restore", "status": "Successful", "vault": "20240409T113357", "err": ""}
   ```

## Data Migration without Database Replacement

This scheme helps to restore database `B` from `source` environment to `target` environment.

To achieve this, follow the guide:

1. Make a backup of the database `B` on `source` environment:

   ```bash
   curl -XPOST -u ${USERNAME}:${PASSWORD} -v -H "Content-Type: application/json" -d '{"dbs":["B"]}'  http://opensearch-curator:8080/backup
   ```

   As a response you will receive `${BACKUP_NAME}` to check its status. For example,

   ```bash
   20240409T113357
   ```

2. Track backup status with the following command:

   ```bash
   curl -u ${USERNAME}:${PASSWORD} -XGET http://opensearch-curator:8080/jobstatus/${BACKUP_NAME}
   ```

   Wait for the backup to complete successfully:

   ```bash
   {"task_id": "20240409T113357", "type": "backup", "status": "Successful", "vault": "20240409T113357", "err": ""}
   ```

3. [Transfer Snapshots Data Between Storages](#transfer-snapshots-data-between-storages) depending on used storages on the source and target environments.
4. If database `B` does not exist in DBaaS aggregator of `target` environment, create it with the following command:

   ```bash
   curl -u ${DBAAS_USERNAME}:${DBAAS_PASSWORD} -XPUT "${DBAAS_URL}/api/v3/dbaas/${NAMESPACE}/databases" -H "Content-Type: application/json" -d'
   {
     "classifier": {
       "microserviceName": "${MICROSERVICE_NAME}",
       "scope": "service",
       "namespace": "${NAMESPACE}"
     },
     "originService": "${MICROSERVICE_NAME}",
     "physicalDatabaseId": "${PHYSICAL_DATABASE_ID}",
     "namePrefix": "B",
     "settings": {
       "resourcePrefix": true,
       "createOnly": ["user"]
     },
     "type": "opensearch"
   }'
   ```

5. Restore database `B` on `target` environment with removing `B` database:

   ```bash
   curl -XPOST -u ${USERNAME}:${PASSWORD} -v -H "Content-Type: application/json" -d  '{"vault":"${BACKUP_NAME}", "dbs":["B"], "clean":"true"}' http://opensearch-curator:8080/restore
   ```

   `clean` option removes indices, templates and aliases of database that should be restored. In the above scenario, database `B` will be removed before restoring data.

   As a response you will receive `${TASK_ID}` to check its status. For example,

   ```bash
   ecaef6f0-2e2f-4361-ab96-f956d89e30c4
   ```

6. Track recovery status with the following command:

   ```bash
   curl -u ${USERNAME}:${PASSWORD} -XGET http://opensearch-curator:8080/jobstatus/${TASK_ID}
   ```

   Wait for the recovery to complete successfully:

   ```bash
   {"task_id": "ecaef6f0-2e2f-4361-ab96-f956d89e30c4", "type": "restore", "status": "Successful", "vault": "20240409T113357", "err": ""}
   ```

Where:

* `${USERNAME}` is the username for OpenSearch backup daemon.
* `${PASSWORD}` is the password for OpenSearch backup daemon.
* `${DBAAS_URL}` is the URL to DBaaS aggregator. For example, `http://dbaas-aggregator.dbaas:8080`.
* `${DBAAS_USERNAME}` is the username for DBaaS aggregator.
* `${DBAAS_PASSWORD}` is the password for DBaaS aggregator.
* `${MICROSERVICE_NAME}` is the name of microservice that requires database creation. For example, `test-microservice`.
* `${NAMESPACE}` is the namespace of `${MICROSERVICE_NAME}` microservice. For example, `test-namespace`.
* `${PHYSICAL_DATABASE_ID}` is the identifier of OpenSearch DBaaS adapter registered in DBaaS aggregator. For example, `opensearch-service`.

# Transfer Snapshots Data Between Storages

**Important**: Snapshots data transfer involves the loss of the data in the `target` environment.

## Transfer From NFS to NFS

To transfer data between NFS persistent volumes, follow the guide:

1. Use `source` environment context in `kubectl` tool.

2. Transfer data from `source` NFS persistent volume to local environment using `kubectl` tool and the following command:

   ```bash
   kubectl cp ${CURATOR_POD_NAME_A}:/backup-storage ./snapshots -n ${NAMESPACE_A}
   ```

3. Use `target` environment context in `kubectl` tool.

4. Remove everything inside `target` NFS persistent volume using the following command:

   ```bash
   kubectl exec ${CURATOR_POD_NAME_B} -n ${NAMESPACE_B} -- sh -c 'rm -rf /backup-storage/*'
   ```

5. Transfer data from local environment to the `target` NFS persistent volume using `tar` and `kubectl` tools and the following command:

   ```bash
   tar -cf - snapshots | kubectl exec -i -n ${NAMESPACE_B} ${CURATOR_POD_NAME_B} -- tar -xf - -C /backup-storage --strip-components 1
   ```

6. Actualize snapshots data for OpenSearch and OpenSearch curator on `target` environment. You need to remove `snapshots` repository by running the following command in any OpenSearch node:

   ```bash
   curl -u $OPENSEARCH_USERNAME:$OPENSEARCH_PASSWORD -k -XDELETE https://localhost:9200/_snapshot/snapshots
   ```

   and then restart `opensearch-service-operator` and `opensearch-curator` pods manually or using the following command:

   ```bash
   kubectl delete pod ${CURATOR_POD_NAME_B} ${OPERATOR_POD_NAME_B} -n ${NAMESPACE_B}
   ```

Where:

* `${CURATOR_POD_NAME_A}` is the name of OpenSearch curator pod from which data needs to be transferred. For example, `opensearch-curator-5bbb49f5d6-dtj6m`.
* `${NAMESPACE_A}` is the namespace where `${CURATOR_POD_NAME_A}` OpenSearch curator pod is located.
* `${CURATOR_POD_NAME_B}` is the name of OpenSearch curator pod to which data needs to be transferred. For example, `opensearch-curator-6b44fc6597-jw6z8`.
* `${OPERATOR_POD_NAME_B}` is the name of OpenSearch service operator pod in `target` environment. For example, `opensearch-service-operator-85677888d8-kxx87`.
* `${NAMESPACE_B}` is the namespace where `${CURATOR_POD_NAME_B}` OpenSearch curator pod is located.

## Transfer From NFS to S3

To transfer data from NFS persistent volume to S3, follow the guide:

1. Use `source` environment context in `kubectl` tool.

2. Transfer data from `source` NFS persistent volume to local environment using `kubectl` tool and the following command:

   ```bash
   kubectl cp ${CURATOR_POD_NAME_A}:/backup-storage ./${BUCKET_NAME_B} -n ${NAMESPACE_A}
   ```

3. On local environment shift folders with backups (full and granular) to `backup-storage` folder. You can do it manually, or use the following commands:

   ```bash
   mkdir ${BUCKET_NAME_B}/backup-storage
   mv ${BUCKET_NAME_B}/[0-9]*T*[0-9] ${BUCKET_NAME_B}/granular ${BUCKET_NAME_B}/backup-storage
   ```

4. Add S3 server credentials for `target` environment to configuration file of `mc` tool:

   ```bash
   mc alias set ${ALIAS_B} ${URL_B} ${ACCESSKEY_B} ${SECRETKEY_B}
   ```

5. Check whether necessary bucket exists on S3:

   ```bash
   mc ls ${ALIAS_B}
   ```

   If it exists, you need to remove everything inside using the following command:

   ```bash
   mc rm -r --force --versions ${ALIAS_B}/${BUCKET_NAME_B}/
   ```

   If the bucket does not exist, create it using the following command:

   ```bash
   mc mb ${ALIAS_B}/${BUCKET_NAME_B}
   ```

6. Transfer data from local environment to S3 of `target` environment using `mc` tool and the following command:

   ```bash
   mc cp -r ./${BUCKET_NAME_B}/ ${ALIAS_B}
   ```

7. Use `target` environment context in `kubectl` tool.

8. Actualize snapshots data for OpenSearch and OpenSearch curator on `target` environment. You need to remove `snapshots` repository by running the following command in any OpenSearch node:

   ```bash
   curl -u $OPENSEARCH_USERNAME:$OPENSEARCH_PASSWORD -k -XDELETE https://localhost:9200/_snapshot/snapshots
   ```

   and then restart `opensearch-service-operator` pod manually or using the following command:

   ```bash
   kubectl delete pod ${OPERATOR_POD_NAME_B} -n ${NAMESPACE_B}
   ```

Where:

* `${CURATOR_POD_NAME_A}` is the name of OpenSearch curator pod from which data needs to be transferred. For example, `opensearch-curator-5bbb49f5d6-dtj6m`.
* `${NAMESPACE_A}` is the namespace where `${CURATOR_POD_NAME_A}` OpenSearch curator pod is located.
* `${OPERATOR_POD_NAME_B}` is the name of OpenSearch service operator pod in `target` environment. For example, `opensearch-service-operator-85677888d8-kxx87`.
* `${NAMESPACE_B}` is the namespace where `${OPERATOR_POD_NAME_B}` OpenSearch service operator pod is located.
* `${BUCKET_NAME_B}` is the name of bucket on S3 to which data needs to be transferred. For example, `opensearch-env-b`.
* `${ALIAS_B}` is the unique alias for S3 to use in `mc` tool. For example, `env-b`.
* `${URL_B}` is the URL to S3 where data needs to be transferred. For example, `https://minio-ingress-minio-service.env-b.openshift.sdntest.example.com`.
* `${ACCESSKEY_B}` is the access key for the `${URL_B}` S3.
* `${SECRETKEY_B}` is the secret key for the `${URL_B}` S3.

## Transfer From S3 to NFS

To transfer data from S3 to NFS persistent volume, follow the guide:

1. Add S3 server credentials for `source` environment to configuration file of `mc` tool:

   ```bash
   mc alias set ${ALIAS_A} ${URL_A} ${ACCESSKEY_A} ${SECRETKEY_A}
   ```

2. Transfer data from S3 of `source` environment to local environment using `mc` tool and the following command:

    ```bash
    mc cp -r ${ALIAS_A}/${BUCKET_NAME_A}/ ./snapshots
    ```

3. On local environment shift folders with backups (full and granular) from `backup-storage` folder. You can do it manually, or use the following commands:

   ```bash
   mv snapshots/backup-storage/* ./snapshots/
   rm -rf snapshots/backup-storage
   ```

4. Use `target` environment context in `kubectl` tool.

5. Remove everything inside `target` NFS persistent volume using the following command:

   ```bash
   kubectl exec ${CURATOR_POD_NAME_B} -n ${NAMESPACE_B} -- sh -c 'rm -rf /backup-storage/*'
   ```

6. Transfer data from local environment to the `target` NFS persistent volume using `tar` and `kubectl` tools and the following command:

   ```bash
   tar -cf - snapshots | kubectl exec -i -n ${NAMESPACE_B} ${CURATOR_POD_NAME_B} -- tar -xf - -C /backup-storage --strip-components 1
   ```

7. Actualize snapshots data for OpenSearch and OpenSearch curator on `target` environment. You need to remove `snapshots` repository by running the following command in any OpenSearch node:

   ```bash
   curl -u $OPENSEARCH_USERNAME:$OPENSEARCH_PASSWORD -k -XDELETE https://localhost:9200/_snapshot/snapshots
   ```

   and then restart `opensearch-service-operator` and `opensearch-curator` pods manually or using the following command:

   ```bash
   kubectl delete pod ${CURATOR_POD_NAME_B} ${OPERATOR_POD_NAME_B} -n ${NAMESPACE_B}
   ```

Where:

* `${BUCKET_NAME_A}` is the name of bucket on S3 from which data needs to be transferred. For example, `opensearch-env-a`.
* `${ALIAS_A}` is the unique alias for S3 to use in `mc` tool. For example, `env-a`.
* `${URL_A}` is the URL to S3 from which data needs to be transferred. For example, `https://minio-ingress-minio-service.env-a.openshift.sdntest.example.com`.
* `${ACCESSKEY_A}` is the access key for the `${URL_A}` S3.
* `${SECRETKEY_A}` is the secret key for the `${URL_A}` S3.
* `${CURATOR_POD_NAME_B}` is the name of OpenSearch curator pod to which data needs to be transferred. For example, `opensearch-curator-6b44fc6597-jw6z8`.
* `${OPERATOR_POD_NAME_B}` is the name of OpenSearch service operator pod in `target` environment. For example, `opensearch-service-operator-85677888d8-kxx87`.
* `${NAMESPACE_B}` is the namespace where `${CURATOR_POD_NAME_B}` OpenSearch curator pod is located.

## Transfer From S3 to S3

To transfer data between S3 servers, follow the guide:

1. Add S3 server credentials for `source` environment to configuration file of `mc` tool:

   ```bash
   mc alias set ${ALIAS_A} ${URL_A} ${ACCESSKEY_A} ${SECRETKEY_A}
   ```

2. Add S3 server credentials for `target` environment to configuration file of `mc` tool:

   ```bash
   mc alias set ${ALIAS_B} ${URL_B} ${ACCESSKEY_B} ${SECRETKEY_B}
   ```

3. Check whether necessary bucket exists on S3 of `target` environment:

   ```bash
   mc ls ${ALIAS_B}
   ```

   If it exists, you need to remove everything inside using the following command:

   ```bash
   mc rm -r --force --versions ${ALIAS_B}/${BUCKET_NAME_B}/
   ```

   If the bucket does not exist, create it using the following command:

   ```bash
   mc mb ${ALIAS_B}/${BUCKET_NAME_B}
   ```

4. Transfer data from S3 of `source` environment to S3 of `target` environment using `mc` tool and the following command:

    ```bash
    mc cp -r ${ALIAS_A}/${BUCKET_NAME_A}/ ${ALIAS_B}/${BUCKET_NAME_B}
    ```

5. Actualize snapshots data for OpenSearch and OpenSearch curator on `target` environment. You need to remove `snapshots` repository by running the following command in any OpenSearch node:

   ```bash
   curl -u $OPENSEARCH_USERNAME:$OPENSEARCH_PASSWORD -k -XDELETE https://localhost:9200/_snapshot/snapshots
   ```

   and then restart `opensearch-service-operator` pod manually or using the following command:

   ```bash
   kubectl delete pod ${OPERATOR_POD_NAME_B} -n ${NAMESPACE_B}
   ```

Where:

* `${BUCKET_NAME_A}` is the name of bucket on S3 from which data needs to be transferred. For example, `opensearch-env-a`.
* `${ALIAS_A}` is the unique alias for S3 to use in `mc` tool. For example, `env-a`.
* `${URL_A}` is the URL to S3 from which data needs to be transferred. For example, `https://minio-ingress-minio-service.env-a.openshift.sdntest.example.com`.
* `${ACCESSKEY_A}` is the access key for the `${URL_A}` S3.
* `${SECRETKEY_A}` is the secret key for the `${URL_A}` S3.
* `${BUCKET_NAME_B}` is the name of bucket on S3 to which data needs to be transferred. For example, `opensearch-env-b`.
* `${ALIAS_B}` is the unique alias for S3 to use in `mc` tool. For example, `env-b`.
* `${URL_B}` is the URL to S3 where data needs to be transferred. For example, `https://minio-ingress-minio-service.env-b.openshift.sdntest.example.com`.
* `${ACCESSKEY_B}` is the access key for the `${URL_B}` S3.
* `${SECRETKEY_B}` is the secret key for the `${URL_B}` S3.
* `${OPERATOR_POD_NAME_B}` is the name of OpenSearch service operator pod in `target` environment. For example, `opensearch-service-operator-85677888d8-kxx87`.
* `${NAMESPACE_B}` is the namespace where `${OPERATOR_POD_NAME_B}` OpenSearch service operator pod is located.
