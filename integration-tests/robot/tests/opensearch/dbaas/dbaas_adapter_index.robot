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
${SLEEP_TIME}                            5s

*** Settings ***
Library  DateTime
Library  String
Resource  ../shared/keywords.robot
Test Setup  Prepare

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

Create Index By Dbaas Agent
    [Arguments]  ${prefix}  ${db_name}  ${username}=  ${password}=
    ${data}=  Set Variable  {"dbName":"${db_name}","metadata":{},"settings":{"index":{"number_of_shards":3,"number_of_replicas":1}},"namePrefix":"${prefix}","username":"${username}","password":"${password}"}
    ${response}=  Post Request  dbaassession  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/databases  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

Delete Resources By Dbaas Agent
    [Arguments]  ${data}
    ${response}=  Post Request  dbaassession  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/resources/bulk-drop  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

*** Test Cases ***
Create Index By Dbaas Adapter
    [Tags]  dbaas  dbaas_opensearch  dbaas_index  dbaas_create_index  dbaas_v1
    ${prefix}=  Generate Random String  5  [LOWER]
    ${db_name}=  Set Variable  dbaas-index
    ${index_name}=  Catenate  SEPARATOR=_  ${prefix}  ${db_name}
    Delete OpenSearch Index  ${index_name}
    ${response}=  Create Index By Dbaas Agent  ${prefix}  ${db_name}
    Check OpenSearch Index Exists  ${index_name}
    [Teardown]  Delete Resources By Dbaas Agent  ${response['resources']}

Delete Index By Dbaas Adapter
    [Tags]  dbaas  dbaas_opensearch  dbaas_index  dbaas_delete_index  dbaas_v1
    ${prefix}=  Generate Random String  5  [LOWER]
    ${db_name}=  Set Variable  dbaas-index
    ${index_name}=  Catenate  SEPARATOR=_  ${prefix}  ${db_name}
    ${response}=  Create Index By Dbaas Agent  ${prefix}  ${db_name}
    Check OpenSearch Index Exists  ${index_name}
    Delete Resources By Dbaas Agent  ${response['resources']}
    Check OpenSearch Index Does Not Exist  ${index_name}
    [Teardown]  Delete OpenSearch Index  ${index_name}

Create Index By Dbaas Adapter And Write Data
    [Tags]  dbaas  dbaas_opensearch  dbaas_index  dbaas_create_index_and_write_data  dbaas_v1
    ${prefix}=  Generate Random String  5  [LOWER]
    ${db_name}=  Set Variable  dbaas-index
    ${index_name}=  Catenate  SEPARATOR=_  ${prefix}  ${db_name}
    Delete OpenSearch Index  ${index_name}
    ${response}=  Create Index By Dbaas Agent  ${prefix}  ${db_name}
    Log  ${response}
    ${username}=  Set Variable  ${response['connectionProperties']['username']}
    ${password}=  Set Variable  ${response['connectionProperties']['password']}
    ${resources}=  Set Variable  ${response['resources']}
    Check OpenSearch Index Exists  ${index_name}

    Login To OpenSearch  ${username}  ${password}
    ${document}=  Set Variable  {"name": "John", "age": "25"}
    Create Document ${document} For Index ${index_name}
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${index_name}  name  John
    Should Be Equal As Strings  ${document['age']}  25
    Run Keyword And Expect Error  *  Create Document ${document} For Index test-${index_name}
    ${response}=  Create OpenSearch Index  test-${prefix}
    Should Be Equal As Strings  ${response.status_code}  403
    ${response}=  Create OpenSearch Index  test-${index_name}
    Should Be Equal As Strings  ${response.status_code}  403

    [Teardown]  Run Keywords  Delete Resources By Dbaas Agent  ${resources}
                ...  AND  Delete OpenSearch Index  ${index_name}*
                ...  AND  Delete OpenSearch Index  test-${index_name}
                ...  AND  Delete OpenSearch Index  test-${prefix}


Create Index With User By Dbaas Adapter And Write Data
    [Tags]  dbaas  dbaas_opensearch  dbaas_index  dbaas_create_index_with_user_and_write_data  dbaas_v1
    ${prefix}=  Generate Random String  5  [LOWER]
    ${db_name}=  Set Variable  dbaas-index
    ${index_name}=  Catenate  SEPARATOR=_  ${prefix}  ${db_name}
    Delete OpenSearch Index  ${index_name}
    ${response}=  Create Index By Dbaas Agent  ${prefix}  ${db_name}  testuser  testPassword_1
    Check OpenSearch Index Exists  ${index_name}

    Login To OpenSearch  testuser  testPassword_1
    ${document}=  Set Variable  {"name": "John", "age": "25"}
    Create Document ${document} For Index ${index_name}
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${index_name}  name  John
    Should Be Equal As Strings  ${document['age']}  25

    [Teardown]  Delete Resources By Dbaas Agent  ${response['resources']}