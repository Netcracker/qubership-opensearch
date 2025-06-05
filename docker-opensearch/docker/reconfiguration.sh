#!/usr/bin/env bash

#Logs message to file.
#
#$1 - message.
log() {
  echo "$1" >>"${RECONFIGURATION_LOG_FILE}"
}

RECONFIGURATION_LOG_FILE=/usr/share/opensearch/logs/reconfiguration.log
truncate -s 0 ${RECONFIGURATION_LOG_FILE}
backup_command="${OPENSEARCH_HOME}/plugins/opensearch-security/tools/securityadmin.sh -backup ${OPENSEARCH_SECURITY_CONFIG_PATH}/backup -cert ${OPENSEARCH_CONFIG_PATH}/admin-crt.pem -cacert ${OPENSEARCH_CONFIG_PATH}/admin-root-ca.pem -key ${OPENSEARCH_CONFIG_PATH}/admin-key.pem"
if ! result=$($backup_command); then
  log "The backup of security failed. The command output is ${result}."
  exit 1
fi
log "The backup of security is successfully completed: \n${result}"
if grep -q "reserved: true" "${OPENSEARCH_SECURITY_CONFIG_PATH}/backup/internal_users.yml"; then
  sed -i 's/reserved: true/reserved: false/' "${OPENSEARCH_SECURITY_CONFIG_PATH}/backup/internal_users.yml"
  configure_users_command="${OPENSEARCH_HOME}/plugins/opensearch-security/tools/securityadmin.sh -f ${OPENSEARCH_SECURITY_CONFIG_PATH}/backup/internal_users.yml -t internalusers -cert ${OPENSEARCH_CONFIG_PATH}/admin-crt.pem -cacert ${OPENSEARCH_CONFIG_PATH}/admin-root-ca.pem -key ${OPENSEARCH_CONFIG_PATH}/admin-key.pem"
  if ! result=$($configure_users_command); then
    log "The dynamic reconfiguration of internal users failed. The command output is ${result}."
    exit 1
  fi
  log "The dynamic reconfiguration of internal users is successfully completed: \n${result}"
else
  log "${OPENSEARCH_SECURITY_CONFIG_PATH}/backup/internal_users.yml doesn't contain reserved users"
fi
