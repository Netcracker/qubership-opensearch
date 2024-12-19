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

uname -a
cat /etc/os-release

# Ensure pip is linked to Python 3.8.6
python3.8 -m pip install --upgrade pip
python3.8 -m pip install pyyaml

python3.8 ./charts/helm/opensearch-service/merge.py
rm ./charts/helm/opensearch-service/values.override.yaml

mkdir -p deployments/charts/opensearch-service
cp -R ./charts/helm/opensearch-service/* deployments/charts/opensearch-service
cp ./charts/deployment-configuration.json deployments/deployment-configuration.json




echo "Archive artifacts"
zip -r ${TARGET_DIR}/${HELM_ARTIFACT_NAME}.zip charts/helm/opensearch-service

SCRIPTS=scripts
DIST_FILE="${SCRIPTS}/migration-artifacts.zip"
DIST_CONTENTS="migration-artifacts"

rm -rf ./${SCRIPTS}
mkdir ${SCRIPTS}

zip -qr "$DIST_FILE" $DIST_CONTENTS