{{- define "name" -}}
gardener-extension-shoot-auditlog-service
{{- end -}}

{{- define "auditlogconfig" -}}
---
apiVersion: shoot-auditlog-service.extensions.config.gardener.cloud/v1alpha1
kind: Configuration
{{- end }}

{{-  define "image" -}}
  {{- if hasPrefix "sha256:" .Values.image.tag }}
  {{- printf "%s@%s" .Values.image.repository .Values.image.tag }}
  {{- else }}
  {{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
  {{- end }}
{{- end }}
