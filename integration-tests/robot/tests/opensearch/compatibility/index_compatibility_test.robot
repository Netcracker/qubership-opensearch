*** Variables ***
${BWC_IDX}              bwc_compat_index
${BWC_ALIAS}            bwc_compat_alias
${BWC_TMPL}             bwc_compat_template
${BWC_LEGACY_TMPL}      bwc_compat_legacy_tmpl
${BWC_COMP_TMPL}        bwc_compat_comp_tmpl
${BWC_COMPOSED_TMPL}    bwc_compat_composed_tmpl
${BWC_SPECIAL_IDX}      bwc-special.dots.and-dashes
${SLEEP_TIME}           3s

*** Settings ***
Library  Collections
Library  String
Resource  ../shared/keywords.robot
Suite Setup  Prepare
Suite Teardown  Cleanup

*** Keywords ***
Prepare
    Prepare OpenSearch

Cleanup
    Run Keyword And Ignore Error  Delete OpenSearch Index  ${BWC_IDX}
    Run Keyword And Ignore Error  Delete OpenSearch Index  ${BWC_SPECIAL_IDX}
    Run Keyword And Ignore Error  Delete OpenSearch Index Template  ${BWC_TMPL}
    Run Keyword And Ignore Error  Delete OpenSearch Index Template  ${BWC_COMPOSED_TMPL}
    Run Keyword And Ignore Error  Delete OpenSearch Component Template  ${BWC_COMP_TMPL}
    Run Keyword And Ignore Error  Delete OpenSearch Template  ${BWC_LEGACY_TMPL}

*** Test Cases ***
Create Index With Settings And Mappings
    [Tags]  compatibility  index
    ${body}=  Set Variable  {"settings":{"number_of_shards":1,"number_of_replicas":0},"mappings":{"properties":{"title":{"type":"text"},"status":{"type":"keyword"},"count":{"type":"integer"}}}}
    ${response}=  Create OpenSearch Index  ${BWC_IDX}  ${body}
    Should Be Equal As Strings  ${response.status_code}  200

Verify Index Exists
    [Tags]  compatibility  index
    Check OpenSearch Index Exists  ${BWC_IDX}

Get Index Settings
    [Tags]  compatibility  index
    ${settings}=  Get Index Settings  ${BWC_IDX}
    Should Be Equal As Strings  ${settings['${BWC_IDX}']['settings']['index']['number_of_shards']}  1
    Should Be Equal As Strings  ${settings['${BWC_IDX}']['settings']['index']['number_of_replicas']}  0

Get Index Mappings
    [Tags]  compatibility  index
    ${response}=  GET On Session  opensearch  /${BWC_IDX}/_mapping
    Should Be Equal As Strings  ${response.status_code}  200
    ${content}=  Convert Json ${response.content} To Type
    Dictionary Should Contain Key  ${content['${BWC_IDX}']['mappings']['properties']}  title
    Dictionary Should Contain Key  ${content['${BWC_IDX}']['mappings']['properties']}  status
    Dictionary Should Contain Key  ${content['${BWC_IDX}']['mappings']['properties']}  count

Update Index Settings
    [Tags]  compatibility  index
    ${body}=  Set Variable  {"index":{"number_of_replicas":1}}
    ${response}=  PUT On Session  opensearch  /${BWC_IDX}/_settings  data=${body}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${settings}=  Get Index Settings  ${BWC_IDX}
    Should Be Equal As Strings  ${settings['${BWC_IDX}']['settings']['index']['number_of_replicas']}  1

Create And Verify Alias
    [Tags]  compatibility  index  alias
    ${response}=  Create OpenSearch Alias  ${BWC_IDX}  ${BWC_ALIAS}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias  ${BWC_ALIAS}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Alias For Index  ${BWC_IDX}  ${BWC_ALIAS}
    Should Be Equal As Strings  ${response.status_code}  200

Close And Reopen Index
    [Tags]  compatibility  index
    ${response}=  POST On Session  opensearch  /${BWC_IDX}/_close  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}
    ${response}=  POST On Session  opensearch  /${BWC_IDX}/_open  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    Sleep  ${SLEEP_TIME}
    Check OpenSearch Index Exists  ${BWC_IDX}

Create And Verify Index Template
    [Tags]  compatibility  index  template
    ${body}=  Set Variable  {"index_patterns":["bwc-tmpl-*"],"template":{"settings":{"number_of_shards":1,"number_of_replicas":0}}}
    ${response}=  PUT On Session  opensearch  /_index_template/${BWC_TMPL}  data=${body}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index Template  ${BWC_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200

Delete Index Template
    [Tags]  compatibility  index  template
    ${response}=  Delete OpenSearch Index Template  ${BWC_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200

Create And Verify Legacy Index Template
    [Tags]  compatibility  index  template
    ${body}=  Set Variable  {"index_patterns":["bwc-legacy-*"],"settings":{"number_of_shards":1,"number_of_replicas":0}}
    ${response}=  PUT On Session  opensearch  /_template/${BWC_LEGACY_TMPL}  data=${body}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Template  ${BWC_LEGACY_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200

Delete Legacy Index Template
    [Tags]  compatibility  index  template
    ${response}=  Delete OpenSearch Template  ${BWC_LEGACY_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200

Create Component Template And Composed Index Template
    [Tags]  compatibility  index  template
    ${comp_body}=  Set Variable  {"template":{"settings":{"number_of_shards":1}}}
    ${response}=  PUT On Session  opensearch  /_component_template/${BWC_COMP_TMPL}  data=${comp_body}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Component Template  ${BWC_COMP_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200
    ${composed_body}=  Set Variable  {"index_patterns":["bwc-composed-*"],"composed_of":["${BWC_COMP_TMPL}"]}
    ${response}=  PUT On Session  opensearch  /_index_template/${BWC_COMPOSED_TMPL}  data=${composed_body}  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Get OpenSearch Index Template  ${BWC_COMPOSED_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200

Delete Composed And Component Templates
    [Tags]  compatibility  index  template
    ${response}=  Delete OpenSearch Index Template  ${BWC_COMPOSED_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200
    ${response}=  Delete OpenSearch Component Template  ${BWC_COMP_TMPL}
    Should Be Equal As Strings  ${response.status_code}  200

Set Index Read Only And Back
    [Tags]  compatibility  index
    ${response}=  Make Index Read Only  ${BWC_IDX}
    Should Be Equal As Strings  ${response.status_code}  200
    ${settings}=  Get Index Settings  ${BWC_IDX}
    Should Be Equal As Strings  ${settings['${BWC_IDX}']['settings']['index']['blocks']['write']}  true
    ${response}=  Make Index Read Write  ${BWC_IDX}
    Should Be Equal As Strings  ${response.status_code}  200

Create Index With Special Characters In Name
    [Tags]  compatibility  index
    ${response}=  Create OpenSearch Index  ${BWC_SPECIAL_IDX}
    Should Be Equal As Strings  ${response.status_code}  200
    Check OpenSearch Index Exists  ${BWC_SPECIAL_IDX}
    ${response}=  Delete OpenSearch Index  ${BWC_SPECIAL_IDX}
    Should Be Equal As Strings  ${response.status_code}  200

Force Merge Index
    [Tags]  compatibility  index
    ${response}=  POST On Session  opensearch  url=/${BWC_IDX}/_forcemerge?max_num_segments=1  headers=${headers}
    Should Be Equal As Strings  ${response.status_code}  200

Delete Index And Verify
    [Tags]  compatibility  index
    ${response}=  Delete OpenSearch Index  ${BWC_IDX}
    Should Be Equal As Strings  ${response.status_code}  200
    Check OpenSearch Index Does Not Exist  ${BWC_IDX}
