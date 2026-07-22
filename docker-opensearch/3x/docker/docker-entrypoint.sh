#!/bin/bash

set -e

resolve_secret_value() {
  local env_name="$1"
  local path="${OPENSEARCH_SECRETS_DIR:-/etc/secrets/opensearch-pod-secrets}/${env_name}"
  if [[ -r "${path}" ]]; then
    tr -d '\r' < "${path}"
    return 0
  fi
  printf '%s' "${!env_name:-}"
}

OPENSEARCH_USERNAME="$(resolve_secret_value OPENSEARCH_USERNAME)"
OPENSEARCH_PASSWORD="$(resolve_secret_value OPENSEARCH_PASSWORD)"
export OPENSEARCH_USERNAME OPENSEARCH_PASSWORD

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
"${JAVA_HOME}"/bin/keytool --importkeystore -noprompt \
        -srckeystore "$KEYSTORE_PATH" \
        -srcstorepass changeit \
        -destkeystore "$DESTINATION_KEYSTORE_PATH" \
        -deststorepass changeit &> /dev/null

if [[ "$(ls $PUBLIC_CERTS_DIR)" ]]; then
    for filename in "$PUBLIC_CERTS_DIR"/*; do
        echo "Import $filename certificate to Java cacerts"
        keytool -import -trustcacerts -keystore $DESTINATION_KEYSTORE_PATH -storepass changeit -noprompt -alias "$filename" -file "$filename"
    done;
fi

OS_TRUST_ANCHORS_DIR=/etc/pki/ca-trust/source/anchors

if [[ "$(ls $S3_CERTS_DIR)" ]]; then
    for filename in "$S3_CERTS_DIR"/*; do
        echo "Import $filename certificate to Java cacerts"
        keytool -import -trustcacerts -keystore "$DESTINATION_KEYSTORE_PATH" -storepass changeit -noprompt -alias "$filename" -file "$filename"
        keytool -import -trustcacerts -keystore "$KEYSTORE_PATH" -storepass changeit -noprompt -alias "$filename" -file "$filename"
    done;

    # The async client of repository-s3 plugin (AWS CRT) is a native library which does not
    # use the JVM truststore and trusts only the OS trust store, so S3 certificates must be
    # imported there as well, otherwise repository verification and snapshot deletion fail
    if [[ -w "$OS_TRUST_ANCHORS_DIR" ]]; then
        for filename in "$S3_CERTS_DIR"/*; do
            echo "Import $filename certificate to OS trust store"
            cp "$filename" "$OS_TRUST_ANCHORS_DIR/s3-$(basename "$filename")"
        done;
        update-ca-trust extract || echo "WARNING: failed to update OS trust store, S3 connections with custom certificates may fail"
    else
        echo "WARNING: $OS_TRUST_ANCHORS_DIR is not writable, S3 certificates are not imported to OS trust store"
    fi
fi

exec "$@"
