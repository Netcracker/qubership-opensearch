FROM ghcr.io/netcracker/qubership-docker-integration-tests:main

# Switch to root for installation
USER root

ENV ROBOT_OUTPUT=/opt/robot/output \
    SERVICE_CHECKER_SCRIPT=${ROBOT_HOME}/opensearch_pods_checker.py

ENV STATUS_CUSTOM_RESOURCE_GROUP=apps
ENV STATUS_CUSTOM_RESOURCE_VERSION=v1
ENV STATUS_CUSTOM_RESOURCE_PLURAL=deployments
ENV STATUS_CUSTOM_RESOURCE_NAME=opensearch-integration-tests

RUN mkdir -p ${ROBOT_OUTPUT}

COPY docker/requirements.txt ${ROBOT_HOME}/requirements.txt
COPY docker/opensearch_pods_checker.py ${ROBOT_HOME}/opensearch_pods_checker.py
COPY robot ${ROBOT_HOME}

# We need to set grants explicitly for ha test data generation. Must be refactored in case of readonly fs feature.
RUN chmod -R 775 ${ROBOT_HOME}/tests/opensearch/ha/test-data-resources/*

RUN set -x \
    && pip3 install -r ${ROBOT_HOME}/requirements.txt \
    && rm -rf /var/cache/apk/*

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk add --update --no-cache apk-tools && apk upgrade --no-cache --available

# Fix permissions for non-root user
RUN chown -R 1000:0 ${ROBOT_HOME} && chmod -R 775 ${ROBOT_HOME}

# Switch back to non-root user
USER 1000:0

EXPOSE 8080
VOLUME ["${ROBOT_OUTPUT}"]
