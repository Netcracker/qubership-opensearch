*** Variables ***
${ICU_ANALYZE_BODY}         {"tokenizer": "icu_tokenizer", "text": "Über die Straße läuft ein naïver Café-Besitzer"}
${KUROMOJI_ANALYZE_BODY}    {"tokenizer": "kuromoji_tokenizer", "text": "東京都は日本の首都です"}
${PLUGIN_IDX}               plugin_analyzer_test
${SLEEP_TIME}               3s

*** Settings ***
Library  Collections
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Setup
Suite Teardown  Cleanup

*** Keywords ***
Setup
    Prepare OpenSearch
    ${body}=  Set Variable  {"settings":{"analysis":{"analyzer":{"icu_custom":{"type":"custom","tokenizer":"icu_tokenizer","filter":["icu_normalizer"]},"kuromoji_custom":{"type":"custom","tokenizer":"kuromoji_tokenizer"}}}},"mappings":{"properties":{"de_text":{"type":"text","analyzer":"icu_custom"},"jp_text":{"type":"text","analyzer":"kuromoji_custom"}}}}
    ${response}=  Create OpenSearch Index  ${PLUGIN_IDX}  ${body}
    Should Be Equal As Strings  ${response.status_code}  200

Cleanup
    Run Keyword And Ignore Error  Delete OpenSearch Index  ${PLUGIN_IDX}

*** Test Cases ***
Test Analysis ICU Plugin
    [Tags]  plugins  analysis_icu
    ${response}=  POST On Session  opensearch  /_analyze  data=${ICU_ANALYZE_BODY}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    ${tokens}=  Get From Dictionary  ${content}  tokens
    Should Not Be Empty  ${tokens}

Test Analysis Kuromoji Plugin
    [Tags]  plugins  analysis_kuromoji
    ${response}=  POST On Session  opensearch  /_analyze  data=${KUROMOJI_ANALYZE_BODY}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    ${tokens}=  Get From Dictionary  ${content}  tokens
    Should Not Be Empty  ${tokens}

Search ICU Analyzed Field
    [Tags]  plugins  analysis_icu  search
    ${doc}=  Set Variable  {"de_text":"Über die Straße läuft ein naïver Café-Besitzer","jp_text":""}
    ${response}=  PUT On Session  opensearch  /${PLUGIN_IDX}/_doc/1  data=${doc}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    POST On Session  opensearch  /${PLUGIN_IDX}/_refresh  headers=${headers}
    ${query}=  Set Variable  {"query":{"match":{"de_text":"café"}}}
    ${response}=  POST On Session  opensearch  /${PLUGIN_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  1
    ${query2}=  Set Variable  {"query":{"match":{"de_text":"strasse"}}}
    ${response2}=  POST On Session  opensearch  /${PLUGIN_IDX}/_search  data=${query2}  headers=${headers}
    Should Be Equal As Strings  ${response2.status_code}  200
    ${content2}=  Convert Json ${response2.content} To Type
    Should Be Equal As Integers  ${content2['hits']['total']['value']}  1

Search Kuromoji Analyzed Field
    [Tags]  plugins  analysis_kuromoji  search
    ${doc}=  Set Variable  {"de_text":"","jp_text":"東京都は日本の首都です"}
    ${response}=  PUT On Session  opensearch  /${PLUGIN_IDX}/_doc/2  data=${doc}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  201
    POST On Session  opensearch  /${PLUGIN_IDX}/_refresh  headers=${headers}
    ${query}=  Set Variable  {"query":{"match":{"jp_text":"東京"}}}
    ${response}=  POST On Session  opensearch  /${PLUGIN_IDX}/_search  data=${query}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Should Be Equal As Integers  ${content['hits']['total']['value']}  1
    ${query2}=  Set Variable  {"query":{"match":{"jp_text":"首都"}}}
    ${response2}=  POST On Session  opensearch  /${PLUGIN_IDX}/_search  data=${query2}  headers=${headers}
    Should Be Equal As Strings  ${response2.status_code}  200
    ${content2}=  Convert Json ${response2.content} To Type
    Should Be Equal As Integers  ${content2['hits']['total']['value']}  1
