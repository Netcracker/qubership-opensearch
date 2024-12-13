1. Specify required version of opensource Qubership OpenSearch transfer version in `build.env`.
2. Go to the `/.gitlab-ci/build/opensearch` and run the command:

    ```bash
    ./build.sh
    ```
    Merged Helm chart will be placed to `/.gitlab-ci/build/opensearch/helm` folder.

3. Prepare `sample.yaml` with parameters and use `helm install`.