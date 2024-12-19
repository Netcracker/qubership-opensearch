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

## Install EPEL repository if not already installed
#if ! yum repolist | grep -q epel; then
#    echo "Installing EPEL repository..."
#    yum install -y epel-release
#fi

# Install Python and pip
if ! command -v python3 &>/dev/null; then
    echo "Python3 not found. Installing..."
    yum install -y python3
fi

# Ensure pip is up-to-date
echo "Upgrading pip..."
pip3 install --upgrade pip

# Install PyYAML
echo "Installing PyYAML..."
pip3 install pyyaml

python ./charts/helm/opensearch-service/merge.py
rm deployments/charts/opensearch-service/values.override.yaml

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