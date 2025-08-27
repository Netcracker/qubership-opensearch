*** Variables ***
${MONITORED_IMAGES}         %{MONITORED_IMAGES}

*** Settings ***
Resource  ../shared/keywords.robot

*** Test Cases ***
Test Hardcoded Images
  [Tags]  opensearch  opensearch_images
  ${stripped_resources}=  Strip String  ${MONITORED_IMAGES}  characters=,  mode=right
  @{list_resources}=  Split String  ${stripped_resources}  ,
  FOR  ${resource}  IN  @{list_resources}
    ${type}  ${name}  ${container_name}  ${image}=  Split String  ${resource}
    ${resource_image}=  Get Resource Image  ${type}  ${name}  %{OPENSEARCH_NAMESPACE}  ${container_name}

    ${expected_tag}=  Get Image Tag  ${image}
    ${actual_tag}=    Get Image Tag  ${resource_image}

    Log To Console  \n[COMPARE] ${resource}: Expected tag = ${expected_tag}, Actual tag = ${actual_tag}
    
    Should Be Equal   ${actual_tag}   ${expected_tag}
  END
  