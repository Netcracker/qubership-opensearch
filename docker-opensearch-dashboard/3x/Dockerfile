FROM opensearchproject/opensearch-dashboards:3.7.0

USER 0

RUN rm -f /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml \
    && ln -s /tmp/dashboards/opensearch_dashboards.yml /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml

COPY docker/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

USER 1000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["opensearch-dashboards"]
