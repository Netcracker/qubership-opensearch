{{/*
Find a busybox image in various places.
*/}}
{{- define "busybox.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.busybox -}}
      {{- printf "%s" .Values.busybox -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.busybox.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.opensearch.initContainer.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find a kubectl image in various places.
*/}}
{{- define "kubectl.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.kubectl -}}
      {{- printf "%s" .Values.kubectl -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.kubectl.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.podScheduler.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch Dashboards image in various places.
*/}}
{{- define "dashboards.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.dockerDashboards -}}
      {{- printf "%s" .Values.dockerDashboards -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.dockerDashboards.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.dashboards.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch image in various places.
*/}}
{{- define "opensearch.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.dockerOpensearch -}}
      {{- printf "%s" .Values.dockerOpensearch -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.dockerOpensearch.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.opensearch.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch image in various places.
*/}}
{{- define "tls-init.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.opensearchTlsInit -}}
      {{- printf "%s" .Values.opensearchTlsInit -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchTlsInit.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.opensearch.dockerTlsInitImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch monitoring image in various places.
*/}}
{{- define "monitoring.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.opensearchMonitoring -}}
      {{- printf "%s" .Values.opensearchMonitoring -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchMonitoring.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.monitoring.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find a DBaaS OpenSearch adapter image in various places.
*/}}
{{- define "dbaas-adapter.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.opensearchDbaasAdapter -}}
      {{- printf "%s" .Values.opensearchDbaasAdapter -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchDbaasAdapter.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.dbaasAdapter.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch curator image in various places.
*/}}
{{- define "curator.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.opensearchCurator -}}
      {{- printf "%s" .Values.opensearchCurator -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchCurator.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.curator.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch indices cleaner image in various places.
*/}}
{{- define "indices-cleaner.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.opensearchIndicesCleaner -}}
      {{- printf "%s" .Values.opensearchIndicesCleaner -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchIndicesCleaner.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.curator.dockerIndicesCleanerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch operator image in various places.
Image can be found from:
* SaaS/App deployer (or groovy.deploy.v3) from .Values.deployDescriptor "opensearch-service" "image"
* DP.Deployer from .Values.deployDescriptor.opensearchOperator.image
* or from default values .Values.operator.dockerImage
*/}}
{{- define "operator.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.opensearchServiceOperator -}}
      {{- printf "%s" .Values.opensearchServiceOperator -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchServiceOperator.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.operator.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an OpenSearch integration tests image in various places.
*/}}
{{- define "integration-tests.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.opensearchIntegrationTests -}}
      {{- printf "%s" .Values.opensearchIntegrationTests -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchIntegrationTests.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.integrationTests.dockerImage -}}
  {{- end -}}
{{- end -}}

{{/*
Find an Opensearch disaster recovery service operator image in various places.
Image can be found from:
* SaaS/App deployer (or groovy.deploy.v3) from .Values.disasterRecoveryImage
* DP.Deployer from .Values.deployDescriptor.disasterRecoveryImage.image
* or from default values .Values.global.disasterRecovery.image
*/}}
{{- define "disasterRecovery.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.disasterRecoveryImage -}}
      {{- printf "%s" .Values.disasterRecoveryImage -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.disasterRecoveryImage.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.global.disasterRecovery.image -}}
  {{- end -}}
{{- end -}}

{{/*
Find a Deployment Status Provisioner image in various places.
*/}}
{{- define "deployment-status-provisioner.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.deploymentStatusProvisioner -}}
      {{- printf "%s" .Values.deploymentStatusProvisioner -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.deploymentStatusProvisioner.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.statusProvisioner.dockerImage -}}
  {{- end -}}
{{- end -}}

{{ define "opensearch-service.findImage" }}
  {{- $root := index . 0 -}}
  {{- $service_name := index . 1 -}}
  {{- if index $root.Values.deployDescriptor $service_name }}
  {{- index $root.Values.deployDescriptor $service_name "image" }}
  {{- else }}
  {{- "not_found" }}
  {{- end }}
{{- end }}

{{- define "opensearch-service.monitoredImages" -}}
  {{- printf "deployment %s-service-operator %s-service-operator %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "opensearch-service.findImage" (list . "opensearch-service")) -}}
  {{- if and (not .Values.global.externalOpensearch.enabled) .Values.opensearch.master.enabled -}}
    {{- printf "statefulset %s opensearch %s, " ( include "master-nodes" . ) (include "opensearch-service.findImage" (list . "prod.platform.elasticstack_docker-opensearch")) -}}
  {{- end -}}
  {{- if .Values.curator.enabled -}}
    {{- printf "deployment %s-curator %s-curator %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "opensearch-service.findImage" (list . "docker-elastic-curator")) -}}
    {{- printf "deployment %s-curator %s-indices-cleaner %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "opensearch-service.findImage" (list . "prod.platform.elasticstack_docker-elastic-curator")) -}}
  {{- end -}}
  {{- if .Values.dashboards.enabled -}}
    {{- printf "deployment %s-dashboards %s-dashboards %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "opensearch-service.findImage" (list . "opensearch-dashboards")) -}}
  {{- end -}}
  {{- if .Values.monitoring.enabled -}}
    {{- printf "deployment %s-monitoring %s-monitoring %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "opensearch-service.findImage" (list . "elasticsearch-monitoring")) -}}
  {{- end -}}
  {{- if .Values.dbaasAdapter.enabled -}}
    {{- printf "deployment %s %s %s, " (include "dbaas-adapter.name" .) (include "dbaas-adapter.name" .) (include "opensearch-service.findImage" (list . "prod.platform.elasticstack_dbaas-opensearch-adapter")) -}}
  {{- end -}}
  {{- if .Values.integrationTests.enabled -}}
    {{- printf "deployment %s-integration-tests %s-integration-tests %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (index .Values "opensearchIntegrationTests") -}}
  {{- end -}}
  {{- if (eq (include "opensearch.enableDisasterRecovery" .) "true") -}}
    {{- printf "deployment %s-service-operator %s-disaster-recovery %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "opensearch-service.findImage" (list . "prod.platform.streaming_disaster-recovery-daemon")) -}}
  {{- end -}}
{{- end -}}
