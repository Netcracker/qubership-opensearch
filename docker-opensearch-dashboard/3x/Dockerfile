FROM opensearchproject/opensearch-dashboards:3.9.0@sha256:46c649e9c4412a56f49bd052c2ca458a2a4933e1706a1843b832b0806e3885dc

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
