*** Variables ***
${OPENSEARCH_BACKUP_INDEX}   opensearch_s3_backup_index
${S3_BUCKET}                 %{S3_BUCKET}
${BACKUP_STORAGE_PATH}       /backup-storage
${TIMEOUT}                   10

*** Settings ***
Resource          ../shared/keywords.robot
Resource          backup_keywords.robot
Suite Setup       Prepare
Test Teardown     Delete Data

Library           S3BackupLibrary  url=%{S3_URL}
...               bucket=%{S3_BUCKET}
...               key_id=%{S3_KEY_ID}
...               key_secret=%{S3_KEY_SECRET}
...               ssl_verify=false

*** Test Cases ***
Full Backup And Restore On S3 Storage
    [Tags]  opensearch  backup  full_backup  s3_storage  full_backup_s3
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}
    ${backup_id}=  Full Backup
    # Creating backup files takes some time
    sleep  ${TIMEOUT}
    Delete Data

    #Check backup created in S3
    ${backup_file_exist}=  Check Backup Exists    path=${BACKUP_STORAGE_PATH}    backup_id=${backup_id}
    Should Be True  ${backup_file_exist}

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}  name  ${document_name}

    #Remove backup from S3
    Delete Backup  ${backup_id}
    # evicting backup files takes some time
    sleep  ${TIMEOUT}
    ${backup_file_exist}=  Check Backup Exists    path=${BACKUP_STORAGE_PATH}    backup_id=${backup_id}
    Should Not Be True  ${backup_file_exist}
    [Teardown]  Run Keywords  Delete Data

Granular Backup And Restore On S3 Storage
    [Tags]  opensearch  backup  granular_backup  s3_storage  granular_backup_s3
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-1
    Create Index With Generated Data  ${OPENSEARCH_BACKUP_INDEX}-2
    ${backup_id}=  Granular Backup
    # Creating backup files takes some time
    sleep  ${TIMEOUT}

    ${response}=  Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-1
    Should Be Equal As Strings  ${response.status_code}  200
    ${document}=  Set Variable  {"age": "1"}
    Update Document ${document} For Index ${OPENSEARCH_BACKUP_INDEX}-2

    #Check backup created in S3
    ${backup_file_exist}=  Check Backup Exists    path=${BACKUP_STORAGE_PATH}/granular    backup_id=${backup_id}/
    Should Be True  ${backup_file_exist}

    Full Restore  ${backup_id}  ["${OPENSEARCH_BACKUP_INDEX}-1", "${OPENSEARCH_BACKUP_INDEX}-2"]
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-1
    Check OpenSearch Index Exists  ${OPENSEARCH_BACKUP_INDEX}-2
    Check That Document Exists By Field  ${OPENSEARCH_BACKUP_INDEX}-2  age  10

    #Remove backup from S3
    Delete Backup  ${backup_id}
    # evicting backup files takes some time
    sleep  ${TIMEOUT}
    ${backup_file_exist}=  Check Backup Exists    path=${BACKUP_STORAGE_PATH}/granular    backup_id=${backup_id}/
    Should Not Be True  ${backup_file_exist}
    [Teardown]  Run Keywords  Delete Data
