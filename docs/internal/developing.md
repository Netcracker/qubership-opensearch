# Developer Guide

## Dev-Kit

### Operator Framework Terminal

The common developer case is changing CRD structure. After the changes, the all related Golang files should be
re-generated. To simplify work with Golang Operator Framework, the dev-kit has specific terminal.

To run it, need to go to `dev-kit/` directory and run `terminal.sh` file.

The `terminal.sh` file runs docker-compose with operator-framework image and mount to the project.

In the running terminal the developer can go to `opensearch-service-operator` directory and work with it.

For example, to re-generate code and CRDs need to run the following commands:

```sh
make generate
make manifests
```

More information can be found in [Operator guide](/docs/internal/operator-guide.md).

### JSON Schema

The Values JSON Schema is file which defines the structure of data that can be placed in `values.yaml`.

To generate or validate JSON Schema need to use [Netcracker values-schema generator](https://git.netcracker.com/Personal.Streaming.Platform/values-schema-generator).

The developer should install it on the host machine using [command line option](https://git.netcracker.com/Personal.Streaming.Platform/values-schema-generator#command-line).

When generator is installed, then developer can use the following scripts:

* `schema/make_json_schema.sh` - generates JSON Schema, which is placed in
  `charts/helm/opensearch-service` directory.
* `schema/json_schema_linter.sh` - runs generator in linter mode to validate current JSON Schema.
