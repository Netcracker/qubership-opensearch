FROM artifactorycn.netcracker.com:17064/alpine:3.19.1

USER 0

ENV OPENSEARCH_TLS_INIT_HOME=/opt/opensearch-tls-init \
    OPENSEARCH_CONFIGS=/opt/opensearch-tls-init/config

COPY docker/alpine-repositories /etc/apk/repositories
COPY docker/generate-certificates.sh ${OPENSEARCH_TLS_INIT_HOME}/bin/generate-certificates.sh

# Install the tools we need: openssl to create certificates and corresponding openssl-libs.
RUN set -x \
    && apk add --update --no-cache build-base apk-tools bash openssl curl jq \
    && rm -rf /var/cache/apk/* \
    && mkdir ${OPENSEARCH_CONFIGS} \
    && chmod 777 ${OPENSEARCH_TLS_INIT_HOME} \
    && for path in \
            /opt/opensearch-tls-init/bin \
            /opt/opensearch-tls-init/config \
       ; do \
         chmod -R 777 "$path"; \
       done

WORKDIR ${OPENSEARCH_TLS_INIT_HOME}

USER 1000:0