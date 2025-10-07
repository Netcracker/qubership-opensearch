FROM opensearchproject/opensearch-dashboards:3.2.0

USER 0:0


ADD https://registry.npmjs.org/form-data/-/form-data-4.0.4.tgz /tmp/form-data.tgz
ADD https://registry.npmjs.org/axios/-/axios-0.30.1.tgz /tmp/axios.tgz

RUN rm -rf /usr/share/opensearch-dashboards/node_modules/form-data \
    && mkdir -p /usr/share/opensearch-dashboards/node_modules/form-data \
    && tar -xzf /tmp/form-data.tgz -C /usr/share/opensearch-dashboards/node_modules/form-data --strip-components=1 \
    \
    && rm -rf /usr/share/opensearch-dashboards/plugins/assistantDashboards/node_modules/form-data \
    && mkdir -p /usr/share/opensearch-dashboards/plugins/assistantDashboards/node_modules/form-data \
    && tar -xzf /tmp/form-data.tgz -C /usr/share/opensearch-dashboards/plugins/assistantDashboards/node_modules/form-data --strip-components=1 \
    \
    && rm -rf /usr/share/opensearch-dashboards/plugins/reportsDashboards/node_modules/form-data \
    && mkdir -p /usr/share/opensearch-dashboards/plugins/reportsDashboards/node_modules/form-data \
    && tar -xzf /tmp/form-data.tgz -C /usr/share/opensearch-dashboards/plugins/reportsDashboards/node_modules/form-data --strip-components=1


RUN rm -rf /usr/share/opensearch-dashboards/node_modules/axios \
    && mkdir -p /usr/share/opensearch-dashboards/node_modules/axios \
    && tar -xzf /tmp/axios.tgz -C /usr/share/opensearch-dashboards/node_modules/axios --strip-components=1

USER 1000

ENTRYPOINT ["./opensearch-dashboards-docker-entrypoint.sh"]
CMD ["opensearch-dashboards"]
