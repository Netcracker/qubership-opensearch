*** Variables ***
${MONITORED_IMAGES}         %{MONITORED_IMAGES}

*** Settings ***
Resource  ../shared/keywords.robot

*** Keywords ***
Get Image Tag
    [Arguments]  ${image}
    ${parts}=  Split String  ${image}  :
    ${length}=  Get Length  ${parts}
    Run Keyword If  ${length} > 1  Return From Keyword  ${parts[2]}  
    Run Keywords
    ...  Log To Console  \n[ERROR] Image ${parts} has no tag: ${image}\nMonitored images list: ${MONITORED_IMAGES}
    ...  AND  Fail  Some images were not found, please check your .helpers template and description.yaml in the repository

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
    
    Run Keyword And Continue On Failure  Should Be Equal   ${actual_tag}   ${expected_tag}
  END
  
