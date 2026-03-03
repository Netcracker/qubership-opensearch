*** Variables ***
${SEARCH_IDX}    bwc_compat_search

*** Settings ***
Library  Collections
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Setup Search Data
Suite Teardown  Cleanup

*** Keywords ***
Setup Search Data
    Prepare OpenSearch
    ${body}=  Set Variable  {"settings":{"number_of_shards":1,"number_of_replicas":0},"mappings":{"properties":{"name":{"type":"text"},"age":{"type":"integer"},"city":{"type":"keyword"},"registered":{"type":"date"}}}}
    ${response}=  Create OpenSearch Index  ${SEARCH_IDX}  ${body}
    Should Be Equal As Strings  ${response.status_code}  200
    ${bulk}=  Set Variable  {"index":{"_id":"1"}}\n{"name":"Alice","age":30,"city":"Berlin","registered":"2024-01-15"}\n{"index":{"_id":"2"}}\n{"name":"Bob","age":25,"city":"London","registered":"2024-03-20"}\n{"index":{"_id":"3"}}\n{"name":"Charlie","age":35,"city":"Berlin","registered":"2024-06-01"}\n{"index":{"_id":"4"}}\n{"name":"Diana","age":28,"city":"Paris","registered":"2024-09-10"}\n{"index":{"_id":"5"}}\n{"name":"Eve","age":22,"city":"London","registered":"2025-01-05"}\n
    &{ndjson_headers}=  Create Dictionary  Content-Type=application/x-ndjson
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_bulk  data=${bulk}  headers=${ndjson_headers}
    Should Be Equal As Strings  ${response.status_code}  200
    POST On Session  opensearch  /${SEARCH_IDX}/_refresh  headers=${headers}
    Wait Until Keyword Succeeds  30s  3s  Verify All Documents Available

Verify All Documents Available
    ${query}=  Set Variable  {"query":{"match_all":{}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  5

Cleanup
    Run Keyword And Ignore Error  Delete OpenSearch Index  ${SEARCH_IDX}

*** Test Cases ***
Search Match All
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"match_all":{}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  5

Search Match Query
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"match":{"name":"Alice"}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  1
    Should Be Equal As Strings  ${content['hits']['hits'][0]['_source']['name']}  Alice

Search Term Query On Keyword Field
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"term":{"city":"Berlin"}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  2

Search Bool Query With Must And Filter
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"bool":{"must":[{"term":{"city":"London"}}],"filter":[{"range":{"age":{"gte":20}}}]}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  2

Search Range Query
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"range":{"age":{"gte":28,"lte":35}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  3

Search With Terms Aggregation
    [Tags]  compatibility  search  aggregation
    ${query}=  Set Variable  {"size":0,"aggs":{"cities":{"terms":{"field":"city"}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    ${buckets}=  Set Variable  ${content['aggregations']['cities']['buckets']}
    Length Should Be  ${buckets}  3

Search With Sort
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"match_all":{}},"sort":[{"age":{"order":"asc"}}]}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Strings  ${content['hits']['hits'][0]['_source']['name']}  Eve
    Should Be Equal As Strings  ${content['hits']['hits'][4]['_source']['name']}  Charlie

Count API
    [Tags]  compatibility  search
    ${response}=  GET On Session  opensearch  /${SEARCH_IDX}/_count
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['count']}  5

Search Wildcard Query Case Insensitive
    [Documentation]  Covers OpenSearch 2.5.0 breaking change: case_insensitive fix for wildcard on text fields
    [Tags]  compatibility  search  wildcard
    ${query}=  Set Variable  {"query":{"wildcard":{"name":{"value":"ali*","case_insensitive":true}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  1
    Should Be Equal As Strings  ${content['hits']['hits'][0]['_source']['name']}  Alice

Search Wildcard Case Sensitive Should Not Match
    [Documentation]  Wildcard without case_insensitive must NOT match capitalized text tokens
    [Tags]  compatibility  search  wildcard
    ${query}=  Set Variable  {"query":{"wildcard":{"city":{"value":"ber*"}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  0

Search Wildcard Case Insensitive On Keyword
    [Tags]  compatibility  search  wildcard
    ${query}=  Set Variable  {"query":{"wildcard":{"city":{"value":"ber*","case_insensitive":true}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  2

Search Multi Match Query
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"multi_match":{"query":"Berlin","fields":["name","city"]}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be True  ${content['hits']['total']['value']} >= 2

Search Exists Query
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"exists":{"field":"city"}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  5

Search With Highlight
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"match":{"name":"Alice"}},"highlight":{"fields":{"name":{}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Dictionary Should Contain Key  ${content['hits']['hits'][0]}  highlight

Search Date Range Query
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"range":{"registered":{"gte":"2024-06-01","lte":"2025-12-31"}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  3

Search With Size And From Pagination
    [Tags]  compatibility  search
    ${query}=  Set Variable  {"query":{"match_all":{}},"size":2,"from":0,"sort":[{"age":"asc"}]}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Length Should Be  ${content['hits']['hits']}  2
    Should Be Equal As Strings  ${content['hits']['hits'][0]['_source']['name']}  Eve
    ${query2}=  Set Variable  {"query":{"match_all":{}},"size":2,"from":2,"sort":[{"age":"asc"}]}
    ${response2}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query2}  headers=${headers}
    ${content2}=  Convert Json ${response2.content} To Type
    Length Should Be  ${content2['hits']['hits']}  2
    Should Be Equal As Strings  ${content2['hits']['hits'][0]['_source']['name']}  Diana

Search With Avg Aggregation
    [Tags]  compatibility  search  aggregation
    ${query}=  Set Variable  {"size":0,"aggs":{"avg_age":{"avg":{"field":"age"}}}}
    ${response}=  POST On Session  opensearch  /${SEARCH_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Numbers  ${content['aggregations']['avg_age']['value']}  28.0

Search Query String Syntax
    [Tags]  compatibility  search
    ${response}=  GET On Session  opensearch  /${SEARCH_IDX}/_search?q=name:Bob
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  1
