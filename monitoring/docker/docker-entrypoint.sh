#!/bin/bash

set -e

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

if [[ -n "$ELASTICSEARCH_CREDENTIALS" ]]; then
  echo "Credentials are taken from ELASTICSEARCH_CREDENTIALS environment variable"
  IFS=: read -r username password <<< "$ELASTICSEARCH_CREDENTIALS"
  export ELASTICSEARCH_USERNAME=${username}
  export ELASTICSEARCH_PASSWORD=${password}
fi

/sbin/tini -- /entrypoint.sh telegraf