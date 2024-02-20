*** Variables ***
${OPENSEARCH_SLOW_QUERIES_INDEX}  opensearch_slow_query_index
${SLOW_QUERY_METRIC}              opensearch_slow_query_took_millis
${RETRY_TIME}                     ${%{SLOW_QUERIES_INTERVAL_MINUTES} + 1}m
${RETRY_INTERVAL}                 10s
${SLEEP}                          5s

*** Settings ***
Library  MonitoringLibrary  host=%{PROMETHEUS_URL}
...                         username=%{PROMETHEUS_USER}
...                         password=%{PROMETHEUS_PASSWORD}
Resource  ../shared/keywords.robot
Suite Setup  Prepare OpenSearch

*** Keywords ***
Check Metric In Prometheus
    [Arguments]  ${index_name}
    ${data}=  Get Metric Values  ${SLOW_QUERY_METRIC}
    ${metric}=  Create Dictionary
    FOR  ${result}  IN  @{data['result']}
        IF  "${result['metric']['index']}" != "${index_name}"  CONTINUE
        ${metric}=  Set Variable  ${result['metric']}
    END
    Should Not Be Empty  ${metric}
    [Return]  ${metric}

*** Test Cases ***
Produce Slow Query Metric
    [Tags]  opensearch  prometheus  slow_query
    ${index_name}=  Generate Index Name  ${OPENSEARCH_SLOW_QUERIES_INDEX}
    ${response}=  Create OpenSearch Index  ${index_name}
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Set Variable  {"name": "Fox", "age": "62"}
    Create Document ${document} For Index ${index_name}
    Sleep  ${SLEEP}

    ${response}=  Enable Slow Log  ${index_name}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Search Document  ${index_name}
    ${metric}=  Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Metric In Prometheus  ${index_name}

    Should Be Equal As Strings  ${metric['index']}  ${index_name}
    Should Be Equal As Strings  ${metric['shard']}  0
    Should Be Equal As Strings  ${metric['query']}  {}
    Should Be Equal As Strings  ${metric['total_hits']}  1

    [Teardown]  Delete OpenSearch Index  ${index_name}
