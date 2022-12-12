*** Variables ***
${OPENSEARCH_CURATOR_PROTOCOL}   %{OPENSEARCH_CURATOR_PROTOCOL}
${OPENSEARCH_CURATOR_HOST}       %{OPENSEARCH_CURATOR_HOST}
${OPENSEARCH_CURATOR_PORT}       %{OPENSEARCH_CURATOR_PORT}
${OPENSEARCH_CURATOR_USERNAME}   %{OPENSEARCH_CURATOR_USERNAME=}
${OPENSEARCH_CURATOR_PASSWORD}   %{OPENSEARCH_CURATOR_PASSWORD=}
${RETRY_TIME}                    120s
${RETRY_INTERVAL}                5s
${OPENSEARCH_BACKUP_INDEX}       opensearch_backup_index

*** Settings ***
Library  String
Library  Collections
Library  RequestsLibrary
Resource  ../shared/keywords.robot
Suite Setup  Prepare
Test Teardown  Delete Data

*** Keywords ***
Prepare
    Prepare OpenSearch
    Prepare Curator
    Delete Data

Prepare Curator
    ${auth}=  Create List  ${OPENSEARCH_CURATOR_USERNAME}  ${OPENSEARCH_CURATOR_PASSWORD}
    ${verify}=  Set Variable If  '${OPENSEARCH_CURATOR_PROTOCOL}' == 'https'  /certs/curator/root-ca.pem  ${True}
    Create Session  curatorsession  ${OPENSEARCH_CURATOR_PROTOCOL}://${OPENSEARCH_CURATOR_HOST}:${OPENSEARCH_CURATOR_PORT}  auth=${auth}  verify=${verify}

Delete Data
    Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}
    Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-1
    Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-2

Full Backup
    ${response}=  Post Request  curatorsession  /backup
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${response.content}
    [Return]  ${response.content}

Granular Backup
    ${data}=  Set Variable  {"dbs":["${OPENSEARCH_BACKUP_INDEX}-1","${OPENSEARCH_BACKUP_INDEX}-2"]}
    ${response}=  Post Request  curatorsession  /backup  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${response.content}
    [Return]  ${response.content}

Delete Backup
    [Arguments]  ${backup_id}
    ${response}=  Post Request  curatorsession  /evict/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200

Full Restore
    [Arguments]  ${backup_id}  ${indices_list}
    ${restore_data}=  Set Variable  {"vault":"${backup_id}","dbs":${indices_list}}
    ${response}=  Post Request  curatorsession  /restore  data=${restore_data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status  ${response}

Check Backup Status
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['failed']}  False

Check Restore Status
    [Arguments]  ${task_id}
    ${response}=  Get Request  curatorsession  /jobstatus/${task_id}
    Should Contain  str(${response.content})  Successful

Check Backup Absence By Curator
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  404

Check Backup Absence By OpenSearch
    [Arguments]  ${backup_id}
    ${backup_id_in_lowercase}=  Convert To Lowercase  ${backup_id}
    ${response}=  Get Request  opensearch  /_snapshot/snapshots/${backup_id_in_lowercase}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  404

Create Index With Generated Data
    [Arguments]  ${index_name}
    ${response}=  Create OpenSearch Index  ${index_name}
    Should Be Equal As Strings  ${response.status_code}  200
    Generate And Add Unique Data To Index  ${index_name}

Generate And Add Unique Data To Index
    [Arguments]  ${index_name}
    ${document_name}=  Generate Random String  10
    Set Global Variable  ${document_name}
    ${document}=  Set Variable  {"name": "${document_name}", "age": "10"}
    Create Document ${document} For Index ${index_name}

*** Test Cases ***
Full Backup And Restore
    [Tags]  opensearch  backup  full_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}
    ${backup_id}=  Full Backup

    Delete Data

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}  name  ${document_name}

Granular Backup And Restore
    [Tags]  opensearch  backup  granular_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup

    ${response}=  Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-1
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Set Variable  {"age": "1"}
    Update Document ${document} For Index ${OPENSEARCH_BACKUP_INDEX}-2

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}-1", "${OPENSEARCH_BACKUP_INDEX}-2"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-1
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-2
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}-2  age  10

Delete Backup By ID
    [Tags]  opensearch  backup  backup_deletion
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup
    Delete Backup  ${backup_id}
    Check Backup Absence By Curator  ${backup_id}
    Check Backup Absence By OpenSearch  ${backup_id}

Unauthorized Access
    [Tags]  opensearch  backup  unauthorized_access
    Create Session  curator_unauthorized  ${OPENSEARCH_CURATOR_PROTOCOL}://${OPENSEARCH_CURATOR_HOST}:${OPENSEARCH_CURATOR_PORT}
    ...  disable_warnings=1
    ${response}=  Post Request  curator_unauthorized  /backup
    Should Be Equal As Strings  ${response.status_code}  401