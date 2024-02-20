*** Variables ***
${OPENSEARCH_IS_DEGRADED_ALERT_NAME}  OpenSearchIsDegradedAlert
${OPENSEARCH_IS_DOWN_ALERT_NAME}      OpenSearchIsDownAlert
${ALERT_RETRY_TIME}                   5min
${ALERT_RETRY_INTERVAL}               10s
${SLEEP_TIME}                         10s

*** Settings ***
Library  MonitoringLibrary  host=%{PROMETHEUS_URL}
...                         username=%{PROMETHEUS_USER}
...                         password=%{PROMETHEUS_PASSWORD}
Resource  ../shared/keywords.robot

*** Keywords ***
Check That Prometheus Alert Is Active
    [Arguments]  ${alert_name}
    ${status}=  Get Alert Status  ${alert_name}  ${OPENSEARCH_NAMESPACE}
    Should Be Equal As Strings  ${status}  pending

Check That Prometheus Alert Is Inactive
    [Arguments]  ${alert_name}
    ${status}=  Get Alert Status  ${alert_name}  ${OPENSEARCH_NAMESPACE}
    Should Be Equal As Strings  ${status}  inactive

Scale Up Master Stateful Set
    [Arguments]  ${replicas}
    Set Replicas For Stateful Set  ${OPENSEARCH_MASTER_NODES_NAME}  ${OPENSEARCH_NAMESPACE}  ${replicas}
    Sleep  ${SLEEP_TIME}
    ${result}=  Check Service Of Stateful Sets Is Scaled  ${OPENSEARCH_MASTER_NODES_NAME}  ${OPENSEARCH_NAMESPACE}
    Should Be True  ${result}

*** Test Cases ***
OpenSearch Is Degraded Alert
    [Tags]  opensearch  prometheus  opensearch_prometheus_alert  opensearch_is_degraded_alert
    ${replicas}=  Get Stateful Set Replica Counts  ${OPENSEARCH_MASTER_NODES_NAME}  ${OPENSEARCH_NAMESPACE}
    Pass Execution If  ${replicas} < 3  OpenSearch cluster has less than 3 master nodes
    Scale Down Stateful Set  ${OPENSEARCH_MASTER_NODES_NAME}  ${OPENSEARCH_NAMESPACE}
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Active  ${OPENSEARCH_IS_DEGRADED_ALERT_NAME}
    Scale Up Master Stateful Set  ${replicas}
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Inactive  ${OPENSEARCH_IS_DEGRADED_ALERT_NAME}
    [Teardown]  Scale Up Master Stateful Set  ${replicas}

OpenSearch Is Down Alert
    [Tags]  opensearch  prometheus  opensearch_prometheus_alert  opensearch_is_down_alert
    ${replicas}=  Get Stateful Set Replica Counts  ${OPENSEARCH_MASTER_NODES_NAME}  ${OPENSEARCH_NAMESPACE}
    Pass Execution If  ${replicas} < 3  OpenSearch cluster has less than 3 master nodes
    Set Replicas For Stateful Set  ${OPENSEARCH_MASTER_NODES_NAME}  ${OPENSEARCH_NAMESPACE}  1
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Active  ${OPENSEARCH_IS_DOWN_ALERT_NAME}
    Scale Up Master Stateful Set  ${replicas}
    Wait Until Keyword Succeeds  ${ALERT_RETRY_TIME}  ${ALERT_RETRY_INTERVAL}
    ...  Check That Prometheus Alert Is Inactive  ${OPENSEARCH_IS_DOWN_ALERT_NAME}
    [Teardown]  Scale Up Master Stateful Set  ${replicas}
