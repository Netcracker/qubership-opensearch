*** Variables ***
${DOC_IDX}       bwc_compat_doc
${SLEEP_TIME}    3s

*** Settings ***
Library  Collections
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Setup
Suite Teardown  Cleanup

*** Keywords ***
Setup
    Prepare OpenSearch
    ${body}=  Set Variable  {"settings":{"number_of_shards":1,"number_of_replicas":0},"mappings":{"properties":{"title":{"type":"text"},"count":{"type":"integer"}}}}
    ${response}=  Create OpenSearch Index  ${DOC_IDX}  ${body}
    Should Be Equal As Strings  ${response.status_code}  200

Cleanup
    Run Keyword And Ignore Error  Delete OpenSearch Index  ${DOC_IDX}

*** Test Cases ***
Index Document With Explicit ID
    [Tags]  compatibility  document
    ${doc}=  Set Variable  {"title":"First Document","count":1}
    ${response}=  PUT On Session  opensearch  /${DOC_IDX}/_doc/doc1  data=${doc}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201

Get Document By ID
    [Tags]  compatibility  document
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/doc1
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be True  ${content['found']}
    Should Be Equal As Strings  ${content['_source']['title']}  First Document
    Should Be Equal As Integers  ${content['_source']['count']}  1

Create Document With Auto Generated ID
    [Tags]  compatibility  document
    ${doc}=  Set Variable  {"title":"Auto ID Document","count":2}
    ${response}=  POST On Session  opensearch  /${DOC_IDX}/_doc  data=${doc}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${content}=  Convert Json ${response.content} To Type
    Set Suite Variable  ${AUTO_DOC_ID}  ${content['_id']}
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/${AUTO_DOC_ID}
    Should Be Equal As Strings  ${response.status_code}  200

Update Document Via Update API
    [Tags]  compatibility  document
    ${body}=  Set Variable  {"doc":{"count":10,"title":"Updated Document"}}
    ${response}=  POST On Session  opensearch  /${DOC_IDX}/_update/doc1  data=${body}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/doc1
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['_source']['count']}  10
    Should Be Equal As Strings  ${content['_source']['title']}  Updated Document

Delete Document By ID
    [Tags]  compatibility  document
    ${response}=  DELETE On Session  opensearch  /${DOC_IDX}/_doc/${AUTO_DOC_ID}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/${AUTO_DOC_ID}  expected_status=404
    Should Be Equal As Strings  ${response.status_code}  404

Bulk Index Documents
    [Tags]  compatibility  document
    ${bulk}=  Set Variable  {"index":{"_id":"b1"}}\n{"title":"Bulk One","count":100}\n{"index":{"_id":"b2"}}\n{"title":"Bulk Two","count":200}\n{"index":{"_id":"b3"}}\n{"title":"Bulk Three","count":300}\n
    &{ndjson_headers}=  Create Dictionary  Content-Type=application/x-ndjson
    ${response}=  POST On Session  opensearch  /${DOC_IDX}/_bulk  data=${bulk}  headers=${ndjson_headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Not Be True  ${content['errors']}

Get Document With Source Filtering
    [Tags]  compatibility  document
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/doc1?_source_includes=title
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Dictionary Should Contain Key  ${content['_source']}  title
    Dictionary Should Not Contain Key  ${content['_source']}  count

Get Document With Source Excludes
    [Tags]  compatibility  document
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/doc1?_source_excludes=count
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Dictionary Should Contain Key  ${content['_source']}  title
    Dictionary Should Not Contain Key  ${content['_source']}  count

Index Document With Unicode Content
    [Tags]  compatibility  document
    ${doc}=  Set Variable  {"title":"Über café naïve résumé —�都","count":42}
    ${response}=  PUT On Session  opensearch  /${DOC_IDX}/_doc/unicode1  data=${doc}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/unicode1
    ${content}=  Convert Json ${response.content} To Type
    Should Contain  ${content['_source']['title']}  café

Index Document With Long ID
    [Tags]  compatibility  document
    ${long_id}=  Generate Random String  500  [LETTERS]
    ${doc}=  Set Variable  {"title":"Long ID Doc","count":99}
    ${response}=  PUT On Session  opensearch  /${DOC_IDX}/_doc/${long_id}  data=${doc}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_doc/${long_id}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be True  ${content['found']}

Refresh And Multi Get Documents
    [Tags]  compatibility  document
    ${response}=  POST On Session  opensearch  /${DOC_IDX}/_refresh  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${body}=  Set Variable  {"ids":["doc1","b1","b2","b3"]}
    ${response}=  POST On Session  opensearch  /${DOC_IDX}/_mget  data=${body}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Length Should Be  ${content['docs']}  4
    FOR  ${doc}  IN  @{content['docs']}
        Should Be True  ${doc['found']}
    END

Verify Document Count Via Count API
    [Tags]  compatibility  document
    ${response}=  GET On Session  opensearch  /${DOC_IDX}/_count
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be True  ${content['count']} >= 5
