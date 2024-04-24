*** Variables ***
${DBAAS_ADAPTER_TYPE}                    %{DBAAS_ADAPTER_TYPE}
${OPENSEARCH_DBAAS_ADAPTER_HOST}         %{OPENSEARCH_DBAAS_ADAPTER_HOST}
${OPENSEARCH_DBAAS_ADAPTER_PORT}         %{OPENSEARCH_DBAAS_ADAPTER_PORT}
${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}     %{OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}
${OPENSEARCH_DBAAS_ADAPTER_USERNAME}     %{OPENSEARCH_DBAAS_ADAPTER_USERNAME}
${OPENSEARCH_DBAAS_ADAPTER_PASSWORD}     %{OPENSEARCH_DBAAS_ADAPTER_PASSWORD}
${OPENSEARCH_DBAAS_ADAPTER_REPOSITORY}   %{OPENSEARCH_DBAAS_ADAPTER_REPOSITORY}
${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}  %{OPENSEARCH_DBAAS_ADAPTER_API_VERSION=v1}
${RETRY_TIME}                            20s
${RETRY_INTERVAL}                        1s

*** Settings ***
Library  DateTime
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Prepare

*** Keywords ***
Prepare
    Prepare OpenSearch
    Prepare Dbaas Adapter

Prepare Dbaas Adapter
    ${auth}=  Create List  ${OPENSEARCH_DBAAS_ADAPTER_USERNAME}  ${OPENSEARCH_DBAAS_ADAPTER_PASSWORD}
    ${root_ca_path}=  Set Variable  /certs/dbaas-adapter/ca.crt
    ${root_ca_exists}=  File Exists  ${root_ca_path}
    ${verify}=  Set Variable If  ${root_ca_exists}  ${root_ca_path}  ${True}
    Create Session  dbaassession  ${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}://${OPENSEARCH_DBAAS_ADAPTER_HOST}:${OPENSEARCH_DBAAS_ADAPTER_PORT}  auth=${auth}  verify=${verify}

Generate Name
    [Arguments]  ${name}
    ${prefix}=  Generate Random String  5  [LOWER]
    [Return]  dbaas_integration_test-${prefix}-${name}

Create Backup By Dbaas Agent
    [Arguments]  ${indices_list}
    ${response}=  Post Request  dbaassession  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/collect  data=${indices_list}  headers=${headers}
    ${content}=  Convert Json ${response.content} To Type
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${content['trackId']}
    [Return]  ${content['trackId']}

Delete Backup By Dbaas Agent
    [Arguments]  ${backup_id}
    ${response}=  Delete Request  dbaassession  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/${backup_id}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Restore Indices From Backup By Dbaas Agent
    [Arguments]  ${backup_id}  ${indices_list}
    ${response}=  Post Request  dbaassession  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/${backup_id}/restore  data=${indices_list}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status  ${content['trackId']}

Check Backup Status
    [Arguments]  ${backup_id}
    ${restore_status}=  Get Request  dbaassession  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/track/backup/${backup_id}
    Should Contain  str(${restore_status.content})  SUCCESS

Check Restore Status
    [Arguments]  ${backup_id}
    ${restore_status}=  Get Request  dbaassession  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/backups/track/restore/${backup_id}
    Should Contain  str(${restore_status.content})  SUCCESS

Delete OpenSearch Backup
    [Arguments]  ${backup_id}
    ${response}=  Delete Request  opensearch  /_snapshot/${OPENSEARCH_DBAAS_ADAPTER_REPOSITORY}/${backup_id}
    ${boolean_success_result}=  Evaluate  ${response.status_code} in [200, 404]
    Should Be True  ${boolean_success_result}

Check OpenSearch Backup Exists
    [Arguments]  ${backup_id}
    # DBaaS adapter manipulates backups via Curator.
    # The backup_id in backup-daemon and the real snapshot name in opensearch are different.
    ${opensearch_backup_id}=  Convert To Lowercase  ${backup_id}
    ${response}=  Get Request  opensearch  /_snapshot/${OPENSEARCH_DBAAS_ADAPTER_REPOSITORY}/${opensearch_backup_id}
    Should Be Equal As Strings  ${response.status_code}  200

Check OpenSearch Backup Does Not Exist
    [Arguments]  ${backup_id}
    ${response}=  Get Request  opensearch  /_snapshot/${OPENSEARCH_DBAAS_ADAPTER_REPOSITORY}/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  404

*** Test Cases ***
Create Backup By Dbaas Adapter
    [Tags]  dbaas  dbaas_backup  dbaas_create_backup
    ${index_name}=  Generate Name  dbaas-backup-index
    ${dbs_prefix}=  Set Variable  dbaas_integration_test
    Create OpenSearch Index  ${index_name}
    Sleep  5s  reason=Index should be created
    Check OpenSearch Index Exists  ${index_name}
    ${backup_id}=  Create Backup By Dbaas Agent  ["${dbs_prefix}"]
    Check OpenSearch Backup Exists  ${backup_id}
    [Teardown]  Run Keywords  Delete OpenSearch Backup  ${backup_id}
                ...  AND  Delete OpenSearch Index  ${index_name}

Delete Backup By Dbaas Adapter
    [Tags]  dbaas  dbaas_backup  dbaas_delete_backup
    ${index_name}=  Generate Name  dbaas-backup-index
    ${dbs_prefix}=  Set Variable  dbaas_integration_test
    Create OpenSearch Index  ${index_name}
    Check OpenSearch Index Exists  ${index_name}
    ${backup_id}=  Create Backup By Dbaas Agent  ["${dbs_prefix}"]
    Delete Backup By Dbaas Agent  ${backup_id}
    Check OpenSearch Backup Does Not Exist  ${backup_id}
    [Teardown]  Run Keywords  Delete OpenSearch Backup  ${backup_id}
                ...  AND  Delete OpenSearch Index  ${index_name}

Restore Backup By Dbaas Adapter
    [Tags]  dbaas  dbaas_backup  dbaas_restore_backup
    ${index_name_first}=  Generate Name  dbaas-restore-index-first
    ${index_name_second}=  Generate Name  dbaas-restore-index-second
    ${dbs_prefix}=  Set Variable  dbaas_integration_test
    Create OpenSearch Index  ${index_name_first}
    Create OpenSearch Index  ${index_name_second}
    ${document_first}=  Set Variable  {"age": "1", "name": "first"}
    ${document_second}=  Set Variable  {"age": "2", "name": "second"}
    Create Document ${document_first} For Index ${index_name_first}
    Create Document ${document_second} For Index ${index_name_second}

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Run Keywords
    ...  Check That Document Exists By Field  ${index_name_first}  age  1  AND
    ...  Check That Document Exists By Field  ${index_name_second}  age  2

    ${backup_id}=  Create Backup By Dbaas Agent  ["${dbs_prefix}"]

    ${update_document_second}=  Set Variable  {"surname": "surname"}

    Delete Document From Index By Id  ${index_name_first}  1
    Update Document ${update_document_second} For Index ${index_name_second}

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Run Keywords
    ...  Check That Document Does Not Exist By Field  ${index_name_first}  age  1  AND
    ...  Check That Document Exists By Field  ${index_name_second}  surname  surname

    Restore Indices From Backup By Dbaas Agent  ${backup_id}  ["${dbs_prefix}"]

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Run Keywords
    ...  Check That Document Does Not Exist By Field  ${index_name_second}  surname  surname  AND
    ...  Check That Document Exists By Field  ${index_name_first}  age  1

    [Teardown]  Run Keywords  Delete OpenSearch Backup  ${backup_id}
                ...  AND  Delete OpenSearch Index  ${index_name_first}
                ...  AND  Delete OpenSearch Index  ${index_name_second}