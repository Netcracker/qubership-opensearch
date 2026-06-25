#!/bin/bash

set -e

# 1. шаблон из ConfigMap (read-only mount)
INJECTED_CONFIG="/etc/opensearch-dashboards/config-injected/opensearch_dashboards.yml"
# Рабочий конфиг в /tmp/dashboards; symlink в образе указывает на него же из
# /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml
CONFIG_FILE="/tmp/dashboards/opensearch_dashboards.yml"

resolve_secret_value() {
  local env_name="$1"
  local path="${DASHBOARDS_SECRETS_DIR:-/etc/secrets/opensearch-dashboards-pod-secrets}/${env_name}"
  if [[ -r "${path}" ]]; then
    tr -d '\r' < "${path}"
    return 0
  fi
  printf '%s' "${!env_name:-}"
}

quote_yaml() {
  local value="$1"
  if [[ "${value}" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    printf '%s' "${value}"
  else
    value="${value//\\/\\\\}"
    value="${value//\"/\\\"}"
    printf '"%s"' "${value}"
  fi
}

process_config_file() {
  local file="$1"
  local tmp
  tmp="$(mktemp "${file}.XXXXXX")"

  local placeholders=(
    "__OPENSEARCH_USERNAME__:OPENSEARCH_USERNAME"
    "__OPENSEARCH_PASSWORD__:OPENSEARCH_PASSWORD"
    "__COOKIE_PASS__:COOKIE_PASS"
    "__KEY_PASSPHRASE__:KEY_PASSPHRASE"
  )

  cp "${file}" "${tmp}"

  local entry placeholder secret_name value quoted
  for entry in "${placeholders[@]}"; do
    placeholder="${entry%%:*}"
    secret_name="${entry##*:}"
    value="$(resolve_secret_value "${secret_name}")"
    value="${value//$'\n'/}"
    value="${value%"${value##*[![:space:]]}"}"
    value="${value#"${value%%[![:space:]]*}"}"

    if [[ -z "${value}" ]]; then
      grep -Fv "${placeholder}" "${tmp}" > "${file}" || true
    else
      quoted="$(quote_yaml "${value}")"
      while IFS= read -r line || [[ -n "${line}" ]]; do
        line="${line//${placeholder}/${quoted}}"
        printf '%s\n' "${line}"
      done < "${tmp}" > "${file}"
    fi
    cp "${file}" "${tmp}"
  done

  rm -f "${tmp}"
}

if [[ ! -f "${INJECTED_CONFIG}" ]]; then
  echo "OpenSearch Dashboards config template is missing: ${INJECTED_CONFIG}" >&2
  exit 1
fi

mkdir -p "$(dirname "${CONFIG_FILE}")"
cp "${INJECTED_CONFIG}" "${CONFIG_FILE}"
process_config_file "${CONFIG_FILE}"

if grep -qE '__OPENSEARCH_USERNAME__|__OPENSEARCH_PASSWORD__' "${CONFIG_FILE}"; then
  echo "OpenSearch Dashboards config is missing required opensearch credentials after secret substitution" >&2
  exit 1
fi

exec /usr/share/opensearch-dashboards/opensearch-dashboards-docker-entrypoint.sh "$@"
