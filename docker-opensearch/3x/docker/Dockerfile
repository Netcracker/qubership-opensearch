ARG OPENSEARCH_BASE='opensearchproject/opensearch:3.7.0'

FROM $OPENSEARCH_BASE

USER 0

ENV DISABLE_INSTALL_DEMO_CONFIG=true \
    OPENSEARCH_HOME=/usr/share/opensearch \
    OPENSEARCH_CONFIGS=/usr/share/opensearch/config

COPY 3x/docker/health.sh ${OPENSEARCH_HOME}/bin
COPY 3x/docker/reconfiguration.sh ${OPENSEARCH_HOME}/bin
COPY 3x/docker/relocate-shards.sh ${OPENSEARCH_HOME}/bin
COPY 3x/docker/docker-entrypoint.sh /

# Install the tools we need
RUN yum update --assumeyes --skip-broken \
    && yum install -y jq-1.7.1 \
    && yum clean all \
    && rm -rf /var/cache/yum/*

RUN echo "Install OpenSearch plugins..." \
    && ${OPENSEARCH_HOME}/bin/opensearch-plugin install --batch --verbose repository-s3 repository-gcs

RUN ${OPENSEARCH_HOME}/bin/opensearch-plugin install --batch --verbose analysis-icu analysis-kuromoji
# Adapt grants
# /etc/pki/ca-trust must be writable so that docker-entrypoint.sh can add custom S3 CA
# certificates to the OS trust store used by the native AWS CRT client of repository-s3
RUN set -x \
    && chmod 777 ${OPENSEARCH_HOME} \
    && chmod +x ${OPENSEARCH_HOME}/bin/*.sh \
    && chmod +x ${OPENSEARCH_HOME}/plugins/opensearch-security/tools/*.sh \
    && chmod +x /docker-entrypoint.sh \
    && for path in \
        /usr/share/opensearch/logs \
        /usr/share/opensearch/config \
        /usr/share/opensearch/bin \
        /usr/share/opensearch/data \
        /usr/share/opensearch/jdk \
        /usr/share/opensearch/plugins \
        /etc/pki/ca-trust \
        ; do \
            chmod -R 777 "$path"; \
        done

USER 1000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./opensearch-docker-entrypoint.sh"]
