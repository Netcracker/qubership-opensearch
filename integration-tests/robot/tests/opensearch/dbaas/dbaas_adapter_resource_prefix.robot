*** Variables ***
${DBAAS_ADAPTER_TYPE}                    %{DBAAS_ADAPTER_TYPE}
${OPENSEARCH_DBAAS_ADAPTER_HOST}         %{OPENSEARCH_DBAAS_ADAPTER_HOST}
${OPENSEARCH_DBAAS_ADAPTER_PORT}         %{OPENSEARCH_DBAAS_ADAPTER_PORT}
${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}     %{OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}
${OPENSEARCH_DBAAS_ADAPTER_USERNAME}     %{OPENSEARCH_DBAAS_ADAPTER_USERNAME}
${OPENSEARCH_DBAAS_ADAPTER_PASSWORD}     %{OPENSEARCH_DBAAS_ADAPTER_PASSWORD}
${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}  %{OPENSEARCH_DBAAS_ADAPTER_API_VERSION=v1}
${OPENSEARCH_HOST}                       %{OPENSEARCH_HOST}
${OPENSEARCH_PORT}                       %{OPENSEARCH_PORT}
${OPENSEARCH_PROTOCOL}                   %{OPENSEARCH_PROTOCOL}
${RETRY_TIME}                            20s
${RETRY_INTERVAL}                        1s
${SLEEP_TIME}                            5s

*** Settings ***
Library  DateTime
Library  String
Resource  ../shared/keywords.robot
Library  ../shared/lib/JsonpathLibrary.py
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
    Create Session  dbaas_admin_session  ${OPENSEARCH_DBAAS_ADAPTER_PROTOCOL}://${OPENSEARCH_DBAAS_ADAPTER_HOST}:${OPENSEARCH_DBAAS_ADAPTER_PORT}  auth=${auth}  verify=${verify}

Login To OpenSearch As User
    [Arguments]  ${username}  ${password}
    ${auth}=  Create List  ${username}  ${password}
    Create Session  dbaas_user_session  ${OPENSEARCH_PROTOCOL}://${OPENSEARCH_HOST}:${OPENSEARCH_PORT}  auth=${auth}  disable_warnings=1
    [Return]  opensearch_user_session

Create Database Resource Prefix By Dbaas Agent
    [Arguments]
    ${data}=  Set Variable  {"settings":{"resourcePrefix": true,"createOnly": ["user"]}}
    ${response}=  Post Request  dbaas_admin_session  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/databases  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

Create Database Resource Prefix By Dbaas Agent With Credentials
    [Arguments]  ${username}  ${password}
    ${data}=  Set Variable  {"settings":{"resourcePrefix": true,"createOnly": ["user"]}, "username": "${username}", "password": "${password}"}
    ${response}=  Post Request  dbaas_admin_session  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/databases  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

Delete Database Resource Prefix Dbaas Agent
    [Arguments]  ${prefix}
    ${data}=  Set Variable  [{"kind":"resourcePrefix","name":"${prefix}"}]
    ${response}=  Post Request  dbaas_admin_session  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/resources/bulk-drop  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

*** Test Cases ***
Create Database Resource Prefix
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_create_resource_prefix  dbaas_v1
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${username}=  Set Variable  ${response['connectionProperties']['username']}
    ${password}=  Set Variable  ${response['connectionProperties']['password']}
    ${resourcePrefix}=  Set Variable  ${response['connectionProperties']['resourcePrefix']}
    Login To OpenSearch  ${username}  ${password}

    Create OpenSearch Template  ${resourcePrefix}-template  ${resourcePrefix}*  {"number_of_shards":3}

    ${response}=  Create OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200

    ${document}=  Set Variable  {"name": "John", "age": "25"}
    Create Document ${document} For Index ${resourcePrefix}-test
    Sleep  ${SLEEP_TIME}

    ${document}=  Find Document By Field  ${resourcePrefix}-test  name  John
    Should Be Equal As Strings  ${document['age']}  25
    ${response}=  Create OpenSearch Alias  ${resourcePrefix}-test  ${resourcePrefix}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Find Document By Field  ${resourcePrefix}-alias  name  John
    Should Be Equal As Strings  ${document['age']}  25

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Database Resource Prefix Authorization
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_resource_prefix_authorization  dbaas_v1
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${username_first}=  Set Variable  ${response['connectionProperties']['username']}
    ${password_first}=  Set Variable  ${response['connectionProperties']['password']}
    ${resourcePrefix_first}=  Set Variable  ${response['connectionProperties']['resourcePrefix']}

    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${username_second}=  Set Variable  ${response['connectionProperties']['username']}
    ${password_second}=  Set Variable  ${response['connectionProperties']['password']}
    ${resourcePrefix_second}=  Set Variable  ${response['connectionProperties']['resourcePrefix']}

    Login To OpenSearch  ${username_first}  ${password_first}
    Create OpenSearch Template  ${resourcePrefix_first}-template  ${resourcePrefix_first}*  {"number_of_shards":3}
    Create OpenSearch Index Template  ${resourcePrefix_first}-index-template  ${resourcePrefix_first}*  {"number_of_shards":3}
    ${response}=  Create OpenSearch Index  ${resourcePrefix_first}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Create OpenSearch Index  test-${resourcePrefix_first}
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "John", "age": "25"}
    Create Document ${document} For Index ${resourcePrefix_first}-test
    ${response}=  Make Index Read Only  ${resourcePrefix_first}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Clone Index  ${resourcePrefix_first}-test  ${resourcePrefix_first}-test-new
    Should Be Equal As Strings  ${response.status_code}  200
    Check OpenSearch Index Exists  ${resourcePrefix_first}-test-new

#    Uncomment it when (if) OpenSearch issue https://github.com/opensearch-project/security/issues/429 fixed.
#    {response}=  Clone Index  ${resourcePrefix_first}-test  custom-test-new
#    Should Be Equal As Strings  ${response.status_code}  403

    ${response}=  Make Index Read Write  ${resourcePrefix_first}-test
    Should Be Equal As Strings  ${response.status_code}  200

    Login To OpenSearch  ${username_second}  ${password_second}
    Create OpenSearch Template  ${resourcePrefix_second}-template  ${resourcePrefix_second}*  {"number_of_shards":3}
    Create OpenSearch Index Template  ${resourcePrefix_second}-index-template  ${resourcePrefix_second}*  {"number_of_shards":3}
    ${response}=  Create OpenSearch Index  ${resourcePrefix_second}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Create OpenSearch Index  ${resourcePrefix_first}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${response}=  Get Request  opensearch  /${resourcePrefix_first}-test/_search
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "John", "age": "26"}
    ${response}=  Update Document ${document} For Index ${resourcePrefix_first}-test
    Should Be Equal As Strings  ${response.status_code}  403
    ${response}=  Create OpenSearch Alias  ${resourcePrefix_first}-test  ${resourcePrefix_first}-alias2
    Should Be Equal As Strings  ${response.status_code}  403
    ${response}=  Create OpenSearch Alias  ${resourcePrefix_first}-test  ${resourcePrefix_second}-alias2
    Should Be Equal As Strings  ${response.status_code}  403
    ${response}=  Clone Index  ${resourcePrefix_first}-test  ${resourcePrefix_second}-test-new
    Should Be Equal As Strings  ${response.status_code}  403

    [Teardown]  Run Keywords  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix_first}
           ...  AND  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix_second}

Delete Database Resource Prefix
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_delete_resource_prefix  dbaas_v1
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${username}=  Set Variable  ${response['connectionProperties']['username']}
    ${password}=  Set Variable  ${response['connectionProperties']['password']}
    ${resourcePrefix}=  Set Variable  ${response['connectionProperties']['resourcePrefix']}
    Login To OpenSearch  ${username}  ${password}

    Create OpenSearch Index Template  ${resourcePrefix}-index-template  ${resourcePrefix}*  {"number_of_shards":3}
    Create OpenSearch Template  ${resourcePrefix}-template  ${resourcePrefix}*  {"number_of_shards":3}
    Create OpenSearch Index  ${resourcePrefix}-test
    Create OpenSearch Alias  ${resourcePrefix}-test  ${resourcePrefix}-alias
    Make Index Read Only  ${resourcePrefix}-test
    Clone Index  ${resourcePrefix}-test  ${resourcePrefix}-test-new
    Sleep  ${SLEEP_TIME}

    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    ${response}=  Get OpenSearch User  ${username}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Role  ${resourcePrefix}_role
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test-new
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index Template  ${resourcePrefix}-index-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Template  ${resourcePrefix}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${resourcePrefix}-test  ${resourcePrefix}-alias
    Should Be Equal As Strings  ${response.status_code}  200

    Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}
    Sleep  ${SLEEP_TIME}

    ${response}=  Get OpenSearch User  ${username}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Role  ${resourcePrefix}_role
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test-new
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index Template  ${resourcePrefix}-index-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Template  ${resourcePrefix}-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias  ${resourcePrefix}-test  ${resourcePrefix}-alias
    Should Be Equal As Strings  ${response.status_code}  404

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Create Database Resource Prefix for Multiple Users
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_create_resource_prefix_for_multiple_users  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}

    ${username_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].username
    ${password_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].password

    ${username_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].username
    ${password_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].password

    ${username_readonly}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="readonly")].username
    ${password_readonly}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="readonly")].password

    Login To OpenSearch  ${username_admin}  ${password_admin}
    ${response}=  Create OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Set Variable  {"name": "John", "age": "25"}
    Create Document ${document} For Index ${resourcePrefix}-test
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${resourcePrefix}-test  name  John
    Should Be Equal As Strings  ${document['age']}  25

    Login To OpenSearch  ${username_dml}  ${password_dml}
    ${response}=  Create OpenSearch Index  ${resourcePrefix}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "Jack", "age": "26"}
    ${response}=  Update Document ${document} For Index ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${resourcePrefix}-test  name  Jack
    Should Be Equal As Strings  ${document['age']}  26

    Login To OpenSearch  ${username_readonly}  ${password_readonly}
    ${response}=  Create OpenSearch Index  ${resourcePrefix}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "James", "age": "27"}
    ${response}=  Update Document ${document} For Index ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Find Document By Field  ${resourcePrefix}-test  name  Jack
    Should Be Equal As Strings  ${document['age']}  26

    Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}
    Sleep  ${SLEEP_TIME}
    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}

    ${response}=  Get OpenSearch User  ${username_admin}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_dml}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_readonly}
    Should Be Equal As Strings  ${response.status_code}  404

    ${response}=  Get OpenSearch Role  ${resourcePrefix}_admin
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Role  ${resourcePrefix}_dml
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Role  ${resourcePrefix}_readonly
    Should Be Equal As Strings  ${response.status_code}  404

    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  404

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}