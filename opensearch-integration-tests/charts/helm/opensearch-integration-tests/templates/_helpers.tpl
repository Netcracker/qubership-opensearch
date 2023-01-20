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

