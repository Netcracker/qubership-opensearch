#!/bin/sh

set -e

if [ "${TLS_HTTP_ENABLED}" = "true" ]; then
  ROOT_CA_CERTIFICATE=/trusted-certs/root-ca.pem
  if [ -f "${ROOT_CA_CERTIFICATE}" ]; then
    echo "TLS Certificate loaded from path '${ROOT_CA_CERTIFICATE}'"
    export ROOT_CA_CERTIFICATE="${ROOT_CA_CERTIFICATE}"
  else
    echo "Warning: Cannot load valid trusted TLS certificates from path '${ROOT_CA_CERTIFICATE}'. SSL_NO_VALIDATE mode is used. Do not use this mode in production."
    export ROOT_CA_CERTIFICATE=""
  fi
else
  export ROOT_CA_CERTIFICATE=""
fi

exec /opt/backup/backup-daemon \
  --custom-vars backup_info:nothing \
  --backup-cmd "/opt/elasticsearch-curator/backup.py {{.data_folder}} {{.dbs}}" \
  --restore-cmd "/opt/elasticsearch-curator/restore.py {{.data_folder}} {{.skip_users_recovery}} {{.dbs}} {{.dbmap}} {{.clean}}" \
  --dblist-cmd "/opt/elasticsearch-curator/list_instances_in_vault.py {{.data_folder}}" \
  --evict-cmd "/opt/elasticsearch-curator/evict.py {{.data_folder}}" \
  --tls-enabled "${INTERNAL_TLS_ENABLED}" \
  --certs-path "${INTERNAL_TLS_PATH}" \
  "$@"