*** Variables ***
${SMOKE_TEST_INDEX_NAME}  smoke_test
${SLEEP_TIME}             5s
${secret_name}            opensearch-secret
${secret_name_old}        opensearch-secret-old
${body}                   [{"op" : "replace" ,"path" : "/data/username" ,"value" : "UUEtZ29vZC1wYXNzd29yZDEhLUFU" ,"path" : "/data/password" ,"value" : "T3BlbnNlYXJjaC1hZG1pbjEhLUFU"}]

*** Settings ***
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Prepare

*** Keywords ***
Prepare
    Prepare OpenSearch

*** Test Cases ***

Change Password for User
    [Tags]  smoke
    ${response}=  Get Secret  ${secret_name}  ${namespace}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Patch Secret  ${secret_name}  ${namespace} ${body}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get Secret  ${secret_name_old}  ${namespace}
    Should Be Equal As Strings  ${response.status_code}  200

Create Index
    [Tags]  smoke  index  create_index
    ${response}=  Create OpenSearch Index  ${SMOKE_TEST_INDEX_NAME}
    Should Be Equal As Strings  ${response.status_code}  200

Get Index
    [Tags]  smoke  index  get_index
    ${response}=  Get OpenSearch Index  ${SMOKE_TEST_INDEX_NAME}
    Should Be Equal As Strings  ${response.status_code}  200

Create Document
    [Tags]  smoke  document  create_document
    Set Global Variable  ${name}  John
    Set Global Variable  ${age}  25
    ${document}=  Set Variable  {"age": "${age}", "name": "${name}"}
    Create Document ${document} For Index ${SMOKE_TEST_INDEX_NAME}

Search Document
    [Tags]  smoke  document  search_document
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${SMOKE_TEST_INDEX_NAME}  name  ${name}
    Should Be Equal As Strings  ${document['age']}  ${age}

Update Document
    [Tags]  smoke  document  update_document
    ${newAge}=  Set Variable  26
    ${document}=  Set Variable  {"age": "${newAge}", "name": "${name}"}
    ${response}=  Update document ${document} For Index ${SMOKE_TEST_INDEX_NAME}
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}
    ${document}=  Find Document By Field  ${SMOKE_TEST_INDEX_NAME}  name  ${name}
    Should Be Equal As Strings  ${document['age']}  ${newAge}

Delete Document
    [Tags]  smoke  document  delete_document
    Delete Document For Index ${SMOKE_TEST_INDEX_NAME}

Delete Index
    [Tags]  smoke  index  delete_index
    ${response}=  Delete OpenSearch Index  ${SMOKE_TEST_INDEX_NAME}
    Should Be Equal As Strings  ${response.status_code}  200