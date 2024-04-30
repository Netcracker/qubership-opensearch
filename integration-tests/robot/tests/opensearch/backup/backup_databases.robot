*** Variables ***
${OPENSEARCH_CURATOR_PROTOCOL}   %{OPENSEARCH_CURATOR_PROTOCOL}
${OPENSEARCH_CURATOR_HOST}       %{OPENSEARCH_CURATOR_HOST}
${OPENSEARCH_CURATOR_PORT}       %{OPENSEARCH_CURATOR_PORT}
${OPENSEARCH_CURATOR_USERNAME}   %{OPENSEARCH_CURATOR_USERNAME=}
${OPENSEARCH_CURATOR_PASSWORD}   %{OPENSEARCH_CURATOR_PASSWORD=}
${RETRY_TIME}                    300s
${RETRY_INTERVAL}                5s

*** Settings ***
Resource  ../shared/keywords.robot
Suite Setup  Prepare
Test Setup  Prepare Databases

*** Keywords ***
Prepare
    Prepare OpenSearch
    Prepare Curator

Prepare Curator
    ${auth}=  Create List  ${OPENSEARCH_CURATOR_USERNAME}  ${OPENSEARCH_CURATOR_PASSWORD}
    ${root_ca_path}=  Set Variable  /certs/curator/root-ca.pem
    ${root_ca_exists}=  File Exists  ${root_ca_path}
    ${verify}=  Set Variable If  ${root_ca_exists}  ${root_ca_path}  ${True}
    Create Session  curatorsession  ${OPENSEARCH_CURATOR_PROTOCOL}://${OPENSEARCH_CURATOR_HOST}:${OPENSEARCH_CURATOR_PORT}  auth=${auth}  verify=${verify}

Prepare Databases
    ${database}=  Generate Database Name
    Set Test Variable  ${database}
    ${database_two}=  Generate Database Name
    Set Test Variable  ${database_two}
    ${renaming_database}=  Generate Database Name
    Set Test Variable  ${renaming_database}

Delete Databases
    FOR  ${db}  IN  ${database}  ${database_two}  ${renaming_database}
        Delete OpenSearch Index  ${db}*
        Delete OpenSearch Index Template  ${db}*
        Delete OpenSearch Component Template  ${db}*
        Delete OpenSearch Template  ${db}*
        ${response}=  Get OpenSearch Index  ${db}*
        Check Response Is Empty  ${response}
        ${response}=  Get OpenSearch Index Template  ${db}*
        Should Be Equal As Strings  ${response.status_code}  404
        ${response}=  Get OpenSearch Component Template  ${db}*
        Should Be Equal As Strings  ${response.status_code}  404
        ${response}=  Get OpenSearch Template  ${db}*
        Should Be Equal As Strings  ${response.status_code}  404
        ${response}=  Get OpenSearch Alias  ${db}*
        Check Response Is Empty  ${response}
    END

Create Index With Generated Data
    [Arguments]  ${index_name}  ${data}=${None}
    ${response}=  Create OpenSearch Index  ${index_name}  ${data}
    Should Be Equal As Strings  ${response.status_code}  200
    Run Keyword And Return  Generate And Add Unique Data To Index By Id  ${index_name}

Generate And Add Unique Data To Index By Id
    [Arguments]  ${index_name}  ${id}=1
    ${document_name}=  Generate Random String  10
    ${document}=  Set Variable  {"name": "${document_name}", "age": "10"}
    Add Document To Index By Id  ${index_name}  ${document}  ${id}
    [Return]  ${document_name}

Granular Backup
    [Arguments]  ${databases}
    ${data}=  Set Variable  {"dbs":${databases}}
    ${response}=  Post Request  curatorsession  /backup  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${response.content}
    [Return]  ${response.text}

Check Backup Status
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['failed']}  False

Granular Restore
    [Arguments]  ${backup_id}  ${dbs_list}  ${renames}={}  ${clean}=false
    ${restore_data}=  Set Variable  {"vault":"${backup_id}","dbs":${dbs_list},"changeDbNames":${renames},"clean":"${clean}"}
    ${response}=  Post Request  curatorsession  /restore  data=${restore_data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status  ${response.content}

Check Restore Status
    [Arguments]  ${task_id}
    ${response}=  Get Request  curatorsession  /jobstatus/${task_id}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['status']}  Successful

Delete Backup
    [Arguments]  ${backup_id}
    ${response}=  Post Request  curatorsession  /evict/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200

*** Test Cases ***
Granular Backup And Restore With Alias
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_alias
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}
    Create OpenSearch Alias  ${index_name}  ${database}-alias

    ${backup_id}=  Granular Backup  ["${database}"]
    Delete Databases
    Granular Restore    ${backup_id}    ["${database}"]

    Check OpenSearch Index Exists    ${index_name}
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    ${response}=  Get OpenSearch Index Template  ${database}*
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Restore With Template
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_template
    Create OpenSearch Index Template  ${database}-template  ${database}*  {"number_of_shards":3, "number_of_replicas": 1}
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    ${backup_id}=  Granular Backup  ["${database}"]
    ${document_name_2}=  Generate And Add Unique Data To Index By Id  ${index_name}  3
    Granular Restore    ${backup_id}    ["${database}"]

    Check OpenSearch Index Exists    ${index_name}
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    Check That Document Does Not Exist By Field  ${index_name}  name  ${document_name_2}
    ${response}=  Get OpenSearch Index Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  404
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Restore With Component Templates
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_component_templates
    Create OpenSearch Component Template  ${database}-settings-template  settings={"number_of_shards":2, "number_of_replicas": 1}
    Create OpenSearch Component Template  ${database}-alias-template  aliases={"${database}-alias": {}}
    Create OpenSearch Index Template  ${database}-template  ${database}*  composed_of=["${database}-settings-template", "${database}-alias-template"]
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    ${backup_id}=  Granular Backup  ["${database}"]
    ${document_name_2}=  Generate And Add Unique Data To Index By Id  ${index_name}  3
    Delete Databases
    Granular Restore    ${backup_id}    ["${database}"]

    Check OpenSearch Index Exists    ${index_name}
    ${settings}=  Get Index Settings  ${index_name}
    Should Be Equal As Strings  ${settings['${index_name}']['settings']['index']['number_of_shards']}  2
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    Check That Document Does Not Exist By Field  ${index_name}  name  ${document_name_2}
    ${response}=  Get OpenSearch Index Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Component Template  ${database}-settings-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Component Template  ${database}-alias-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Restore With Obsolete Template
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_obsolete_template
    Create OpenSearch Template  ${database}-obsolete-template  ${database}*  {"number_of_shards":5, "number_of_replicas": 1}
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    ${backup_id}=  Granular Backup  ["${database}"]
    ${document_name_2}=  Generate And Add Unique Data To Index By Id  ${index_name}  3
    Granular Restore    ${backup_id}    ["${database}"]

    Check OpenSearch Index Exists    ${index_name}
    ${settings}=  Get Index Settings  ${index_name}
    Should Be Equal As Strings  ${settings['${index_name}']['settings']['index']['number_of_shards']}  5
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    Check That Document Does Not Exist By Field  ${index_name}  name  ${document_name_2}
    ${response}=  Get OpenSearch Template  ${database}-obsolete-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  404
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Restore With Template And Alias
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_alias_and_template
    Create OpenSearch Index Template  ${database}-template  ${database}*  {"number_of_shards":3, "number_of_replicas": 1}  {"${database}-alias": {}}
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    ${backup_id}=  Granular Backup  ["${database}"]
    Delete Databases
    Granular Restore    ${backup_id}    ["${database}"]

    Check OpenSearch Index Exists    ${index_name}
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    ${response}=  Get OpenSearch Index Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Renaming Restore
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_rename
    Create OpenSearch Component Template  ${database}-alias-template  aliases={"${database}-alias": {}}
    Create OpenSearch Index Template  ${database}-template  ${database}*  {"number_of_shards":3, "number_of_replicas": 1}  composed_of=["${database}-alias-template"]
    ${index_name_1}=  Generate Index Name  ${database}
    ${document_name_1}=  Create Index With Generated Data  ${index_name_1}
    ${index_name_2}=  Generate Index Name  ${database}
    ${document_name_2}=  Create Index With Generated Data  ${index_name_2}

    ${index_name_3}=  Replace String  ${index_name_1}  ${database}  ${renaming_database}
    ${document_name_3}=  Create Index With Generated Data  ${index_name_3}  {"settings": {"index": {"number_of_shards":3, "number_of_replicas": 1}}}

    ${backup_id}=  Granular Backup  ["${database}"]
    Granular Restore    ${backup_id}    ["${database}"]  renames={"${database}": "${renaming_database}"}

    Check OpenSearch Index Exists  ${index_name_1}
    Check That Document Exists By Field  ${index_name_1}  name  ${document_name_1}
    Check OpenSearch Index Exists  ${index_name_2}
    Check That Document Exists By Field  ${index_name_2}  name  ${document_name_2}
    ${response}=  Get OpenSearch Index Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  200

    Check OpenSearch Index Exists  ${index_name_3}
    Check That Document Exists By Field  ${index_name_3}  name  ${document_name_1}
    Check That Document Does Not Exist By Field  ${index_name_3}  name  ${document_name_3}
    ${new_index_name}=  Replace String  ${index_name_2}  ${database}  ${renaming_database}
    Check OpenSearch Index Exists    ${new_index_name}
    Check That Document Exists By Field  ${new_index_name}  name  ${document_name_2}
    ${response}=  Get OpenSearch Index Template  ${renaming_database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Component Template  ${renaming_database}-alias-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${renaming_database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Renaming Restore With Manual Data Deletion
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_rename_after_data_deletion
    Create OpenSearch Index Template  ${database}-template  ${database}*  {"number_of_shards":3, "number_of_replicas": 1}  {"${database}-alias": {}}
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    ${backup_id}=  Granular Backup  ["${database}"]
    Delete Databases
    Granular Restore    ${backup_id}    ["${database}"]  renames={"${database}": "${renaming_database}"}

    Check OpenSearch Index Does Not Exist    ${index_name}
    ${response}=  Get OpenSearch Index Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  404

    ${new_index_name}=  Replace String  ${index_name}  ${database}  ${renaming_database}
    Check OpenSearch Index Exists    ${new_index_name}
    Check That Document Exists By Field  ${new_index_name}  name  ${document_name}
    ${response}=  Get OpenSearch Index Template  ${renaming_database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${renaming_database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Partial Renaming Restore
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_partial_rename
    Create OpenSearch Index Template  ${database}-template  ${database}*  {"number_of_shards":3, "number_of_replicas": 1}  {"${database}-alias": {}}
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    Create OpenSearch Template  ${database_two}-template  ${database_two}*  {"number_of_shards":4, "number_of_replicas": 1}  {"${database_two}-alias": {}}
    ${index_name_two}=  Generate Index Name  ${database_two}
    ${document_name_two}=  Create Index With Generated Data  ${index_name_two}

    ${backup_id}=  Granular Backup  ["${database}", "${database_two}"]
    Delete Databases
    Granular Restore    ${backup_id}    ["${database}", "${database_two}"]  renames={"${database_two}": "${renaming_database}"}

    Check OpenSearch Index Exists  ${index_name}
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    ${response}=  Get OpenSearch Index Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  200

    Check OpenSearch Index Does Not Exist  ${index_name_two}
    ${response}=  Get OpenSearch Index Template  ${database_two}-template
    Should Be Equal As Strings  ${response.status_code}  404
    ${response}=  Get OpenSearch Alias  ${database_two}-alias
    Should Be Equal As Strings  ${response.status_code}  404
    ${new_index_name}=  Replace String  ${index_name_two}  ${database_two}  ${renaming_database}
    Check OpenSearch Index Exists    ${new_index_name}
    ${settings}=  Get Index Settings  ${new_index_name}
    Should Be Equal As Strings  ${settings['${new_index_name}']['settings']['index']['number_of_shards']}  4
    Check That Document Exists By Field  ${new_index_name}  name  ${document_name_two}
    ${response}=  Get OpenSearch Template  ${renaming_database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${renaming_database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Restore With Cleanup
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_cleanup
    Create OpenSearch Component Template  ${database}-settings-template  {"number_of_shards":3, "number_of_replicas": 1}
    Create OpenSearch Index Template  ${database}-template  ${database}*  aliases={"${database}-alias": {}}  composed_of=["${database}-settings-template"]
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    ${backup_id}=  Granular Backup  ["${database}"]
    Delete Databases
    Create OpenSearch Component Template  ${database}-settings-template  {"number_of_shards":4, "number_of_replicas": 1}
    Create OpenSearch Index Template  ${database}-template  ${database}*  aliases={"${database}-new-alias": {}}  composed_of=["${database}-settings-template"]
    ${index_name_2}=  Generate Index Name  ${database}
    ${document_name_2}=  Create Index With Generated Data  ${index_name_2}
    ${document_name_3}=  Create Index With Generated Data  ${index_name}

    Granular Restore    ${backup_id}    ["${database}"]    clean=true

    Check OpenSearch Index Exists  ${index_name}
    ${settings}=  Get Index Settings  ${index_name}
    Should Be Equal As Strings  ${settings['${index_name}']['settings']['index']['number_of_shards']}  3
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    Check That Document Does Not Exist By Field  ${index_name}  name  ${document_name_3}
    Check OpenSearch Index Does Not Exist  ${index_name_2}
    ${response}=  Get OpenSearch Index Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Component Template  ${database}-settings-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-new-alias
    Should Be Equal As Strings  ${response.status_code}  404
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}

Granular Backup And Renaming Restore With Cleanup
    [Tags]  opensearch  backup  granular_backup  backup_databases  restore_with_rename_and_cleanup
    Create OpenSearch Template  ${database}-template  ${database}*  {"number_of_shards":3, "number_of_replicas": 1}  {"${database}-alias": {}}
    ${index_name}=  Generate Index Name  ${database}
    ${document_name}=  Create Index With Generated Data  ${index_name}

    Create OpenSearch Index Template  ${database_two}-template  ${database_two}*  {"number_of_shards":2, "number_of_replicas": 1}  {"${database_two}-alias": {}}
    ${index_name_two}=  Generate Index Name  ${database_two}
    ${document_name_two}=  Create Index With Generated Data  ${index_name_two}

    Create OpenSearch Template  ${renaming_database}-template  ${renaming_database}*  {"number_of_shards":4, "number_of_replicas": 1}  {"${renaming_database}-alias": {}}
    ${index_name_three}=  Generate Index Name  ${renaming_database}
    ${document_name_three}=  Create Index With Generated Data  ${index_name_three}

    ${backup_id}=  Granular Backup  ["${database}","${database_two}"]
    Granular Restore    ${backup_id}    ["${database}","${database_two}"]  renames={"${database}": "${renaming_database}"}  clean=true

    Check OpenSearch Index Exists  ${index_name_two}
    Check That Document Exists By Field  ${index_name_two}  name  ${document_name_two}
    ${response}=  Get OpenSearch Index Template  ${database_two}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database_two}-alias
    Should Be Equal As Strings  ${response.status_code}  200

    Check OpenSearch Index Exists    ${index_name}
    Check That Document Exists By Field  ${index_name}  name  ${document_name}
    ${response}=  Get OpenSearch Template  ${database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${database}-alias
    Should Be Equal As Strings  ${response.status_code}  200

    Check OpenSearch Index Does Not Exist  ${index_name_three}
    ${new_index_name}=  Replace String  ${index_name}  ${database}  ${renaming_database}
    Check OpenSearch Index Exists    ${new_index_name}
    ${settings}=  Get Index Settings  ${new_index_name}
    Should Be Equal As Strings  ${settings['${new_index_name}']['settings']['index']['number_of_shards']}  3
    Check That Document Exists By Field  ${new_index_name}  name  ${document_name}
    ${response}=  Get OpenSearch Template  ${renaming_database}-template
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${renaming_database}-alias
    Should Be Equal As Strings  ${response.status_code}  200
    [Teardown]  Run Keywords  Delete Databases  AND  Delete Backup  ${backup_id}
