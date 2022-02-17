#!/bin/sh
set -e
set -x

if [[ "$CI_BUILD_REF_NAME" = "master" ]]; then
  TAG_NAME="latest"
else
  TAG_NAME=$(echo "${CI_BUILD_REF_NAME}"| sed 's/\//_/g')
fi