FROM opensearchproject/opensearch-dashboards:3.9.0@sha256:4bdb8ded547ce7f4c379a10261f1b4aad58726900c215be570a7d1fe86f2a7d8

USER 0

RUN yum update --assumeyes --skip-broken \
    && yum clean all \
    && rm -rf /var/cache/yum/*

RUN rm -f /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml \
    && ln -s /tmp/dashboards/opensearch_dashboards.yml /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml

COPY docker/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

USER 1000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["opensearch-dashboards"]
