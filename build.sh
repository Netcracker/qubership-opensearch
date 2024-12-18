#!/bin/sh

DOCKER_FILE=Dockerfile
TARGET_DIR=target
HELM_ARTIFACT_NAME=opensearch-service-operator-helm-artifacts

mkdir -p ${TARGET_DIR}

echo "Build docker image"
for docker_image_name in ${DOCKER_NAMES}; do
  echo "Docker image name: $docker_image_name"
  docker build \
    --file=${DOCKER_FILE} \
    --pull \
    -t ${docker_image_name} \
    .
done

mkdir -p deployments/charts/opensearch-service
cp -R ./charts/helm/opensearch-service/* deployments/charts/opensearch-service
cp ./charts/deployment-configuration.json deployments/deployment-configuration.json

#wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/bin/yq && chmod +x /usr/bin/yq

yq eval-all 'select(filename == "values.yaml") * select(filename == "values.override.yaml")' values.yaml values.override.yaml -i

rm deployments/charts/opensearch-service/values.override.yaml

echo "Archive artifacts"
zip -r ${TARGET_DIR}/${HELM_ARTIFACT_NAME}.zip charts/helm/opensearch-service

SCRIPTS=scripts
DIST_FILE="${SCRIPTS}/migration-artifacts.zip"
DIST_CONTENTS="migration-artifacts"

rm -rf ./${SCRIPTS}
mkdir ${SCRIPTS}

zip -qr "$DIST_FILE" $DIST_CONTENTS