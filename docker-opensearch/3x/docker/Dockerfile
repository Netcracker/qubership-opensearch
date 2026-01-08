ARG OPENSEARCH_BASE='opensearchproject/opensearch:3.2.0'

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

RUN echo "Download OpenSearch plugins..." \
&& curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-s3/3.2.0/repository-s3-3.2.0.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-3.2.0.zip \
&& curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-gcs/3.2.0/repository-gcs-3.2.0.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-3.2.0.zip \
&& if [ -f ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-3.2.0.zip ] && [ -f ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-3.2.0.zip ]; then \
    echo "Plugins downloaded successfully."; \
else \
    echo "Plugin download failed"; \
    exit 1; \
fi

# Adapt grants
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
        /usr/share/opensearch/dist \
        ; do \
            chmod -R 777 "$path"; \
        done

USER 1000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./opensearch-docker-entrypoint.sh"]
