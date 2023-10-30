# Build the manager binary
FROM artifactorycn.netcracker.com:17064/golang:1.20.10-alpine3.18 as builder

ENV GOPROXY=https://artifactorycn.netcracker.com/pd.sandbox-staging.go.group \
    GOSUMDB=off

WORKDIR /workspace
COPY alpine-repositories /etc/apk/repositories
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY disasterrecovery/ disasterrecovery/
COPY util/ util/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM artifactorycn.netcracker.com:17064/alpine:3.18.4

ENV USER_UID=1001 \
    USER_NAME=opensearch-service-operator \
    GROUP_NAME=opensearch-service-operator

WORKDIR /
COPY alpine-repositories /etc/apk/repositories
COPY --from=builder --chown=${USER_UID} /workspace/manager .

# Avoiding vulnerabilities
RUN set -x \
    && apk add --upgrade --no-cache apk-tools grep curl

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN addgroup ${GROUP_NAME} && adduser -D -G ${GROUP_NAME} -u ${USER_UID} ${USER_NAME}
USER ${USER_UID}

ENTRYPOINT ["/manager"]
