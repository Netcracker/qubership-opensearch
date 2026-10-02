FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

USER 0

ENV OPENSEARCH_TLS_INIT_HOME=/opt/opensearch-tls-init \
    OPENSEARCH_CONFIGS=/opt/opensearch-tls-init/config

COPY docker/generate-certificates.sh ${OPENSEARCH_TLS_INIT_HOME}/bin/generate-certificates.sh

# Install the tools we need: openssl to create certificates and corresponding openssl-libs.
RUN set -x \
    && apk add --update --no-cache build-base apk-tools coreutils bash openssl curl jq \
    && rm -rf /var/cache/apk/* \
    && mkdir ${OPENSEARCH_CONFIGS} \
    && chmod 777 ${OPENSEARCH_TLS_INIT_HOME} \
    && for path in \
            /opt/opensearch-tls-init/bin \
            /opt/opensearch-tls-init/config \
       ; do \
         chmod -R 777 "$path"; \
       done

RUN apk add --update --upgrade --no-cache \
    libcurl curl

WORKDIR ${OPENSEARCH_TLS_INIT_HOME}

USER 1000:0
