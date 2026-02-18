FROM --platform=$BUILDPLATFORM golang:1.25.6-alpine3.23 as builder
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GOSUMDB=off

WORKDIR /workspace
# Copy the Go Modules manifests
COPY docker/backup-docker/indices_migrator/go.mod go.mod
COPY docker/backup-docker/indices_migrator/go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY docker/backup-docker/indices_migrator/main.go main.go
# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o migrator main.go

FROM ghcr.io/netcracker/qubership-backup-daemon:main_alpine

WORKDIR /
ENV ELASTICSEARCH_CURATOR_HOME=/opt/elasticsearch-curator

COPY backup-daemon.conf /etc/backup-daemon.conf
COPY docker/backup-docker/scripts ${ELASTICSEARCH_CURATOR_HOME}
COPY docker/backup-docker/config/curator.yml ${ELASTICSEARCH_CURATOR_HOME}/config/curator.yml
COPY docker/backup-docker/actions ${ELASTICSEARCH_CURATOR_HOME}/actions
COPY docker/backup-docker/docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=builder --chown=${USER_UID} /workspace/migrator ${ELASTICSEARCH_CURATOR_HOME}

RUN set -x \
    && pip3 install --upgrade pip setuptools wheel \
    && pip3 install -U opensearch-py==2.5.0 elasticsearch==7.17.12 curator-opensearch==0.0.13 \
    && pip3 install --upgrade pip urllib3 click \
    && rm -rf /var/cache/apk/*

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
        pcre2 libcurl curl

USER 1000:0

WORKDIR ${ELASTICSEARCH_CURATOR_HOME}

ENTRYPOINT ["/docker-entrypoint.sh"]
