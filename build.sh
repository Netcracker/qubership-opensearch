#!/bin/sh

source ./build.env

TARGET_DIR=target
HELM_ARTIFACT_NAME=opensearch-service-operator-helm-artifacts

mkdir -p ${TARGET_DIR}

docker pull ${OPENSEARCH_TRANSFER_IMAGE}

echo "Build docker image"
for docker_image_name in ${DOCKER_NAMES}; do
  echo "Docker image name: $docker_image_name"
  docker tag ${OPENSEARCH_TRANSFER_IMAGE} ${docker_image_name}
done

docker run --name opensearch-transfer ${OPENSEARCH_TRANSFER_IMAGE} /bin/true || true

# Enrich with github chart
mkdir -p temporary_directory
docker cp opensearch-transfer:charts temporary_directory
cp -rn temporary_directory/* ./
rm -rf temporary_directory
docker stop opensearch-transfer
docker rm opensearch-transfer

# Ensure pip is linked to Python 3.8.6
python3.8 -m pip install --upgrade pip
python3.8 -m pip install pyyaml

python3.8 ./merge.py "./charts/helm/opensearch-service/values.yaml" "./charts/helm/opensearch-service/values.override.yaml"
rm ./charts/helm/opensearch-service/values.override.yaml

mkdir -p deployments/charts/opensearch-service
cp -R ./charts/helm/opensearch-service/* deployments/charts/opensearch-service
cp ./charts/deployment-configuration.json deployments/deployment-configuration.json

echo "Archive artifacts"
zip -r ${TARGET_DIR}/${HELM_ARTIFACT_NAME}.zip charts/helm/opensearch-service