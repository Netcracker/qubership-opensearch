#!/usr/bin/env bash

#Logs message to file.
#
#$1 - message.
log() {
  echo "$1" >>"${HEALTH_LOG_FILE}"
}

#Returns 0 if POD has http port.
has_http_port() {
  node_roles=$(env | grep ^node\\.roles= | cut -d= -f2-)
  if [[ "$node_roles" == *"cluster_manager"* || "$node_roles" == *"ingest"* ]]; then
    log "Has http port: $node_roles"
    echo 0
  else
    log "Does not have http port: $node_roles"
    echo 1
  fi
}

if [ -f "/usr/share/opensearch/credentials/username" ]; then
    OPENSEARCH_USERNAME=$(cat /usr/share/opensearch/credentials/username)
fi

if [ -f "/usr/share/opensearch/credentials/password" ]; then
    OPENSEARCH_PASSWORD=$(cat /usr/share/opensearch/credentials/password)
fi
#Handles Kubernetes container readiness probe.
readiness_probe() {
  HEALTH_LOG_FILE=/usr/share/opensearch/logs/health_readiness_probe.log
  truncate -s 0 ${HEALTH_LOG_FILE}
  log "[readiness-probe] start"
  if [ "$(has_http_port)" -eq 0 ]; then
    command="curl -Is -u "${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD}" -XGET http://localhost:9200/_cat/health"
    if [[ ${TLS_ENABLED} == "true" ]]; then
      command="curl -Is -u "${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD}" -XGET https://localhost:9200/_cat/health --cacert ${OPENSEARCH_CONFIGS}/rest-crt.pem"
    fi
    http_status_code=$(${command} | head -1 | grep HTTP/1.1 | cut -d " " -f 2)
    log "http status code: [$http_status_code]"

    if [ -z "$http_status_code" ]; then
      log "Error: localhost is not available, because received http status code is unknown!"
      exit 1
    fi
    if [ "$http_status_code" -eq 200 ]; then
      if [[ "$(/usr/share/opensearch/bin/opensearch-keystore list )" != *gcs.client.default.credentials_file* && -f /usr/share/opensearch/gcs/key.json ]]; then
        log "adding gcs key to keystore."
        /usr/share/opensearch/bin/opensearch-keystore add-file gcs.client.default.credentials_file /usr/share/opensearch/gcs/key.json
        log "reloading secure settings."
        reload_command="curl -Is -u "${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD}" -XPOST http://localhost:9200/_nodes/reload_secure_settings"
        if [[ ${TLS_ENABLED} == "true" ]]; then
          reload_command="curl -Is -u "${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD}" -XPOST https://localhost:9200/_nodes/reload_secure_settings --cacert ${OPENSEARCH_CONFIGS}/rest-crt.pem"
        fi
        reload_command_http_status_code=$(${reload_command} | head -1 | grep HTTP/1.1 | cut -d " " -f 2)
        log "reload secure settings status code: [$reload_command_http_status_code]"
      else
        log "gcs key already added to keystore or there is no need to add gcs key."
      fi
      log "localhost is available."
    fi
  fi
  log "[readiness-probe] stop"
}

case $1 in
readiness-probe)
  readiness_probe
  exit $?
  ;;
esac
