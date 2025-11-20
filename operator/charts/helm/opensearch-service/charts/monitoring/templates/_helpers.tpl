{{- define "defaultAlerts" -}}
{{ .Release.Namespace }}-{{ .Release.Name }}:
  rules:
    OpenSearchCPULoadAlert:
      expr: max(opensearch_process_cpu_percent{namespace="{{ .Release.Namespace }}"}) > 95
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
      annotations:
        description: 'OpenSearch CPU usage is above 95 percent.'
        summary: OpenSearch's CPU usage is above 95 percent
    OpenSearchDiskUsageAbove75percentAlert:
      annotations:
        description: 'OpenSearch disk usage is above 75 percent'
        summary: OpenSearch's disk usage is above 75 percent
      expr: 1 - sum(opensearch_fs_total_free_in_bytes{namespace="{{ .Release.Namespace }}"}) / sum(opensearch_fs_total_total_in_bytes{namespace="{{ .Release.Namespace }}"}) > 0.75 <= 0.85
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchDiskUsageAbove85percentAlert:
      annotations:
        description: 'OpenSearch disk usage is above 85 percent'
        summary: OpenSearch's disk usage is above 85 percent
      expr: 1 - sum(opensearch_fs_total_free_in_bytes{namespace="{{ .Release.Namespace }}"}) / sum(opensearch_fs_total_total_in_bytes{namespace="{{ .Release.Namespace }}"}) > 0.85 <= 0.95
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchDiskUsageAbove95percentAlert:
      annotations:
        description: 'OpenSearch disk usage is above 95 percent'
        summary: OpenSearch's disk usage is above 95 percent
      expr: 1 - sum(opensearch_fs_total_free_in_bytes{namespace="{{ .Release.Namespace }}"}) / sum(opensearch_fs_total_total_in_bytes{namespace="{{ .Release.Namespace }}"}) > 0.95
      for: 3m
      labels:
        severity: critical
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchHeapMemoryUsageAlert:
      annotations:
        description: 'OpenSearch heap memory usage is above 95 percent.'
        summary: OpenSearch's heap memory usage is above 95 percent
      expr: max(opensearch_jvm_mem_heap_used_percent{namespace="{{ .Release.Namespace }}"}) > 95
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchIsDegradedAlert:
      annotations:
        description: 'OpenSearch is Degraded.'
        summary: Some of OpenSearch Service pods are down
      expr: opensearch_cluster_health_status_code{namespace="{{ .Release.Namespace }}"} == 6
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchIsDownAlert:
      annotations:
        description: 'OpenSearch is Down.'
        summary: All of OpenSearch Service pods are down
      expr: opensearch_cluster_health_status_code{namespace="{{ .Release.Namespace }}"} == 10
      for: 3m
      labels:
        severity: critical
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchDBaaSIsDownAlert:
      annotations:
        description: 'OpenSearch DBaaS agent is Down.'
        summary: OpenSearch DBaaS agent is Down
      expr: opensearch_dbaas_health_status{namespace="{{ .Release.Namespace }}"} == 1
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchLastBackupHasFailedAlert:
      annotations:
        description: 'OpenSearch Last Backup Has Failed.'
        summary: OpenSearch Last Backup Has Failed
      expr: opensearch_backups_metric_last_backup_status{namespace="{{ .Release.Namespace }}"} != 1
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchQueryIsTooSlowAlert:
      annotations:
        description: 'OpenSearch Query Is Too Slow.'
        summary: OpenSearch Query Is Too Slow
      expr: opensearch_slow_query_took_millis{namespace="{{ .Release.Namespace }}"} > 10 * 1000
      for: 1m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchReplicationFailedAlert:
      annotations:
        description: 'OpenSearch Replication has Failed.'
        summary: OpenSearch Replication has Failed
      expr: opensearch_replication_metric_status{namespace="{{ .Release.Namespace }}"} == 4
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchReplicationDegradedAlert:
      annotations:
        description: 'OpenSearch Replication has Degraded.'
        summary: OpenSearch Replication has Degraded
      expr: opensearch_replication_metric_status{namespace="{{ .Release.Namespace }}"} == 2
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchReplicationLeaderConnectionLostAlert:
      annotations:
        description: 'OpenSearch Replication Leader connection lost.'
        summary: OpenSearch Replication Follower lost connection with Leader side
      expr: opensearch_replication_metric_status{namespace="{{ .Release.Namespace }}"} == -1
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
    OpenSearchReplicationTooHighLagAlert:
      annotations:
        description: 'OpenSearch Replication has Index with too high Lag.'
        summary: OpenSearch Replication has Index with Lag higher than expected Maximum.
      expr: max(opensearch_replication_metric_index_lag{namespace="{{ .Release.Namespace }}"}) > -1
      for: 3m
      labels:
        severity: warning
        namespace: {{ .Release.Namespace }}
        service: {{ .Release.Name }}
  {{- end }}