FROM quay.io/operator-framework/operator-sdk:v1.42

RUN microdnf install -y --nodocs \
    gcc \
    git \
    && microdnf clean all

ENTRYPOINT ["/bin/bash"]
