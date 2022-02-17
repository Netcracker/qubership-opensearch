#!/bin/sh
set -e
set -x

IMAGE_NAME="${DOCKER_IMAGE_NAME}:${CI_PIPELINE_ID}"

docker build \
  --pull \
  --file=${DOCKER_FILE} \
  -t ${IMAGE_NAME} \
  --no-cache \
  .

docker inspect ${IMAGE_NAME}
