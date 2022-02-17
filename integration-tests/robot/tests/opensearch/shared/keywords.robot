*** Variables ***
${OPENSEARCH_HOST}               %{OPENSEARCH_HOST}
${OPENSEARCH_PORT}               %{OPENSEARCH_PORT}
${OPENSEARCH_USERNAME}           %{OPENSEARCH_USERNAME}
${OPENSEARCH_PASSWORD}           %{OPENSEARCH_PASSWORD}
${OPENSEARCH_MASTER_NODES_NAME}  %{OPENSEARCH_MASTER_NODES_NAME}
${OPENSEARCH_NAMESPACE}          %{OPENSEARCH_NAMESPACE}

*** Settings ***
Library  Collections
Library  ./lib/OpenSearchUtils.py
Library  PlatformLibrary  managed_by_operator=true
Library  RequestsLibrary
Library  json

*** Keywords ***
Prepare OpenSearch
    [Arguments]  ${need_auth}=True
    ${auth}=  Run Keyword If  ${need_auth}  Create List  ${OPENSEARCH_USERNAME}  ${OPENSEARCH_PASSWORD}
    Create Session  opensearch  http://${OPENSEARCH_HOST}:${OPENSEARCH_PORT}  auth=${auth}  disable_warnings=1
    &{headers}=  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Global Variable  ${headers}

Create OpenSearch Index
    [Arguments]  ${name}  ${data}=${None}
    ${json}=  Run Keyword If  ${data}  To Json  ${data}
    ${response}=  Put Request  opensearch  /${name}  data=${json}  headers=${headers}
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
    ${response}=  Get Request  opensearch  _cat/master?h=node  timeout=10
    ${content}=  Decode Bytes To String  ${response.content}  UTF-8
    Should Be Equal As Strings  ${response.status_code}  200  OpenSearch returned ${response.status_code} code. Master node is not recognized
    [Return]  ${content.strip()}

