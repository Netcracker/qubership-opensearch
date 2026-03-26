*** Variables ***
${OPENSEARCH_BACKUP_V2_INDEX}             opensearch_backup_v2_index
${BACKUP_STORAGE_NAME}                    s3
${BACKUP_BLOB_PATH}                       /backup-storage/v2
${DBAAS_ADAPTER_TYPE}                     %{DBAAS_ADAPTER_TYPE}
${OPENSEARCH_DBAAS_ADAPTER_HOST}          %{OPENSEARCH_DBAAS_ADAPTER_HOST}
${OPENSEARCH_DBAAS_ADAPTER_PORT}          %{OPENSEARCH_DBAAS_ADAPTER_PORT}
${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}      %{OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}
${OPENSEARCH_DBAAS_ADAPTER_USERNAME}      %{OPENSEARCH_DBAAS_ADAPTER_USERNAME}
${OPENSEARCH_DBAAS_ADAPTER_PASSWORD}      %{OPENSEARCH_DBAAS_ADAPTER_PASSWORD}
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
    [Arguments]  ${database_name}
    ${data}=  Set Variable  {"storageName":"${BACKUP_STORAGE_NAME}","blobPath":"${BACKUP_BLOB_PATH}","databases":[{"databaseName":"${database_name}"}]}
    ${response}=  Post Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/backup  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  202
    ${backup_id}=  Get Track Id  ${response.content}
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status V2  ${backup_id}
    RETURN  ${backup_id}

Check Backup Status V2
    [Arguments]  ${backup_id}
    ${response}=  Get Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/backup/${backup_id}?blobPath=${BACKUP_BLOB_PATH}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Should Contain  str(${response.content})  completed

Restore Backup V2
    [Arguments]  ${backup_id}  ${database_name}
    ${data}=  Set Variable  {"storageName":"${BACKUP_STORAGE_NAME}","blobPath":"${BACKUP_BLOB_PATH}","databases":[{"microserviceName":"integration-tests","databaseName":"${database_name}","namespace":"${OPENSEARCH_NAMESPACE}"}]}
    ${response}=  Post Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/backup/${backup_id}/restore?dryRun=false  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  202
    ${restore_id}=  Get Track Id  ${response.content}
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status V2  ${restore_id}

Check Restore Status V2
    [Arguments]  ${restore_id}
    ${response}=  Get Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/restore/${restore_id}?blobPath=${BACKUP_BLOB_PATH}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Should Contain  str(${response.content})  completed

Delete Backup V2
    [Arguments]  ${backup_id}
    ${response}=  Delete Request  dbaas_v2_session  /api/v2/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/backup/${backup_id}?blobPath=${BACKUP_BLOB_PATH}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  204

Delete Backup V2 If Exists
    [Arguments]  ${backup_id}
    Run Keyword If  "${backup_id}" != "${None}"  Run Keyword And Ignore Error  Delete Backup V2  ${backup_id}

*** Test Cases ***
Full Backup And Restore V2
    [Tags]  opensearch  backup  backup_v2
    ${backup_id}=  Set Variable  ${None}
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_V2_INDEX}
    ${backup_id}=  Create Backup V2  ${OPENSEARCH_BACKUP_V2_INDEX}
    Delete Data  ${OPENSEARCH_BACKUP_V2_INDEX}
    Restore Backup V2  ${backup_id}  ${OPENSEARCH_BACKUP_V2_INDEX}
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_V2_INDEX}
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_V2_INDEX}  name  ${document_name}
    Delete Backup V2  ${backup_id}
    [Teardown]  Run Keywords  Delete Data  ${OPENSEARCH_BACKUP_V2_INDEX}  AND  Delete Backup V2 If Exists  ${backup_id}
