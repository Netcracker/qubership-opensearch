#!/bin/bash

set -e

# Prepares necessary entities for certificates generation
prepare() {
  token=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
  subject_common="/OU=Opensearch/O=Opensearch/L=Opensearch/C=CA"

  # Generate extension file for certificates
  cat >"${OPENSEARCH_CONFIGS}/opensearch.ext" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
RID = 1.2.3.4.5.5
EOF
  # Add additional IP addresses to extension file
  local index=0
  for address in $ADDITIONAL_IP_ADDRESSES; do
    echo "IP.$index = $address" >>"${OPENSEARCH_CONFIGS}/opensearch.ext"
    index=$((index + 1))
  done
  # Add additional DNS names to extension file
  local index=0
  for name in $ADDITIONAL_DNS_NAMES; do
    echo "DNS.$index = $name" >>"${OPENSEARCH_CONFIGS}/opensearch.ext"
    index=$((index + 1))
  done
}

# Automatically generates certificates if it is necessary
generate_certificates() {
  local duration_days=14600
  if [[ ! -s ${root_ca} ]]; then
    # Generate CA's private key and self-signed certificate
    openssl req -x509 -newkey rsa:2048 -nodes -keyout "${OPENSEARCH_CONFIGS}/root-ca.key" -out "${root_ca}" -days ${duration_days} -subj "/CN=opensearch"
  fi
  if [[ ! -s ${private_key} || ! -s ${certificate} ]]; then
    # Generate web server's private key and certificate signing request (CSR)
    openssl req -newkey rsa:2048 -nodes -keyout "${private_key}" -out "${OPENSEARCH_CONFIGS}/req.csr" -subj ${subject}
    # Use CA's private key to sign web server's CSR and get back the signed certificate
    if [[ "$use_extension" == "true" ]]; then
      openssl x509 -req -in "${OPENSEARCH_CONFIGS}/req.csr" -CA "${root_ca}" -CAkey "${OPENSEARCH_CONFIGS}/root-ca.key" -CAcreateserial -out "${certificate}" -days ${duration_days} -sha256 -extfile "${OPENSEARCH_CONFIGS}/opensearch.ext"
    else
      openssl x509 -req -in "${OPENSEARCH_CONFIGS}/req.csr" -CA "${root_ca}" -CAkey "${OPENSEARCH_CONFIGS}/root-ca.key" -CAcreateserial -out "${certificate}" -days ${duration_days} -sha256
    fi
  fi
}

# Creates or updates secret with generated certificates
#
# $1 - the type of generating certificates
# $2 - the name of the secret for generated certificates
create_certificates() {
  local type=$1
  local secret_name=$2
  local private_key_name=${type}-key.pem
  local root_ca_name=${type}-root-ca.pem
  local certificate_name=${type}-crt.pem
  root_ca=${OPENSEARCH_CONFIGS}/root-ca.pem
  private_key=${OPENSEARCH_CONFIGS}/${private_key_name}
  certificate=${OPENSEARCH_CONFIGS}/${certificate_name}

  echo "Generating '${type}' certificates"
  generate_certificates
  echo "'${type}' certificates are generated"
  if [[ $(secret_exists $secret_name) == false ]]; then
    # Creates secret
    result=$(curl -sSk -X POST -H "Authorization: Bearer $token" \
      "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -d "{ \"kind\": \"Secret\", \"apiVersion\": \"v1\", \"metadata\": { \"name\": \"${secret_name}\", \"namespace\": \"${NAMESPACE}\" }, \"data\": { \"${certificate_name}\": \"$(cat ${certificate} | base64 | tr -d '\n')\", \"${private_key_name}\": \"$(cat ${private_key} | base64 | tr -d '\n')\", \"${root_ca_name}\": \"$(cat ${root_ca} | base64 | tr -d '\n')\" } }")
  else
    # Updates secret
    result=$(curl -sSk -X PUT -H "Authorization: Bearer $token" \
      "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${secret_name}" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -d "{ \"kind\": \"Secret\", \"apiVersion\": \"v1\", \"metadata\": { \"name\": \"${secret_name}\", \"namespace\": \"${NAMESPACE}\" }, \"data\": { \"${certificate_name}\": \"$(cat ${certificate} | base64 | tr -d '\n')\", \"${private_key_name}\": \"$(cat ${private_key} | base64 | tr -d '\n')\", \"${root_ca_name}\": \"$(cat ${root_ca} | base64 | tr -d '\n')\" } }")
  fi
  local code=$(echo "${result}" | jq -r ".code")
  local message=$(echo "${result}" | jq -r ".message")
  if [[ "$code" -ne "null" ]]; then
    echo "Certificates cannot be generated because of error with '$code' code and '$message' message"
    exit 1
  fi
}

# Checks secret with specified name exists
#
# $1 - the name of the secret
secret_exists() {
  local name=$1
  local secret_response=$(curl -sSk -X GET -H "Authorization: Bearer $token" \
    "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${name}")
  local code=$(echo "${secret_response}" | jq -r ".code")
  local message=$(echo "${secret_response}" | jq -r ".message")
  if [[ "$code" -eq "null" ]]; then
    echo true
  elif [[ "$code" -eq "404" ]]; then
    echo false
  else
    echo 2>&1 "Secret cannot be obtained because of error with '$code' code and '$message' message"
    exit 1
  fi
}

cert_expires() {
  local type=$1
  local secret=$2
  if [[ $(secret_exists ${secret}) == true ]]; then
    curl -sSk -X GET -H "Authorization: Bearer $token" "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${secret}" | jq --arg type "$(echo "$type-crt.pem")" '.data[$type]' | tr -d '"' | base64 --decode > crt.pem
    if [[ $(($(openssl x509 -enddate -noout -in crt.pem | awk '{print $4}') - $(date | awk '{print $6}'))) -lt 10  && "${RENEW_CERTS}" == "true" ]]; then
      echo true
    else
      echo false
    fi
    rm crt.pem
  else
    echo true
  fi
}

delete_pods() {
  local response=$(curl -sSk -X GET -H "Authorization: Bearer $token" "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/pods/")
  local pods=$(echo "${response}" | jq ".items[].metadata.name")
  pods=$(echo $pods | tr -d '"')
  local podsarray=( $pods )
  for pod in ${podsarray[@]}; do
    if [[ ($pod == ${OPENSEARCH_FULLNAME}* || $pod == dbaas-${OPENSEARCH_FULLNAME}*) && ! $pod =~ "tls-init" ]]; then
      curl -sSk -X DELETE -H "Authorization: Bearer $token" "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/pods/${pod}"
    fi
  done
}

prepare
if [[ $(cert_expires "transport" $TRANSPORT_CERTIFICATES_SECRET_NAME) == true || $(cert_expires "admin" $ADMIN_CERTIFICATES_SECRET_NAME) == true || -n "$REST_CERTIFICATES_SECRET_NAME" && $(cert_expires "rest" $REST_CERTIFICATES_SECRET_NAME) == true ]]; then
  # Creates secret with transport certificates
  use_extension="true"
  subject="/CN=opensearch-node${subject_common}"
  create_certificates "transport" "$TRANSPORT_CERTIFICATES_SECRET_NAME"

  # Creates secret with admin certificates
  use_extension="false"
  subject="/CN=opensearch-admin${subject_common}"
  create_certificates "admin" "$ADMIN_CERTIFICATES_SECRET_NAME"

  # Creates secret with REST certificates if secret name is specified
  if [[ -n "$REST_CERTIFICATES_SECRET_NAME" ]]; then
    use_extension="true"
    subject="/CN=opensearch${subject_common}"
    create_certificates "rest" "$REST_CERTIFICATES_SECRET_NAME"
  fi
  delete_pods
fi