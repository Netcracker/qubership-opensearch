*** Variables ***
${DBAAS_METADATA_INDEX}                  dbaas_opensearch_metadata
${RETRY_TIME}                            20s
${RETRY_INTERVAL}                        1s
${SLEEP_TIME}                            5s

*** Settings ***
Resource  ./keywords.robot
Suite Setup  Prepare

*** Keywords ***
Change User Password By Dbaas Agent
    [Arguments]  ${username}  ${password}  ${role_type}
    ${data}=  Set Variable  {"password": "${password}", "role": "${role_type}"}
    ${response}=  Put Request  dbaas_admin_session  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/users/${username}  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

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

Create Database Resource Prefix With Metadata
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_create_resource_prefix_with_metadata  dbaas_v1
    &{metadata}=  Create Dictionary  scope=service-v1  microserviceName=integration-tests
    ${response}=  Create Database Resource Prefix By Dbaas Agent  metadata=${metadata}
    Log  ${response}
    ${resourcePrefix}=  Set Variable  ${response['connectionProperties']['resourcePrefix']}
    Log  Resource Prefix: ${resourcePrefix}
    Sleep  ${SLEEP_TIME}

    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    ${document}=  Find Document By Field  ${DBAAS_METADATA_INDEX}  _id  ${resourcePrefix}
    Should Be Equal As Strings  ${document['microserviceName']}  integration-tests
    Should Be Equal As Strings  ${document['scope']}  service-v1

    Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}
    Sleep  ${SLEEP_TIME}

    Check That Document Does Not Exist By Field  ${DBAAS_METADATA_INDEX}  _id  ${resourcePrefix}
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

    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test-new
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index Template  ${resourcePrefix}-index-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Template  ${resourcePrefix}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${resourcePrefix}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias For Index  ${resourcePrefix}-test  ${resourcePrefix}-alias
    Should Be Equal As Strings  ${response.status_code}  200

    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    ${response}=  Get OpenSearch User  ${username}
    Should Be Equal As Strings  ${response.status_code}  200
    Check That Document Does Not Exist By Field  ${DBAAS_METADATA_INDEX}  _id  ${resourcePrefix}

    Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}
    Sleep  ${SLEEP_TIME}

    ${response}=  Get OpenSearch User  ${username}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test-new
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index Template  ${resourcePrefix}-index-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Template  ${resourcePrefix}-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias  ${resourcePrefix}-alias
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias For Index  ${resourcePrefix}-test  ${resourcePrefix}-alias
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
    ${response}=  Create OpenSearch Alias  ${resourcePrefix}-test  ${resourcePrefix}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Tasks
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Task By ID  SZcaJdObTeu2srh12Uwv0Q:1
    Should Be Equal As Strings  ${response.status_code}  404

    Login To OpenSearch  ${username_dml}  ${password_dml}
    ${response}=  Create OpenSearch Index  ${resourcePrefix}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "Jack", "age": "26"}
    ${response}=  Update Document ${document} For Index ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${resourcePrefix}-test  name  Jack
    Should Be Equal As Strings  ${document['age']}  26
    ${response}=  Get OpenSearch Tasks
    Should Be Equal As Strings  ${response.status_code}  403
    ${response}=  Get OpenSearch Task By ID  SZcaJdObTeu2srh12Uwv0Q:1
    Should Be Equal As Strings  ${response.status_code}  404

    Login To OpenSearch  ${username_readonly}  ${password_readonly}
    ${response}=  Create OpenSearch Index  ${resourcePrefix}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "James", "age": "27"}
    ${response}=  Update Document ${document} For Index ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Find Document By Field  ${resourcePrefix}-test  name  Jack
    Should Be Equal As Strings  ${document['age']}  26
    ${document}=  Find Document By Field  ${resourcePrefix}-alias  name  Jack
    Should Be Equal As Strings  ${document['age']}  26
    ${response}=  Get OpenSearch Task By ID  SZcaJdObTeu2srh12Uwv0Q:1
    Should Be Equal As Strings  ${response.status_code}  403
    ${response}=  Get OpenSearch Index Exists  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index Exists  test
    Should Be Equal As Strings  ${response.status_code}  403

    Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}
    Sleep  ${SLEEP_TIME}
    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}

    ${response}=  Get OpenSearch User  ${username_admin}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_dml}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_readonly}
    Should Be Equal As Strings  ${response.status_code}  404

    ${response}=  Get OpenSearch Index  ${resourcePrefix}-test
    Should Be Equal As Strings  ${response.status_code}  404

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Create Database Resource Prefix With Metadata for Multiple Users
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_create_resource_prefix_with_metadata_for_multiple_users  dbaas_v2
    &{metadata}=  Create Dictionary  scope=service-v2  microserviceName=integration-tests
    ${response}=  Create Database Resource Prefix By Dbaas Agent  metadata=${metadata}
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}
    Sleep  ${SLEEP_TIME}

    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    ${document}=  Find Document By Field  ${DBAAS_METADATA_INDEX}  _id  ${resourcePrefix}
    Should Be Equal As Strings  ${document['microserviceName']}  integration-tests
    Should Be Equal As Strings  ${document['scope']}  service-v2

    Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}
    Sleep  ${SLEEP_TIME}

    Check That Document Does Not Exist By Field  ${DBAAS_METADATA_INDEX}  _id  ${resourcePrefix}
    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Create Database With Custom Resource Prefix for Multiple Users
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_create_with_custom_resource_prefix_for_multiple_users  dbaas_v2
    ${resource_prefix}=  Set Variable  custom-2b9e-4cbc-8f7f-b98684b51073
    Log  Resource Prefix: ${resource_prefix}
    ${response}=  Create Database Resource Prefix By Dbaas Agent  prefix=${resource_prefix}
    Log  ${response}
    ${created_resource_prefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].resourcePrefix
    Should Be Equal As Strings  ${resource_prefix}  ${created_resource_prefix}

    ${username_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].username
    ${password_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].password

    ${username_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].username
    ${password_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].password

    ${username_readonly}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="readonly")].username
    ${password_readonly}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="readonly")].password

    Login To OpenSearch  ${username_admin}  ${password_admin}
    ${response}=  Create OpenSearch Index  ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Set Variable  {"name": "Jared", "age": "51"}
    Create Document ${document} For Index ${resource_prefix}-test
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${resource_prefix}-test  name  Jared
    Should Be Equal As Strings  ${document['age']}  51

    Login To OpenSearch  ${username_dml}  ${password_dml}
    ${response}=  Create OpenSearch Index  ${resource_prefix}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "Jonathan", "age": "37"}
    ${response}=  Update Document ${document} For Index ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${resource_prefix}-test  name  Jonathan
    Should Be Equal As Strings  ${document['age']}  37

    Login To OpenSearch  ${username_readonly}  ${password_readonly}
    ${response}=  Create OpenSearch Index  ${resource_prefix}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "James", "age": "27"}
    ${response}=  Update Document ${document} For Index ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Find Document By Field  ${resource_prefix}-test  name  Jonathan
    Should Be Equal As Strings  ${document['age']}  37

    Delete Database Resource Prefix Dbaas Agent  ${resource_prefix}
    Sleep  ${SLEEP_TIME}
    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}

    ${response}=  Get OpenSearch User  ${username_admin}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_dml}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_readonly}
    Should Be Equal As Strings  ${response.status_code}  404

    ${response}=  Get OpenSearch Index  ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  404

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resource_prefix}

Change Password for DML User
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_change_password_for_dml_user  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resource_prefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].resourcePrefix
    Log  Resource Prefix: ${resource_prefix}

    ${username_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].username
    ${password_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].password

    ${username_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].username
    ${password_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].password

    Login To OpenSearch  ${username_admin}  ${password_admin}
    ${response}=  Create OpenSearch Index  ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  200

    Login To OpenSearch  ${username_dml}  ${password_dml}
    ${response}=  Create OpenSearch Index  ${resource_prefix}-test2
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "Joanne", "age": "57"}
    Create Document ${document} For Index ${resource_prefix}-test
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${resource_prefix}-test  name  Joanne
    Should Be Equal As Strings  ${document['age']}  57

    ${new_password_dml} =  Set Variable  eX5l#RqbdQ
    ${content}=  Change User Password By Dbaas Agent  ${username_dml}  ${new_password_dml}  dml
    Should Be Equal As Strings  ${content['connectionProperties']['resourcePrefix']}  ${resource_prefix}
    Should Be Equal As Strings  ${content['connectionProperties']['username']}  ${username_dml}
    Should Be Equal As Strings  ${content['connectionProperties']['password']}  ${new_password_dml}

    Login To OpenSearch  ${username_dml}  ${new_password_dml}
    ${response}=  Create OpenSearch Index  ${resource_prefix}-test3
    Should Be Equal As Strings  ${response.status_code}  403
    ${document}=  Set Variable  {"name": "Jesus", "age": "37"}
    ${response}=  Update Document ${document} For Index ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${resource_prefix}-test  name  Jesus
    Should Be Equal As Strings  ${document['age']}  37

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resource_prefix}

Delete Database Resource Prefix for Multiple Users
    [Tags]  dbaas  dbaas_opensearch  dbaas_resource_prefix  dbaas_delete_resource_prefix_for_multiple_users  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resource_prefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].resourcePrefix
    Log  Resource Prefix: ${resource_prefix}

    ${username_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].username
    ${password_admin}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="admin")].password

    ${username_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].username
    ${password_dml}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="dml")].password

    ${username_readonly}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="readonly")].username
    ${password_readonly}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="readonly")].password

    Login To OpenSearch  ${username_admin}  ${password_admin}
    Create OpenSearch Index Template  ${resource_prefix}-index-template  ${resource_prefix}*  {"number_of_shards":3}
    Create OpenSearch Template  ${resource_prefix}-template  ${resource_prefix}*  {"number_of_shards":3}
    Create OpenSearch Index  ${resource_prefix}-test
    Create OpenSearch Alias  ${resource_prefix}-test  ${resource_prefix}-alias
    Make Index Read Only  ${resource_prefix}-test
    Clone Index  ${resource_prefix}-test  ${resource_prefix}-test-new
    Sleep  ${SLEEP_TIME}

    ${response}=  Get OpenSearch Index  ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index  ${resource_prefix}-test-new
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index Template  ${resource_prefix}-index-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Template  ${resource_prefix}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${resource_prefix}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias For Index  ${resource_prefix}-test  ${resource_prefix}-alias
    Should Be Equal As Strings  ${response.status_code}  200

    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    ${response}=  Get OpenSearch User  ${username_admin}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch User  ${username_dml}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch User  ${username_readonly}
    Should Be Equal As Strings  ${response.status_code}  200
    Check That Document Does Not Exist By Field  ${DBAAS_METADATA_INDEX}  _id  ${resourcePrefix}

    Delete Database Resource Prefix Dbaas Agent  ${resource_prefix}
    Sleep  ${SLEEP_TIME}

    ${response}=  Get OpenSearch User  ${username_admin}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_dml}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch User  ${username_readonly}
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index  ${resource_prefix}-test
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index  ${resource_prefix}-test-new
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Index Template  ${resource_prefix}-index-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Template  ${resource_prefix}-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias  ${resource_prefix}-alias
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias For Index  ${resource_prefix}-test  ${resource_prefix}-alias
    Should Be Equal As Strings  ${response.status_code}  404

    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resource_prefix}
