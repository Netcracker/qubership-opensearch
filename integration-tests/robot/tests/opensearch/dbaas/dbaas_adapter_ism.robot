*** Variables ***
${RETRY_TIME}        180s
${RETRY_INTERVAL}    20s

*** Settings ***
Resource  ./keywords.robot
Suite Setup  Prepare

*** Keywords ***
Set ISM Job Interval
    [Arguments]  ${interval}
    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    ${response}=  Update Cluster Settings  {"persistent" : {"plugins.index_state_management.job_interval": ${interval}}}
    Should Be Equal As Strings  ${response.status_code}  200

Check Managed Index State
    [Arguments]  ${index_name}  ${policy_name}  ${state}
    ${content}=  Explain Index  ${index_name}
    Should Be Equal As Strings  ${content["${index_name}"]["policy_id"]}  ${policy_name}
    Should Be Equal As Strings  ${content["${index_name}"]["state"]["name"]}  ${state}

*** Test Cases ***
Policy CRUD
    [Tags]  dbaas  dbaas_opensearch  dbaas_ism  dbaas_ism_policy_crud  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}
    ${username_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].username
    ${password_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].password

    Login To OpenSearch  ${username_ism}  ${password_ism}
    ${policy_name}=  Set Variable  ${resource_prefix}-policy
    # create policy
    ${response}=  Create Policy  ${policy_name}  {"policy": {"description": "Test rollover", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 2}}]}]}}
    Should Be Equal As Strings  ${response.status_code}  201
    # read policy
    ${content}=  Get Policy  ${policy_name}
    Should Be Equal As Strings  ${content["policy"]["policy_id"]}  ${policy_name}
    Should Be Equal As Strings  ${content["policy"]["description"]}  Test rollover
    ${seq_no}=  Set Variable  ${content["_seq_no"]}
    ${primary_term}=  Set Variable  ${content["_primary_term"]}
    # update policy
    ${response}=  Update Policy  ${policy_name}  ${seq_no}  ${primary_term}  {"policy": {"description": "Testing rollover", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 2}}]}], "ism_template": {"index_patterns": ["${resource_prefix}*"], "priority": 100}}}
    Should Be Equal As Strings  ${response.status_code}  200
    # read policy
    ${content}=  Get Policy  ${policy_name}
    Should Be Equal As Strings  ${content["policy"]["policy_id"]}  ${policy_name}
    Should Be Equal As Strings  ${content["policy"]["description"]}  Testing rollover
    Should Be Equal As Strings  ${content["policy"]["ism_template"][0]["index_patterns"][0]}  ${resource_prefix}*
    # read all policies - forbidden
    ${response}=  Get Policies
    Should Be Equal As Strings  ${response.status_code}  403
    # delete policy
    ${response}=  Remove Policy  ${policy_name}
    Should Be Equal As Strings  ${response.status_code}  200
    # read policy
    ${response}=  Get Policy  ${policy_name}  False
    Should Be Equal As Strings  ${response.status_code}  404
    [Teardown]  Run Keywords  Remove Policy  ${policy_name}
    ...  AND  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Policy CRUD With Unallowed Resource Prefix
    [Tags]  dbaas  dbaas_opensearch  dbaas_ism  dbaas_ism_unallowed_policy_crud  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}
    ${username_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].username
    ${password_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].password

    Login To OpenSearch  ${username_ism}  ${password_ism}
    ${impermissible_resource_prefix}=  Set Variable  custom-2b9e-4cbc-8f7f-b98684b51073
    ${policy_name}=  Set Variable  ${impermissible_resource_prefix}-policy
    # create policy
    ${response}=  Create Policy  ${policy_name}  {"policy": {"description": "Test rollover", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 2}}]}]}}
    Should Be Equal As Strings  ${response.status_code}  403
    # read policy
    ${response}=  Get Policy  ${policy_name}  False
    Should Be Equal As Strings  ${response.status_code}  403
    ${seq_no}=  Set Variable  0
    ${primary_term}=  Set Variable  1
    # update policy
    ${response}=  Update Policy  ${policy_name}  ${seq_no}  ${primary_term}  {"policy": {"description": "Testing rollover", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 2}}]}], "ism_template": {"index_patterns": ["${resource_prefix}*"], "priority": 100}}}
    Should Be Equal As Strings  ${response.status_code}  403
    # delete policy
    ${response}=  Remove Policy  ${policy_name}
    Should Be Equal As Strings  ${response.status_code}  403
    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Managed Index CRUD
    [Tags]  dbaas  dbaas_opensearch  dbaas_ism  dbaas_ism_index_crud  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}
    ${username_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].username
    ${password_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].password

    Set ISM Job Interval  1

    Login To OpenSearch  ${username_ism}  ${password_ism}
    ${policy_name}=  Set Variable  ${resource_prefix}-policy
    ${response}=  Create Policy  ${policy_name}  {"policy": {"description": "Test rollover", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 2}}]}, {"name": "delete", "actions": [{"delete": {}}], "transitions": []}]}}
    Should Be Equal As Strings  ${response.status_code}  201
    ${response}=  Create Policy  ${policy_name}-2  {"policy": {"description": "My rollover", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 20}}]}, {"name": "delete", "actions": [{"delete": {}}], "transitions": []}]}}
    Should Be Equal As Strings  ${response.status_code}  201
    ${index_name}=  Set Variable  ${resource_prefix}-000001
    ${response}=  Create OpenSearch Index  ${index_name}  {"settings": {"plugins.index_state_management.rollover_alias": "${resource_prefix}"}, "aliases": {"${resource_prefix}": {"is_write_index": "true"}}}
    Should Be Equal As Strings  ${response.status_code}  200
    # add policy to index
    ${response}=  Add Policy To Index  ${index_name}  ${policy_name}
    Should Be Equal As Strings  ${response.status_code}  200
    # read index
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Managed Index State  ${index_name}  ${policy_name}  test_rollover
    # update index policy
    ${response}=  Change Index Policy  ${index_name}  {"policy_id": "${policy_name}-2"}
    Should Be Equal As Strings  ${response.status_code}  200
    # read index
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Managed Index State  ${index_name}  ${policy_name}-2  test_rollover
    # retry failed index
    ${response}=  Retry Failed Index  ${index_name}  {"state": "delete"}
    Should Be Equal As Strings  ${response.status_code}  200
    # delete policy from index
    ${response}=  Remove Policy From Index  ${index_name}
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Remove Policy  ${policy_name}
    ...  AND  Remove Policy  ${policy_name}-2
    ...  AND  Set ISM Job Interval  null
    ...  AND  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Managed Index CRUD With Unallowed Resource Prefix
    [Tags]  dbaas  dbaas_opensearch  dbaas_ism  dbaas_ism_unallowed_index_crud  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}
    ${username_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].username
    ${password_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].password

    Login To OpenSearch  ${username_ism}  ${password_ism}
    ${impermissible_resource_prefix}=  Set Variable  custom-2b9e-4cbc-8f7f-b98684b51073
    ${policy_name}=  Set Variable  ${impermissible_resource_prefix}-policy
    ${index_name}=  Set Variable  ${impermissible_resource_prefix}-test
    # add policy to index
    ${response}=  Add Policy To Index  ${index_name}  ${policy_name}
    Should Be Equal As Strings  ${response.status_code}  403
    # read index
    ${response}=  Explain Index  ${index_name}  False
    Should Be Equal As Strings  ${response.status_code}  403
    # update index policy
    ${response}=  Change Index Policy  ${index_name}  {"policy_id": "${policy_name}", "state": "delete", "include": [{"state": "test_rollover"}]}
    Should Be Equal As Strings  ${response.status_code}  403
    # retry failed index
    ${response}=  Retry Failed Index  ${index_name}  {"state": "delete"}
    Should Be Equal As Strings  ${response.status_code}  403
    # delete policy from index
    ${response}=  Remove Policy From Index  ${index_name}
    Should Be Equal As Strings  ${response.status_code}  403
    [Teardown]  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Roll Over And Delete
    [Tags]  dbaas  dbaas_opensearch  dbaas_ism  dbaas_ism_rollover_and_delete  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}

    ${username_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].username
    ${password_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].password

    Set ISM Job Interval  1

    Login To OpenSearch  ${username_ism}  ${password_ism}
    ${policy_name}=  Set Variable  ${resource_prefix}-policy
    ${response}=  Create Policy  ${policy_name}  {"policy": {"description": "Rollover and remove", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 2}}], "transitions": [{"state_name": "remove", "conditions": {"min_doc_count": 3}}]}, {"name": "remove", "actions": [{"delete": {}}]}]}}
    Should Be Equal As Strings  ${response.status_code}  201
    Create OpenSearch Index Template  ${resource_prefix}-index-template  ${resource_prefix}*  {"plugins.index_state_management.rollover_alias": "${resource_prefix}"}
    ${index_name}=  Set Variable  ${resource_prefix}-000001
    ${response}=  Create OpenSearch Index  ${index_name}  {"aliases": {"${resource_prefix}": {"is_write_index": "true"}}}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Add Policy To Index  ${index_name}  ${policy_name}
    Add Document To Index By Id  ${index_name}  {"test": "first"}  1
    Add Document To Index By Id  ${index_name}  {"test": "second"}  2
    Add Document To Index By Id  ${index_name}  {"test": "third"}  3
    Add Document To Index By Id  ${index_name}  {"test": "fourth"}  4

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check OpenSearch Index Exists  ${resource_prefix}-000002

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check OpenSearch Index Does Not Exist  ${index_name}

    [Teardown]  Run Keywords  Remove Policy  ${policy_name}
    ...  AND  Set ISM Job Interval  null
    ...  AND  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}

Roll Over And Delete Index With ISM Template
    [Tags]  dbaas  dbaas_opensearch  dbaas_ism  dbaas_ism_rollover_and_delete_with_template  dbaas_v2
    ${response}=  Create Database Resource Prefix By Dbaas Agent
    Log  ${response}
    ${resourcePrefix}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].resourcePrefix
    Log  Resource Prefix: ${resourcePrefix}

    ${username_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].username
    ${password_ism}=  Get Items By Path  ${response}  $.connectionProperties[?(@.role=="ism")].password

    Set ISM Job Interval  1

    Login To OpenSearch  ${username_ism}  ${password_ism}
    ${policy_name}=  Set Variable  ${resource_prefix}-policy
    ${response}=  Create Policy  ${policy_name}  {"policy": {"description": "Rollover and remove", "default_state": "test_rollover", "states": [{"name": "test_rollover", "actions": [{"rollover": {"min_doc_count": 2}}], "transitions": [{"state_name": "remove", "conditions": {"min_doc_count": 3}}]}, {"name": "remove", "actions": [{"delete": {}}]}], "ism_template": {"index_patterns": ["${resource_prefix}*"], "priority": 100}}}
    Should Be Equal As Strings  ${response.status_code}  201
    Sleep  5s
    Create OpenSearch Index Template  ${resource_prefix}-index-template  ${resource_prefix}*  {"plugins.index_state_management.rollover_alias": "${resource_prefix}"}
    ${index_name}=  Set Variable  ${resource_prefix}-000001
    ${response}=  Create OpenSearch Index  ${index_name}  {"aliases": {"${resource_prefix}": {"is_write_index": "true"}}}
    Should Be Equal As Strings  ${response.status_code}  200
    Add Document To Index By Id  ${index_name}  {"test": "first"}  1
    Add Document To Index By Id  ${index_name}  {"test": "second"}  2
    Add Document To Index By Id  ${index_name}  {"test": "third"}  3
    Add Document To Index By Id  ${index_name}  {"test": "fourth"}  4

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check OpenSearch Index Exists  ${resource_prefix}-000002

    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check OpenSearch Index Does Not Exist  ${index_name}

    [Teardown]  Run Keywords  Remove Policy  ${policy_name}
    ...  AND  Set ISM Job Interval  null
    ...  AND  Delete Database Resource Prefix Dbaas Agent  ${resourcePrefix}
