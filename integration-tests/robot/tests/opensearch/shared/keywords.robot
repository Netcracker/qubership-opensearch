*** Variables ***
${OPENSEARCH_HOST}               %{OPENSEARCH_HOST}
${OPENSEARCH_PORT}               %{OPENSEARCH_PORT}
${OPENSEARCH_PROTOCOL}           %{OPENSEARCH_PROTOCOL}
${OPENSEARCH_USERNAME}           %{OPENSEARCH_USERNAME}
${OPENSEARCH_PASSWORD}           %{OPENSEARCH_PASSWORD}
${OPENSEARCH_MASTER_NODES_NAME}  %{OPENSEARCH_MASTER_NODES_NAME}
${OPENSEARCH_NAMESPACE}          %{OPENSEARCH_NAMESPACE}
${OPENSEARCH_CURATOR_USERNAME}   %{OPENSEARCH_CURATOR_USERNAME=}
${OPENSEARCH_CURATOR_PASSWORD}   %{OPENSEARCH_CURATOR_PASSWORD=}
${OPENSEARCH_CURATOR_PROTOCOL}   %{OPENSEARCH_CURATOR_PROTOCOL}
${OPENSEARCH_CURATOR_HOST}       %{OPENSEARCH_CURATOR_HOST}
${OPENSEARCH_CURATOR_PORT}       %{OPENSEARCH_CURATOR_PORT}
${RETRY_TIME}                    300s
${RETRY_INTERVAL}                10s

*** Settings ***
Library  Collections
Library  ./lib/FileSystemLibrary.py
Library  ./lib/OpenSearchUtils.py
Library  PlatformLibrary  managed_by_operator=true
Library  RequestsLibrary
Library  String
Library  json

*** Keywords ***
Prepare
    Prepare OpenSearch
    Prepare Curator
    Delete Data

Prepare OpenSearch
    [Arguments]  ${need_auth}=True
    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}  ${need_auth}

Prepare Curator
    ${auth}=  Create List  ${OPENSEARCH_CURATOR_USERNAME}  ${OPENSEARCH_CURATOR_PASSWORD}
    ${root_ca_path}=  Set Variable  /certs/curator/root-ca.pem
    ${root_ca_exists}=  File Exists  ${root_ca_path}
    ${verify}=  Set Variable If  ${root_ca_exists}  ${root_ca_path}  ${True}
    Create Session  curatorsession  ${OPENSEARCH_CURATOR_PROTOCOL}://${OPENSEARCH_CURATOR_HOST}:${OPENSEARCH_CURATOR_PORT}  auth=${auth}  verify=${verify}

Login To OpenSearch
    [Arguments]  ${username}  ${password}  ${need_auth}=True
    ${auth}=  Run Keyword If  ${need_auth}  Create List  ${username}  ${password}
    ${root_ca_path}=  Set Variable  /certs/opensearch/root-ca.pem
    ${root_ca_exists}=  File Exists  ${root_ca_path}
    ${verify}=  Set Variable If  ${root_ca_exists}  ${root_ca_path}  ${True}
    Create Session  opensearch  ${OPENSEARCH_PROTOCOL}://${OPENSEARCH_HOST}:${OPENSEARCH_PORT}  auth=${auth}  verify=${verify}  disable_warnings=1
    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Global Variable  ${headers}

Generate Index Name
    [Arguments]  ${index_name}
    ${suffix}=  Generate Random String  5  [LOWER]
    [Return]  ${index_name}-${suffix}

Create OpenSearch Index
    [Arguments]  ${name}  ${data}=${None}
    ${json}=  Run Keyword If  ${data}  To Json  ${data}
    ${response}=  Put Request  opensearch  /${name}  data=${json}  headers=${headers}
    Log  ${response.content}
    [Return]  ${response}

Get OpenSearch Index
    [Arguments]  ${name}  ${timeout}=${None}
    ${response}=  Get Request  opensearch  /${name}  timeout=${timeout}
    [Return]  ${response}

Delete OpenSearch Index
    [Arguments]  ${name}
    ${response}=  Delete Request  opensearch  /${name}
    [Return]  ${response}

Delete Data
    Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}
    Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-1
    Delete OpenSearch Index  ${OPENSEARCH_BACKUP_INDEX}-2
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Run Keywords
    ...  Check OpenSearch Index Does Not Exist  ${OPENSEARCH_BACKUP_INDEX}  AND
    ...  Check OpenSearch Index Does Not Exist  ${OPENSEARCH_BACKUP_INDEX}-1  AND
    ...  Check OpenSearch Index Does Not Exist  ${OPENSEARCH_BACKUP_INDEX}-2

Check OpenSearch Index Exists
    [Arguments]  ${name}
    ${response}=  Get OpenSearch Index  ${name}
    Should Be Equal As Strings  ${response.status_code}  200

Check OpenSearch Index Does Not Exist
    [Arguments]  ${name}
    ${response}=  Get OpenSearch Index  ${name}
    Should Be Equal As Strings  ${response.status_code}  404

Bulk Update Index Data
    [Arguments]  ${index_name}  ${binary_data}  ${timeout}=${None}
    &{local_headers}=  Create Dictionary  Content-Type=application/x-ndjson
    ${response}=  Post Request  opensearch  /${index_name}/_bulk  data=${binary_data}  headers=${local_headers}  timeout=${timeout}
    [Return]  ${response}

Bulk Update Data
    [Arguments]  ${binary_data}  ${timeout}=${None}
    &{local_headers}=  Create Dictionary  Content-Type=application/x-ndjson
    ${response}=  Post Request  opensearch  /_bulk  data=${binary_data}  headers=${local_headers}  timeout=${timeout}
    [Return]  ${response}

Create Document ${document} For Index ${index_name}
    Add Document To Index By Id  ${index_name}  ${document}  1

Add Document To Index By Id
    [Arguments]  ${index_name}  ${document}  ${id}
    ${response}=  Put Request  opensearch  /${index_name}/_create/${id}  data=${document}  headers=${headers}
    Log  ${response.content}
    Should Be Equal As Strings  ${response.status_code}  201

Update Document ${document} For Index ${index_name}
    ${document}=  Set Variable  {"doc":${document}}
    ${response}=  Post Request  opensearch  /${index_name}/_update/1  data=${document}  headers=${headers}
    [Return]  ${response}

Search Document
    [Arguments]  ${index_name}  ${timeout}=${None}
    ${response}=  Get Request  opensearch  /${index_name}/_search  timeout=${timeout}
    [Return]  ${response.content}

Search Document By Field
    [Arguments]  ${index_name}  ${field_name}  ${field_value}
    ${response}=  Get Request  opensearch  /${index_name}/_search?q=${field_name}:${field_value}
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

Find Document By Field
    [Arguments]  ${index_name}  ${field_name}  ${field_value}
    ${content}=  Search Document By Field  ${index_name}  ${field_name}  ${field_value}
    [Return]  ${content['hits']['hits'][0]['_source']}

Check That Document Exists By Field
    [Arguments]  ${index_name}  ${field_name}  ${field_value}
    ${content}=  Search Document By Field  ${index_name}  ${field_name}  ${field_value}
    Should Be True  ${content['hits']['total']['value']} > 0

Check That Document Does Not Exist By Field
    [Arguments]  ${index_name}  ${field_name}  ${field_value}
    ${content}=  Search Document By Field  ${index_name}  ${field_name}  ${field_value}
    Should Be True  ${content['hits']['total']['value']} == 0

Delete Document For Index ${index_name}
    Delete Document From Index By Id  ${index_name}  1

Delete Document From Index By Id
    [Arguments]  ${index_name}  ${id}
    ${response}=  Delete Request  opensearch  /${index_name}/_doc/${id}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Convert Json ${json} To Type
    ${json_dictionary}=  Evaluate  json.loads('''${json}''')  json
    [Return]  ${json_dictionary}

Get OpenSearch Status
    ${response}=  Get Request  opensearch  _cat/health?h=status
    ${content}=  Decode Bytes To String  ${response.content}  UTF-8
    [Return]  ${content.strip()}

Check OpenSearch Is Green
    ${status}=  Get OpenSearch Status
    Should Be Equal As Strings  ${status}  green

Get Index Uuid
    [Arguments]  ${index_name}
    ${response}=  Get Request  opensearch  _cat/indices/${index_name}?h=uuid
    ${content}=  Decode Bytes To String  ${response.content}  UTF-8
    [Return]  ${content.strip()}

Get Index Information
    [Arguments]  ${index_name}
    ${response}=  Get Request  opensearch  _cat/shards/${index_name}?v&h=shard,prirep,node&format=json
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

Get Master Node Name
    ${response}=  Get Request  opensearch  _cat/cluster_manager?h=node  timeout=10
    ${content}=  Decode Bytes To String  ${response.content}  UTF-8
    Should Be Equal As Strings  ${response.status_code}  200  OpenSearch returned ${response.status_code} code. Master node is not recognized
    [Return]  ${content.strip()}

Create OpenSearch Alias
    [Arguments]  ${index_name}  ${alias}
    ${response}=  Put Request  opensearch  /${index_name}/_alias/${alias}
    [Return]  ${response}

Create OpenSearch Index Template
    [Arguments]  ${template_name}  ${index_pattern}  ${settings}={"number_of_shards":1}
    ${template}=  Set Variable  {"index_patterns":["${index_pattern}"],"template": {"settings":${settings}}}
    ${response}=  Put Request  opensearch  /_index_template/${template_name}  data=${template}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Create OpenSearch Template
    [Arguments]  ${template_name}  ${index_pattern}  ${settings}={"number_of_shards":1}
    ${template}=  Set Variable  {"index_patterns":["${index_pattern}"],"settings":${settings}}
    ${response}=  Put Request  opensearch  /_template/${template_name}  data=${template}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Get OpenSearch Alias
    [Arguments]  ${alias}
    ${response}=  Get Request  opensearch  /_alias/${alias}
    [Return]  ${response}

Get OpenSearch Alias For Index
    [Arguments]  ${index_name}  ${alias}
    ${response}=  Get Request  opensearch  /${index_name}/_alias/${alias}
    [Return]  ${response}

Get OpenSearch Template
    [Arguments]  ${template_name}
    ${response}=  Get Request  opensearch  /_template/${template_name}
    [Return]  ${response}

Get OpenSearch Index Template
    [Arguments]  ${template_name}
    ${response}=  Get Request  opensearch  /_index_template/${template_name}
    [Return]  ${response}

Get OpenSearch User
    [Arguments]  ${username}
    ${response}=  Get Request  opensearch  /_plugins/_security/api/internalusers/${username}
    [Return]  ${response}

Check OpenSearch User Exists
    [Arguments]  ${username}
    ${response}=  Get OpenSearch User  ${username}
    Should Be Equal As Strings  ${response.status_code}  200

Make Index Read Only
    [Arguments]  ${index_name}
    ${body}=  Set Variable  {"settings":{"index.blocks.write":true}}
    ${response}=  Put Request  opensearch  /${index_name}/_settings  data=${body}  headers=${headers}
    [Return]  ${response}

Make Index Read Write
    [Arguments]  ${index_name}
    ${body}=  Set Variable  {"settings":{"index.blocks.write":false}}
    ${response}=  Put Request  opensearch  /${index_name}/_settings  data=${body}  headers=${headers}
    [Return]  ${response}

Enable Slow Log
    [Arguments]  ${index_name}
    ${body}=  Set Variable  {"search":{"slowlog":{"threshold":{"query":{"info":"0s"}}}}}
    ${response}=  Put Request  opensearch  /${index_name}/_settings  data=${body}  headers=${headers}
    [Return]  ${response}

Clone Index
    [Arguments]  ${index_name}  ${clone_index_name}
    ${response}=  Put Request  opensearch  /${index_name}/_clone/${clone_index_name}  headers=${headers}
    [Return]  ${response}

Full Backup
    ${response}=  Post Request  curatorsession  /backup
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${response.content}
    [Return]  ${response.content}

Granular Backup
    ${data}=  Set Variable  {"dbs":["${OPENSEARCH_BACKUP_INDEX}-1","${OPENSEARCH_BACKUP_INDEX}-2"]}
    ${response}=  Post Request  curatorsession  /backup  data=${data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Backup Status  ${response.content}
    [Return]  ${response.content}

Delete Backup
    [Arguments]  ${backup_id}
    ${response}=  Post Request  curatorsession  /evict/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200

Full Restore
    [Arguments]  ${backup_id}  ${indices_list}
    ${restore_data}=  Set Variable  {"vault":"${backup_id}","dbs":${indices_list}}
    ${response}=  Post Request  curatorsession  /restore  data=${restore_data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status  ${response.content}

Full Restore By Timestamp
    [Arguments]  ${backup_ts}
    ${restore_data}=  Set Variable  {"ts":"${backup_ts}"}
    ${response}=  Post Request  curatorsession  /restore  data=${restore_data}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Wait Until Keyword Succeeds  ${RETRY_TIME}  ${RETRY_INTERVAL}
    ...  Check Restore Status  ${response.content}

Get Backup Timestamp
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content['ts']}

Find Backup ID By Timestamp
    [Arguments]  ${backup_ts}
    ${find_data}=  Create Dictionary  ts=${backup_ts}
    ${response}=  Get Request  curatorsession  /find  json=${find_data}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content['id']}

Check Backup Status
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['failed']}  False

Check Restore Status
    [Arguments]  ${task_id}
    ${response}=  Get Request  curatorsession  /jobstatus/${task_id}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['status']}  Successful

Check Backup Absence By Curator
    [Arguments]  ${backup_id}
    ${response}=  Get Request  curatorsession  /listbackups/${backup_id}
    Should Be Equal As Strings  ${response.status_code}  404

Check Backup Absence By OpenSearch
    [Arguments]  ${backup_id}
    ${backup_id_in_lowercase}=  Convert To Lowercase  ${backup_id}
    ${response}=  Get Request  opensearch  /_snapshot/snapshots/${backup_id_in_lowercase}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  404

Create Index With Generated Data
    [Arguments]  ${index_name}
    ${response}=  Create OpenSearch Index  ${index_name}
    Should Be Equal As Strings  ${response.status_code}  200
    Generate And Add Unique Data To Index  ${index_name}

Generate And Add Unique Data To Index
    [Arguments]  ${index_name}
    ${document_name}=  Generate Random String  10
    Set Global Variable  ${document_name}
    ${document}=  Set Variable  {"name": "${document_name}", "age": "10"}
    Create Document ${document} For Index ${index_name}