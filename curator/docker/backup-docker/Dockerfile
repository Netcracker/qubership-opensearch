FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.23 as builder
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GOSUMDB=off

# Build context must be repo root (see .qubership/docker-build-config.cfg: context ".").
# Layout matches go.mod replace ../../../../dbaas-adapter => /workspace/dbaas-adapter
WORKDIR /workspace
COPY dbaas-adapter /workspace/dbaas-adapter
COPY curator/docker/backup-docker/indices_migration_tool/go.mod curator/docker/backup-docker/indices_migration_tool/go.sum /workspace/curator/docker/backup-docker/indices_migration_tool/
WORKDIR /workspace/curator/docker/backup-docker/indices_migration_tool
RUN go mod download

COPY curator/docker/backup-docker/indices_migration_tool/main.go main.go
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o migration_tool main.go

FROM ghcr.io/netcracker/qubership-backup-daemon-go:main

ARG TARGETOS
ARG TARGETARCH

WORKDIR /
ENV ELASTICSEARCH_CURATOR_HOME=/opt/elasticsearch-curator

COPY curator/backup-daemon.conf /etc/backup-daemon.conf
COPY curator/docker/backup-docker/scripts ${ELASTICSEARCH_CURATOR_HOME}
COPY curator/docker/backup-docker/config/curator.yml ${ELASTICSEARCH_CURATOR_HOME}/config/curator.yml
COPY curator/docker/backup-docker/actions ${ELASTICSEARCH_CURATOR_HOME}/actions
COPY curator/docker/backup-docker/docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=builder --chown=${USER_UID} /workspace/curator/docker/backup-docker/indices_migration_tool/migration_tool ${ELASTICSEARCH_CURATOR_HOME}

RUN apk add --no-cache wget python3 python3-dev py3-pip \
    && apk update \
    && apk upgrade

#https://peps.python.org/pep-0668/
RUN apk add --no-cache wget python3 py3-pip \
    && pip install --break-system-packages --upgrade pip setuptools==80.10.2 wheel==0.46.2 \
    && pip install --break-system-packages \
        opensearch-py==2.5.0 \
        elasticsearch==7.17.12 \
        curator-opensearch==0.0.13 \
        urllib3 \
        click

#Download necessary tools
RUN set -x \
    && wget \
        --no-check-certificate \
        -nv \
        -O /usr/bin/kubectl \
        "https://dl.k8s.io/v1.35.1/bin/linux/${TARGETARCH}/kubectl" \
    && chmod +x /usr/bin/kubectl

#Download jq
RUN apk add --update --no-cache curl apk-tools \
    && apk add --update --no-cache wget \
    && wget -O "jq-linux32" \
    "https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux32" \
    && mv "jq-linux32" "/usr/local/bin/jq" \
    && chmod +x /docker-entrypoint.sh \
    && chmod -R 777 ${ELASTICSEARCH_CURATOR_HOME} \
    && chmod +x "/usr/local/bin/jq"


RUN set -x \
    && python3 -m unittest discover -s ${ELASTICSEARCH_CURATOR_HOME} -p "*_tests.py"

RUN chmod +x ${ELASTICSEARCH_CURATOR_HOME}/*

# Upgrade tools from edge to avoid vulnerabilities
RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/main" > /etc/apk/repositories  \
    && echo "https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
    && apk add --update --upgrade --no-cache \
        pcre2 libcurl curl py3-setuptools py3-setuptools-pyc musl-utils

USER 1000:0

WORKDIR ${ELASTICSEARCH_CURATOR_HOME}


ENTRYPOINT ["/docker-entrypoint.sh"]
