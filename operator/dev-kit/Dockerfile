FROM quay.io/operator-framework/operator-sdk:v1.42@sha256:b7378eb179b97fbdf1ff1304e86f62e9ff9f9c2d0089464111ce8eddd39e2013

RUN microdnf install -y --nodocs \
    gcc \
    git \
    && microdnf clean all

ENTRYPOINT ["/bin/bash"]
