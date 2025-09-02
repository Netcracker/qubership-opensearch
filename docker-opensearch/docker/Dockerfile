FROM amazoncorretto:11-alpine AS builder

RUN apk add --no-cache maven git

WORKDIR /build

COPY opensearch-filter-plugin/pom.xml .
RUN mvn dependency:go-offline

COPY opensearch-filter-plugin /build
RUN mvn clean package

FROM opensearchproject/opensearch:2.19.2

USER 0

ENV DISABLE_INSTALL_DEMO_CONFIG=true \
    OPENSEARCH_HOME=/usr/share/opensearch \
    OPENSEARCH_CONFIGS=/usr/share/opensearch/config

#COPY docker/config/yum-repositories.repo /etc/yum.repos.d
COPY docker/health.sh ${OPENSEARCH_HOME}/bin
COPY docker/reconfiguration.sh ${OPENSEARCH_HOME}/bin
COPY docker/relocate-shards.sh ${OPENSEARCH_HOME}/bin
COPY docker/docker-entrypoint.sh /

# Install the tools we need
#RUN yum install --disablerepo=* --enablerepo=paas,os,openstack,nc_updates --nogpgcheck -y jq \
#    && yum clean all \
RUN yum install --nogpgcheck -y jq \
    && yum clean all

# Upgrade all tools to avoid vulnerabilities
#RUN yum update --disablerepo=* --enablerepo=os,nc_updates --nogpgcheck --assumeyes --skip-broken
RUN yum update --nogpgcheck --assumeyes --skip-broken

RUN echo "Download OpenSearch plugins..." \
  && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-s3/2.19.2/repository-s3-2.19.2.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-2.19.2.zip \
  && curl --create-dirs -s "https://artifacts.opensearch.org/releases/plugins/repository-gcs/2.19.2/repository-gcs-2.19.2.zip" \
    -o ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-2.19.2.zip \
  && if [ -f ${OPENSEARCH_HOME}/dist/repository-s3/repository-s3-2.19.2.zip ] && [ -f ${OPENSEARCH_HOME}/dist/repository-gcs/repository-gcs-2.19.2.zip ]; then \
       echo "Plugins downloaded successfully."; \
     else \
       echo "Plugin download failed"; \
       exit 1; \
     fi

COPY --from=builder /build/target/opensearch-filter-plugin-2.19.2.0.zip  ${OPENSEARCH_HOME}/dist/opensearch-filter-plugin/opensearch-filter-plugin-2.19.2.0.zip

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
