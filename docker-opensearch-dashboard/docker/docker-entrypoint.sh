#!/bin/bash

set -e

# 1. шаблон из ConfigMap (read-only mount)
INJECTED_CONFIG="/etc/opensearch-dashboards/config-injected/opensearch_dashboards.yml"
# 2. рабочая копия для подстановки секретов
WORKDIR="/tmp/dashboards/opensearch_dashboards.yml"
# 3. итоговый конфиг, откуда читает OpenSearch Dashboards
FINAL_CONFIG="/usr/share/opensearch-dashboards/config/opensearch_dashboards.yml"

process_config_file() {
  local file="$1"
  node - "${file}" <<'NODE'
const fs = require('fs');

const file = process.argv[1];
const secretsDir = process.env.DASHBOARDS_SECRETS_DIR || '/etc/secrets/opensearch-dashboards-pod-secrets';

function readSecret(name) {
  try {
    const path = require('path').join(secretsDir, name);
    if (fs.existsSync(path)) {
      return fs.readFileSync(path, 'utf8').replace(/\r/g, '').trim();
    }
  } catch (_) {}
  return process.env[name] || '';
}

function quoteYaml(value) {
  if (/^[a-zA-Z0-9._-]+$/.test(value)) {
    return value;
  }
  return '"' + value.replace(/\\/g, '\\\\').replace(/"/g, '\\"') + '"';
}

const placeholders = {
  '__OPENSEARCH_USERNAME__': readSecret('OPENSEARCH_USERNAME'),
  '__OPENSEARCH_PASSWORD__': readSecret('OPENSEARCH_PASSWORD'),
  '__COOKIE_PASS__': readSecret('COOKIE_PASS'),
  '__KEY_PASSPHRASE__': readSecret('KEY_PASSPHRASE'),
};

let content = fs.readFileSync(file, 'utf8');
for (const [placeholder, value] of Object.entries(placeholders)) {
  if (!value) {
    content = content.split('\n').filter((line) => !line.includes(placeholder)).join('\n');
  } else {
    content = content.split(placeholder).join(quoteYaml(value));
  }
}
fs.writeFileSync(file, content);
NODE
}

if [[ ! -f "${INJECTED_CONFIG}" ]]; then
  echo "OpenSearch Dashboards config template is missing: ${INJECTED_CONFIG}" >&2
  exit 1
fi

mkdir -p "$(dirname "${WORKDIR}")"
cp "${INJECTED_CONFIG}" "${WORKDIR}"
process_config_file "${WORKDIR}"
cp "${WORKDIR}" "${FINAL_CONFIG}"

if grep -qE '__OPENSEARCH_USERNAME__|__OPENSEARCH_PASSWORD__' "${FINAL_CONFIG}"; then
  echo "OpenSearch Dashboards config is missing required opensearch credentials after secret substitution" >&2
  exit 1
fi

exec /usr/share/opensearch-dashboards/opensearch-dashboards-docker-entrypoint.sh "$@"
