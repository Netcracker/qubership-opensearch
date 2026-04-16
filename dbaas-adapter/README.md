This section provides information about REST API of the DBaaS OpenSearch adapter.

- [Introduction](#introduction)
- [Paths](#paths)
    - [Force physical database registration](#force-physical-database-registration)
    - [Physical database information](#physical-database-information)
    - [Support Info](#support-info)
    - [Health](#health)
    - [Create Database](#create-database)
    - [Create Database v2](#create-database-v2)
    - [List Databases](#list-databases)
    - [Update Database Metadata](#update-database-metadata)
    - [Create User with Generated Name](#create-user-with-generated-name)
    - [Create User with Specified Name](#create-user-with-specified-name)
    - [Recover Users](#recover-users)
    - [Users Recovery State](#users-recovery-state)
    - [Drop Created Resources](#drop-created-resources)
    - [Drop Created Resources v2](#drop-created-resources-v2)
    - [Collect Backup](#collect-backup)
    - [Track Backup](#track-backup)
    - [Restore Backup](#restore-backup)
    - [Track Restore From Track ID](#track-restore-from-track-id)
    - [Track Restore From Indices](#track-restore-from-indices)
- [Definitions](#definitions)
    - [RegistrationPhysicalRequest](#registrationphysicalrequest)
    - [Supports](#supports)
    - [HealthStatus](#healthstatus)
    - [DBCreateRequest](#dbcreaterequest)
    - [Settings](#settings)
    - [CreatedDatabase](#createddatabase)
    - [CreatedDatabase v2](#createddatabase-v2)
    - [UserCreateRequest](#usercreaterequest)
    - [CreatedUser](#createduser)
    - [UsersToRecover](#userstorecover)
    - [ConnectionProperties](#connectionproperties)
    - [ConnectionProperties v2](#connectionproperties-v2)
    - [DBResource](#dbresource)
    - [DBResourceDeleteStatus](#dbresourcedeletestatus)
    - [ActionTrack](#actiontrack)
    - [Details](#details)

# Introduction

OpenSearch does not have databases or any other logical entity which could combine the indexes of one microservice. In terms of the DBaaS OpenSearch adapter the database is a set of indices, aliases and templates starting with one `resourcePrefix` and managed by one microservice and its user. 

The DBaaS OpenSearch adapter has 2 versions of API: `v1` and `v2`. The `v1` version allows to create users only with `admin` permissions, but the `v2` version creates 4 users with different roles (`admin`, `dml`, `readonly`, `ism`) on each corresponding request. You can find out more about roles in [Multiple Roles](#multiple-roles) section.

The migration between these versions is uni-directional. It means if you are upgraded DBaaS OpenSearch adapter from `v1` to `v2` version, you must not downgrade it.

## Security

The DBaaS OpenSearch adapter allows to create users which have access only for indices with specific Resource Prefix (`resourcePrefix`) which is also generated with Database Creation.

**Note:** At this moment OpenSearch security does not allow to configure granular security for Index Templates. Users can create any Index Template with any `index_pattern` inside or cannot create them all. It is strongly recommended to create templates starting with `resourcePrefix`.

### Multiple Roles

The DBaaS OpenSearch adapter in `v2` version supports the following roles:

* `readonly` role allows getting information about specific indices, searching by indices and aliases
* `dml` role allows the same as `readonly` role and writing to specific indices.
* `admin` role allows the same as `dml` role and creating, updating, deleting specific indices, aliases and any templates.
* `ism` role allows the same as `admin` role and access to OpenSearch Index State Management API. 

# Paths

## Force physical database registration

```
GET /api/v1/dbaas/adapter/physical_database/force_registration
```

### Description

This API forces the adapter to immediately register itself in `DBaaS aggregator`. The adapter initiates
background task that tries to register physical database in `DBaaS aggregator`, and responds with
`202` status before the background task finishes.

### Responses

| HTTP Code | Description                                                          | Schema |
|-----------|----------------------------------------------------------------------|--------|
| **202**   | Physical database registration process has been started successfully | string |

### Example

```
curl -u <username>:<password> -XGET http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/physical_database/force_registration
```

## Physical database information

```
GET /api/v1/dbaas/adapter/opensearch/physical_database
```

### Description

This API sends adapter physical database information.

### Responses

| HTTP Code | Description                              | Schema                                                      |
|-----------|------------------------------------------|-------------------------------------------------------------|
| **200**   | Own physical database information        | [RegistrationPhysicalRequest](#registrationphysicalrequest) |
| **500**   | Error occurred while getting information | No Content                                                  |

### Example

Request:

```
curl -u <username>:<password> -XGET http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/physical_database
```

Response:

```
{"id":"opensearch-service","labels":null}
```

## Support Info

```
GET /api/v1/dbaas/adapter/opensearch/supports
```

### Description

This API describes what features supported by adapter.

### Responses

| HTTP Code | Description                                    | Schema                |
|-----------|------------------------------------------------|-----------------------|
| **200**   | Provided values would be used instead defaults | [Supports](#supports) |

### Example

Request:

```
curl -XGET http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/supports
```

Response:

```
{"users":true,"settings":true,"describeDatabases":false}
```

## Health

```
GET /health
```

### Description

This API provides information about OpenSearch and DBaaS aggregator health statuses.

### Responses

| HTTP Code | Description                                     | Schema                        |
|-----------|-------------------------------------------------|-------------------------------|
| **200**   | Provided values would be used instead defaults  | [HealthStatus](#healthstatus) |
| **500**   | Error occurred while getting health information | string                        |

### Example

Request:

```
curl -XGET http://dbaas-opensearch-adapter:8080/health
```

Response:

```
{"status":"UP","opensearchHealth":{"status":"UP"},"dbaasAggregatorHealth":{"status":"OK"}}
```

## Create Database
```

POST /api/v1/dbaas/adapter/opensearch/databases
```

### Description

This API creates user with permissions to read and write to the set of entities with generated prefix. The operation returns connection parameters including generated prefix, username, password.

In the terms of DBaaS OpenSearch the database is a logical scope of entities (indices, aliases and templates) with the same prefix name.
When you create database the DBaaS adapter generates prefix and creates user with right for all entities of generated prefix name.

This API does not assume that database is an index, how it was in previous approach. You still have an ability to use previous API in [Create Database Old](#create-database-old).

### Parameters

| Type     | Name                              | Description                                               | Schema                              |
|----------|-----------------------------------|-----------------------------------------------------------|-------------------------------------|
| **Body** | **createRequest**  <br>*required* | The model for adding the OpenSearch database in the DBaaS | [DBCreateRequest](#dbcreaterequest) |

To init this option the request parameter `settings.resourcePrefix` must be `true`.

**Note**: It is not possible to specify `username` or `namePrefix` for this API. They are always generated automatically for security reasons.

### Responses

| HTTP Code | Description                                          | Schema                              |
|-----------|------------------------------------------------------|-------------------------------------|
| **201**   | Database is created                                  | [CreatedDatabase](#createddatabase) |
| **400**   | Provided `namePrefix` does not meet the requirements | string                              |
| **500**   | Error occurred while creating database               | string                              |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/databases -d'
{
  "settings": {
    "resourcePrefix": true,
    "createOnly": [
      "user"
    ]
  }
}'
```

Response:

```
{
  "name": "",
  "connectionProperties": {
    "dbName": "",
    "host": "opensearch",
    "port": 443,
    "url": "https://opensearch:9200/",
    "username": "578b078c-925f-490e-bf65-39cf374747c0",
    "password": "nU!FwA4XvL",
    "resourcePrefix": "578b078c-925f-490e-bf65-39cf374747c0"
  },
  "resources": [
    {
      "kind": "resourcePrefix",
      "name": "578b078c-925f-490e-bf65-39cf374747c0"
    },
    {
      "kind": "user",
      "name": "578b078c-925f-490e-bf65-39cf374747c0"
    },
    {
      "kind": "role",
      "name": "578b078c-925f-490e-bf65-39cf374747c0_role"
    }
  ]
}
```

## Create Database Old

```
POST /api/v1/dbaas/adapter/opensearch/databases
```

### Description

This API creates database and user with permissions to read and write to the database. The operation returns connection parameters including database name, username, password.

### Parameters

| Type     | Name                              | Description                                               | Schema                              |
|----------|-----------------------------------|-----------------------------------------------------------|-------------------------------------|
| **Body** | **createRequest**  <br>*required* | The model for adding the OpenSearch database in the DBaaS | [DBCreateRequest](#dbcreaterequest) |

### Responses

| HTTP Code | Description                                          | Schema                              |
|-----------|------------------------------------------------------|-------------------------------------|
| **201**   | Database is created                                  | [CreatedDatabase](#createddatabase) |
| **400**   | Provided `namePrefix` does not meet the requirements | string                              |
| **500**   | Error occurred while creating database               | string                              |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/databases -d'{
  "dbName": "index_name",
  "metadata": {
    "key": "value"
  },
  "namePrefix": "dbaas_prefix",
  "password": "pass",
  "settings": {
    "indexSettings": {
      "settings": {
        "index": {
          "number_of_shards": 5,
          "number_of_replicas": 1,
          "blocks.write": true
        }
      }
    }
  },
  "username": "new-user"
}'
```

Response:

```
{"name":"dbaas_prefix-index_name","connectionProperties":{"dbName":"dbaas_prefix-index_name","host":"opensearch","port":9200,"url":"http://opensearch:9200/dbaas_prefix-index_name","username":"new-user","password":"pass"},"resources":[{"kind":"index","name":"dbaas_prefix-index_name"},{"kind":"metadataDocument","name":"dbaas_prefix-index_name"},{"kind":"role","name":"dbaas_prefix-index_name-role"},{"kind":"user","name":"new-user"}]}
```

## List Databases

```
GET /api/v1/dbaas/adapter/opensearch/databases
```

### Description

This API returns list of database names excluding service ones.

### Responses

| HTTP Code | Description                            | Schema       |
|-----------|----------------------------------------|--------------|
| **200**   | List of database names                 | list<string> |
| **500**   | Error occurred while finding databases | string       |

### Example

Request:

```
curl -u <username>:<password> -XGET http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/databases
```

Response:

```
["dbaas_opensearch_metadata","testmine","test-newsty","test-new","dbaas_metadata","test-news","testme","dbaas_prefix-index_name"]
```

## Update Database Metadata

```
PUT /api/v1/dbaas/adapter/opensearch/databases/{dbName}/metadata
```

### Description

This API changes metadata for `{dbName}` specified in `dbaas_opensearch_metadata` index.

### Parameters

| Type     | Name                        | Description                                 | Schema              |
|----------|-----------------------------|---------------------------------------------|---------------------|
| **Path** | **dbName** <br>*required*   | Database name to update                     | string              |
| **Body** | **metadata** <br>*required* | JSON object with information about database | map<string, object> |

### Responses

| HTTP Code | Description                            | Schema |
|-----------|----------------------------------------|--------|
| **200**   | Metadata update is successful          | string |
| **500**   | Error occurred while updating metadata | string |

### Example

Request:

```
curl -u <username>:<password> -XPUT http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/databases/testmine/metadata -d'{
  "doc": {
    "data": "new data"
  }
}'
```

## Create User with Generated Name

```
PUT /api/v1/dbaas/adapter/opensearch/users
```

### Description

This API creates or updates OpenSearch user with read/write access to specific database. This operation returns connection parameters including database name, username, password. If user exists, password will be changed to the specified one.

### Parameters

| Type     | Name                            | Description                                        | Schema                                  |
|----------|---------------------------------|----------------------------------------------------|-----------------------------------------|
| **Body** | **userRequest**  <br>*optional* | Data used to create a user with appropriate grants | [UserCreateRequest](#usercreaterequest) |

### Responses

| HTTP Code | Description                        | Schema                      |
|-----------|------------------------------------|-----------------------------|
| **201**   | User is successfully created       | [CreatedUser](#createduser) |
| **500**   | Error occurred while user creation | string                      |

### Example

Request:

```
curl -u <username>:<password> -XPUT http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/users -d'{
  "dbName": "test-newsty",
  "password": "psswrd"
}'
```

Response:

```
{"connectionProperties":{"dbName":"test-newsty","host":"opensearch","port":9200,"url":"http://opensearch:9200/test-newsty","username":"dbaas_c71f1a63193c40328281e4901efb647f","password":"psswrd"},"name":"test-newsty","resources":[{"kind":"role","name":"test-newsty-role"},{"kind":"index","name":"test-newsty"},{"kind":"user","name":"dbaas_c71f1a63193c40328281e4901efb647f"}]}
```

## Create User with Specified Name

```
PUT /api/v1/dbaas/adapter/opensearch/users/{name}
```

### Description

This API creates or updates OpenSearch user with read/write access to specific database. This operation returns connection parameters including database name, username, password. If user exists, password will be changed to the specified one.

### Parameters

| Type     | Name                            | Description                                        | Schema                                  |
|----------|---------------------------------|----------------------------------------------------|-----------------------------------------|
| **Path** | **name**  <br>*required*        | The name of user to create or update               | string                                  |                      
| **Body** | **userRequest**  <br>*optional* | Data used to create a user with appropriate grants | [UserCreateRequest](#usercreaterequest) |

### Responses

| HTTP Code | Description                        | Schema                      |
|-----------|------------------------------------|-----------------------------|
| **201**   | User is successfully created       | [CreatedUser](#createduser) |
| **500**   | Error occurred while user creation | string                      |

### Example

Request:

```
curl -u <username>:<password> -XPUT http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/users/usertest -d'{
  "dbName": "test-news"
}'
```

Response:

```
{"connectionProperties":{"dbName":"test-news","host":"opensearch","port":9200,"url":"http://opensearch:9200/test-news","username":"usertest","password":"a02e2104fead496c8a4c6ef84c4ae70b"},"name":"test-news","resources":[{"kind":"role","name":"test-news-role"},{"kind":"index","name":"test-news"},{"kind":"user","name":"usertest"}]}
```

## Recover Users

```
POST /api/v2/dbaas/adapter/opensearch/users/restore-password
```

### Description

This API runs the OpenSearch users recovery process which creates or updates users passed in the body.

### Parameters

| Type     | Name                          | Description                              | Schema                            |
|----------|-------------------------------|------------------------------------------|-----------------------------------|
| **Body** | **resources**  <br>*required* | Data used to recover users in OpenSearch | [UsersToRecover](#userstorecover) |

### Responses

| HTTP Code | Description                                                   |
|-----------|---------------------------------------------------------------|
| **200**   | The OpenSearch users recovery process is successfully started |
| **500**   | Error occurred while running the recovery process             |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v2/dbaas/adapter/opensearch/users/restore-password -d'{
  "settings": {},
  "connectionProperties": [
    {
      "dbName": "",
      "host": "opensearch.opensearch-service",
      "port": 9200,
      "url": "https://opensearch.opensearch-service:9200",
      "username": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45-admin-user",
      "password": "dmnpsswrd",
      "resourcePrefix": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45",
      "role": "admin",
      "tls": true
    },
    {
      "dbName": "",
      "host": "opensearch.opensearch-service",
      "port": 9200,
      "url": "https://opensearch.opensearch-service:9200",
      "username": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45-dml-user",
      "password": "dmlpsswrd",
      "resourcePrefix": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45",
      "role": "dml",
      "tls": true
    },
    {
      "dbName": "",
      "host": "opensearch.opensearch-service",
      "port": 9200,
      "url": "https://opensearch.opensearch-service:9200",
      "username": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45-readonly-user",
      "password": "rdnlpsswrd",
      "resourcePrefix": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45",
      "role": "readonly",
      "tls": true
    },
    {
      "dbName": "",
      "host": "opensearch.opensearch-service",
      "port": 9200,
      "url": "https://opensearch.opensearch-service:9200",
      "username": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45-ism-user",
      "password": "smpsswrd",
      "resourcePrefix": "7a84ddf6-4f26-4282-94ba-bb13e44a3d45",
      "role": "ism",
      "tls": true
    }
  ]
}'
```

## Users Recovery State

```
GET /api/v2/dbaas/adapter/opensearch/users/restore-password/state
```

### Description

This API returns the current state of the OpenSearch users recovery process.

### Responses

| HTTP Code | Description                                                                                | Schema |
|-----------|--------------------------------------------------------------------------------------------|--------|
| **200**   | The state of recovery process. The possible values are `idle`, `running`, `failed`, `done` | string |

### Example

Request:

```
curl -u <username>:<password> -XGET /api/v2/dbaas/adapter/opensearch/users/restore-password/state
```

Response:

```
running
```

## Drop Created Resources

```
POST /api/v1/dbaas/adapter/opensearch/resources/bulk-drop
```

### Description

This API deletes any previously created resources such as user or database.

### Parameters

| Type     | Name                          | Description         | Schema                          |
|----------|-------------------------------|---------------------|---------------------------------|
| **Body** | **resources**  <br>*required* | Resources to delete | list<[DBResource](#dbresource)> |

### Responses

| HTTP Code | Description                             | Schema                                      |
|-----------|-----------------------------------------|---------------------------------------------|
| **200**   | All resources are successfully deleted  | list<[DBResourceDeleteStatus](#dbresource)> |
| **500**   | Error occurred while removing resources | list<[DBResourceDeleteStatus](#dbresource)> |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/resources/bulk-drop -d'[
  {
    "kind": "role",
    "name": "test-newsty-role"
  },
  {
    "kind": "index",
    "name": "test-newsty"
  },
  {
    "kind": "user",
    "name": "dbaas_c71f1a63193c40328281e4901efb647f"
  }
]'
```

Response:

```
[{"kind":"role","name":"test-newsty-role","status":"DELETED","errorMessage":""},{"kind":"user","name":"dbaas_c71f1a63193c40328281e4901efb647f","status":"DELETED","errorMessage":""},{"kind":"index","name":"test-newsty","status":"DELETED","errorMessage":""}]
```

## Collect Backup

```
POST /api/v1/dbaas/adapter/opensearch/backups/collect
```

### Description

This API requests to collect backup for specified database prefixes.

### Parameters

| Type     | Name                          | Description                         | Schema       |
|----------|-------------------------------|-------------------------------------|--------------|
| **Body** | **databases**  <br>*required* | List of database prefixes to backup | list<string> |

### Responses

| HTTP Code | Description                            | Schema                      |
|-----------|----------------------------------------|-----------------------------|
| **202**   | Backup is in progress                  | [ActionTrack](#actiontrack) |
| **500**   | Error occurred while collecting backup | string                      |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/backups/collect -d '["db1"]'
```

Response:

```
{"action":"BACKUP","details":{"localId":"20240322T091826"},"status":"PROCEEDING","trackId":"20240322T091826","changedNameDb":null,"trackPath":null}
```

## Track Backup

```
GET /api/v1/dbaas/adapter/opensearch/backups/track/backup/{trackId}
```

### Description

This API provides information about requested backup action.

### Parameters

| Type     | Name                        | Description                          | Schema |
|----------|-----------------------------|--------------------------------------|--------|
| **Path** | **trackId**  <br>*required* | Identifier to track backup procedure | string |

### Responses

| HTTP Code | Description                          | Schema                      |
|-----------|--------------------------------------|-----------------------------|
| **200**   | Information about backup action      | [ActionTrack](#actiontrack) |
| **500**   | Error occurred while tracking backup | string                      |

### Example

Request:

```
curl -u <username>:<password> -XGET http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/backups/track/backup/dbaas_2022_04_07_t_14_50_02_339486
```

Response:

```
{"action":"BACKUP","details":{"localId":"dbaas_2022_04_07_t_14_50_02_339486"},"status":"SUCCESS","trackId":"dbaas_2022_04_07_t_14_50_02_339486","changedNameDb":null,"trackPath":null}
```

## Restore Backup

```
POST /api/v1/dbaas/adapter/opensearch/backups/{backupId}/restore
```

### Description

This API requests to restore backup for specified databases.

### Parameters

| Type      | Name                                | Description                                                                                                                                                                                                                                                        | Schema       |
|-----------|-------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------|
| **Path**  | **backupId**  <br>*required*        | Backup identifier to be restored                                                                                                                                                                                                                                   | string       |
| **Query** | **regenerateNames**  <br>*optional* | Whether adapter should generate names for each restoring database, and restore databases under new names, which would effectively `clone` databases from backup. This action MUST NOT affect any of `source` databases whether they are present in cluster or not. | boolean      |
| **Body**  | **databases**  <br>*optional*       | List of database prefixes to restore                                                                                                                                                                                                                               | list<string> |

### Responses

| HTTP Code | Description                           | Schema                      |
|-----------|---------------------------------------|-----------------------------|
| **202**   | Restore is in progress                | [ActionTrack](#actiontrack) |
| **500**   | Error occurred while restoring backup | string                      |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/backups/20240322T091826/restore -d'["db1"]'
```

Response:

```
{"action":"RESTORE","details":{"localId":"20240322T091826"},"status":"PROCEEDING","trackId":"20240322T091826","changedNameDb":null,"trackPath":null}
```

## Track Restore From Track ID

```
GET /api/v1/dbaas/adapter/opensearch/backups/track/restore/{trackId}
```

### Description

This API provides information about requested restore action.

### Parameters

| Type     | Name                        | Description                           | Schema |
|----------|-----------------------------|---------------------------------------|--------|
| **Path** | **trackId**  <br>*required* | Identifier to track restore procedure | string |

### Responses

| HTTP Code | Description                           | Schema                      |
|-----------|---------------------------------------|-----------------------------|
| **200**   | Information about restore action      | [ActionTrack](#actiontrack) |
| **500**   | Error occurred while tracking restore | string                      |

### Example

Request:

```
curl -u <username>:<password> -XGET http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/backups/track/restore/20240322T091826
```

Response:

```
{"action":"RESTORE","details":{"localId":"20240322T091826"},"status":"SUCCESS","trackId":"20240322T091826","changedNameDb":null,"trackPath":null}
```

## Track Restore From Indices

```
GET /api/v1/dbaas/adapter/opensearch/backups/track/restoring/backups/{trackId}/indices/{indices}
```

### Description

This API provides information about requested restore action.

### Parameters

| Type     | Name                        | Description                              | Schema |
|----------|-----------------------------|------------------------------------------|--------|
| **Path** | **trackId**  <br>*required* | Identifier to track restore procedure    | string |
| **Path** | **indices**  <br>*required* | Indices which recovery should be tracked | string |

### Responses

| HTTP Code | Description                           | Schema                      |
|-----------|---------------------------------------|-----------------------------|
| **200**   | Information about restore action      | [ActionTrack](#actiontrack) |
| **500**   | Error occurred while tracking restore | string                      |

### Example

Request:

```
curl -u <username>:<password> -XGET http://dbaas-opensearch-adapter:8080/api/v1/dbaas/adapter/opensearch/backups/track/restoring/backups/20240322T091826/indices/testme,testmine
```

Response:

```
{"action":"RESTORE","details":{"localId":"20240322T091826"},"status":"SUCCESS","trackId":"20240322T091826","changedNameDb":null,"trackPath":null}
```

## Create Database v2
```

POST /api/v2/dbaas/adapter/opensearch/databases
```

### Description

This API creates users with permissions according to dbaas aggregator roles with generated prefix (`admin`, `dml`, `readonly` and `ism` by default). The operation returns list of connection parameters including generated prefix, usernames, passwords.

In the terms of DBaaS OpenSearch the database is a logical scope of entities (indices, aliases and templates) with the same prefix name.
When you create database the DBaaS adapter generates prefix and creates users with rights for all entities of generated prefix name. All users and roles created during this operation have the same `resourcePrefix`.

### Parameters

| Type     | Name                              | Description                                               | Schema                              |
|----------|-----------------------------------|-----------------------------------------------------------|-------------------------------------|
| **Body** | **createRequest**  <br>*required* | The model for adding the OpenSearch database in the DBaaS | [DBCreateRequest](#dbcreaterequest) |

To init this option the request parameter `settings.resourcePrefix` must be `true`.

**Note**: It is not possible to specify `username` or `namePrefix` for this API. They are always generated automatically for security reasons.

### Responses

| HTTP Code | Description                                          | Schema                              |
|-----------|------------------------------------------------------|-------------------------------------|
| **201**   | Database is created                                  | [CreatedDatabase](#createddatabase) |
| **400**   | Provided `namePrefix` does not meet the requirements | string                              |
| **500**   | Error occurred while creating database               | string                              |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v2/dbaas/adapter/opensearch/databases -d'
{
  "settings": {
    "resourcePrefix": true,
    "createOnly": [
      "user"
    ]
  },
  "metadata": {
    "classifier": {
       "namespace": "test-namespace"
   },
    "microserviceName": "test-service"
  }
}'
```

Response:

```
{
  "name": "",
  "connectionProperties": [
    {
      "dbName": "",
      "host": "opensearch.opensearch-service-v2",
      "port": 9200,
      "url": "https://opensearch.opensearch-service-v2:9200/",
      "username": "test-service_test-namespace_115500463160424_d2eece51b026479ca02fdcdee8712c27",
      "password": "THkP8pbC_I",
      "resourcePrefix": "test-service_test-namespace_115500463160424",
      "role": "readonly"
    },
    {
      "dbName": "",
      "host": "opensearch.opensearch-service-v2",
      "port": 9200,
      "url": "https: //opensearch.opensearch-service-v2:9200/",
      "username": "test-service_test-namespace_115500463160424_b35a8109fe204de48d1bf281d6a7a7af",
      "password": "CNwKIW#t9O",
      "resourcePrefix": "test-service_test-namespace_115500463160424",
      "role": "dml"
    },
    {
      "dbName": "",
      "host": "opensearch.opensearch-service-v2",
      "port": 9200,
      "url": "https://opensearch.opensearch-service-v2:9200/",
      "username": "test-service_test-namespace_115500463160424_0d74f547bc474da0a16a59da1f0cb56a",
      "password": "QghV6vlOD#",
      "resourcePrefix": "test-service_test-namespace_115500463160424",
      "role": "admin"
    },
    {
      "dbName": "",
      "host": "opensearch.opensearch-service-v2",
      "port": 9200,
      "url": "https://opensearch.opensearch-service-v2:9200/",
      "username": "test-service_test-namespace_111220703270524_3727c2a0f6524d97baae31be10848f68",
      "password": "oL#3xuAmBi",
      "resourcePrefix": "test-service_test-namespace_115500463160424",
      "role": "ism"
    }
  ],
  "resources": [
    {
      "kind": "resourcePrefix",
      "name": "test-service_test-namespace_115500463160424"
    },
    {
      "kind": "user",
      "name": "test-service_test-namespace_115500463160424_d2eece51b026479ca02fdcdee8712c27"
    },
    {
      "kind": "user",
      "name": "test-service_test-namespace_115500463160424_b35a8109fe204de48d1bf281d6a7a7af"
    },
    {
      "kind": "user",
      "name": "test-service_test-namespace_115500463160424_0d74f547bc474da0a16a59da1f0cb56a"
    },
    {
      "kind": "user",
      "name": "test-service_test-namespace_111220703270524_3727c2a0f6524d97baae31be10848f68"
    },
    {
      "kind": "metadataDocument",
      "name": "test-service_test-namespace_115500463160424"
    }
  ]
} 
```

## Drop Created Resources v2

```
POST /api/v2/dbaas/adapter/opensearch/resources/bulk-drop
```

### Description

This API deletes any previously created resources such as user or database. If `resourcePrefix` provided for deletion, all users and roles created during database creating deleted by prefix.

### Parameters

| Type     | Name                          | Description         | Schema                          |
|----------|-------------------------------|---------------------|---------------------------------|
| **Body** | **resources**  <br>*required* | Resources to delete | list<[DBResource](#dbresource)> |

### Responses

| HTTP Code | Description                             | Schema                                      |
|-----------|-----------------------------------------|---------------------------------------------|
| **200**   | All resources are successfully deleted  | list<[DBResourceDeleteStatus](#dbresource)> |
| **500**   | Error occurred while removing resources | list<[DBResourceDeleteStatus](#dbresource)> |

### Example

Request:

```
curl -u <username>:<password> -XPOST http://dbaas-opensearch-adapter:8080/api/v2/dbaas/adapter/opensearch/resources/bulk-drop -d'[
  {
    "kind": "role",
    "name": "test-newsty-role"
  },
  {
    "kind": "resourcePrefix",
    "name": "prefix"
  },
  {
    "kind": "user",
    "name": "dbaas_c71f1a63193c40328281e4901efb647f"
  }
]'
```

Response:

```
[{"kind":"role","name":"test-newsty-role","status":"DELETED","errorMessage":""},{"kind":"user","name":"dbaas_c71f1a63193c40328281e4901efb647f","status":"DELETED","errorMessage":""},{"kind":"user","name":"prefix-user","status":"DELETED","errorMessage":""},{"kind":"role","name":"prefix-role","status":"DELETED","errorMessage":""}]
```

# Definitions

## RegistrationPhysicalRequest

| Name                      | Description                                                                                                                                                                                          | Schema              |
|---------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------|
| **id** <br>*required*     | Physical database identifier. The parameter is permanent and specified during deployment.                                                                                                            | string              |
| **labels** <br>*optional* | Additional information that is sent when handshake process is done. The information is read from file which is located in `/app/config/dbaas.physical_databases.registration.labels.json` directory. | map<string, string> |

## Supports

| Name                                  | Description                                                                                             | Schema  |
|---------------------------------------|---------------------------------------------------------------------------------------------------------|---------|
| **describeDatabases**  <br>*required* | Identifies whether the adapter supports databases description endpoint. By default, it is not supported | boolean |
| **settings**  <br>*required*          | Identifies whether the adapter supports `settings` field in database creation request.                  | boolean |
| **users**  <br>*required*             | Identifies whether the adapter supports user creation endpoint.                                         | boolean |

## HealthStatus

| Name                                      | Description                                                                                      | Schema              |
|-------------------------------------------|--------------------------------------------------------------------------------------------------|---------------------|
| **dbaasAggregatorHealth**  <br>*required* | DBaaS aggregator health status. The possible values are as follows: `OK`, `PROBLEM`, `UNKNOWN`   | map<string, string> |
| **opensearchHealth**  <br>*required*      | OpenSearch health status. The possible values are as follows: `DOWN`, `PROBLEM`, `UP`, `WARNING` | map<string, string> |
| **status**  <br>*required*                | Result of aggregation of DBaaS aggregator and OpenSearch health statuses                         | string              |

## DBCreateRequest

| Name                           | Description                                                                                                                                                                                           | Schema                |
|--------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------|
| **dbName**  <br>*optional*     | Database name to create. If it is not specified, it will be generated starting from `namePrefix`.                                                                                                     | string                |
| **metadata**  <br>*optional*   | JSON object with any information about database. Adapter saves it in `dbaas_opensearch_metadata` index.                                                                                               | object                |
| **namePrefix**  <br>*optional* | Name prefix is used to generate database name. If prefix is empty, it does not participate in the formation of the database name. If prefix is not specified, default value (`dbaas`) is used. | string                |
| **password**  <br>*optional*   | Password for user to be created with the database. If password is not specified, it will be generated.                                                                                                | string                |
| **settings**  <br>*optional*   | Additional settings to create database.                                                                                                                                                               | [Settings](#settings) |
| **username**  <br>*optional*   | Username for user to be created with the database. If username is not specified, it is generated as follows: `dbaas_UUID`.                                                                            | string                |

## Settings

| Name                               | Description                                                                                                                                                | Schema              |
|------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------|
| **createOnly**  <br>*optional*     | List of resource types to create. The possible values are `user` and `index`. For example, `["user", "index"]`                                             | list<string>        |
| **indexSettings**  <br>*optional*  | Creation parameters map for the database: [Index Settings](https://opensearch.org/docs/latest/opensearch/rest-api/index-apis/create-index/#index-settings) | map<string, string> |
| **resourcePrefix**  <br>*optional* | Whether to generate prefix for all created resources. Must be `true` for [Create Database](#create-database).                                              | boolean             |

## CreatedDatabase

| Name                                     | Description                                                                     | Schema                                        |
|------------------------------------------|---------------------------------------------------------------------------------|-----------------------------------------------|
| **connectionProperties**  <br>*optional* | Properties to connect to database                                               | [ConnectionProperties](#connectionproperties) |
| **name**  <br>*optional*                 | Name of created database                                                        | string                                        |
| **resources**  <br>*optional*            | List of resources created during database creation and used during its deletion | list<[DbResource](#dbresource)>               |

## CreatedDatabase v2

| Name                                     | Description                                                                     | Schema                                        |
|------------------------------------------|---------------------------------------------------------------------------------|-----------------------------------------------|
| **connectionProperties**  <br>*optional* | List of properties to connect to database                                               | [ConnectionProperties](#connectionproperties) |
| **name**  <br>*optional*                 | Name of created database                                                        | string                                        |
| **resources**  <br>*optional*            | List of resources created during database creation and used during its deletion | list<[DbResource](#dbresource)>               |


## UserCreateRequest

| Name                         | Description                                                                                                   | Schema |
|------------------------------|---------------------------------------------------------------------------------------------------------------|--------|
| **dbName**  <br>*optional*   | Database to grant read/write access to. If it is not specified, user will be created without any permissions. | string |
| **password**  <br>*optional* | Password for user to be created or updated. If password is absent, it will be generated.                      | string |

## CreatedUser

| Name                                     | Description                                                                                                | Schema                                        |
|------------------------------------------|------------------------------------------------------------------------------------------------------------|-----------------------------------------------|
| **connectionProperties**  <br>*optional* | Properties to connect to database with created user                                                        | [ConnectionProperties](#connectionproperties) |
| **name**  <br>*optional*                 | Name of database accessed by created or updated user. If it is not requested, database name will be `null` | string                                        |
| **resources**  <br>*optional*            | List of resources created during user creation                                                             | list<[DbResource](#dbresource)>               |

## UsersToRecover

| Name                                     | Description                                          | Schema                                        |
|------------------------------------------|------------------------------------------------------|-----------------------------------------------|
| **connectionProperties**  <br>*required* | Properties to connect to database with specific user | [ConnectionProperties](#connectionproperties) |
| **settings**  <br>*optional*             | Additional settings to recover users                 | map[string]string                             |

## ConnectionProperties

| Name                               | Description                                                    | Schema         |
|------------------------------------|----------------------------------------------------------------|----------------|
| **host**  <br>*optional*           | Hostname of OpenSearch cluster where database has been created | string         |
| **name**  <br>*optional*           | Name of created database                                       | string         |
| **password**  <br>*optional*       | Password of created user with read/write access to database    | string         |
| **port**  <br>*optional*           | Port of OpenSearch cluster where database has been created     | integer(int32) |
| **url**  <br>*optional*            | URL to connect to database                                     | string         |
| **username**  <br>*optional*       | Username of created user with read/write access to database    | string         |
| **resourcePrefix**  <br>*optional* | Generated prefix that is used for created resources            | string         |

## ConnectionProperties v2

| Name                               | Description                                                    | Schema         |
|------------------------------------|----------------------------------------------------------------|----------------|
| **host**  <br>*optional*           | Hostname of OpenSearch cluster where database has been created | string         |
| **name**  <br>*optional*           | Name of created database                                       | string         |
| **password**  <br>*optional*       | Password of created user with read/write access to database    | string         |
| **port**  <br>*optional*           | Port of OpenSearch cluster where database has been created     | integer(int32) |
| **url**  <br>*optional*            | URL to connect to database                                     | string         |
| **username**  <br>*optional*       | Username of created user with read/write access to database    | string         |
| **resourcePrefix**  <br>*optional* | Generated prefix that is used for created resources            | string         |
| **role**  <br>*optional*           | Role provided to user for data access                          | string         |

## DBResource

| Name                     | Description                                                                                                                   | Schema |
|--------------------------|-------------------------------------------------------------------------------------------------------------------------------|--------|
| **kind**  <br>*optional* | Kind of resource. Possible values are as follows: `index`, `metadataDocument`, `user`, `role`, `resourcePrefix`               | string |
| **name**  <br>*required* | Name of the resource. If `kind` is `resourcePrefix`, value should contain prefix for resources to delete. For example, `test` | string |

## DBResourceDeleteStatus

| Name                            | Description                                                                                                                         | Schema |
|---------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|--------|
| **errorMessage** <br>*optional* | Message of error occurred during resource deletion                                                                                  | string |                          
| **kind**  <br>*optional*        | Kind of resource. Possible values are as follows: `index`, `metadataDocument`, `user`, `role`, `template`, `indexTemplate`, `alias` | string |
| **name**  <br>*required*        | Name of the resource                                                                                                                | string |
| **status** <br>*optional*       | Resource deletion status                                                                                                            | string |

## ActionTrack

| Name                              | Description                                                                                                                                                                                               | Schema                          |
|-----------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------|
| **action** <br>*optional*         | Type of action. The possible values are as follows: `BACKUP`, `RESTORE`                                                                                                                                   | string                          |                          
| **changedNameDb**  <br>*optional* | If the parameter `regenerateNames` is passed with value `true`, this field should contain associative array, where `key` is name of backup database, `value` is a new name of database with the same data | map<string, string>             |
| **details**  <br>*optional*       | Additional information about running procedure                                                                                                                                                            | [Details](#details)             |
| **status** <br>*optional*         | Processing status                                                                                                                                                                                         | enum(FAIL, SUCCESS, PROCEEDING) |
| **trackId** <br>*optional*        | Identifier to track the process                                                                                                                                                                           | string                          |

## Details

| Name                       | Description                    | Schema |
|----------------------------|--------------------------------|--------|
| **localId** <br>*optional* | Identifier of backup procedure | string |                          
