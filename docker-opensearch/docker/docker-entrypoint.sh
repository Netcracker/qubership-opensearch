#!/bin/bash

set -e

${OPENSEARCH_HOME}/bin/opensearch-plugin install \
  --batch --verbose "file://${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-2.17.1.zip"
${OPENSEARCH_HOME}/bin/opensearch-plugin install \
  --batch --verbose "file://${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-2.17.1.zip"
${OPENSEARCH_HOME}/bin/opensearch-plugin install \
  --batch --verbose "file://${OPENSEARCH_HOME}/dist/opensearch-filter-plugin/opensearch-filter-plugin-2.17.1.0.zip"
rm -rf ${OPENSEARCH_HOME}/dist

if [[ -n "$OPENSEARCH_SECURITY_CONFIG_PATH" ]]; then
  # Set internal users
  password=$("${OPENSEARCH_HOME}/plugins/opensearch-security/tools/hash.sh" -p "${OPENSEARCH_PASSWORD}" | grep -v "\*\*")
  cat >"${OPENSEARCH_SECURITY_CONFIG_PATH}/internal_users.yml" <<EOF
_meta:
  type: "internalusers"
  config_version: 2

# Define your internal users here
${OPENSEARCH_USERNAME}:
  hash: "${password}"
  reserved: false
  backend_roles:
  - "admin"
  description: "Admin user"
EOF
fi

export OPENSEARCH_JAVA_OPTS="$OPENSEARCH_JAVA_OPTS -Dopensearch.allow_insecure_settings=true"

echo "Import trustcerts to application keystore"

PUBLIC_CERTS_DIR=/usr/share/opensearch/config/trustcerts
S3_CERTS_DIR=/usr/share/opensearch/config/s3certs
DESTINATION_KEYSTORE_PATH=/usr/share/opensearch/config/cacerts

KEYSTORE_PATH=${JAVA_HOME}/lib/security/cacerts

echo "Copy Java cacerts to $DESTINATION_KEYSTORE_PATH"
${JAVA_HOME}/bin/keytool --importkeystore -noprompt \
        -srckeystore $KEYSTORE_PATH \
        -srcstorepass changeit \
        -destkeystore $DESTINATION_KEYSTORE_PATH \
        -deststorepass changeit &> /dev/null

if [[ "$(ls $PUBLIC_CERTS_DIR)" ]]; then
    for filename in $PUBLIC_CERTS_DIR/*; do
        echo "Import $filename certificate to Java cacerts"
        keytool -import -trustcacerts -keystore $DESTINATION_KEYSTORE_PATH -storepass changeit -noprompt -alias $filename -file $filename
    done;
fi

if [[ "$(ls $S3_CERTS_DIR)" ]]; then
    for filename in $S3_CERTS_DIR/*; do
        echo "Import $filename certificate to Java cacerts"
        keytool -import -trustcacerts -keystore $DESTINATION_KEYSTORE_PATH -storepass changeit -noprompt -alias $filename -file $filename
        keytool -import -trustcacerts -keystore $KEYSTORE_PATH -storepass changeit -noprompt -alias $filename -file $filename
    done;
fi

exec "$@"