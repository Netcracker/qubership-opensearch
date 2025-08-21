# OpenSearch Monitoring

## Description

This service provides custom Telegraf sh plugin which collect metrics from cluster of OpenSearch.

### Custom scripts

* `exec-scripts/backup_metric.py` - collect metrics about OpenSearch snapshots.
* `exec-scripts/dbaas_health_metric.py` - collect metrics about DBaaS cluster health and status.
* `exec-scripts/health_metric.py` - collect metrics about cluster health, status and total nodes count.
* `exec-scripts/replication_metric.py` - collect metrics about replication process.
* `exec-scripts/slow_queries_metric.py` - collect metrics about slow queries to a search engine.