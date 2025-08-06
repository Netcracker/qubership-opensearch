FROM telegraf:1.35.1-alpine

ENV ELASTICSEARCH_MONITORING_HOME=/opt/elasticsearch-monitoring

RUN mkdir -p ${ELASTICSEARCH_MONITORING_HOME}

COPY docker/config/requirements.txt ${ELASTICSEARCH_MONITORING_HOME}/requirements.txt
COPY exec-scripts/ ${ELASTICSEARCH_MONITORING_HOME}/exec-scripts/
COPY docker/docker-entrypoint.sh /docker-entrypoint.sh

RUN set -x \
    && apk add --update --no-cache \
         bash \
         tini \
         python3 \
         apk-tools \
         busybox

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN rm /usr/lib/python3.12/EXTERNALLY-MANAGED

# Install misc tools
RUN set -x \
    && python3 -m ensurepip \
    && rm -r /usr/lib/python*/ensurepip \
    && pip3 install --upgrade pip setuptools==70.0.0 \
    && pip3 install -r ${ELASTICSEARCH_MONITORING_HOME}/requirements.txt \
    && pip3 install --upgrade urllib3==1.26.17 \
    && chmod -R 777 ${ELASTICSEARCH_MONITORING_HOME}/exec-scripts \
    && chmod +x /docker-entrypoint.sh \
    && python3 -m unittest discover -s ${ELASTICSEARCH_MONITORING_HOME}/exec-scripts -p "test_*.py" \
    && rm -fr ${ELASTICSEARCH_MONITORING_HOME}/exec-scripts/test_*.py \
    && rm -fr ${ELASTICSEARCH_MONITORING_HOME}/exec-scripts/test_resources \
    && rm -rf /var/cache/apk/*

WORKDIR ${ELASTICSEARCH_MONITORING_HOME}

#Telegraf try to read this file and throw exception, because this file doesn't exist
RUN touch /var/run/utmp

USER 1000:0

ENTRYPOINT ["/docker-entrypoint.sh"]
