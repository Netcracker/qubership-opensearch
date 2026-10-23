FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

USER 0

ENV OPENSEARCH_TLS_INIT_HOME=/opt/opensearch-tls-init \
    OPENSEARCH_CONFIGS=/opt/opensearch-tls-init/config

COPY docker/generate-certificates.sh ${OPENSEARCH_TLS_INIT_HOME}/bin/generate-certificates.sh

# Install the tools we need: openssl to create certificates and corresponding openssl-libs.
RUN set -x \
    && apk add --update --no-cache build-base binutils apk-tools coreutils bash openssl curl jq \
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
