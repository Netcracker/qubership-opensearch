*** Variables ***
${OPENSEARCH_BACKUP_V2_INDEX}             opensearch_backup_v2_index
${BACKUP_STORAGE_NAME}                    s3
${BACKUP_BLOB_PATH}                       /backup-storage/v2
${BACKUP_BLOB_PATH_ALIAS_TEST}            /backup-storage/v2-alias-default
${S3_ALIASES_SECRET_NAME}                 %{S3_ALIASES_SECRET_NAME=opensearch-s3-aliases}
${S3_DEFAULT_ALIAS_NAME}                  %{S3_DEFAULT_ALIAS_NAME=default}
${DBAAS_ADAPTER_TYPE}                     %{DBAAS_ADAPTER_TYPE}
${OPENSEARCH_DBAAS_ADAPTER_HOST}          %{OPENSEARCH_DBAAS_ADAPTER_HOST}
${OPENSEARCH_DBAAS_ADAPTER_PORT}          %{OPENSEARCH_DBAAS_ADAPTER_PORT}
${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}      %{OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}
${RETRY_TIME}                             300s
${RETRY_INTERVAL}                         10s

*** Settings ***
Resource          ../shared/keywords.robot
Resource          backup_keywords.robot
Suite Setup       Prepare

*** Keywords ***
Prepare
    Prepare OpenSearch
    Prepare Dbaas Adapter
    Delete Data  ${OPENSEARCH_BACKUP_V2_INDEX}

Prepare Dbaas Adapter
    ${auth}=  Create List  ${OPENSEARCH_DBAAS_ADAPTER_USERNAME}  ${OPENSEARCH_DBAAS_ADAPTER_PASSWORD}
    ${root_ca_path}=  Set Variable  /certs/dbaas-adapter/ca.crt
    ${root_ca_exists}=  File Exists  ${root_ca_path}
    ${verify}=  Set Variable If  ${root_ca_exists}  ${root_ca_path}  ${True}
    Create Session  dbaas_v2_session  ${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}://${OPENSEARCH_DBAAS_ADAPTER_HOST}:${OPENSEARCH_DBAAS_ADAPTER_PORT}  auth=${auth}  verify=${verify}

Get Track Id
    [Arguments]  ${response_content}
    ${content}=  Convert Json ${response_content} To Type
    ${track_id}=  Evaluate  $content.get('backupId') or $content.get('restoreId') or $content.get('trackId') or $content.get('id')
    Should Not Be Equal  ${track_id}  ${None}
    RETURN  ${track_id}

Create Backup V2
    [Arguments]  ${database_name}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${data}=  Set Variable  {"storageName":"${BACKUP_STORAGE_NAME}","blobPath":"${blob_path}","databases":[{"databaseName":"${database_name}"}]}
    ${response}=  Post Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/backup  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  202
    ${backup_id}=  Get Track Id  ${response.content}
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status V2  ${backup_id}  ${blob_path}
    RETURN  ${backup_id}

Check Backup Status V2
    [Arguments]  ${backup_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${response}=  Get Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/backup/${backup_id}?blobPath=${blob_path}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Should Contain  str(${response.content})  completed

Delete Backup V2
    [Arguments]  ${backup_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${response}=  Delete Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/backup/${backup_id}?blobPath=${blob_path}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  204

Delete Backup V2 If Exists
    [Arguments]  ${backup_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    Run Keyword If  "${backup_id}" != "${None}"  Run Keyword And Ignore Error  Delete Backup V2  ${backup_id}  ${blob_path}

Ensure S3 Aliases Config Available
    ${secret_exists}=  Run Keyword And Return Status  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${OPENSEARCH_NAMESPACE}
    Pass Execution If  not ${secret_exists}  S3 aliases secret is absent, skip alias routing test
    ${secret}=  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${OPENSEARCH_NAMESPACE}
    ${has_alias_config}=  Evaluate  bool($secret.data) and 's3_aliases.json' in $secret.data and bool($secret.data['s3_aliases.json'])
    Pass Execution If  not ${has_alias_config}  S3 aliases config is empty, skip alias routing test

Get Default S3 Alias Config
    ${secret}=  Check Secret  ${S3_ALIASES_SECRET_NAME}  ${OPENSEARCH_NAMESPACE}
    ${aliases_base64}=  Set Variable  ${secret.data['s3_aliases.json']}
    ${aliases_json}=  Evaluate  base64.b64decode($aliases_base64).decode("utf-8")  modules=base64
    ${aliases}=  Convert Json ${aliases_json} To Type
    ${default_alias_name}=  Evaluate  next((name for name, cfg in $aliases.items() if cfg.get("default") is True), None)
    ${default_alias_name}=  Run Keyword If  "${default_alias_name}" == "${None}"  Set Variable  ${S3_DEFAULT_ALIAS_NAME}  ELSE  Set Variable  ${default_alias_name}
    ${default_alias}=  Evaluate  $aliases.get($default_alias_name)
    Should Not Be Equal  ${default_alias}  ${None}
    RETURN  ${default_alias}

Check Backup Exists In Default Alias Bucket
    [Arguments]  ${backup_id}  ${blob_path}=${BACKUP_BLOB_PATH}
    ${default_alias}=  Get Default S3 Alias Config
    ${s3_url}=  Evaluate  $default_alias.get("s3Url") or $default_alias.get("storageServerUrl")
    ${s3_bucket}=  Evaluate  $default_alias.get("bucketName") or $default_alias.get("storageBucket")
    ${s3_key_id}=  Evaluate  $default_alias.get("accessKeyId") or $default_alias.get("storageUsername")
    ${s3_key_secret}=  Evaluate  $default_alias.get("accessKeySecret") or $S3_KEY_SECRET
    Should Not Be Empty  ${s3_url}
    Should Not Be Empty  ${s3_bucket}
    Should Not Be Empty  ${s3_key_id}
    Should Not Be Empty  ${s3_key_secret}
    Import Library  S3BackupLibrary  url=${s3_url}  bucket=${s3_bucket}  key_id=${s3_key_id}  key_secret=${s3_key_secret}  WITH NAME  DefaultAliasS3
    ${backup_file_exist}=  DefaultAliasS3.Check Backup Exists  path=${blob_path}  backup_id=${backup_id}
    Should Be True  ${backup_file_exist}

Check Backup Does Not Exist In Default Alias Bucket
    [Arguments]  ${backup_id}  ${blob_path}
    ${default_alias}=  Get Default S3 Alias Config
    ${s3_url}=  Evaluate  $default_alias.get("s3Url") or $default_alias.get("storageServerUrl")
    ${s3_bucket}=  Evaluate  $default_alias.get("bucketName") or $default_alias.get("storageBucket")
    ${s3_key_id}=  Evaluate  $default_alias.get("accessKeyId") or $default_alias.get("storageUsername")
    ${s3_key_secret}=  Evaluate  $default_alias.get("accessKeySecret") or $S3_KEY_SECRET
    Import Library  S3BackupLibrary  url=${s3_url}  bucket=${s3_bucket}  key_id=${s3_key_id}  key_secret=${s3_key_secret}  WITH NAME  DefaultAliasS3
    ${backup_file_exist}=  DefaultAliasS3.Check Backup Exists  path=${blob_path}  backup_id=${backup_id}
    Should Not Be True  ${backup_file_exist}

*** Test Cases ***
V2 Backup Uses Default S3 Alias Container
    [Tags]  opensearch  backup  backup_v2  backup_v2_aliases
    ${backup_id}=  Set Variable  ${None}
    Ensure S3 Aliases Config Available
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_V2_INDEX}
    ${backup_id}=  Create Backup V2  ${OPENSEARCH_BACKUP_V2_INDEX}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
    Check Backup Exists In Default Alias Bucket  ${backup_id}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
    Check Backup Does Not Exist In Default Alias Bucket  ${backup_id}  ${BACKUP_BLOB_PATH}
    Delete Backup V2  ${backup_id}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
    [Teardown]  Run Keywords  Delete Data  ${OPENSEARCH_BACKUP_V2_INDEX}  AND  Delete Backup V2 If Exists  ${backup_id}  ${BACKUP_BLOB_PATH_ALIAS_TEST}
