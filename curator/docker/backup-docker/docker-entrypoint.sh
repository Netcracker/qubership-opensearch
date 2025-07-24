#!/bin/sh

set -e

if [[ ${TLS_HTTP_ENABLED} == "true" ]]; then
  ROOT_CA_CERTIFICATE=/trusted-certs/root-ca.pem
  if [[ -f ${ROOT_CA_CERTIFICATE} ]]; then
    echo "TLS Certificate loaded from path '${ROOT_CA_CERTIFICATE}'"
    export ROOT_CA_CERTIFICATE="${ROOT_CA_CERTIFICATE}"
  else
    echo "Warning: Cannot load valid trusted TLS certificates from path '${ROOT_CA_CERTIFICATE}'. SSL_NO_VALIDATE mode is used. Do not use this mode in production."
    export ROOT_CA_CERTIFICATE=""
  fi
else
  export ROOT_CA_CERTIFICATE=""
fi

python3 /opt/backup/backup-daemon.py