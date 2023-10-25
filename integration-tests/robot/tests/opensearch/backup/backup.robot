*** Variables ***
${OPENSEARCH_CURATOR_PROTOCOL}   %{OPENSEARCH_CURATOR_PROTOCOL}
${OPENSEARCH_CURATOR_HOST}       %{OPENSEARCH_CURATOR_HOST}
${OPENSEARCH_CURATOR_PORT}       %{OPENSEARCH_CURATOR_PORT}
${OPENSEARCH_CURATOR_USERNAME}   %{OPENSEARCH_CURATOR_USERNAME=}
${OPENSEARCH_CURATOR_PASSWORD}   %{OPENSEARCH_CURATOR_PASSWORD=}
${RETRY_TIME}                    300s
${RETRY_INTERVAL}                10s
${OPENSEARCH_BACKUP_INDEX}       opensearch_backup_index

*** Settings ***
Resource  ../shared/keywords.robot
Suite Setup  Prepare
Test Teardown  Delete Data

*** Keywords ***
Prepare
    Prepare OpenSearch
    Prepare Curator
    Delete Data

*** Test Cases ***
Find Backup By Timestamp
    [Tags]  opensearch  backup  find_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup
    ${backup_ts}=  Get Backup Timestamp  ${backup_id}
    ${found_backup_id}=  Find Backup ID By Timestamp  ${backup_ts}
    Should Be Equal As Strings  ${backup_id}  ${found_backup_id}
    [Teardown]  Run Keywords  Delete Data  AND  Delete Backup  ${backup_id}

Full Backup And Restore
    [Tags]  opensearch  backup  full_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}
    ${backup_id}=  Full Backup

    Delete Data

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}  name  ${document_name}
    [Teardown]  Run Keywords  Delete Data  AND  Delete Backup  ${backup_id}

Granular Backup And Restore
    [Tags]  opensearch  backup  granular_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup

    ${response}=  Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-1
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Set Variable  {"age": "1"}
    Update Document ${document} For Index ${OPENSEARCH_BACKUP_INDEX}-2

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}-1", "${OPENSEARCH_BACKUP_INDEX}-2"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-1
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-2
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}-2  age  10
    [Teardown]  Run Keywords  Delete Data  AND  Delete Backup  ${backup_id}

Granular Backup And Restore By Timestamp
    [Tags]  opensearch  backup  granular_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup
    ${backup_ts}=  Get Backup Timestamp  ${backup_id}

    Delete Data

    Full Restore By Timestamp  ${backup_ts}
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-1
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-2
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}-2  age  10
    [Teardown]  Run Keywords  Delete Data  AND  Delete Backup  ${backup_id}

Delete Backup By ID
    [Tags]  opensearch  backup  backup_deletion
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup
    Delete Backup  ${backup_id}
    Check Backup Absence By Curator  ${backup_id}
    Check Backup Absence By OpenSearch  ${backup_id}

Unauthorized Access
    [Tags]  opensearch  backup  unauthorized_access
    Create Session  curator_unauthorized  ${OPENSEARCH_CURATOR_PROTOCOL}://${OPENSEARCH_CURATOR_HOST}:${OPENSEARCH_CURATOR_PORT}
    ...  disable_warnings=1
    ${response}=  Post Request  curator_unauthorized  /backup
    Should Be Equal As Strings  ${response.status_code}  401