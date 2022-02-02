#!/bin/sh
source ./.gitlab-ci/resolve_tag_name.sh
set -e
set -x

IMAGE_NAME="${DOCKER_IMAGE_NAME}:${CI_PIPELINE_ID}"

FULL_IMAGE_NAME="${DOCKER_REPOSITORY}/${DOCKER_IMAGE_NAME}:${TAG_NAME}"

docker login --username="$DOCKER_REGISTRY_USERNAME" --password="$DOCKER_REGISTRY_PASSWORD" ${DOCKER_REPOSITORY}
docker tag ${IMAGE_NAME} ${FULL_IMAGE_NAME}
docker push ${FULL_IMAGE_NAME}
