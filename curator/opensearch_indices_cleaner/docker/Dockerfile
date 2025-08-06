FROM python:3.10.18-alpine3.22

ENV ELASTICSEARCH_INDICES_CLEANER_HOME=/opt/elasticsearch-indices-cleaner

COPY docker/requirements.txt ${ELASTICSEARCH_INDICES_CLEANER_HOME}/requirements.txt
COPY docker/docker-entrypoint.sh /
COPY scripts/indices_cleaner.py ${ELASTICSEARCH_INDICES_CLEANER_HOME}/indices_cleaner.py
COPY scripts/indices_cleaner_tests.py ${ELASTICSEARCH_INDICES_CLEANER_HOME}/indices_cleaner_tests.py

# Return back 5.8.1 version for 'elasticsearch-curator' in requirements when 'botocore' version is compatible with 5.8.1 version of 'elasticsearch-curator'
RUN set -x \
    && apk add --upgrade build-base apk-tools \
    && pip3 install --upgrade pip==22.1.2 setuptools==70.0.0 \
    && pip3 install -r ${ELASTICSEARCH_INDICES_CLEANER_HOME}/requirements.txt \
    && pip3 install --upgrade urllib3==1.26.17 \
    && rm -rf /var/cache/apk/*

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN set -x \
    && python3 -m unittest discover -s ${ELASTICSEARCH_INDICES_CLEANER_HOME} -p "*_tests.py"

RUN set -x \
    && for path in \
         /docker-entrypoint.sh \
    ; do \
        chmod +x "$path"; \
        chgrp 0 "$path"; \
    done

USER 1000:0

WORKDIR ${ELASTICSEARCH_INDICES_CLEANER_HOME}

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["elasticsearch-indices-cleaner"]

