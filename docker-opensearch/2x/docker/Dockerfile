FROM opensearchproject/opensearch:2.19.4

USER 0

ENV DISABLE_INSTALL_DEMO_CONFIG=true \
    OPENSEARCH_HOME=/usr/share/opensearch \
    OPENSEARCH_CONFIGS=/usr/share/opensearch/config

COPY 2x/docker/health.sh ${OPENSEARCH_HOME}/bin
COPY 2x/docker/reconfiguration.sh ${OPENSEARCH_HOME}/bin
COPY 2x/docker/relocate-shards.sh ${OPENSEARCH_HOME}/bin
COPY 2x/docker/docker-entrypoint.sh /

# Install the tools we need
RUN yum update --assumeyes --skip-broken \
    && yum install -y jq-1.7.1 \
    && yum clean all \
    && rm -rf /var/cache/yum/*

RUN echo "Download OpenSearch plugins..." \
    && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-s3/2.19.4/repository-s3-2.19.4.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-2.19.4.zip \
    && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-gcs/2.19.4/repository-gcs-2.19.4.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-2.19.4.zip \
    && if [ -f ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-2.19.4.zip ] && [ -f ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-2.19.4.zip ]; then \
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

RUN set -x \
    && curl \
        --silent \
        --show-error \
        --output-dir ${OPENSEARCH_HOME}/plugins/opensearch-ml \
        -O "https://repo1.maven.org/maven2/org/json/json/20231013/json-20231013.jar" \
    && rm -rf ${OPENSEARCH_HOME}/plugins/opensearch-ml/json-20230227.jar

RUN set -x \
    && curl \
        --silent \
        --show-error \
        --output-dir ${OPENSEARCH_HOME}/plugins/opensearch-observability \
        -O "https://repo1.maven.org/maven2/org/json/json/20231013/json-20231013.jar" \
    && rm -rf ${OPENSEARCH_HOME}/plugins/opensearch-observability/json-20230227.jar

RUN set -x \
    && curl \
        --silent \
        --show-error \
        --output-dir ${OPENSEARCH_HOME}/plugins/opensearch-reports-scheduler \
        -O "https://repo1.maven.org/maven2/org/json/json/20231013/json-20231013.jar" \
    && rm -rf ${OPENSEARCH_HOME}/plugins/opensearch-reports-scheduler/json-20230227.jar

RUN set -x \
    && curl \
        --silent \
        --show-error \
        --output-dir ${OPENSEARCH_HOME}/plugins/opensearch-sql \
        -O "https://repo1.maven.org/maven2/org/json/json/20231013/json-20231013.jar" \
    && rm -rf ${OPENSEARCH_HOME}/plugins/opensearch-sql/json-20230227.jar

RUN set -x \
    && curl \
        --silent \
        --show-error \
        --output-dir ${OPENSEARCH_HOME}/plugins/opensearch-neural-search \
        -O "https://repo1.maven.org/maven2/org/json/json/20231013/json-20231013.jar" \
    && rm -rf ${OPENSEARCH_HOME}/plugins/opensearch-neural-search/json-20230227.jar

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./opensearch-docker-entrypoint.sh"]
