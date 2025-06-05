`OpenSearch Filter` is a plugin for OpenSearch that offers security filtering features.

# Features

## IsmSecurityFilter

`IsmSecurityFilter` is a security filter for API of [Index State Management](https://opensearch.org/docs/latest/im-plugin/ism/index/) plugin. It adds one more level in OpenSearch filters chain to check the user with specific settings has the rights to create, read, update and remove ISM policies and managed indices. This filter is applied in the following cases:

* Received action is an ISM action, i.e. starts with `cluster:admin/opendistro/ism`.
* The user executing request has `dbaas_ism` `backend_role` and specified `attr.internal.resource_prefix` attribute.

In all other cases `IsmSecurityFilter` passes execution to the next filter in the chain.

The current implementation takes into account the following actions:

* `cluster:admin/opendistro/ism/managedindex/add` action corresponds to a request to add a policy to a managed index. The filter checks that the policy name and indices in the request start with the `resource_prefix`.
* `cluster:admin/opendistro/ism/managedindex/change` action corresponds to a request to update a managed index policy to a new policy (or to a new version of the policy). The filter checks that the policy name and indices in the request start with the `resource_prefix`.
* `cluster:admin/opendistro/ism/policy/delete` action corresponds to a request to remove policy. The filter checks policy name in the request starts with `resource_prefix`.
* `cluster:admin/opendistro/ism/managedindex/explain` action corresponds to a request to get the current state of a managed index. The filter checks that indices in the request start with `resource_prefix`.
* `cluster:admin/opendistro/ism/policy/search` action corresponds to a request to get all policies created in OpenSearch. This action is prohibited for DBaaS ISM user.
* `cluster:admin/opendistro/ism/policy/get` action corresponds to a request to get a policy content by specified name. The filter checks that the policy name in the request starts with `resource_prefix`.
* `cluster:admin/opendistro/ism/policy/write` action corresponds to a request to create a policy with specified settings. The filter checks that policy name and indices specified in `ism_template` section of the request start with the `resource_prefix`.
* `cluster:admin/opendistro/ism/managedindex/remove` action corresponds to a request to remove any ISM policy from a managed index. The filter checks that indices in the request start with `resource_prefix`.
* `cluster:admin/opendistro/ism/managedindex/retry` action corresponds to a request to retry a failed managed index. The filter checks that indices in the request start with `resource_prefix`.

Also, the filter changes DBaaS ISM user to user with `admin` permissions in policy or managed index configuration stored in `.opendistro-ism-config` OpenSearch index for the following actions:

* `cluster:admin/opendistro/ism/policy/write` stores user configuration in `policy.user` field.
* `cluster:admin/opendistro/ism/managedindex/add` stores user configuration in `managed_index.policy.user` field.
* `cluster:admin/opendistro/ism/managedindex/change` stores user configuration in `managed_index.policy.user` and `managed_index.change_policy.user` fields.
