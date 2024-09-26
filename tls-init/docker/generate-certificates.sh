#!/bin/bash

set -e

log() {
  echo ${1} >> log.txt
}

print_log() {
  cat log.txt
  sleep 1m
}

exit_with_log() {
  print_log
  exit $1
}

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
  local private_key_name=tls.key
  local root_ca_name=ca.crt
  local certificate_name=tls.crt
  root_ca=${OPENSEARCH_CONFIGS}/root-ca.pem
  private_key=${OPENSEARCH_CONFIGS}/${type}-${private_key_name}
  certificate=${OPENSEARCH_CONFIGS}/${type}-${certificate_name}

  echo "Generating '${type}' certificates"
  generate_certificates
  echo "'${type}' certificates are generated"
  if [[ $(secret_exists $secret_name) == false ]]; then
    # Creates secret
    local secret_type="kubernetes.io/tls"
    result=$(curl -sSk -X POST -H "Authorization: Bearer $token" \
      "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -d "{ \"kind\": \"Secret\", \"apiVersion\": \"v1\", \"metadata\": { \"name\": \"${secret_name}\", \"namespace\": \"${NAMESPACE}\" }, \"type\": \"${secret_type}\", \"data\": { \"${certificate_name}\": \"$(cat ${certificate} | base64 | tr -d '\n')\", \"${private_key_name}\": \"$(cat ${private_key} | base64 | tr -d '\n')\", \"${root_ca_name}\": \"$(cat ${root_ca} | base64 | tr -d '\n')\" } }")
  else
    # Updates secret
    result=$(curl -sSk -X PATCH -H "Authorization: Bearer $token" \
      "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${secret_name}" \
      -H "Content-Type: application/json-patch+json" \
      -H "Accept: application/json" \
      -d "[ { \"op\": \"replace\", \"path\": \"/data/${certificate_name}\", \"value\": \"$(cat ${certificate} | base64 | tr -d '\n')\" }, { \"op\": \"replace\", \"path\": \"/data/${private_key_name}\", \"value\": \"$(cat ${private_key} | base64 | tr -d '\n')\" }, { \"op\": \"replace\", \"path\": \"/data/${root_ca_name}\", \"value\": \"$(cat ${root_ca} | base64 | tr -d '\n')\" } ]")
  fi
  local code=$(echo "${result}" | jq -r ".code")
  local message=$(echo "${result}" | jq -r ".message")
  if [[ "$code" -ne "null" ]]; then
    echo "Certificates cannot be generated because of error with '$code' code and '$message' message"
    exit_with_log 1
  fi
}

# Check the secret has legacy sub paths, where each path has "type" prefix.
#
# $1 - the type of generated certificates
# $2 - the name of the secret for generated certificates
certs_path_are_legacy() {
  log "check legacy paths in $2"

  local type=$1
  local secret=$2
  local secret_file="secret.json"
  curl -sSk -X GET -H "Authorization: Bearer $token" "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${secret}" > ${secret_file}

  if [[ $(jq -r '.type?' ${secret_file}) == "Opaque" ]]; then
    log "Legacy type is used, need to migrate paths"
    echo true
    return
  fi

  local old_key_subpath
  if [[ $(jq --arg subPath "$(echo "$type-key.pem")" -r '.data."$subPath"?' ${secret_file}) != "null" ]]; then
    old_key_subpath=true
  else
    old_key_subpath=false
  fi

  local old_crt_subpath
  if [[ $(jq --arg subPath "$(echo "$type-crt.pem")" -r '.data."$subPath"?' ${secret_file}) != "null" ]]; then
    old_crt_subpath=true
  else
     old_crt_subpath=false
  fi

  local old_root_ca_subpath
  if [[ $(jq --arg subPath "$(echo "$type-root-ca.pem")" -r '.data."$subPath"?' ${secret_file}) != "null" ]]; then
    old_root_ca_subpath=true
  else
    old_root_ca_subpath=false
  fi

  if [[ "$old_key_subpath" == "true" && "$old_crt_subpath" == "true" && "$old_root_ca_subpath" == "true" ]]; then
    log "All paths are legacy, need to migrate them"
    echo true
    return
  fi

  if [[ "$old_key_subpath" == "false" && "$old_crt_subpath" == "false" && "$old_root_ca_subpath" == "false" ]]; then
    log "All paths are actual, no migration need"
    echo false
    return
  fi

  local log_message="Mixed subpath detected, Job can't process it. Please check the secret ${secret} & update it manually."
  log "${log_message}"
  echo 2>&1 "${log_message}"
  exit_with_log 1
}

# Migrates the certificates and key from legacy secret paths to the new
#
# $1 - the type of generated certificates
# $2 - the name of the secret for generated certificates
migrate_paths() {
  log "Running path migration..."

  local type=$1
  local secret_name=$2

  local secret_file="secret.json"

  local private_key_name=tls.key
  local root_ca_name=ca.crt
  local certificate_name=tls.crt

  local tmp_tls_key=tmp-tls.key
  local tmp_tls_crt=tmp-tls.crt
  local tmp_ca_crt=tmp-ca.crt

  local secret_type="kubernetes.io/tls"

  jq --arg type "$(echo "$type-key.pem")" '.data[$type]' ${secret_file} | tr -d '"'     > ${tmp_tls_key}
  jq --arg type "$(echo "$type-crt.pem")" '.data[$type]' ${secret_file} | tr -d '"'     > ${tmp_tls_crt}
  jq --arg type "$(echo "$type-root-ca.pem")" '.data[$type]' ${secret_file} | tr -d '"' > ${tmp_ca_crt}

  local secret_response
  local code
  local delay=3s

  # Create TEMP-secret to save certs to avoid certs lost at all. It can be used for manual recovering
  secret_response=$(curl -sSk -X POST -H "Authorization: Bearer $token" \
    "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d "{ \"kind\": \"Secret\", \"apiVersion\": \"v1\", \"metadata\": { \"name\": \"${secret_name}-tmp\", \"namespace\": \"${NAMESPACE}\" }, \"type\": \"${secret_type}\",\"data\": { \"${certificate_name}\": \"$(cat ${tmp_tls_crt})\", \"${private_key_name}\": \"$(cat ${tmp_tls_key})\", \"${root_ca_name}\": \"$(cat ${tmp_ca_crt})\" } }")
  sleep ${delay}

  # Removes real secret, because it might have type Opaque
  secret_response=$(curl -sSk -X DELETE -H "Authorization: Bearer $token" \
    "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${secret_name}")
  sleep ${delay}

  # Create secret again
  secret_response=$(curl -sSk -X POST -H "Authorization: Bearer $token" \
    "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d "{ \"kind\": \"Secret\", \"apiVersion\": \"v1\", \"metadata\": { \"name\": \"${secret_name}\", \"namespace\": \"${NAMESPACE}\" }, \"type\": \"${secret_type}\",\"data\": { \"${certificate_name}\": \"$(cat ${tmp_tls_crt})\", \"${private_key_name}\": \"$(cat ${tmp_tls_key})\", \"${root_ca_name}\": \"$(cat ${tmp_ca_crt})\" } }")
  sleep ${delay}

  # Removes TEMP-secret
  curl -sSk -X DELETE -H "Authorization: Bearer $token" \
    "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${secret_name}-tmp"
  sleep ${delay}

  # Cleanup temp files
  rm ${tmp_tls_key}
  rm ${tmp_tls_crt}
  rm ${tmp_ca_crt}

  log "Path migration done"
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
    exit_with_log 1
  fi
}

cert_expires() {
  local type=$1
  local secret=$2
  if [[ $(secret_exists ${secret}) == true ]]; then
    log "secret $2 exists"
    if [[ $(certs_path_are_legacy ${type} ${secret}) == true ]]; then
      migrate_paths ${type} ${secret}
    fi
    curl -sSk -X GET -H "Authorization: Bearer $token" "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/secrets/${secret}" | jq --arg type "tls.crt" '.data[$type]' | tr -d '"' | base64 --decode > crt.pem
    if [[ $(($(openssl x509 -enddate -noout -in crt.pem | awk '{print $4}') - $(date | awk '{print $6}'))) -lt 10  && "${RENEW_CERTS}" == "true" ]]; then
      log "cert with type $1 was expired"
      echo true
    else
      echo false
    fi
    rm crt.pem
  else
    log "secret $2 not exists"
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

log "BEGIN"
prepare
log "AFTER PREPARE"
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
  legacy_path="transport-root"
  statefulset_mounts=$(curl -sSk -X GET -H "Authorization: Bearer $token" "https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/statefulset/${MASTER_STATEFULSET_NAME}" | jq '.spec.template.spec.containers[0].volumeMounts')
  if ! [[ $statefulset_mounts =~ $legacy_path ]] ; then
        log "secrets type is kubernetes.io/tls. Deleting pods..."
        delete_pods
  fi
fi
log "END"

# Uncomment it to run sleep & log printing
# print_log
