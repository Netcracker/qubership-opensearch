FROM ghcr.io/netcracker/qubership-docker-integration-tests:main

USER root

ENV ROBOT_OUTPUT=/opt/robot/output \
    SERVICE_CHECKER_SCRIPT=${ROBOT_HOME}/opensearch_pods_checker.py \
    STATUS_CUSTOM_RESOURCE_GROUP=apps \
    STATUS_CUSTOM_RESOURCE_VERSION=v1 \
    STATUS_CUSTOM_RESOURCE_PLURAL=deployments

RUN mkdir -p ${ROBOT_OUTPUT}

COPY docker/requirements.txt ${ROBOT_HOME}/requirements.txt
COPY docker/opensearch_pods_checker.py ${ROBOT_HOME}/opensearch_pods_checker.py
COPY robot ${ROBOT_HOME}

RUN chmod -R 775 ${ROBOT_HOME}/tests/opensearch/ha/test-data-resources/*

RUN set -x \
    && pip3 install -r ${ROBOT_HOME}/requirements.txt \
    && apk add --no-cache curl \
    && rm -rf /var/cache/apk/*

RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/main" > /etc/apk/repositories \
    && echo "https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
    && apk add --update --upgrade --no-cache \
        libcurl curl

RUN set -x && apk add --update --no-cache apk-tools && apk upgrade --no-cache --available

RUN chown -R 1000:0 ${ROBOT_HOME} && chmod -R 775 ${ROBOT_HOME}

USER 1000:0

EXPOSE 8080
VOLUME ["${ROBOT_OUTPUT}"]
