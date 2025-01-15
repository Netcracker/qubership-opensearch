# Build the manager binary
# Note: this uses host platform for the build, and we ask go build to target the needed platform, so we do not spend time on qemu emulation when running "go build"
FROM --platform=$BUILDPLATFORM golang:1.22.5-alpine3.20 as builder
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GOSUMDB=off

WORKDIR /workspace
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
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:3.20.2

ENV USER_UID=1001 \
    USER_NAME=opensearch-service-operator \
    GROUP_NAME=opensearch-service-operator

WORKDIR /
COPY --from=builder --chown=${USER_UID} /workspace/manager .

# Avoiding vulnerabilities
RUN set -x \
    && apk add --upgrade --no-cache apk-tools grep curl

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN addgroup ${GROUP_NAME} && adduser -D -G ${GROUP_NAME} -u ${USER_UID} ${USER_NAME}
USER ${USER_UID}

ENTRYPOINT ["/manager"]
