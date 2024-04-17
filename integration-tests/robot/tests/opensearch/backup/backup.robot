*** Variables ***
${OPENSEARCH_BACKUP_INDEX}   opensearch_backup_index

*** Settings ***
Resource  ../shared/keywords.robot
Resource  backup_keywords.robot
Suite Setup  Prepare
Test Teardown  Delete Data  ${OPENSEARCH_BACKUP_INDEX}

*** Test Cases ***
Find Backup By Timestamp
    [Tags]  opensearch  backup  find_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup  ["${OPENSEARCH_BACKUP_INDEX}-1","${OPENSEARCH_BACKUP_INDEX}-2"]
    ${backup_ts}=  Get Backup Timestamp  ${backup_id}
    ${found_backup_id}=  Find Backup ID By Timestamp  ${backup_ts}
    Should Be Equal As Strings  ${backup_id}  ${found_backup_id}
    [Teardown]  Run Keywords  Delete Data  ${OPENSEARCH_BACKUP_INDEX}  AND  Delete Backup  ${backup_id}

Full Backup And Restore
    [Tags]  opensearch  backup  full_backup
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}
    ${backup_id}=  Full Backup

    Delete Data  ${OPENSEARCH_BACKUP_INDEX}
    Create OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-1

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}  name  ${document_name}
    Check OpenSearch Does Not Have Closed Indices
    [Teardown]  Run Keywords  Delete Data  ${OPENSEARCH_BACKUP_INDEX}  AND  Delete Backup  ${backup_id}

Granular Backup And Restore
    [Tags]  opensearch  backup  granular_backup  restore
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup  ["${OPENSEARCH_BACKUP_INDEX}-1","${OPENSEARCH_BACKUP_INDEX}-2"]

    ${response}=  Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-1
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Set Variable  {"age": "1"}
    Update Document ${document} For Index ${OPENSEARCH_BACKUP_INDEX}-2

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}-1", "${OPENSEARCH_BACKUP_INDEX}-2"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-1
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-2
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}-2  age  10
    [Teardown]  Run Keywords  Delete Data  ${OPENSEARCH_BACKUP_INDEX}  AND  Delete Backup  ${backup_id}

Granular Backup And Restore By Timestamp
    [Tags]  opensearch  backup  granular_backup  restore_by_timestamp
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup  ["${OPENSEARCH_BACKUP_INDEX}-1","${OPENSEARCH_BACKUP_INDEX}-2"]
    ${backup_ts}=  Get Backup Timestamp  ${backup_id}

    Delete Data  ${OPENSEARCH_BACKUP_INDEX}

    Full Restore By Timestamp  ${backup_ts}  ["${OPENSEARCH_BACKUP_INDEX}-1", "${OPENSEARCH_BACKUP_INDEX}-2"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-1
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-2
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}-2  age  10
    [Teardown]  Run Keywords  Delete Data  ${OPENSEARCH_BACKUP_INDEX}  AND  Delete Backup  ${backup_id}

Delete Backup By ID
    [Tags]  opensearch  backup  backup_deletion
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup  ["${OPENSEARCH_BACKUP_INDEX}-1","${OPENSEARCH_BACKUP_INDEX}-2"]
    Delete Backup  ${backup_id}
    Check Backup Absence By Curator  ${backup_id}
    Check Backup Absence By OpenSearch  ${backup_id}

Unauthorized Access
    [Tags]  opensearch  backup  unauthorized_access
    Create Session  curator_unauthorized  ${OPENSEARCH_CURATOR_PROTOCOL}://${OPENSEARCH_CURATOR_HOST}:${OPENSEARCH_CURATOR_PORT}
    ...  disable_warnings=1
    ${response}=  Post Request  curator_unauthorized  /backup
    Should Be Equal As Strings  ${response.status_code}  401