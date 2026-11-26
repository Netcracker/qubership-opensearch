# AGENTS.md

Qubership OpenSearch is an enterprise OpenSearch distribution for
Kubernetes/OpenShift. It bundles a Kubebuilder-based operator, a Helm chart,
OpenSearch and OpenSearch Dashboards container images, several auxiliary
services (Go and Python), monitoring assets, and a Robot Framework
integration test suite.

## Repository Layout

Monorepo. `go.mod` files live under `operator/`, `dbaas-adapter/` and
`curator/docker/backup-docker/indices_migration_tool/`. The `curator`
(backup daemon, indices cleaner) and `monitoring` exec scripts are Python.

- `/operator` — Kubebuilder v3 operator (`projectName: opensearch-service`,
  domain `netcracker.com`). Single binary that reconciles one
  `OpenSearchService` CR (`netcracker.com/v1`) per namespace
  (`WATCH_NAMESPACE`).
- `/operator/charts/helm/opensearch-service` — The only Helm chart. Installs
  the operator, an `OpenSearchService` CR, the OpenSearch cluster
  (master/data/arbiter/client roles), and optional components (dashboards,
  curator, DBaaS adapter, monitoring, disaster recovery, TLS init,
  integration tests, status provisioner). CRD lives in `crds/crd.yaml`.
- `/dbaas-adapter` — Go REST service implementing the DBaaS aggregator
  contract (databases/indices, users, roles, role mappings, backups, users
  recovery).
- `/curator` — Python backup daemon (`docker/backup-docker`, built on
  `qubership-backup-daemon-go`) for full and granular snapshot
  backup/restore (PV or S3), plus the Go `indices_migration_tool` and the
  `opensearch_indices_cleaner` job. Unit tests: `curator_tests.py` and
  `indices_cleaner_tests.py` next to the scripts.
- `/monitoring` — Telegraf-based image with Python exec scripts
  (`exec-scripts/*_metric.py`, unit tests alongside as `test_*.py`); Grafana
  dashboards and Prometheus alert rules are shipped in the chart under
  `operator/charts/helm/opensearch-service/monitoring`.
- `/docker-opensearch` — Custom OpenSearch images with security plugin,
  health checks and Prometheus exporter. Subdirs `2x/` and `3x/` build
  OpenSearch 2.x and 3.x.
- `/docker-opensearch-dashboard` — OpenSearch Dashboards images (`2x/`,
  `3x/`).
- `/tls-init` — Job image that generates TLS certificates.
- `/docker-transfer` — `FROM scratch` image used to ship charts and docs.
- `/integration-tests` — Robot Framework suites at `robot/tests/opensearch/`.
- `/demo` — Legacy Docker Compose stacks (curator demo, monitoring demo).
  Not usable for running the integration tests.
- `/openspec` — OpenSpec change proposals and specs.
- `/agent-packages`, `apm.yml` — APM packages for AI agents (the
  `troubleshoot-opensearch` skill).
- `/docs/public` — User-facing documentation (architecture, installation,
  troubleshooting, security, TLS, disaster recovery, etc.).
- `/operator/Makefile` — Drives operator codegen and tests (`make generate`,
  `make manifests`, `make unit-test`).

## Operator Code Organization

- `operator/api/v1` — `OpenSearchService` type. Spec sections:
  `opensearch`, `externalOpenSearch`, `dashboards`, `monitoring`,
  `dbaasAdapter`, `elasticsearchDbaasAdapter`, `curator`,
  `disasterRecovery`.
- `operator/controllers/opensearchservice_controller.go` — Main controller;
  dispatches to one reconciler per component.
- `operator/controllers/<component>_reconciler.go` — Per-component
  reconciliation logic (`opensearch`, `external_opensearch`, `dashboards`,
  `monitoring`, `dbaas_adapter`, `elasticsearch_dbaas_adapter`, `curator`,
  `disaster_recovery`). Shared helpers in `opensearchservice_reconciler.go`
  and `condition.go`.
- `operator/controllers/*_watcher.go`, `replication_manager.go` — Background
  watchers (index settings, slowlog indices, cross-cluster replication).
- `operator/disasterrecovery/` — DR HTTP server and replication checker.
- `operator/util/` — REST client, status updater, hashing, env/secret
  helpers. Operator configuration comes from env vars (`OPENSEARCH_NAME`,
  `OPENSEARCH_PROTOCOL`, `WATCH_NAMESPACE`, `LOG_LEVEL`, …) read in
  `main.go` and `util/util.go`.

## Compatibility Rules

All changes MUST be backward compatible.

### CRDs

- Add new functionality through new optional fields. Do not change the
  semantics or shape of existing fields.
- If a breaking change is unavoidable, stop and ask the user for explicit
  approval before adding a new API version.
- After editing anything under `operator/api/`, run `make generate` and
  `make manifests`, and sync the chart CRD in
  `operator/charts/helm/opensearch-service/crds/crd.yaml`. Use the dev-kit if
  `operator-sdk` is not installed locally (see below).

### Helm Charts

- Every new parameter MUST have a sensible default; never break existing
  values files.
- Document every new parameter in `docs/public/installation.md` and keep
  `values.schema.json` in sync (`operator/dev-kit/make_json_schema.sh`).
- Reuse shared templates in `templates/_helpers.tpl` for labels,
  annotations, images and security contexts.
- Every change must be tested with `helm template`.

## Dev-Kit (Operator Code Generation)

Use the dev-kit when `operator-sdk` is not installed locally. It runs
`operator-sdk:v1.42` in Docker with the project mounted.

```sh
cd operator/dev-kit
./terminal.sh        # Builds and enters the operator-sdk container
make generate        # Regenerate DeepCopy code from operator/api/
make manifests       # Regenerate CRDs in config/crd/bases
make unit-test       # Run operator unit tests (go test -tags unit)
```

## Integration Tests

Robot Framework suites at `integration-tests/robot/tests/opensearch/`
(`smoke`, `authentication`, `backup`, `dbaas`, `ha`, `prometheus`,
`image_tests`). They run against a deployed OpenSearch in Kubernetes: the
chart deploys them as a pod (`templates/integration-tests`), and CI runs
them via `.github/workflows/run_tests.yaml` (Netcracker/qubership-test-pipelines).

### Mandatory – every change must be tested

- Any change in docker folders where Dockerfiles are placed must be tested
  with Docker build.
- Any code change must be covered with unit tests (`go test` for
  `operator` and `dbaas-adapter`, Python `unittest` for `curator` and
  `monitoring/exec-scripts`).
- Any functional feature of components that can be tested without redeploy
  must be covered with Robot Framework tests.
- Before submitting any change to `curator`, `dbaas-adapter` or
  `integration-tests` folders, run **at minimum** the `smoke`, `backup` and
  `dbaas` tagged suites.

```sh
cd operator && make unit-test
cd dbaas-adapter && go test ./...
cd monitoring/exec-scripts && python -m unittest discover -p "test_*.py"
cd curator/docker/backup-docker/scripts && python -m unittest curator_tests
robot -i smokeORbackupORdbaas ./tests   # inside the integration-tests pod
```

See [integration-tests/README.md](integration-tests/README.md) for the
full environment reference (environment variables, tags, deployment).
