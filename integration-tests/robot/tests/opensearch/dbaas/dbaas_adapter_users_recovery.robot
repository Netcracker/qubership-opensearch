*** Variables ***
${RETRY_TIME}                            60s
${RETRY_INTERVAL}                        5s
${SLEEP_TIME}                            5s

*** Settings ***
Resource  ./keywords.robot
Suite Setup  Prepare

*** Keywords ***
Run Users Recovery By Dbaas Agent
    [Arguments]  ${properties}
    ${data}=  Set Variable  {"settings": {}, "connectionProperties": ${properties}}
    ${response}=  Post Request  dbaas_admin_session  api/v2/dbaas/adapter/opensearch/users/restore-password  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Get Users Recovery State By Dbaas Agent
    ${response}=  Get Request  dbaas_admin_session  api/v2/dbaas/adapter/opensearch/users/restore-password/state  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    [Return]  ${response.content}

Check Users Recovery State
    ${state}=  Get Users Recovery State By Dbaas Agent
    Should Be Equal As Strings  ${state}  done

*** Test Cases ***
Recover Users In OpenSearch
    [Tags]  dbaas  dbaas_opensearch  dbaas_recovery  dbaas_recover_users  dbaas_v2
    ${resource_prefix}=  Set Variable  860dde0d-dfcc-480a-9880-19533c5aa7aa
    ${admin_username}=  Set Variable  ${resource_prefix}-admin-user
    ${admin_password}=  Set Variable  dmnpsswrd
    ${dml_username}=  Set Variable  ${resource_prefix}-dml-user
    ${dml_password}=  Set Variable  dmlpsswrd
    ${readonly_username}=  Set Variable  ${resource_prefix}-readonly-user
    ${readonly_password}=  Set Variable  rdnlpsswrd
    ${admin_user}=  Set Variable  {"username": "${admin_username}", "password": "${admin_password}", "resourcePrefix": "${resource_prefix}", "role": "admin", "dbName": "", "host": "${OPENSEARCH_HOST}.${OPENSEARCH_NAMESPACE}", "port": ${OPENSEARCH_DBAAS_ADAPTER_PORT}, "url": "https://${OPENSEARCH_HOST}.${OPENSEARCH_NAMESPACE}:${OPENSEARCH_DBAAS_ADAPTER_PORT}", "tls": true}
    ${dml_user}=  Set Variable  {"username": "${dml_username}", "password": "${dml_password}", "resourcePrefix": "${resource_prefix}", "role": "dml", "dbName": "", "host": "${OPENSEARCH_HOST}.${OPENSEARCH_NAMESPACE}", "port": ${OPENSEARCH_DBAAS_ADAPTER_PORT}, "url": "https://${OPENSEARCH_HOST}.${OPENSEARCH_NAMESPACE}:${OPENSEARCH_DBAAS_ADAPTER_PORT}", "tls": true}
    ${readonly_user}=  Set Variable  {"username": "${readonly_username}", "password": "${readonly_password}", "resourcePrefix": "${resource_prefix}", "role": "readonly", "dbName": "", "host": "${OPENSEARCH_HOST}.${OPENSEARCH_NAMESPACE}", "port": ${OPENSEARCH_DBAAS_ADAPTER_PORT}, "url": "https://${OPENSEARCH_HOST}.${OPENSEARCH_NAMESPACE}:${OPENSEARCH_DBAAS_ADAPTER_PORT}", "tls": true}
    ${properties}=  Set Variable  [${admin_user}, ${dml_user}, ${readonly_user}]

    Run Users Recovery By Dbaas Agent  ${properties}
    Sleep  ${SLEEP_TIME}

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Users Recovery State

    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    Check OpenSearch User Exists  ${admin_username}
    Check OpenSearch User Exists  ${dml_username}
    Check OpenSearch User Exists  ${readonly_username}

    Login To OpenSearch  ${admin_username}  ${admin_password}
    ${response}=  Create OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200

    Login To OpenSearch  ${dml_username}  ${dml_password}
    ${document}=  Set Variable  {"name": "Theodore", "age": "44"}
    Create Document ${document} For Index ${resource_prefix}-test
    Sleep  ${SLEEP_TIME}

    Login To OpenSearch  ${readonly_username}  ${readonly_password}
    ${document}=  Find Document By Field  ${resourcePrefix}-test  name  Theodore
    Should Be Equal As Strings  ${document['age']}  44

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resource_prefix}
