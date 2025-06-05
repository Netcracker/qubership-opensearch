#!/bin/sh

if [[ ${TLS_HTTP_ENABLED} == "true" ]]; then
  ROOT_CA_CERTIFICATE=/trusted-certs/root-ca.pem
  if [[ -f ${ROOT_CA_CERTIFICATE} ]]; then
    echo "TLS Certificate loaded from path '${ROOT_CA_CERTIFICATE}'"
    export ROOT_CA_CERTIFICATE
  else
    echo "Warning: Cannot load valid trusted TLS certificates from path '${ROOT_CA_CERTIFICATE}'."
    export ROOT_CA_CERTIFICATE=""
  fi
else
    export ROOT_CA_CERTIFICATE=""
fi

case $1 in
  elasticsearch-indices-cleaner)
    echo "Elasticsearch Indices Cleaner has started!"
    python3 indices_cleaner.py
esac