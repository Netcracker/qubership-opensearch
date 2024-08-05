FROM artifactorycn.netcracker.com:17014/product/prod.platform.streaming_docker-integration-tests:0.1.14_20240802-014533

ENV ROBOT_OUTPUT=/opt/robot/output \
    SERVICE_CHECKER_SCRIPT=${ROBOT_HOME}/opensearch_pods_checker.py \
    STATUS_CUSTOM_RESOURCE_GROUP=apps \
    STATUS_CUSTOM_RESOURCE_VERSION=v1 \
    STATUS_CUSTOM_RESOURCE_PLURAL=deployments

RUN mkdir -p ${ROBOT_OUTPUT}

COPY docker/pip.conf /etc/pip.conf
COPY docker/requirements.txt ${ROBOT_HOME}/requirements.txt
COPY docker/opensearch_pods_checker.py ${ROBOT_HOME}/opensearch_pods_checker.py
COPY robot ${ROBOT_HOME}

RUN set -x \
    && pip3 install -r ${ROBOT_HOME}/requirements.txt \
    && rm -rf /var/cache/apk/*

USER 1000:0

EXPOSE 8080
VOLUME ["${ROBOT_OUTPUT}"]
