#!/bin/bash

set -e

CONFIG_TEMPLATE="/etc/telegraf/telegraf.conf"
CONFIG_FILE="/tmp/monitoring/telegraf.conf"

resolve_secret_value() {
  local env_name="$1"
  local path="${OPENSEARCH_MONITORING_SECRETS_DIR:-/etc/secrets/monitoring-pod-secrets}/${env_name}"
  if [[ -r "${path}" ]]; then
    tr -d '\r' < "${path}"
    return 0
  fi
  printf '%s' "${!env_name:-}"
}

escape_toml_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/}"
  printf '%s' "${value}"
}

process_config_file() {
  local file="$1"
  local tmp next
  tmp="$(mktemp "${file}.XXXXXX")"

  local placeholders=(
    "__ELASTICSEARCH_USERNAME__:ELASTICSEARCH_USERNAME"
    "__ELASTICSEARCH_PASSWORD__:ELASTICSEARCH_PASSWORD"
  )

  cp "${file}" "${tmp}"

  local entry placeholder secret_name value escaped
  for entry in "${placeholders[@]}"; do
    placeholder="${entry%%:*}"
    secret_name="${entry##*:}"
    value="$(resolve_secret_value "${secret_name}")"
    value="${value//$'\n'/}"
    value="${value%"${value##*[![:space:]]}"}"
    value="${value#"${value%%[![:space:]]*}"}"

    next="${tmp}.new"
    if [[ -z "${value}" ]]; then
      grep -Fv "${placeholder}" "${tmp}" > "${next}" || true
    else
      escaped="$(escape_toml_string "${value}")"
      while IFS= read -r line || [[ -n "${line}" ]]; do
        line="${line//${placeholder}/${escaped}}"
        printf '%s\n' "${line}"
      done < "${tmp}" > "${next}"
    fi
    mv "${next}" "${tmp}"
  done

  cp "${tmp}" "${file}"
  rm -f "${tmp}"
}

if [[ ${ELASTICSEARCH_PROTOCOL} == "https" ]]; then
  ROOT_CA_CERTIFICATE=/trusted-certs/root-ca.pem
  if [[ -f ${ROOT_CA_CERTIFICATE} ]]; then
    echo "TLS Certificate loaded from path '${ROOT_CA_CERTIFICATE}'"
    export INSECURE_SKIP_VERIFY=false
    export ROOT_CA_CERTIFICATE
  else
    echo "Warning: Cannot load valid trusted TLS certificates from path '${ROOT_CA_CERTIFICATE}'. insecure_skip_verify mode is used. Do not use this mode in production."
    export INSECURE_SKIP_VERIFY=true
    export ROOT_CA_CERTIFICATE=""
  fi
else
    export ROOT_CA_CERTIFICATE=""
    export INSECURE_SKIP_VERIFY=true
fi

mkdir -p "${MONITORING_LOGS:-/tmp/monitoring/logs}"

if [[ ! -f "${CONFIG_TEMPLATE}" ]]; then
  echo "Telegraf config template is missing: ${CONFIG_TEMPLATE}" >&2
  exit 1
fi

mkdir -p "$(dirname "${CONFIG_FILE}")"
cp "${CONFIG_TEMPLATE}" "${CONFIG_FILE}"
process_config_file "${CONFIG_FILE}"

if grep -qE '__ELASTICSEARCH_USERNAME__|__ELASTICSEARCH_PASSWORD__' "${CONFIG_FILE}"; then
  echo "Telegraf config is missing required elasticsearch credentials after secret substitution" >&2
  exit 1
fi

exec /sbin/tini -- /entrypoint.sh telegraf --config "${CONFIG_FILE}"
