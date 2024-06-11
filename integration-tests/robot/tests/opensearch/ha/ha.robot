*** Variables ***
${HA_TEST_INDEX_NAME}            ha_test_index
${CHECK_RESULT_RETRY_COUNT}      15x
${CHECK_RESULT_RETRY_INTERVAL}   5s
${SLEEP_TIME}                    20s

*** Settings ***
Library  OperatingSystem
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Prepare
Test Setup  Prepare Test

*** Keywords ***
Prepare
    Prepare OpenSearch

Prepare Test
    Wait Until Keyword Succeeds  ${CHECK_RESULT_RETRY_COUNT}  ${CHECK_RESULT_RETRY_INTERVAL}
    ...  Check OpenSearch Is Green
    ${postfix}=  Generate Random String  5  [LOWER]
    Set Global Variable  ${index_name}  ${HA_TEST_INDEX_NAME}-${postfix}
    &{data_files}=  Get Generated Test Case Data  ${TEST_NAME}  ${index_name}
    Set Global Variable  ${data_files}
    Generate Data  ${data_files}

Generate Data
    [Arguments]  ${data_files}  ${timeout}=${None}
    ${response}=  Get OpenSearch Index  ${index_name}
    Run Keyword If  ${response.status_code} == 200  Delete OpenSearch Index  ${index_name}
    ${data}=  Get File  ${data_files.index_settings_file}
    ${response}=  Create OpenSearch Index  ${index_name}  data=${data}
    Should Be Equal As Strings  ${response.status_code}  200
    ${binary_data}=  Get Binary File  ${data_files.index_create_file}
    ${response}=  Bulk Update Index Data  ${index_name}  ${binary_data}  timeout=${timeout}
    Log  ${response.content}
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}

Check Master Node Changed
    [Arguments]  ${old_master_node}
    ${new_master_node}=  Get Master Node Name
    Should Not Be Equal  ${old_master_node}  ${new_master_node}

Check Replica Shard Become Primary
    [Arguments]  ${index_name}  ${row}
    ${content}=  Search Document  ${index_name}
    ${shards_distribution}=  Get Index Information  ${index_name}
    ${boolean_result}=  Replica Shard Become Primary  ${shards_distribution}  ${row['shard']}  ${row['replica_service']}
    Should Be True  ${boolean_result}

Check Replica Shard Is Relocated
    [Arguments]  ${index_name}  ${row}
    ${binary_data}=  Get Binary File  ${data_files.index_update_file}
    Bulk Update Data  ${binary_data}
    ${shards_distribution}=  Get Index Information  ${index_name}
    ${boolean_result}=  Is Replica Shard Relocated  ${shards_distribution}  ${row['shard']}  ${row['node']}
    Should Be True  ${boolean_result}

*** Test Cases ***
Elected Master Is Crashed
    [Tags]  ha  opensearch_ha  ha_elected_master_is_crashed
    [Setup]  None
    ${master_node}=  Get Master Node Name
    Delete Pod By Pod Name  ${master_node}  ${OPENSEARCH_NAMESPACE}
    Sleep  ${SLEEP_TIME}
    ${statefulset_names}=  Create List  ${OPENSEARCH_MASTER_NODES_NAME}
    Check Service Of Stateful Sets Is Scaled  ${statefulset_names}  ${OPENSEARCH_NAMESPACE}
    Wait Until Keyword Succeeds  ${CHECK_RESULT_RETRY_COUNT}  ${CHECK_RESULT_RETRY_INTERVAL}
    ...  Check Master Node Changed  ${master_node}
    Sleep  ${SLEEP_TIME}

Data Files Corrupted On Primary Shard
    [Tags]  ha  opensearch_ha  ha_data_files_corrupted_on_primary_shard
    ${uuid}=  Get Index Uuid  ${index_name}
    ${index_information}=  Get Index Information  ${index_name}
    ${row}=  Get Primary Shard Description To Corrupt  ${index_information}
    ${command}=  Get Command To Corrupt Shard  ${uuid}  ${row['shard']}
    Execute Command In Pod  ${row['node']}  ${OPENSEARCH_NAMESPACE}  ${command}
    Sleep  ${SLEEP_TIME}
    # OpenSearch doesn’t detect problems with data files before reading.
    # After several requests OpenSearch reassigns shard with corrupted files and all requests finish successfully.
    Wait Until Keyword Succeeds  ${CHECK_RESULT_RETRY_COUNT}  ${CHECK_RESULT_RETRY_INTERVAL}
    ...  Check Replica Shard Become Primary  ${index_name}  ${row}
    [Teardown]  Delete OpenSearch Index  ${index_name}

Data Files Corrupted On Replica Shard
    [Tags]  ha  opensearch_ha  ha_data_files_corrupted_on_replica_shard
    ${uuid}=  Get Index Uuid  ${index_name}
    ${index_information}=  Get Index Information  ${index_name}
    ${row}=  Get Replica Shard Description To Corrupt  ${index_information}
    ${command}=  Get Command to Corrupt Shard  ${uuid}  ${row['shard']}
    Execute Command In Pod  ${row['node']}  ${OPENSEARCH_NAMESPACE}  ${command}
    Sleep  ${SLEEP_TIME}
    Wait Until Keyword Succeeds  ${CHECK_RESULT_RETRY_COUNT}  ${CHECK_RESULT_RETRY_INTERVAL}
    ...  Check Replica Shard Is Relocated  ${index_name}  ${row}
    [Teardown]  Delete OpenSearch Index  ${index_name}
