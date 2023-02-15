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
{{- if .Values.global.externalOpensearch.enabled }}
  {{- .Values.global.externalOpensearch.dataNodesCount }}
{{- else }}
  {{- if .Values.opensearch.data.dedicatedPod.enabled }}
    {{- .Values.opensearch.data.replicas }}
  {{- else }}
    {{- .Values.opensearch.master.replicas }}
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
Whether TLS for OpenSearch is enabled
*/}}
{{- define "opensearch.tlsEnabled" -}}
  {{- and (not .Values.global.externalOpensearch.enabled) .Values.global.tls.enabled .Values.opensearch.tls.enabled -}}
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
Define the path to the transport certificate in secret.
*/}}
{{- define "opensearch.transport-cert-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "tls.crt" "transport-crt.pem" }}
{{- else }}
  {{- .Values.opensearch.tls.transport.existingCertSecretCertSubPath -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the transport private key in secret.
*/}}
{{- define "opensearch.transport-key-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "tls.key" "transport-key.pem" }}
{{- else }}
  {{- .Values.opensearch.tls.transport.existingCertSecretKeySubPath -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the transport root CA in secret.
*/}}
{{- define "opensearch.transport-root-ca-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "ca.crt" "transport-root-ca.pem" }}
{{- else -}}
  {{- .Values.opensearch.tls.transport.existingCertSecretRootCASubPath -}}
{{- end -}}
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
Define the path to the admin certificate in secret.
*/}}
{{- define "opensearch.admin-cert-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "tls.crt" "admin-crt.pem" }}
{{- else -}}
  {{- .Values.opensearch.tls.admin.existingCertSecretCertSubPath -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the admin private key in secret.
*/}}
{{- define "opensearch.admin-key-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "tls.key" "admin-key.pem" }}
{{- else -}}
  {{- .Values.opensearch.tls.admin.existingCertSecretKeySubPath -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the admin root CA in secret.
*/}}
{{- define "opensearch.admin-root-ca-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "ca.crt" "admin-root-ca.pem" }}
{{- else -}}
  {{- .Values.opensearch.tls.admin.existingCertSecretRootCASubPath -}}
{{- end -}}
{{- end -}}

{{/*
Define the name of the REST certificates secret.
*/}}
{{- define "opensearch.rest-cert-secret-name" -}}
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

{{/*
Define the path to the REST certificate in secret.
*/}}
{{- define "opensearch.rest-cert-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "tls.crt" "rest-crt.pem" }}
{{- else -}}
  {{- .Values.opensearch.tls.rest.existingCertSecretCertSubPath -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the REST private key in secret.
*/}}
{{- define "opensearch.rest-key-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "tls.key" "rest-key.pem" }}
{{- else -}}
  {{- .Values.opensearch.tls.rest.existingCertSecretKeySubPath -}}
{{- end -}}
{{- end -}}

{{/*
Define the path to the REST root CA in secret.
*/}}
{{- define "opensearch.rest-root-ca-path" -}}
{{- if .Values.global.tls.generateCerts.enabled }}
  {{- eq (include "certProvider" .) "cert-manager" | ternary "ca.crt" "rest-root-ca.pem" }}
{{- else -}}
  {{- .Values.opensearch.tls.rest.existingCertSecretRootCASubPath -}}
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
TLS secret name for Disaster Recovery
*/}}
{{- define "disasterRecovery.certSecretName" -}}
{{- if and (not .Values.global.tls.generateCerts.enabled) .Values.global.disasterRecovery.tls.secretName }}
  {{- .Values.global.disasterRecovery.tls.secretName -}}
{{- else }}
  {{- template "opensearch.fullname" . -}}-drd-tls-secret
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
Elastic protocol for dbaas adapter
*/}}
{{- define "dbaas-adapter.elasticsearch-protocol" -}}
{{- if .Values.global.externalOpensearch.enabled }}
  {{- if contains "https" .Values.global.externalOpensearch.url }}
    {{- printf "https" }}
 {{- else }}
    {{- printf "http" }}
 {{- end -}}
{{- else }}
    {{- default "http" .Values.elasticsearchDbaasAdapter.opensearchProtocol }}
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
TLS secret name for OpenSearch curator
*/}}
{{- define "curator.certSecretName" -}}
{{- if and (not .Values.global.tls.generateCerts.enabled) .Values.curator.tls.secretName }}
  {{- .Values.curator.tls.secretName -}}
{{- else }}
  {{- template "opensearch.fullname" . }}-curator-tls-secret
{{- end -}}
{{- end }}

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
  {{- if and .Values.global.tls.enabled .Values.dbaasAdapter.tls.enabled -}}
    {{- printf "true" -}}
  {{- else -}}
    {{- printf "false" -}}
  {{- end -}}
{{- end -}}

{{/*
TLS secret name for OpenSearch DBaaS Adapter
*/}}
{{- define "dbaas-adapter.tlsSecretName" -}}
  {{- if and (not .Values.global.tls.generateCerts.enabled) .Values.dbaasAdapter.tls.secretName -}}
    {{- .Values.dbaasAdapter.tls.secretName -}}
  {{- else }}
    {{- template "dbaas-adapter.name" . }}-tls-secret
  {{- end -}}
{{- end }}

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
    {{- printf "Deployment %s-service-operator, " (include "opensearch.fullname" .) -}}
    {{- if and (not .Values.global.externalOpensearch.enabled) .Values.dashboards.enabled }}
    {{- printf "Deployment %s-dashboards, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- if not (or (eq .Values.global.disasterRecovery.mode "standby") (eq .Values.global.disasterRecovery.mode "disabled")) -}}
    {{- if .Values.curator.enabled }}
    {{- printf "Deployment %s-curator, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- if .Values.dbaasAdapter.enabled }}
    {{- printf "Deployment dbaas-%s-adapter, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- if .Values.elasticsearchDbaasAdapter.enabled }}
    {{- printf "Deployment %s, " .Values.elasticsearchDbaasAdapter.name -}}
    {{- end }}
    {{- if .Values.integrationTests.enabled }}
    {{- printf "Deployment %s-integration-tests, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- end }}
    {{- if .Values.monitoring.enabled }}
    {{- printf "Deployment %s-monitoring, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- if not .Values.global.externalOpensearch.enabled -}}
    {{- if eq (include "joint-mode" .) "true" }}
    {{- printf "StatefulSet %s, " (include "opensearch.fullname" .) -}}
    {{- else }}
    {{- printf "StatefulSet %s-master, " (include "opensearch.fullname" .) -}}
    {{- if and .Values.opensearch.data.enabled .Values.opensearch.data.dedicatedPod }}
    {{- printf "StatefulSet %s-data, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- if .Values.opensearch.arbiter.enabled }}
    {{- printf "StatefulSet %s-data, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- if and .Values.opensearch.client.enabled .Values.opensearch.client.dedicatedPod }}
    {{- printf "Deployment %s-client, " (include "opensearch.fullname" .) -}}
    {{- end }}
    {{- end }}
    {{ end }}
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
Find a DBaaS Elasticsearch adapter image in various places.
*/}}
{{- define "elasticsearch-dbaas-adapter.image" -}}
  {{- if .Values.deployDescriptor -}}
    {{- if .Values.elasticsearchDbaasAdapterImage -}}
      {{- printf "%s" .Values.elasticsearchDbaasAdapterImage -}}
    {{- else -}}
      {{- printf "%s" (index .Values.deployDescriptor.elasticsearchDbaasAdapterImage.image) -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s" .Values.elasticsearchDbaasAdapter.dockerImage -}}
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
Configure Open Distro Elasticsearch statefulset and deployment names in disaster recovery health check format.
*/}}
{{- define "opensearch.statefulsetNames" -}}
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