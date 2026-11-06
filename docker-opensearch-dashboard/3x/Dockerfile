FROM opensearchproject/opensearch-dashboards:3.9.0@sha256:256d4cea5928868ac09bca642a0ac8285a2b7e4b7fdd45c4ec06e75a5cf3184c

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
