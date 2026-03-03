ARG OPENSEARCH_VERSION=3.4.0
ARG OPENSEARCH_BASE="opensearchproject/opensearch:${OPENSEARCH_VERSION}"

FROM $OPENSEARCH_BASE

ARG OPENSEARCH_VERSION

USER 0

ENV DISABLE_INSTALL_DEMO_CONFIG=true \
    OPENSEARCH_HOME=/usr/share/opensearch \
    OPENSEARCH_CONFIGS=/usr/share/opensearch/config \
    OPENSEARCH_VERSION=${OPENSEARCH_VERSION}

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
    && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-s3/${OPENSEARCH_VERSION}/repository-s3-${OPENSEARCH_VERSION}.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-${OPENSEARCH_VERSION}.zip \
    && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-gcs/${OPENSEARCH_VERSION}/repository-gcs-${OPENSEARCH_VERSION}.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-${OPENSEARCH_VERSION}.zip \
    && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/analysis-icu/${OPENSEARCH_VERSION}/analysis-icu-${OPENSEARCH_VERSION}.zip" \
    -o ${OPENSEARCH_HOME}/dist/analysis-icu/analysis-icu-${OPENSEARCH_VERSION}.zip \
    && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/analysis-kuromoji/${OPENSEARCH_VERSION}/analysis-kuromoji-${OPENSEARCH_VERSION}.zip" \
    -o ${OPENSEARCH_HOME}/dist/analysis-kuromoji/analysis-kuromoji-${OPENSEARCH_VERSION}.zip \
    && if [ -f ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-${OPENSEARCH_VERSION}.zip ] \
    && [ -f ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-${OPENSEARCH_VERSION}.zip ] \
    && [ -f ${OPENSEARCH_HOME}/dist/analysis-icu/analysis-icu-${OPENSEARCH_VERSION}.zip ] \
    && [ -f ${OPENSEARCH_HOME}/dist/analysis-kuromoji/analysis-kuromoji-${OPENSEARCH_VERSION}.zip ]; then \
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
