FROM quay.io/operator-framework/operator-sdk:v1.41

RUN microdnf install --nodocs \
    gcc \
    git \
    && microdnf clean all

ENTRYPOINT ["/bin/bash"]
