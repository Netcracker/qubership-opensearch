{{/*
Expand the name of the chart.
*/}}
{{- define "opensearch.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "opensearch.fullname" -}}
{{- if .Values.fullnameOverride -}}
  {{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
  {{- $name := default .Chart.Name .Values.nameOverride -}}
  {{- if contains $name .Release.Name -}}
    {{- .Release.Name | trunc 63 | trimSuffix "-" -}}
  {{- else -}}
    {{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Define standard labels for frequently used metadata.
*/}}
{{- define "opensearch.labels.standard" -}}
app: {{ template "opensearch.fullname" . }}
chart: "{{ .Chart.Name }}-{{ .Chart.Version }}"
release: "{{ .Release.Name }}"
heritage: "{{ .Release.Service }}"
{{- end -}}

{{/*
Define labels for Deployment/StatefulSet selectors.
We cannot have the chart label here as it will prevent upgrades.
*/}}
{{- define "opensearch.labels.selector" -}}
app: {{ template "opensearch.fullname" . }}
release: "{{ .Release.Name }}"
heritage: "{{ .Release.Service }}"
{{- end -}}

{{/*
Define readiness probe for OpenSearch Deployment/StatefulSet.
*/}}
{{- define "opensearch.readiness.probe" -}}
exec:
  command:
    - '/bin/bash'
    - '-c'
    - '/usr/share/opensearch/bin/health.sh readiness-probe'
initialDelaySeconds: 40
periodSeconds: 20
timeoutSeconds: 20
successThreshold: 1
failureThreshold: 5
{{- end -}}

{{/*
Define OpenSearch data nodes count.
*/}}
{{- define "opensearch.dataNodes.count" -}}
{{- if .Values.opensearch.data.dedicatedPod.enabled }}
  {{- .Values.opensearch.data.replicas }}
{{- else }}
  {{- .Values.opensearch.master.replicas }}
{{- end }}
{{- end -}}

{{/*
Define OpenSearch total nodes count.
*/}}
{{- define "opensearch.nodes.count" -}}
{{- $masterNodes := .Values.opensearch.master.replicas }}
{{- $dataNodes := 0 }}
{{- if .Values.opensearch.data.dedicatedPod.enabled }}
  {{- $dataNodes = .Values.opensearch.data.replicas | int }}
{{- end }}
{{- $clientNodes := 0 }}
{{- if .Values.opensearch.client.dedicatedPod.enabled }}
  {{- $clientNodes = .Values.opensearch.client.replicas | int }}
{{- end }}
{{- add $masterNodes $dataNodes $clientNodes }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "dashboards.serviceAccountName" -}}
{{- if .Values.dashboards.serviceAccount.create -}}
  {{ default (include "opensearch.fullname" .) .Values.dashboards.serviceAccount.name }}-dashboards
{{- else -}}
  {{ default "default" .Values.dashboards.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "opensearch.serviceAccountName" -}}
{{- if .Values.opensearch.serviceAccount.create -}}
  {{ default (include "opensearch.fullname" .) .Values.opensearch.serviceAccount.name }}
{{- else -}}
  {{ default "default" .Values.opensearch.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Define if OpenSearch is to be deployed in 'joint' mode.
*/}}
{{- define "joint-mode" -}}
{{- if or .Values.opensearch.data.dedicatedPod.enabled .Values.opensearch.client.dedicatedPod.enabled }}
  {{- "false" -}}
{{- else }}
  {{- "true" -}}
{{- end -}}
{{- end -}}

{{/*
Define the name of the transport certificates secret.
*/}}
{{- define "opensearch.transport-cert-secret-name" -}}
{{- if .Values.opensearch.ssl.transport.existingCertSecret }}
  {{- .Values.opensearch.ssl.transport.existingCertSecret -}}
{{- else }}
  {{- template "opensearch.fullname" . -}}-transport-certs
{{- end -}}
{{- end -}}

{{/*
Define the path to the transport certificate in secret.
*/}}
{{- define "opensearch.transport-cert-path" -}}
{{- if .Values.opensearch.ssl.transport.existingCertSecret }}
  {{- .Values.opensearch.ssl.transport.existingCertSecretCertSubPath -}}
{{- else }}
  {{- "transport-crt.pem" -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the transport private key in secret.
*/}}
{{- define "opensearch.transport-key-path" -}}
{{- if .Values.opensearch.ssl.transport.existingCertSecret }}
  {{- .Values.opensearch.ssl.transport.existingCertSecretKeySubPath -}}
{{- else }}
  {{- "transport-key.pem" -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the transport root CA in secret.
*/}}
{{- define "opensearch.transport-root-ca-path" -}}
{{- if .Values.opensearch.ssl.transport.existingCertSecret }}
  {{- .Values.opensearch.ssl.transport.existingCertSecretRootCASubPath -}}
{{- else }}
  {{- "transport-root-ca.pem" -}}
{{- end -}}
{{- end -}}

{{/*
Define the name of the admin certificates secret.
*/}}
{{- define "opensearch.admin-cert-secret-name" -}}
{{- if .Values.opensearch.ssl.admin.existingCertSecret }}
  {{- .Values.opensearch.ssl.admin.existingCertSecret -}}
{{- else }}
  {{- template "opensearch.fullname" . -}}-admin-certs
{{- end -}}
{{- end -}}

{{/*
Define the path to the admin certificate in secret.
*/}}
{{- define "opensearch.admin-cert-path" -}}
{{- if .Values.opensearch.ssl.admin.existingCertSecret }}
  {{- .Values.opensearch.ssl.admin.existingCertSecretCertSubPath -}}
{{- else }}
  {{- "admin-crt.pem" -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the admin private key in secret.
*/}}
{{- define "opensearch.admin-key-path" -}}
{{- if .Values.opensearch.ssl.admin.existingCertSecret }}
  {{- .Values.opensearch.ssl.admin.existingCertSecretKeySubPath -}}
{{- else }}
  {{- "admin-key.pem" -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the admin root CA in secret.
*/}}
{{- define "opensearch.admin-root-ca-path" -}}
{{- if .Values.opensearch.ssl.admin.existingCertSecret }}
  {{- .Values.opensearch.ssl.admin.existingCertSecretRootCASubPath -}}
{{- else }}
  {{- "admin-root-ca.pem" -}}
{{- end -}}
{{- end -}}

{{/*
Define name for OpenSearch master nodes.
*/}}
{{- define "master-nodes" -}}
{{- if eq (include "joint-mode" .) "true" }}
  {{- template "opensearch.fullname" . -}}
{{- else }}
  {{- template "opensearch.fullname" . -}}-master
{{- end }}
{{- end -}}

{{/*
Define the list of full names of OpenSearch master nodes.
*/}}
{{- define "initial-master-nodes" -}}
{{- $replicas := .Values.opensearch.master.replicas | int }}
  {{- range $i, $e := untilStep 0 $replicas 1 -}}
    {{ template "master-nodes" $ }}-{{ $i }},
  {{- end -}}
{{- if .Values.opensearch.arbiter.enabled }}
{{- $arbiter_replicas := .Values.opensearch.arbiter.replicas | int }}
  {{- range $i, $e := untilStep 0 $arbiter_replicas 1 -}}
    {{ template "opensearch.fullname" $ }}-arbiter-{{ $i }},
  {{- end -}}
{{- end }}
{{- end -}}

{{/*
Define if persistent volumes are to be enabled for OpenSearch master nodes.
*/}}
{{- define "master-nodes-volumes-enabled" -}}
{{- if and .Values.opensearch.master.persistence.enabled .Values.opensearch.master.persistence.nodes }}
  {{- "true" -}}
{{- else }}
  {{- "false" -}}
{{- end -}}
{{- end -}}

{{/*
Define if persistent volumes are to be enabled for OpenSearch arbiter nodes.
*/}}
{{- define "arbiter-nodes-volumes-enabled" -}}
{{- if and .Values.opensearch.arbiter.persistence.enabled .Values.opensearch.arbiter.persistence.nodes }}
  {{- "true" -}}
{{- else }}
  {{- "false" -}}
{{- end -}}
{{- end -}}

{{/*
Define if persistent volumes are to be enabled for OpenSearch data nodes.
*/}}
{{- define "data-nodes-volumes-enabled" -}}
{{- if and .Values.opensearch.data.persistence.enabled .Values.opensearch.data.persistence.nodes }}
  {{- "true" -}}
{{- else }}
  {{- "false" -}}
{{- end -}}
{{- end -}}

{{/*
Configure OpenSearch service 'enableDisasterRecovery' property
*/}}
{{- define "opensearch.enableDisasterRecovery" -}}
{{- if or (eq .Values.global.disasterRecovery.mode "active") (eq .Values.global.disasterRecovery.mode "standby") (eq .Values.global.disasterRecovery.mode "disabled") -}}
  {{- printf "true" }}
{{- else -}}
  {{- printf "false" }}
{{- end -}}
{{- end -}}

{{- define "pod-scheduler-enabled" -}}
{{- if and .Values.podScheduler.enabled (or (eq (include "master-nodes-volumes-enabled" .) "true") (eq (include "data-nodes-volumes-enabled" .) "true")) }}
  {{- "true" -}}
{{- else }}
  {{- "false" -}}
{{- end -}}
{{- end -}}

{{/*
Define the name of DBaaS OpenSearch adapter.
*/}}
{{- define "dbaas-adapter.name" -}}
{{ printf "dbaas-%s-adapter" (include "opensearch.fullname" .) }}
{{- end -}}

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
    {{- if index .Values.deployDescriptor "opensearch-service" -}}
      {{- printf "%s" (index .Values.deployDescriptor "opensearch-service" "image") -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.opensearchOperator.image) -}}
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