*** Variables ***
${IDENTITY_PROVIDER_URL}                 %{IDENTITY_PROVIDER_URL}
${IDENTITY_PROVIDER_REGISTRATION_TOKEN}  %{IDENTITY_PROVIDER_REGISTRATION_TOKEN}
${IDENTITY_PROVIDER_USERNAME}            %{IDENTITY_PROVIDER_USERNAME}
${IDENTITY_PROVIDER_PASSWORD}            %{IDENTITY_PROVIDER_PASSWORD}
${CLIENT_NAME}                           opensearch-integration-tests-client

*** Settings ***
Library  String
Library  OAuthLibrary  url=${IDENTITY_PROVIDER_URL}
...                    registration_token=${IDENTITY_PROVIDER_REGISTRATION_TOKEN}
...                    username=${IDENTITY_PROVIDER_USERNAME}
...                    password=${IDENTITY_PROVIDER_PASSWORD}
Resource  ../shared/keywords.robot

*** Keywords ***
Send Request With Basic Authentication
    [Arguments]  ${path}
    ${response}=  Get Request  opensearch  ${path}
    [Return]  ${response}

Register New Client
    [Documentation]  Registers new Identity Provider client
    ${client}=  Register Client  ${CLIENT_NAME}
    [Return]  ${client['client_id']}

Get New Token
    [Documentation]  Provides new access token or refresh existed (creates new client if it is necessary)
    ${client_id}=  Register New Client
    ${token}=  Get Token  ${client_id}
    [Return]  ${token}

Send Request With OAuth Token
    [Arguments]  ${endpoint}  ${token}  ${data}=${None}
    &{headers}=  Create Dictionary  Authorization=Bearer ${token}
    ${response}=  Get Request  opensearch  ${endpoint}  headers=${headers}
    [Return]  ${response}

*** Test Cases ***
Basic Authentication With Valid Credentials
    [Tags]  authentication  basic_authentication  regression
    Prepare OpenSearch
    ${response} =  Send Request With Basic Authentication  /_cat
    Should Be Equal As Strings  ${response.status_code}  200

Basic Authentication With Invalid Credentials
    [Tags]  authentication  basic_authentication  regression
    Prepare OpenSearch  False
    ${response} =  Send Request With Basic Authentication  /_cat
    Should Be Equal As Strings  ${response.status_code}  401

OAuth Request With Valid Token
    [Tags]  authentication  oauth  regression
    Prepare OpenSearch  False
    ${token}=  Get New Token
    ${response}=  Send Request With OAuth Token  /_cat  ${token}
    Should Be Equal As Strings  ${response.status_code}  200

OAuth Request With Invalid Token
    [Tags]  authentication  oauth  regression
    Prepare OpenSearch  False
    ${invalid_token}=  Set Variable  invalid_token
    ${response}=  Send Request With OAuth Token  /_cat  ${invalid_token}
    Should Be Equal As Strings  ${response.status_code}  401