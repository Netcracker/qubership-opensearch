# Elasticsearch Indices Cleaner Configuration

## Overview

The Elasticsearch Indices Cleaner deletes Elasticsearch indices which match with predefined patterns. To delete Elasticsearch indices
it organizes special scheduling process. Special scheduler executes script time by time. To specify scheduling process you should 
change `INDICES_CLEANER_SCHEDULER_UNIT` and `INDICES_CLEANER_SCHEDULER_UNIT_COUNT` deployments environment variables. To specify patterns for deleting indices
you should change OpenShift config map `es-curator-configuration`.

## Patterns configuration

OpenShift config map `es-curator-configuration` should contain config dictionary with one pair 
where key is value of `INDICES_CLEANER_CONFIGURATION_KEY` config parameter and value is list of "configuration items" or is absent.
By default there are no configuration items at all and they should be written by project side.
Configuration item is simple dictionary (without nested structures for values) which contains the following list of keys:

* `name` (Optional) - the name of particular configuration item.

* `filter_kind` (Required) - specifies kind of filtration for found Elasticsearch indices. Possible values are `prefix`, `postfix`.

* `filter_value` (Required) - specifies the regexp value for filtration. For example, `streaming` and `filter_kind` as `prefix` allows
  exclude all indices which does not starts with  `streaming`.
  
* `filter_direction` (Required) - specifies direction for "age filtration". Possible values are `older`, `younger`.

* `filter_unit` (Required) - specifies "age filtration" unit. Possible values are `days`, `hours`, `minutes`, `monthes`, `years`.

* `filter_unit_count` (Required) - specifies "age filtration" unit count. Should contains an integer value. 
  For example, if it is `5` and `filter_unit` is  `days` and `filter_direction` is `older` all indices which have been created before than
  5 days ago will be deleted.
  
Example:

```
patterns_to_delete:
  - name: zipkin
    filter_kind: prefix
    filter_value: streaming
    filter_direction: older
    filter_unit: days
    filter_unit_count: 1
```    

this configuration is a part of OpenShift config map. It contains one configuration item - "zipkin" and Elasticsearch Indices Cleaner
will delete all Elasticsearch indices which start with "streaming" and have been created before than 1 day ago on every iteration.
`Note!` On OpenShift 1.5 environment scale down and scale up for Elasticsearch Curator deployment config needed after changes in the Config Map. 
  