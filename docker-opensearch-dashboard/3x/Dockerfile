FROM opensearchproject/opensearch-dashboards:3.7.0@sha256:1c9e0a50472123bd2cde615ffb7157ef436db794e082dba09e42c3a6d3ce91a3

USER 0

RUN rm -f /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml \
    && ln -s /tmp/dashboards/opensearch_dashboards.yml /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml

COPY docker/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

USER 1000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["opensearch-dashboards"]
