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

*** Settings ***
Library  DateTime
Library  String
Resource  ../shared/keywords.robot
Library  ../shared/lib/JsonpathLibrary.py

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

Create Database Resource Prefix By Dbaas Agent
    [Arguments]  ${prefix}=  ${metadata}={}
    @{create_only}=  Create List  user
    &{settings}=  Create Dictionary  resourcePrefix=${True}  createOnly=${create_only}
    &{data}=  Create Dictionary  settings=${settings}
    Run Keyword If  "${prefix}" != "${EMPTY}"  Set To Dictionary  ${data}  namePrefix=${prefix}
    Run Keyword If  ${metadata}  Set To Dictionary  ${data}  metadata=${metadata}
    ${response}=  Post Request  dbaas_admin_session  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/databases  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

Delete Database Resource Prefix Dbaas Agent
    [Arguments]  ${prefix}
    ${data}=  Set Variable  [{"kind":"resourcePrefix","name":"${prefix}"}]
    ${response}=  Post Request  dbaas_admin_session  /api/${OPENSEARCH_DBAAS_ADAPTER_API_VERSION}/dbaas/adapter/${DBAAS_ADAPTER_TYPE}/resources/bulk-drop  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
