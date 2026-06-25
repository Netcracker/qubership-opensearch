FROM opensearchproject/opensearch-dashboards:3.6.0

USER 0

RUN rm -f /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml \
    && ln -s /tmp/dashboards/opensearch_dashboards.yml /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml

COPY docker/docker-entrypoint.sh /usr/share/opensearch-dashboards/qubership-docker-entrypoint.sh
RUN chmod +x /usr/share/opensearch-dashboards/qubership-docker-entrypoint.sh

USER 1000

WORKDIR /usr/share/opensearch-dashboards
ENTRYPOINT ["./qubership-docker-entrypoint.sh"]
CMD ["opensearch-dashboards"]
