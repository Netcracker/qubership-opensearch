*** Variables ***
${RETRY_TIME}                            60s
${RETRY_INTERVAL}                        5s
${SLEEP_TIME}                            5s
${CHECK_RESULT_RETRY_COUNT}              15x
${CHECK_RESULT_RETRY_INTERVAL}           5s
${secret_name}                           opensearch-secret
${secret_name_old}                       opensearch-secret-old
${host}                                  ${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}://${OPENSEARCH_DBAAS_ADAPTER_HOST}:${OPENSEARCH_DBAAS_ADAPTER_PORT}/health

*** Settings ***
Library    Process
Resource  ./keywords.robot
Suite Setup  Prepare

*** Keywords ***
Run Users Recovery By Dbaas Agent
    [Arguments]  ${properties}
    ${data}=  Set Variable  {"settings": {}, "connectionProperties": ${properties}}
    ${response}=  POST On Session  dbaas_admin_session  /api/v2/dbaas/adapter/opensearch/users/restore-password  data=${data}  headers=${headers}

Get Users Recovery State By Dbaas Agent
    ${response}=  GET On Session  dbaas_admin_session  /api/v2/dbaas/adapter/opensearch/users/restore-password/state  headers=${headers}
    RETURN  ${response.content}

Check Users Recovery State
    ${state}=  Get Users Recovery State By Dbaas Agent
    Should Be Equal As Strings  ${state}  done

DBaaS Adapter Is Up
    ${health} =  GET On Session  dbaas_admin_session  /health
    Should Be Equal As Strings  ${health.content}  {"status":"UP","opensearchHealth":{"status":"UP"},"dbaasAggregatorHealth":{"status":"OK"}}

Check DBaaS Adapter State
    Wait Until Keyword Succeeds  ${CHECK_RESULT_RETRY_COUNT}  ${CHECK_RESULT_RETRY_INTERVAL}
    ...  DBaaS Adapter Is Up

*** Test Cases ***
Change Password for User and Healthcheck Dbaas Pod
    [Tags]   dbaas  dbaas_opensearch  dbaas_recovery  dbaas_recover_users  dbaas_v2
    ${secret}=  Get Secret  ${secret_name}  ${OPENSEARCH_NAMESPACE}
    ${secret_body}=  Set Variable  {"data": {"password": "UUEtZ29vZC1wYXNzd29yZDEhLUFU", "username": "T3BlbnNlYXJjaC1hZG1pbjEhLUFU"}}
    ${response}=  Patch Secret  ${secret_name}  ${OPENSEARCH_NAMESPACE}  ${secret_body}
    ${old_secret}=  Get Secret  ${secret_name_old}  ${OPENSEARCH_NAMESPACE}
    Should Be Equal As Strings  ${old_secret.data}  ${secret_body}
    Check DBaaS Adapter State
    [Teardown]  Run Keywords    Patch Secret  ${secret_name}  ${OPENSEARCH_NAMESPACE}  ${secret.data}
    ...  AND  Check DBaaS Adapter State

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
