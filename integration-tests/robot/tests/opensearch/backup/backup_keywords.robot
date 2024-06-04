*** Variables ***
${OPENSEARCH_CURATOR_USERNAME}   %{OPENSEARCH_CURATOR_USERNAME}
${OPENSEARCH_CURATOR_PASSWORD}   %{OPENSEARCH_CURATOR_PASSWORD}
${OPENSEARCH_CURATOR_PROTOCOL}   %{OPENSEARCH_CURATOR_PROTOCOL}
${OPENSEARCH_CURATOR_HOST}       %{OPENSEARCH_CURATOR_HOST}
${OPENSEARCH_CURATOR_PORT}       %{OPENSEARCH_CURATOR_PORT}
${RETRY_TIME}                    300s
${RETRY_INTERVAL}                10s

*** Keywords ***
Prepare
    Prepare OpenSearch
    Prepare Curator
    Delete Data  ${OPENSEARCH_BACKUP_INDEX}

Prepare Curator
    ${auth}=  Create List  ${OPENSEARCH_CURATOR_USERNAME}  ${OPENSEARCH_CURATOR_PASSWORD}
    ${root_ca_path}=  Set Variable  /certs/curator/root-ca.pem
    ${root_ca_exists}=  File Exists  ${root_ca_path}
    ${verify}=  Set Variable If  ${root_ca_exists}  ${root_ca_path}  ${True}
    Create Session  curatorsession  ${OPENSEARCH_CURATOR_PROTOCOL}://${OPENSEARCH_CURATOR_HOST}:${OPENSEARCH_CURATOR_PORT}  auth=${auth}  verify=${verify}

Delete Data
    [Arguments]  ${prefix}
    Delete OpenSearch Index  ${prefix}
    Delete OpenSearch Index  ${prefix}-1
    Delete OpenSearch Index  ${prefix}-2
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Run Keywords
    ...  Check OpenSearch Index Does Not Exist  ${prefix}  AND
    ...  Check OpenSearch Index Does Not Exist  ${prefix}-1  AND
    ...  Check OpenSearch Index Does Not Exist  ${prefix}-2

Full Backup
    ${response}=  Post Request  curatorsession  /backup
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${response.content}
    [Return]  ${response.text}

Granular Backup
    [Arguments]  ${indices_list}
    ${data}=  Set Variable  {"dbs":${indices_list}}
    ${response}=  Post Request  curatorsession  /backup  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${response.content}
    [Return]  ${response.text}

Delete Backup
    [Arguments]  ${backup_id}
    ${response}=  Post Request  curatorsession  /evict/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200

Delete Backup If Exists
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    Run Keyword If    ${response.status_code}==200    Delete Backup  ${backup_id}

Full Restore
    [Arguments]  ${backup_id}  ${indices_list}
    ${restore_data}=  Set Variable  {"vault":"${backup_id}","dbs":${indices_list}}
    ${response}=  Post Request  curatorsession  /restore  data=${restore_data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status  ${response.content}

Full Restore By Timestamp
    [Arguments]  ${backup_ts}  ${indices_list}
    ${restore_data}=  Set Variable  {"ts":"${backup_ts}","dbs":${indices_list}}
    ${response}=  Post Request  curatorsession  /restore  data=${restore_data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status  ${response.content}

Get Backup Timestamp
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content['ts']}

Find Backup ID By Timestamp
    [Arguments]  ${backup_ts}
    ${find_data}=  Create Dictionary  ts=${backup_ts}
    ${response}=  Get Request  curatorsession  /find  json=${find_data}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content['id']}

Check Backup Status
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['failed']}  False

Check Restore Status
    [Arguments]  ${task_id}
    ${response}=  Get Request  curatorsession  /jobstatus/${task_id}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['status']}  Successful

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
