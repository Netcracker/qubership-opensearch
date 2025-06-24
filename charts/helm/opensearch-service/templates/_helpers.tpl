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
The most common OpenSearch resources labels
*/}}
{{- define "opensearch-service.coreLabels" -}}
app.kubernetes.io/version: '{{ .Values.ARTIFACT_DESCRIPTOR_VERSION | trunc 63 | trimAll "-_." }}'
app.kubernetes.io/part-of: '{{ .Values.PART_OF }}'
{{- end -}}

{{/*
Core OpenSearch resources labels with backend component label
*/}}
{{- define "opensearch-service.defaultLabels" -}}
{{ include "opensearch-service.coreLabels" . }}
app.kubernetes.io/component: 'backend'
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
{{- if .Values.global.externalOpensearch.enabled }}
  {{- .Values.global.externalOpensearch.dataNodesCount }}
{{- else }}
  {{- if .Values.opensearch.data.dedicatedPod.enabled }}
    {{- (include "opensearch.data.replicas" .) }}
  {{- else }}
    {{- (include "opensearch.master.replicas" .) }}
  {{- end }}
{{- end -}}
{{- end -}}

{{/*
Define OpenSearch total nodes count.
*/}}
{{- define "opensearch.nodes.count" -}}
{{- if .Values.global.externalOpensearch.enabled }}
  {{- .Values.global.externalOpensearch.nodesCount }}
{{- else }}
  {{- $masterNodes := (include "opensearch.master.replicas" .) }}
  {{- $dataNodes := 0 }}
  {{- if .Values.opensearch.data.dedicatedPod.enabled }}
    {{- $dataNodes = (include "opensearch.data.replicas" .) | int }}
  {{- end }}
  {{- $clientNodes := 0 }}
  {{- if .Values.opensearch.client.dedicatedPod.enabled }}
    {{- $clientNodes = (include "opensearch.client.replicas" .) | int }}
  {{- end }}
  {{- $arbiterNodes := 0 }}
  {{- if .Values.opensearch.arbiter.enabled }}
    {{- $arbiterNodes = (include "opensearch.arbiter.replicas" .) | int }}
  {{- end }}
  {{- add $masterNodes $dataNodes $clientNodes $arbiterNodes }}
{{- end -}}
{{- end -}}

{{/*
Define log4j configuration for OpenSearch.
*/}}
{{- define "opensearch.log4jConfig" -}}
  {{- if .Values.opensearch.log4jConfig -}}
    {{- .Values.opensearch.log4jConfig }}
  {{- end }}
  {{- if and (not .Values.global.externalOpensearch.enabled) .Values.monitoring.slowQueries.enabled }}
status = error
appender.console.type = Console
appender.console.name = STDOUT
appender.console.layout.type = PatternLayout
appender.console.layout.pattern = [%d{ISO8601}][%-5p][%-25c{1.}] [%node_name]%marker %m%n
appender.rolling.type = RollingFile
appender.rolling.name = RollingFile
appender.rolling.fileName = /usr/share/opensearch/logs/slow_logs.log
appender.rolling.filePattern = /usr/share/opensearch/logs/slowlog_query.log.%d{yyyy-MM-dd}.gz
appender.rolling.layout.type = PatternLayout
appender.rolling.layout.pattern = [%level], %d{yyyy-MM-dd'T'HH:mm:ss}, [%node_name], %msg%n
appender.rolling.policies.type = Policies
appender.rolling.policies.time.type = TimeBasedTriggeringPolicy
appender.rolling.policies.time.interval = 1
appender.rolling.policies.time.modulate = true
appender.rolling.policies.size.type = SizeBasedTriggeringPolicy
appender.rolling.policies.size.size=10MB
appender.rolling.strategy.type = DefaultRolloverStrategy
appender.rolling.strategy.max = 1
logger.rolling.name = index.search.slowlog.query
logger.rolling.appenderRef.rolling.ref = RollingFile
rootLogger.level = info
rootLogger.appenderRef.stdout.ref = STDOUT
  {{- end }}
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
Provider used to generate TLS certificates
*/}}
{{- define "certProvider" -}}
  {{- .Values.global.tls.enabled | ternary (default "dev" .Values.global.tls.generateCerts.certProvider) "dev" }}
{{- end -}}

{{/*
Whether Enhanced Security Plugin for OpenSearch is enabled
*/}}
{{- define "opensearch.enhancedSecurityPluginEnabled" -}}
  {{- if and (not .Values.global.externalOpensearch.enabled) .Values.opensearch.securityConfig.enhancedSecurityPlugin.enabled -}}
    {{- printf "true" -}}
  {{- else -}}
    {{- printf "false" -}}
  {{- end -}}
{{- end -}}

{{/*
Whether TLS for OpenSearch is enabled
*/}}
{{- define "opensearch.tlsEnabled" -}}
  {{- or (and (not .Values.global.externalOpensearch.enabled) .Values.opensearch.tls.enabled) (eq (include "external.useTlsSecret" .) "true") -}}
{{- end -}}

{{/*
OpenSearch configuration
*/}}
{{- define "opensearch.config" -}}
{{ toYaml .Values.opensearch.config }}
{{- if and (eq (include "opensearch.tlsEnabled" .) "true") (or .Values.opensearch.tls.cipherSuites .Values.global.tls.cipherSuites) }}
plugins.security.ssl.http.enabled_ciphers:
{{- range (coalesce .Values.opensearch.tls.cipherSuites .Values.global.tls.cipherSuites) }}
- {{ . | quote }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field
*/}}
{{- define "opensearch.certDnsNames" -}}
  {{- $opensearchName := include "opensearch.fullname" . -}}
  {{- $dnsNames := list "localhost" $opensearchName (printf "%s.%s" $opensearchName .Release.Namespace) (printf "%s.%s.svc" $opensearchName .Release.Namespace) (printf "%s-internal" $opensearchName) (printf "%s-internal.%s" $opensearchName .Release.Namespace) (printf "%s-internal.%s.svc" $opensearchName .Release.Namespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.opensearch.client.ingress.hosts }}
  {{- $dnsNames = concat $dnsNames .Values.opensearch.tls.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{/*
IP addresses used to generate TLS certificate with "Subject Alternative Name" field
*/}}
{{- define "opensearch.certIpAddresses" -}}
  {{- $ipAddresses := (list "127.0.0.1") -}}
  {{- $ipAddresses = concat $ipAddresses .Values.opensearch.tls.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}

{{/*
Define the path to the certificate in secret.
*/}}
{{- define "opensearch.cert-path" -}}
{{- template "opensearch.verifyExistingCertSecretCertSubPath" . -}}
{{- "tls.crt" -}}
{{- end -}}

{{/*
Define the path to the private key in secret.
*/}}
{{- define "opensearch.key-path" -}}
{{- template "opensearch.verifyExistingCertSecretKeySubPath" . -}}
{{- "tls.key" -}}
{{- end -}}

{{/*
Define the path to the root CA in secret.
*/}}
{{- define "opensearch.root-ca-path" -}}
{{- template "opensearch.verifyExistingCertSecretRootCASubPath" . -}}
{{- "ca.crt" -}}
{{- end -}}

{{/*
Whether transport certificates are Specified
*/}}
{{- define "opensearch.transportCertificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.opensearch.tls.transport.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- if and $filled .Values.global.tls.generateCerts.enabled -}}
    {{- fail "Incorrect deployment parameters configuration: Transport TLS certificates are defined in the parameters and global.tls.generateCerts.enabled is true. Please choose one of the TLS deploying configurations." -}}
  {{- end -}}
  {{- $filled -}}
{{- end -}}

{{/*
Define the name of the transport certificates secret.
*/}}
{{- define "opensearch.transport-cert-secret-name" -}}
{{- if and (not .Values.global.tls.generateCerts.enabled) .Values.opensearch.tls.transport.existingCertSecret }}
  {{- .Values.opensearch.tls.transport.existingCertSecret -}}
{{- else }}
  {{- if and .Values.global.tls.generateCerts.enabled (eq (include "certProvider" .) "cert-manager") }}
    {{- template "opensearch.fullname" . -}}-transport-issuer-certs
  {{- else -}}
    {{- template "opensearch.fullname" . -}}-transport-certs
  {{- end }}
{{- end -}}
{{- end -}}


{{/*
Whether admin certificates are Specified
*/}}
{{- define "opensearch.adminCertificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.opensearch.tls.admin.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- if and $filled .Values.global.tls.generateCerts.enabled -}}
    {{- fail "Incorrect deployment parameters configuration: Admin TLS certificates are defined in the parameters and global.tls.generateCerts.enabled is true. Please choose one of the TLS deploying configurations." -}}
  {{- end -}}
  {{- $filled -}}
{{- end -}}

{{/*
Define the name of the admin certificates secret.
*/}}
{{- define "opensearch.admin-cert-secret-name" -}}
{{- if and (not .Values.global.tls.generateCerts.enabled) .Values.opensearch.tls.admin.existingCertSecret }}
  {{- .Values.opensearch.tls.admin.existingCertSecret -}}
{{- else -}}
  {{- if and .Values.global.tls.generateCerts.enabled (eq (include "certProvider" .) "cert-manager") }}
    {{- template "opensearch.fullname" . -}}-admin-issuer-certs
  {{- else -}}
    {{- template "opensearch.fullname" . -}}-admin-certs
  {{- end }}
{{- end -}}
{{- end -}}

{{/*
Whether rest certificates are Specified
*/}}
{{- define "opensearch.restCertificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.opensearch.tls.rest.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- if and $filled .Values.global.tls.generateCerts.enabled -}}
    {{- fail "Incorrect deployment parameters configuration: REST TLS certificates are defined in the parameters and global.tls.generateCerts.enabled is true. Please choose one of the TLS deploying configurations." -}}
  {{- end -}}
  {{- $filled -}}
{{- end -}}

{{/*
Define the name of the REST certificates secret.
*/}}
{{- define "opensearch.rest-cert-secret-name" -}}
{{ if eq (include "external.useTlsSecret" .) "true" }}
  {{- .Values.global.externalOpensearch.tlsSecretName -}}
{{- else }}
  {{- if and (not .Values.global.tls.generateCerts.enabled) .Values.opensearch.tls.rest.existingCertSecret }}
    {{- .Values.opensearch.tls.rest.existingCertSecret -}}
  {{- else }}
    {{- if and .Values.global.tls.generateCerts.enabled (eq (include "certProvider" .) "cert-manager") }}
      {{- template "opensearch.fullname" . -}}-rest-issuer-certs
    {{- else -}}
      {{- template "opensearch.fullname" . -}}-rest-certs
    {{- end }}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Define name for OpenSearch master nodes.
*/}}
{{- define "master-nodes" -}}
{{- if eq (include "joint-mode" .) "true" }}
  {{- template "opensearch.fullname" . -}}
{{- else -}}
  {{- template "opensearch.fullname" . -}}-master
{{- end }}
{{- end -}}

{{/*
Define roles for OpenSearch cluster manager nodes.
*/}}
{{- define "cluster-manager-roles" -}}
{{- $roles := list "cluster_manager" }}
{{- if not .Values.opensearch.client.dedicatedPod.enabled }}
  {{- $roles = concat $roles (list "ingest" "remote_cluster_client") }}
{{- end }}
{{- if not .Values.opensearch.data.dedicatedPod.enabled }}
  {{- $roles = append $roles "data" }}
{{- end }}
{{- join "," $roles }}
{{- end }}

{{/*
Define the list of full names of OpenSearch master nodes.
*/}}
{{- define "initial-master-nodes" -}}
{{- $replicas := (include "opensearch.master.replicas" .) | int }}
  {{- range $i, $e := untilStep 0 $replicas 1 -}}
    {{ template "master-nodes" $ }}-{{ $i }},
  {{- end -}}
{{- if .Values.opensearch.arbiter.enabled }}
{{- $arbiter_replicas := (include "opensearch.arbiter.replicas" .) | int }}
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
{{- else -}}
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

{{/*
Configure OpenSearch service 'replicasForSingleService' property
*/}}
{{- define "opensearch.replicasForSingleService" -}}
{{- if or (eq .Values.global.disasterRecovery.mode "standby") (eq .Values.global.disasterRecovery.mode "disabled") -}}
  {{- 0 }}
{{- else -}}
  {{- 1 }}
{{- end -}}
{{- end -}}

{{/*
Whether TLS for Disaster Recovery is enabled
*/}}
{{- define "disasterRecovery.tlsEnabled" -}}
{{- and (eq (include "opensearch.enableDisasterRecovery" .) "true") .Values.global.tls.enabled .Values.global.disasterRecovery.tls.enabled -}}
{{- end -}}

{{/*
Cipher suites that can be used in Disaster Recovery
*/}}
{{- define "disasterRecovery.cipherSuites" -}}
  {{- join "," (coalesce .Values.global.disasterRecovery.tls.cipherSuites .Values.global.tls.cipherSuites) -}}
{{- end -}}

{{/*
Whether DRD certificates are Specified
*/}}
{{- define "disasterRecovery.certificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.global.disasterRecovery.tls.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- $filled -}}
{{- end -}}

{{/*
TLS secret name for Disaster Recovery
*/}}
{{- define "disasterRecovery.certSecretName" -}}
  {{- if and .Values.global.disasterRecovery.tls.enabled .Values.global.tls.enabled -}}
    {{- if and (or .Values.global.tls.generateCerts.enabled (eq (include "disasterRecovery.certificatesSpecified" .) "true")) (not .Values.global.disasterRecovery.tls.secretName) -}}
      {{- template "opensearch.fullname" . -}}-drd-tls-secret
    {{- else -}}
      {{- required "The TLS secret name should be specified in the 'disasterRecovery.tls.secretName' parameter when the service is deployed with disasterRecovery and TLS enabled, but without certificates generation." .Values.global.disasterRecovery.tls.secretName -}}
    {{- end -}}
  {{- else -}}
     {{/*
       The empty string is needed for correct prometheus rule configuration in `tls_static_metrics.yaml`
     */}}
    {{- "" -}}
  {{- end -}}
{{- end -}}

{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field for Disaster Recovery
*/}}
{{- define "disasterRecovery.certDnsNames" -}}
  {{- $drdNamespace := .Release.Namespace -}}
  {{- $dnsNames := list "localhost" (printf "%s-disaster-recovery" (include "opensearch.fullname" .)) (printf "%s-disaster-recovery.%s" (include "opensearch.fullname" .) .Release.Namespace) (printf "%s-disaster-recovery.%s.svc.cluster.local" (include "opensearch.fullname" .) .Release.Namespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.global.disasterRecovery.tls.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{/*
IP addresses used to generate TLS certificate with "Subject Alternative Name" field for Disaster Recovery
*/}}
{{- define "disasterRecovery.certIpAddresses" -}}
  {{- $ipAddresses := list "127.0.0.1" -}}
  {{- $ipAddresses = concat $ipAddresses .Values.global.disasterRecovery.tls.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}

{{/*
Generate certificates for Disaster Recovery
*/}}
{{- define "disasterRecovery.generateCerts" -}}
{{- $dnsNames := include "disasterRecovery.certDnsNames" . | fromYamlArray -}}
{{- $ipAddresses := include "disasterRecovery.certIpAddresses" . | fromYamlArray -}}
{{- $duration := default 365 .Values.global.tls.generateCerts.durationDays | int -}}
{{- $ca := genCA "opensearch-drd-ca" $duration -}}
{{- $drdName := "drd" -}}
{{- $cert := genSignedCert $drdName $ipAddresses $dnsNames $duration $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end -}}

{{/*
Protocol for DRD
*/}}
{{- define "disasterRecovery.protocol" -}}
{{- if and .Values.global.tls.enabled .Values.global.disasterRecovery.tls.enabled -}}
  {{- "https" -}}
{{- else -}}
  {{- "http" -}}
{{- end -}}
{{- end -}}

{{/*
Service Account for Site Manager depending on smSecureAuth
*/}}
{{- define "disasterRecovery.siteManagerServiceAccount" -}}
  {{- if .Values.global.disasterRecovery.httpAuth.smServiceAccountName -}}
    {{- .Values.global.disasterRecovery.httpAuth.smServiceAccountName -}}
  {{- else -}}
    {{- if .Values.global.disasterRecovery.httpAuth.smSecureAuth -}}
      {{- "site-manager-sa" -}}
    {{- else -}}
      {{- "sm-auth-sa" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
DRD Port
*/}}
{{- define "disasterRecovery.port" -}}
  {{- if and .Values.global.tls.enabled .Values.global.disasterRecovery.tls.enabled -}}
    {{- "8443" -}}
  {{- else -}}
    {{- "8080" -}}
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
Whether forced cleanup of previous opensearch-status-provisioner job is enabled
*/}}
{{- define "opensearch-status-provisioner.cleanupEnabled" -}}
  {{- if .Values.statusProvisioner.enabled -}}
    {{- $cleanupEnabled := .Values.statusProvisioner.cleanupEnabled | toString }}
    {{- if eq $cleanupEnabled "true" -}}
      {{- printf "true" }}
    {{- else if eq $cleanupEnabled "false" -}}
      {{- printf "false" -}}
    {{- else -}}
      {{- if or (gt .Capabilities.KubeVersion.Major "1") (ge .Capabilities.KubeVersion.Minor "21") -}}
        {{- printf "false" -}}
      {{- else -}}
        {{- printf "true" -}}
      {{- end -}}
    {{- end -}}
  {{- else -}}
    {{- printf "false" -}}
  {{- end -}}
{{- end -}}

{{/*
Opensearch protocol for dbaas adapter
*/}}
{{- define "dbaas-adapter.opensearch-protocol" -}}
{{- if .Values.global.externalOpensearch.enabled }}
  {{- if contains "https" .Values.global.externalOpensearch.url }}
    {{- printf "https" }}
 {{- else }}
    {{- printf "http" }}
 {{- end -}}
{{- else }}
  {{- if eq (include "opensearch.tlsEnabled" .) "true" }}
    {{- "https" -}}
  {{- else }}
    {{- default "http" .Values.dbaasAdapter.opensearchProtocol -}}
  {{- end }}
{{- end -}}
{{- end -}}

{{/*
Whether TLS for OpenSearch curator is enabled
*/}}
{{- define "curator.tlsEnabled" -}}
{{- and .Values.curator.enabled .Values.global.tls.enabled .Values.curator.tls.enabled -}}
{{- end -}}

{{/*
OpenSearch curator Port
*/}}
{{- define "curator.port" -}}
  {{- if and .Values.global.tls.enabled .Values.curator.tls.enabled -}}
    {{- "8443" -}}
  {{- else -}}
    {{- "8080" -}}
  {{- end -}}
{{- end -}}

{{/*
OpenSearch curator Name
*/}}
{{- define "curator.name" -}}
{{ printf "%s-curator" (include "opensearch.fullname" .) }}
{{- end -}}

{{/*
OpenSearch curator Protocol
*/}}
{{- define "curator.protocol" -}}
  {{- if and .Values.global.tls.enabled .Values.curator.tls.enabled -}}
    {{- "https" -}}
  {{- else -}}
    {{- "http" -}}
  {{- end -}}
{{- end -}}

{{/*
OpenSearch curator address
*/}}
{{- define "curator.address" -}}
  {{- printf "%s://%s.%s:%s" (include "curator.protocol" .) (include "curator.name" .) .Release.Namespace (include "curator.port" .) -}}
{{- end -}}

{{/*
Whether curator certificates are Specified
*/}}
{{- define "curator.certificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.curator.tls.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- $filled -}}
{{- end -}}

{{/*
TLS secret name for OpenSearch curator
*/}}
{{- define "curator.certSecretName" -}}
  {{- if and .Values.curator.tls.enabled .Values.global.tls.enabled -}}
    {{- if and (or .Values.global.tls.generateCerts.enabled (eq (include "curator.certificatesSpecified" .) "true")) (not .Values.curator.tls.secretName) -}}
      {{- template "opensearch.fullname" . }}-curator-tls-secret
    {{- else -}}
      {{- required "The TLS secret name should be specified in the 'curator.tls.secretName' parameter when the service is deployed with curator and TLS enabled, but without certificates generation." .Values.curator.tls.secretName -}}
    {{- end -}}
  {{- else -}}
     {{/*
       The empty string is needed for correct prometheus rule configuration in `tls_static_metrics.yaml`
     */}}
    {{- "" -}}
  {{- end -}}
{{- end -}}

{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field for OpenSearch curator
*/}}
{{- define "curator.certDnsNames" -}}
  {{- $dnsNames := list "localhost" (printf "%s-curator" (include "opensearch.fullname" .)) (printf "%s-curator.%s" (include "opensearch.fullname" .) .Release.Namespace) (printf "%s-curator.%s.svc" (include "opensearch.fullname" .) .Release.Namespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.curator.tls.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{/*
IP addresses used to generate TLS certificate with "Subject Alternative Name" field for OpenSearch curator
*/}}
{{- define "curator.certIpAddresses" -}}
  {{- $ipAddresses := list "127.0.0.1" -}}
  {{- $ipAddresses = concat $ipAddresses .Values.curator.tls.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}

{{/*
Generate certificates for OpenSearch curator
*/}}
{{- define "curator.generateCerts" -}}
{{- $dnsNames := include "curator.certDnsNames" . | fromYamlArray -}}
{{- $ipAddresses := include "curator.certIpAddresses" . | fromYamlArray -}}
{{- $duration := default 365 .Values.global.tls.generateCerts.durationDays | int -}}
{{- $ca := genCA "opensearch-curator-ca" $duration -}}
{{- $cert := genSignedCert "curator" $ipAddresses $dnsNames $duration $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end -}}

{{/*
Whether TLS for external OpenSearch is enabled
*/}}
{{- define "external.tlsEnabled" -}}
  {{- and .Values.global.externalOpensearch.enabled (contains "https" .Values.global.externalOpensearch.url) -}}
{{- end -}}

{{/*
External Opensearch host
*/}}
{{- define "external.opensearch-host" -}}
{{- $host := .Values.global.externalOpensearch.url -}}
{{- if contains "https" .Values.global.externalOpensearch.url -}}
  {{- $host = trimPrefix "https://" $host -}}
{{- else -}}
  {{- $host = trimPrefix "http://" $host -}}
{{- end -}}
{{- $host = trimSuffix "/" $host -}}
{{- $host }}
{{- end -}}

{{/*
External Opensearch port
*/}}
{{- define "external.opensearch-port" -}}
{{- $host := .Values.global.externalOpensearch.url -}}
{{- if contains "https" .Values.global.externalOpensearch.url -}}
  {{- 443 -}}
{{- else -}}
  {{- 80 -}}
{{- end -}}
{{- end -}}

{{/*
Whether TLS for DBaaS Adapter is enabled
*/}}
{{- define "dbaas-adapter.tlsEnabled" -}}
  {{- if and .Values.global.tls.enabled .Values.dbaasAdapter.tls.enabled (contains "https" (include "dbaas.registrationUrl" .)) -}}
    {{- printf "true" -}}
  {{- else -}}
    {{- printf "false" -}}
  {{- end -}}
{{- end -}}

{{/*
Whether DBaaS Adapter certificates are Specified
*/}}
{{- define "dbaas-adapter.certificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.dbaasAdapter.tls.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- $filled -}}
{{- end -}}

{{/*
TLS secret name for Disaster Recovery
*/}}
{{- define "dbaas-adapter.tlsSecretName" -}}
  {{- if and .Values.dbaasAdapter.tls.enabled .Values.global.tls.enabled -}}
    {{- if and (or .Values.global.tls.generateCerts.enabled (eq (include "dbaas-adapter.certificatesSpecified" .) "true")) (not .Values.dbaasAdapter.tls.secretName) -}}
      {{- template "dbaas-adapter.name" . }}-tls-secret
    {{- else -}}
      {{- required "The TLS secret name should be specified in the 'dbaasAdapter.tls.secretName' parameter when the service is deployed with dbaasAdapter and TLS enabled, but without certificates generation." .Values.dbaasAdapter.tls.secretName -}}
    {{- end -}}
  {{- else -}}
     {{/*
       The empty string is needed for correct prometheus rule configuration in `tls_static_metrics.yaml`
     */}}
    {{- "" -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS Adapter protocol
*/}}
{{- define "dbaas-adapter.protocol" -}}
  {{- if eq (include "dbaas-adapter.tlsEnabled" .) "true" -}}
    {{- printf "https" -}}
  {{- else -}}
    {{- printf "http" -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS Adapter port
*/}}
{{- define "dbaas-adapter.port" -}}
  {{- if eq (include "dbaas-adapter.tlsEnabled" .) "true" -}}
    {{- printf "8443" -}}
  {{- else -}}
    {{- printf "8080" -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS Adapter address
*/}}
{{- define "dbaas-adapter.address" -}}
  {{- printf "%s://%s.%s:%s" (include "dbaas-adapter.protocol" .) (include "dbaas-adapter.name" .) .Release.Namespace (include "dbaas-adapter.port" .) -}}
{{- end -}}

{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field for OpenSearch DBaaS Addapter
*/}}
{{- define "dbaas-adapter.certDnsNames" -}}
  {{- $dnsNames := list "localhost" (include "dbaas-adapter.name" .) (printf "%s.%s" (include "dbaas-adapter.name" .) .Release.Namespace) (printf "%s.%s.svc" (include "dbaas-adapter.name" .) .Release.Namespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.dbaasAdapter.tls.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{/*
IP addresses used to generate TLS certificate with "Subject Alternative Name" field for OpenSearch DBaaS Addapter
*/}}
{{- define "dbaas-adapter.certIpAddresses" -}}
  {{- $ipAddresses := list "127.0.0.1" -}}
  {{- $ipAddresses = concat $ipAddresses .Values.dbaasAdapter.tls.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}

{{/*
Generate certificates for OpenSearch DBaaS Addapter
*/}}
{{- define "dbaas-adapter.generateCerts" -}}
{{- $dnsNames := include "dbaas-adapter.certDnsNames" . | fromYamlArray -}}
{{- $ipAddresses := include "dbaas-adapter.certIpAddresses" . | fromYamlArray -}}
{{- $duration := default 365 .Values.global.tls.generateCerts.durationDays | int -}}
{{- $ca := genCA "opensearch-dbaas-adapter-ca" $duration -}}
{{- $cert := genSignedCert "dbaas-adapter" $ipAddresses $dnsNames $duration $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end -}}
{{/*
Calculates resources that should be monitored during deployment by Deployment Status Provisioner.
*/}}
{{- define "opensearch.monitoredResources" -}}
    {{- $resources := list (printf "Deployment %s-service-operator" (include "opensearch.fullname" .)) }}
    {{- if and (not .Values.global.externalOpensearch.enabled) .Values.dashboards.enabled }}
    {{- $resources = append $resources (printf "Deployment %s-dashboards" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- if not (or (eq .Values.global.disasterRecovery.mode "standby") (eq .Values.global.disasterRecovery.mode "disabled")) -}}
    {{- if .Values.curator.enabled }}
    {{- $resources = append $resources (printf "Deployment %s-curator" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- if eq (include "dbaas.enabled" .) "true" }}
    {{- $resources = append $resources (printf "Deployment dbaas-%s-adapter" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- if .Values.integrationTests.enabled }}
    {{- $resources = append $resources (printf "Deployment %s-integration-tests" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- end }}
    {{- if (eq (include "monitoring.enabled" .) "true") }}
    {{- $resources = append $resources (printf "Deployment %s-monitoring" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- if not .Values.global.externalOpensearch.enabled -}}
    {{- if eq (include "joint-mode" .) "true" }}
    {{- $resources = append $resources (printf "StatefulSet %s" (include "opensearch.fullname" .)) -}}
    {{- else }}
    {{- if and .Values.opensearch.data.enabled .Values.opensearch.data.dedicatedPod.enabled }}
    {{- $resources = append $resources (printf "StatefulSet %s-data" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- $resources = append $resources (printf "StatefulSet %s-master" (include "opensearch.fullname" .)) -}}
    {{- if and .Values.opensearch.client.enabled .Values.opensearch.client.dedicatedPod.enabled }}
    {{- $resources = append $resources (printf "Deployment %s-client" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- end }}
    {{- if .Values.opensearch.arbiter.enabled }}
    {{- $resources = append $resources (printf "StatefulSet %s-arbiter" (include "opensearch.fullname" .)) -}}
    {{- end }}
    {{- end }}
    {{- join ", " $resources }}
{{- end -}}

{{/*
Find a busybox image in various places.
*/}}
{{- define "busybox.image" -}}
    {{- printf "%s" .Values.opensearch.initContainer.dockerImage -}}
{{- end -}}

{{/*
Find a kubectl image in various places.
*/}}
{{- define "kubectl.image" -}}
    {{- printf "%s" .Values.podScheduler.dockerImage -}}
{{- end -}}

{{/*
Find an OpenSearch Dashboards image in various places.
*/}}
{{- define "dashboards.image" -}}
    {{- printf "%s" .Values.dashboards.dockerImage -}}
{{- end -}}

{{/*
Find an OpenSearch image in various places.
*/}}
{{- define "opensearch.image" -}}
    {{- printf "%s" .Values.opensearch.dockerImage -}}
{{- end -}}

{{/*
Find an OpenSearch image in various places.
*/}}
{{- define "tls-init.image" -}}
    {{- printf "%s" .Values.opensearch.dockerTlsInitImage -}}
{{- end -}}

{{/*
Find an OpenSearch monitoring image in various places.
*/}}
{{- define "monitoring.image" -}}
    {{- printf "%s" .Values.monitoring.dockerImage -}}
{{- end -}}

{{/*
Find a DBaaS OpenSearch adapter image in various places.
*/}}
{{- define "dbaas-adapter.image" -}}
    {{- printf "%s" .Values.dbaasAdapter.dockerImage -}}
{{- end -}}

{{/*
Find an OpenSearch curator image in various places.
*/}}
{{- define "curator.image" -}}
    {{- printf "%s" .Values.curator.dockerImage -}}
{{- end -}}

{{/*
Find an OpenSearch indices cleaner image in various places.
*/}}
{{- define "indices-cleaner.image" -}}
    {{- printf "%s" .Values.curator.dockerIndicesCleanerImage -}}
{{- end -}}

{{/*
Find an OpenSearch operator image in various places.
*/}}
{{- define "operator.image" -}}
    {{- printf "%s" .Values.operator.dockerImage -}}
{{- end -}}

{{/*
Find an OpenSearch integration tests image in various places.
*/}}
{{- define "integration-tests.image" -}}
    {{- printf "%s" .Values.integrationTests.dockerImage -}}
{{- end -}}

{{- define "disasterRecovery.image" -}}
    {{- printf "%s" .Values.global.disasterRecovery.image -}}
{{- end -}}

{{/*
Find a Deployment Status Provisioner image in various places.
*/}}
{{- define "deployment-status-provisioner.image" -}}
    {{- printf "%s" .Values.statusProvisioner.dockerImage -}}
{{- end -}}

{{/*
Configure pod annotation for Velero pre-hook backup
*/}}
{{- define "opensearch.velero-pre-hook-backup-flush" -}}
  {{- if eq (include "opensearch.tlsEnabled" .) "true" }}
    {{- printf "'[\"/bin/sh\", \"-c\", \"curl -u ${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD} ${OPENSEARCH_PROTOCOL:-https}://${OPENSEARCH_NAME}:9200/_flush --cacert /certs/crt.pem\"]'" }}
  {{- else }}
    {{- printf "'[\"/bin/sh\", \"-c\", \"curl -u ${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD} ${OPENSEARCH_PROTOCOL:-http}://${OPENSEARCH_NAME}:9200/_flush\"]'" }}
  {{- end }}
{{- end -}}

{{/*
Configure OpenSearch statefulset and deployment names in disaster recovery health check format.
*/}}
{{- define "opensearch.nodeNames" -}}
    {{- $lst := list }}
    {{- if .Values.opensearch.arbiter.enabled }}
        {{- $lst = append $lst (printf "%s %s-%s" "statefulset" (include "opensearch.fullname" . ) "arbiter") }}
    {{- end }}
    {{ if and .Values.opensearch.data.enabled .Values.opensearch.data.dedicatedPod.enabled }}
        {{- $lst = append $lst (printf "%s %s-%s" "statefulset" (include "opensearch.fullname" . ) "data") }}
    {{- end }}
    {{- if .Values.opensearch.master.enabled }}
        {{- $lst = append $lst (printf "%s %s" "statefulset" (include "master-nodes" . )) }}
    {{- end }}
    {{- if and .Values.opensearch.client.enabled .Values.opensearch.client.dedicatedPod.enabled }}
        {{- $lst = append $lst (printf "%s %s-%s" "deployment" (include "opensearch.fullname" . ) "client") }}
    {{- end }}
    {{- join "," $lst }}
{{- end -}}

{{/*
TLS Static Metric secret template
Arguments:
Dictionary with:
* "namespace" is a namespace of application
* "application" is name of application
* "service" is a name of service
* "enableTls" is tls enabled for service
* "secret" is a name of tls secret for service
* "certProvider" is a type of tls certificates provider
* "certificate" is a name of CertManager's Certificate resource for service
Usage example:
{{template "global.tlsStaticMetric" (dict "namespace" .Release.Namespace "application" .Chart.Name "service" .global.name "enableTls" (include "global.enableTls" .) "secret" (include "global.tlsSecretName" .) "certProvider" (include "services.certProvider" .) "certificate" (printf "%s-tls-certificate" (include "global.name")) }}
*/}}
{{- define "global.tlsStaticMetric" -}}
- expr: {{ ternary "1" "0" (eq .enableTls "true") }}
  labels:
    namespace: "{{ .namespace }}"
    application: "{{ .application }}"
    service: "{{ .service }}"
    {{ if eq .enableTls "true" }}
    secret: "{{ .secret }}"
    {{ if eq .certProvider "cert-manager" }}
    certificate: "{{ .certificate }}"
    {{ end }}
    {{ end }}
  record: service:tls_status:info
{{- end -}}

{{- define "opensearch-service.globalPodSecurityContext" -}}
runAsNonRoot: true
seccompProfile:
  type: "RuntimeDefault"
{{- with .Values.global.securityContext }}
{{ toYaml . }}
{{- end -}}
{{- end -}}

{{- define "opensearch-service.globalContainerSecurityContext" -}}
allowPrivilegeEscalation: false
capabilities:
  drop: ["ALL"]
{{- end -}}

{{- define "opensearch-gke-service-name" -}}
{{- printf "%s-%s" (include "opensearch.fullname" .) (index .Values.global.disasterRecovery.serviceExport.region) -}}
{{- end -}}

{{- define "external.useTlsSecret" -}}
{{ and (eq (include "external.tlsEnabled" .) "true") (ne .Values.global.externalOpensearch.tlsSecretName "") }}
{{- end -}}

{{/*
Determines whether OpenSearch data StatefulSet be used or not
*/}}
{{- define "opensearch.useDataNodes" -}}
{{ and (not .Values.global.externalOpensearch.enabled) .Values.opensearch.data.enabled .Values.opensearch.data.dedicatedPod.enabled }}
{{- end -}}

{{/*
Determines update strategy for OpenSearch master nodes
*/}}
{{- define "master.updateStrategy" -}}
{{- if .Values.opensearch.rollingUpdate -}}
  {{- "OnDelete" -}}
{{- else -}}
  {{- .Values.opensearch.master.updateStrategy | default "RollingUpdate" -}}
{{- end -}}
{{- end -}}

{{/*
Determines update strategy for OpenSearch data nodes
*/}}
{{- define "data.updateStrategy" -}}
{{- if .Values.opensearch.rollingUpdate -}}
    {{- "OnDelete" -}}
{{- else -}}
  {{- .Values.opensearch.data.updateStrategy | default "RollingUpdate" -}}
{{- end -}}
{{- end -}}

{{/*
Determines update strategy for OpenSearch arbiter nodes
*/}}
{{- define "arbiter.updateStrategy" -}}
{{- if .Values.opensearch.rollingUpdate -}}
    {{- "OnDelete" -}}
{{- else -}}
  {{- .Values.opensearch.arbiter.updateStrategy | default "RollingUpdate" -}}
{{- end -}}
{{- end -}}

{{/*
Configure OpenSearch statefulset names for rolling update mechanism in operator.
*/}}
{{- define "opensearch.statefulsetNames" -}}
    {{- $lst := list }}
    {{ if and .Values.opensearch.data.enabled .Values.opensearch.data.dedicatedPod.enabled }}
        {{- $lst = append $lst (printf "%s-data" (include "opensearch.fullname" . )) }}
    {{- end }}
    {{- if .Values.opensearch.master.enabled }}
        {{- $lst = append $lst (include "master-nodes" .) }}
    {{- end }}
    {{- if .Values.opensearch.arbiter.enabled }}
        {{- $lst = append $lst (printf "%s-arbiter" (include "opensearch.fullname" . )) }}
    {{- end }}
    {{- join "," $lst }}
{{- end -}}

{{/*
Master storage class from various places.
*/}}
{{- define "opensearch.master.storageClassName" -}}
  {{- if and (ne (.Values.STORAGE_RWO_CLASS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.STORAGE_RWO_CLASS -}}
  {{- else -}}
    {{- .Values.opensearch.master.persistence.storageClass -}}
  {{- end -}}
{{- end -}}

{{/*
Data storage class from various places.
*/}}
{{- define "opensearch.data.storageClassName" -}}
  {{- if and (ne (.Values.STORAGE_RWO_CLASS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.STORAGE_RWO_CLASS -}}
  {{- else -}}
    {{- .Values.opensearch.data.persistence.storageClass -}}
  {{- end -}}
{{- end -}}

{{/*
Arbiter storage class from various places.
*/}}
{{- define "opensearch.arbiter.storageClassName" -}}
  {{- if and (ne (.Values.STORAGE_RWO_CLASS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.STORAGE_RWO_CLASS -}}
  {{- else -}}
    {{- .Values.opensearch.arbiter.persistence.storageClass -}}
  {{- end -}}
{{- end -}}

{{/*
Snapshot storage class from various places.
*/}}
{{- define "opensearch.snapshot.storageClassName" -}}
  {{- if and (ne (.Values.STORAGE_RWX_CLASS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.STORAGE_RWX_CLASS -}}
  {{- else -}}
    {{- .Values.opensearch.snapshots.storageClass -}}
  {{- end -}}
{{- end -}}

{{/*
Master replicas from various places.
*/}}
{{- define "opensearch.master.replicas" -}}
  {{- if and (ne (.Values.INFRA_OPENSEARCH_REPLICAS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_OPENSEARCH_REPLICAS }}
  {{- else -}}
    {{- .Values.opensearch.master.replicas -}}
  {{- end -}}
{{- end -}}

{{/*
Arbiter replicas from various places.
*/}}
{{- define "opensearch.data.replicas" -}}
  {{- if and (ne (.Values.INFRA_OPENSEARCH_REPLICAS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_OPENSEARCH_REPLICAS -}}
  {{- else -}}
    {{- .Values.opensearch.data.replicas -}}
  {{- end -}}
{{- end -}}

{{/*
Data replicas from various places.
*/}}
{{- define "opensearch.arbiter.replicas" -}}
  {{- if and (ne (.Values.INFRA_OPENSEARCH_REPLICAS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_OPENSEARCH_REPLICAS -}}
  {{- else -}}
    {{- .Values.opensearch.arbiter.replicas -}}
  {{- end -}}
{{- end -}}

{{/*
Client replicas from various places.
*/}}
{{- define "opensearch.client.replicas" -}}
  {{- if and (ne (.Values.INFRA_OPENSEARCH_REPLICAS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_OPENSEARCH_REPLICAS -}}
  {{- else -}}
    {{- .Values.opensearch.client.replicas -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS Enabled from various places.
*/}}
{{- define "dbaas.enabled" -}}
  {{- if and (ne (.Values.DBAAS_ENABLED | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.DBAAS_ENABLED -}}
  {{- else -}}
    {{- .Values.dbaasAdapter.enabled -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS registration URL from various places.
*/}}
{{- define "dbaas.registrationUrl" -}}
  {{- if and (ne (.Values.API_DBAAS_ADDRESS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.API_DBAAS_ADDRESS -}}
  {{- else -}}
    {{- .Values.dbaasAdapter.dbaasAggregatorRegistrationAddress -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS registration username from various places.
*/}}
{{- define "dbaas.registrationUsername" -}}
  {{- if and (ne (.Values.DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME -}}
  {{- else -}}
    {{- .Values.dbaasAdapter.registrationAuthUsername -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS registration password from various places.
*/}}
{{- define "dbaas.registrationPassword" -}}
  {{- if and (ne (.Values.DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD -}}
  {{- else -}}
    {{- .Values.dbaasAdapter.registrationAuthPassword -}}
  {{- end -}}
{{- end -}}

{{/*
OpenSearch admin username from various places.
*/}}
{{- define "opensearch.username" -}}
  {{- if and (ne (.Values.INFRA_OPENSEARCH_USERNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_OPENSEARCH_USERNAME -}}
  {{- else -}}
    {{- .Values.opensearch.securityConfig.authc.basic.username -}}
  {{- end -}}
{{- end -}}

{{/*
DBaaS registration password from various places.
*/}}
{{- define "opensearch.password" -}}
  {{- if and (ne (.Values.INFRA_OPENSEARCH_PASSWORD | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_OPENSEARCH_PASSWORD -}}
  {{- else -}}
    {{- .Values.opensearch.securityConfig.authc.basic.password -}}
  {{- end -}}
{{- end -}}

{{/*
Monitoring installation required
*/}}
{{- define "monitoring.enabled" -}}
  {{- if and (ne (.Values.MONITORING_ENABLED | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.MONITORING_ENABLED }}
  {{- else -}}
    {{- .Values.monitoring.enabled -}}
  {{- end -}}
{{- end -}}

{{/*
Whether ingress for OpenSearch enabled
*/}}
{{- define "opensearch.ingressEnabled" -}}
  {{- if and (ne (.Values.PRODUCTION_MODE | toString) "<nil>") .Values.global.cloudIntegrationEnabled}}
    {{- (eq .Values.PRODUCTION_MODE false) }}
  {{- else -}}
    {{- .Values.opensearch.client.ingress.enabled }}
  {{- end -}}
{{- end -}}

{{/*
Ingress host for OpenSearch
*/}}
{{- define "opensearch.ingressHost" -}}
  {{- if .Values.opensearch.client.ingress.hosts }}
    {{- .Values.opensearch.client.ingress.hosts }}
  {{- else -}}
    {{- if and (ne (.Values.SERVER_HOSTNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled }}
      {{- printf "opensearch-%s.%s" .Release.Namespace .Values.SERVER_HOSTNAME | toStrings }}
    {{- end -}}
  {{- end -}}
{{- end -}}


{{- define "opensearch-service.monitoredImages" -}}
  {{- printf "deployment %s-service-operator %s-service-operator %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "operator.image" . ) -}}
  {{- if and (not .Values.global.externalOpensearch.enabled) .Values.opensearch.master.enabled -}}
    {{- printf "statefulset %s opensearch %s, " ( include "master-nodes" . ) (include "opensearch.image" . ) -}}
  {{- end -}}
  {{- if .Values.curator.enabled -}}
    {{- printf "deployment %s-curator %s-curator %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "curator.image" . ) -}}
    {{- printf "deployment %s-curator %s-indices-cleaner %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "indices-cleaner.image" . ) -}}
  {{- end -}}
  {{- if .Values.dashboards.enabled -}}
    {{- printf "deployment %s-dashboards %s-dashboards %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "dashboards.image" . ) -}}
  {{- end -}}
  {{- if .Values.monitoring.enabled -}}
    {{- printf "deployment %s-monitoring %s-monitoring %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "monitoring.image" . ) -}}
  {{- end -}}
  {{- if .Values.dbaasAdapter.enabled -}}
    {{- printf "deployment %s %s %s, " (include "dbaas-adapter.name" .) (include "dbaas-adapter.name" .) (include "dbaas-adapter.image" . ) -}}
  {{- end -}}
  {{- if .Values.integrationTests.enabled -}}
    {{- printf "deployment %s-integration-tests %s-integration-tests %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "integration-tests.image" . ) -}}
  {{- end -}}
  {{- if (eq (include "opensearch.enableDisasterRecovery" .) "true") -}}
    {{- printf "deployment %s-service-operator %s-disaster-recovery %s, " (include "opensearch.fullname" .) (include "opensearch.fullname" .) (include "disasterRecovery.image" . ) -}}
  {{- end -}}
{{- end -}}

{{- define "monitoring.lagAlertThresholdDefined" -}}
  {{- if .Values.monitoring.thresholds.lagAlert }}
    {{- gt (int .Values.monitoring.thresholds.lagAlert) -1 }}
  {{- else -}}
    {{- "false" }}
  {{- end -}}
{{- end -}}

{{- define "opensearch.verifyExistingCertSecretCertSubPath" -}}
  {{- $correctPath := "tls.crt" -}}
  {{- $transport := (.Values.opensearch.tls.transport.existingCertSecretCertSubPath | toString) -}}
  {{- $rest :=      (.Values.opensearch.tls.rest.existingCertSecretCertSubPath      | toString) -}}
  {{- $admin :=     (.Values.opensearch.tls.admin.existingCertSecretCertSubPath     | toString) -}}
  {{- $transportCorrect :=  or (eq $transport "") (eq ($transport | toString) "<nil>") (eq $transport $correctPath) -}}
  {{- $restCorrect :=       or (eq $rest "")      (eq ($rest | toString) "<nil>")      (eq $rest $correctPath) -}}
  {{- $adminCorrect :=      or (eq $admin "")     (eq ($admin | toString) "<nil>")     (eq $admin $correctPath) -}}
  {{- if not (and $transportCorrect $restCorrect $adminCorrect) }}
    {{- fail "Overriden opensearch.tls.*.existingCertSecretCertSubPath parameters are not supported" -}}
  {{- end -}}
{{- end -}}

{{- define "opensearch.verifyExistingCertSecretKeySubPath" -}}
  {{- $correctPath := "tls.key" -}}
  {{- $transport := (.Values.opensearch.tls.transport.existingCertSecretKeySubPath | toString) -}}
  {{- $rest :=      (.Values.opensearch.tls.rest.existingCertSecretKeySubPath      | toString) -}}
  {{- $admin :=     (.Values.opensearch.tls.admin.existingCertSecretKeySubPath     | toString) -}}
  {{- $transportCorrect :=  or (eq $transport "") (eq $transport "<nil>") (eq $transport $correctPath) -}}
  {{- $restCorrect :=       or (eq $rest "")      (eq $rest "<nil>")      (eq $rest $correctPath) -}}
  {{- $adminCorrect :=      or (eq $admin "")     (eq $admin "<nil>")     (eq $admin $correctPath) -}}
  {{- if not (and $transportCorrect $restCorrect $adminCorrect) }}
    {{- fail "Overriden opensearch.tls.*.existingCertSecretKeySubPath parameters are not supported" -}}
  {{- end -}}
{{- end -}}

{{- define "opensearch.verifyExistingCertSecretRootCASubPath" -}}
  {{- $correctPath := "ca.crt" -}}
  {{- $transport := (.Values.opensearch.tls.transport.existingCertSecretRootCASubPath | toString) -}}
  {{- $rest :=      (.Values.opensearch.tls.rest.existingCertSecretRootCASubPath      | toString) -}}
  {{- $admin :=     (.Values.opensearch.tls.admin.existingCertSecretRootCASubPath     | toString) -}}
  {{- $transportCorrect :=  or (eq $transport "") (eq $transport "<nil>") (eq $transport $correctPath) -}}
  {{- $restCorrect :=       or (eq $rest "")      (eq $rest "<nil>")      (eq $rest $correctPath) -}}
  {{- $adminCorrect :=      or (eq $admin "")     (eq $admin "<nil>")     (eq $admin $correctPath) -}}
  {{- if not (and $transportCorrect $restCorrect $adminCorrect) }}
    {{- fail "Overriden opensearch.tls.*.existingCertSecretRootCASubPath parameters are not supported" -}}
  {{- end -}}
{{- end -}}

{{/*
Curator SSL secret name
*/}}
{{- define "curator.s3.tlsSecretName" -}}
  {{- if .Values.opensearch.snapshots.s3.sslCert -}}
    {{- if .Values.opensearch.snapshots.s3.sslSecretName -}}
      {{- .Values.opensearch.snapshots.s3.sslSecretName -}}
    {{- else -}}
      {{- printf "curator-s3-tls-secret" -}}
    {{- end -}}
  {{- else -}}
    {{- if .Values.opensearch.snapshots.s3.sslSecretName -}}
      {{- .Values.opensearch.snapshots.s3.sslSecretName -}}
    {{- else -}}
      {{- printf "" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Restricted environment.
*/}}
{{- define "opensearch.restrictedEnvironment" -}}
  {{- if and (ne (.Values.INFRA_RESTRICTED_ENVIRONMENT | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_RESTRICTED_ENVIRONMENT }}
  {{- else -}}
    {{- .Values.global.restrictedEnvironment -}}
  {{- end -}}
{{- end -}}