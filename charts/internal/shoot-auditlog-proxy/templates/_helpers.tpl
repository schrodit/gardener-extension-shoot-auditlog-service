{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "auditlog-proxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "auditlog-proxy.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "auditlog-proxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "auditlogconfig" -}}
---
apiVersion: proxy.shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1
kind: Configuration

provider: {{ .Values.configuration.provider }}
providerConfig: {{ toJson .Values.configuration.providerConfig }}
webhookConfiguration:
  httpsPort: {{ .Values.configuration.serverPortHttps }}
  httpPort: {{ .Values.configuration.serverPortHttp }}

  tls:
    certFile: /etc/auditlog-proxy/tls/tls.crt
    keyFile: /etc/auditlog-proxy/tls/tls.key
{{- end }}
