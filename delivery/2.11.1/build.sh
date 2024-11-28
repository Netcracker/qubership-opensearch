#!/bin/bash

source ../../build.env

mkdir -p ./helm

cp -R ../../charts/helm/opensearch-service helm/

docker pull ${OPENSEARCH_TRANSFER_IMAGE}

docker run --name opensearch-transfer ${OPENSEARCH_TRANSFER_IMAGE} /bin/true || true

mkdir -p temporary_directory

docker cp opensearch-transfer:charts/helm temporary_directory
cp -rn temporary_directory/* ./
rm -rf temporary_directory
docker stop opensearch-transfer
docker rm opensearch-transfer