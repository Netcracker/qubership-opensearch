FROM alpine:3.24@sha256:3f4d6b32e26703c6082839234f43769d5046b3ac7c910033911d1e71d8cd630a

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
