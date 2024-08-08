*** Variables ***
${OPENSEARCH_HOST}               %{OPENSEARCH_HOST}
${OPENSEARCH_PORT}               %{OPENSEARCH_PORT}
${OPENSEARCH_PROTOCOL}           %{OPENSEARCH_PROTOCOL}
${OPENSEARCH_USERNAME}           %{OPENSEARCH_USERNAME}
${OPENSEARCH_PASSWORD}           %{OPENSEARCH_PASSWORD}
${OPENSEARCH_MASTER_NODES_NAME}  %{OPENSEARCH_MASTER_NODES_NAME}
${OPENSEARCH_NAMESPACE}          %{OPENSEARCH_NAMESPACE}

*** Settings ***
Library  Collections
Library  ./lib/FileSystemLibrary.py
Library  ./lib/OpenSearchUtils.py
Library  PlatformLibrary  managed_by_operator=true
Library  RequestsLibrary
Library  String
Library  json

*** Keywords ***
Prepare OpenSearch
    [Arguments]  ${need_auth}=True
    Login To OpenSearch  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}  ${need_auth}

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

Generate Database Name
    ${suffix}=  Generate Random String  15  [LOWER]
    [Return]  backup-test-${suffix}

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

Check OpenSearch Index Exists
    [Arguments]  ${name}
    ${response}=  Get OpenSearch Index  ${name}
    Should Be Equal As Strings  ${response.status_code}  200

Check OpenSearch Index Does Not Exist
    [Arguments]  ${name}
    ${response}=  Get OpenSearch Index  ${name}
    Should Be Equal As Strings  ${response.status_code}  404

Check OpenSearch Does Not Have Closed Indices
    ${response}=  Get OpenSearch Index  *?expand_wildcards=closed
    Check Response Is Empty  ${response}

Check Response Is Empty
    [Arguments]  ${response}
    Should Be Equal As Strings  ${response.status_code}  200
    Should Be Equal As Strings  ${response.text}  {}

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

Get OpenSearch Alias
    [Arguments]  ${alias}
    ${response}=  Get Request  opensearch  /_alias/${alias}
    [Return]  ${response}

Get OpenSearch Alias For Index
    [Arguments]  ${index_name}  ${alias}
    ${response}=  Get Request  opensearch  /${index_name}/_alias/${alias}
    [Return]  ${response}

Create OpenSearch Component Template
    [Arguments]  ${template_name}  ${settings}={}  ${aliases}={}
    ${template}=  Set Variable  {"template": {"settings":${settings}, "aliases": ${aliases}}}
    ${response}=  Put Request  opensearch  /_component_template/${template_name}  data=${template}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Create OpenSearch Index Template
    [Arguments]  ${template_name}  ${index_pattern}  ${settings}={}  ${aliases}={}  ${composed_of}=[]
    ${template}=  Set Variable  {"index_patterns":["${index_pattern}"],"template": {"settings":${settings}, "aliases": ${aliases}}, "composed_of": ${composed_of}}
    ${response}=  Put Request  opensearch  /_index_template/${template_name}  data=${template}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Create OpenSearch Template
    [Arguments]  ${template_name}  ${index_pattern}  ${settings}={}  ${aliases}={}
    ${template}=  Set Variable  {"index_patterns":["${index_pattern}"],"settings":${settings},"aliases": ${aliases}}
    ${response}=  Put Request  opensearch  /_template/${template_name}  data=${template}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Get OpenSearch Template
    [Arguments]  ${template_name}
    ${response}=  Get Request  opensearch  /_template/${template_name}
    [Return]  ${response}

Delete OpenSearch Template
    [Arguments]  ${template_name}
    ${response}=  Delete Request  opensearch  /_template/${template_name}
    [Return]  ${response}

Get OpenSearch Component Template
    [Arguments]  ${template_name}
    ${response}=  Get Request  opensearch  /_component_template/${template_name}
    [Return]  ${response}

Delete OpenSearch Component Template
    [Arguments]  ${template_name}
    ${response}=  Delete Request  opensearch  /_component_template/${template_name}
    [Return]  ${response}

Get OpenSearch Index Template
    [Arguments]  ${template_name}
    ${response}=  Get Request  opensearch  /_index_template/${template_name}
    [Return]  ${response}

Delete OpenSearch Index Template
    [Arguments]  ${template_name}
    ${response}=  Delete Request  opensearch  /_index_template/${template_name}
    [Return]  ${response}

Get OpenSearch Tasks
    ${response}=  Get Request  opensearch  /_tasks
    [Return]  ${response}

Get OpenSearch Task By ID
    [Arguments]  ${task_id}
    ${response}=  Get Request  opensearch  /_tasks/${task_id}
    [Return]  ${response}

Get OpenSearch Index Exists
    [Arguments]  ${index_name}
    ${response}=  Head Request  opensearch  /${index_name}
    [Return]  ${response}

Get OpenSearch User
    [Arguments]  ${username}
    ${response}=  Get Request  opensearch  /_plugins/_security/api/internalusers/${username}
    [Return]  ${response}

Check OpenSearch User Exists
    [Arguments]  ${username}
    ${response}=  Get OpenSearch User  ${username}
    Should Be Equal As Strings  ${response.status_code}  200

Get Index Settings
    [Arguments]  ${index_name}
    ${response}=  Get Request  opensearch  /${index_name}/_settings
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}

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

Create Policy
    [Arguments]  ${policy_name}  ${body}
    ${response}=  Put Request  opensearch  _plugins/_ism/policies/${policy_name}  data=${body}  headers=${headers}
    Log  ${response.text}
    [Return]  ${response}

Get Policy
    [Arguments]  ${policy_name}  ${with_content}=True
    ${response}=  Get Request  opensearch  _plugins/_ism/policies/${policy_name}
    Log  ${response.text}
    Run Keyword And Return If  ${with_content}  Get Response Content  ${response}
    [Return]  ${response}

Get Policies
    ${response}=  Get Request  opensearch  _plugins/_ism/policies
    [Return]  ${response}

Update Policy
    [Arguments]  ${policy_name}  ${seq_no}  ${primary_term}  ${body}
    ${response}=  Put Request  opensearch  _plugins/_ism/policies/${policy_name}?if_seq_no=${seq_no}&if_primary_term=${primary_term}  data=${body}  headers=${headers}
    [Return]  ${response}

Remove Policy
    [Arguments]  ${policy_name}
    ${response}=  Delete Request  opensearch  _plugins/_ism/policies/${policy_name}  headers=${headers}
    [Return]  ${response}

Add Policy To Index
    [Arguments]  ${index_name}  ${policy_name}
    ${response}=  Post Request  opensearch  _plugins/_ism/add/${index_name}  data={"policy_id": "${policy_name}"}  headers=${headers}
    [Return]  ${response}

Explain Index
    [Arguments]  ${index_name}  ${with_content}=True
    ${response}=  Get Request  opensearch  _plugins/_ism/explain/${index_name}
    Log  ${response.text}
    Run Keyword And Return If  ${with_content}  Get Response Content  ${response}
    [Return]  ${response}

Change Index Policy
    [Arguments]  ${index_name}  ${data}
    ${response}=  Post Request  opensearch  _plugins/_ism/change_policy/${index_name}  data=${data}  headers=${headers}
    [Return]  ${response}

Retry Failed Index
    [Arguments]  ${index_name}  ${data}
    ${response}=  Post Request  opensearch  _plugins/_ism/retry/${index_name}  data=${data}  headers=${headers}
    [Return]  ${response}

Remove Policy From Index
    [Arguments]  ${index_name}
    ${response}=  Post Request  opensearch  _plugins/_ism/remove/${index_name}  headers=${headers}
    [Return]  ${response}

Update Cluster Settings
    [Arguments]  ${body}
    ${response}=  Put Request  opensearch  _cluster/settings  data=${body}  headers=${headers}
    [Return]  ${response}

Get Response Content
    [Arguments]  ${response}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    [Return]  ${content}
