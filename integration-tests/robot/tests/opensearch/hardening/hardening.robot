*** Variables ***
${HARDENING_EXCLUSIONS}          %{HARDENING_EXCLUSIONS={}}

*** Settings ***
Resource  ../shared/keywords.robot

*** Test Cases ***
Test Container Hardening
    [Tags]  hardening
    ${part_of}=  Create List  opensearch-service
    ${exclusions}=  Convert Json ${HARDENING_EXCLUSIONS} To Type
    Check Container Hardening    ${part_of}    ${OPENSEARCH_NAMESPACE}    ${exclusions}
