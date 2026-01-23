{{/*
Expand the name of the chart.
*/}}
{{- define "state-metrics.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "state-metrics.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "state-metrics.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "state-metrics.labels" -}}
helm.sh/chart: {{ include "state-metrics.chart" . }}
{{ include "state-metrics.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "state-metrics.selectorLabels" -}}
app.kubernetes.io/name: {{ include "state-metrics.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "state-metrics.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "state-metrics.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "state-metrics.image" -}}
{{- .Values.image }}
{{- end }}

{{/*
Collector metadata: defines which collectors require leader election
*/}}
{{- define "state-metrics.collectorMetadata" -}}
{{- $metadata := dict }}
{{- $_ := set $metadata "domain" (dict "requiresLeaderElection" true "deploymentMode" "deployment") }}
{{- $_ := set $metadata "node" (dict "requiresLeaderElection" true "deploymentMode" "deployment") }}
{{- $_ := set $metadata "pod" (dict "requiresLeaderElection" true "deploymentMode" "deployment") }}
{{- $_ := set $metadata "imagepull" (dict "requiresLeaderElection" true "deploymentMode" "deployment") }}
{{- $_ := set $metadata "zombie" (dict "requiresLeaderElection" true "deploymentMode" "deployment") }}
{{- $_ := set $metadata "cloudbalance" (dict "requiresLeaderElection" true "deploymentMode" "deployment") }}
{{- $_ := set $metadata "lvm" (dict "requiresLeaderElection" false "deploymentMode" "daemonset") }}
{{- $metadata | toJson }}
{{- end }}

{{/*
Get collectors that require Deployment (leader election)
*/}}
{{- define "state-metrics.deploymentCollectors" -}}
{{- $metadata := include "state-metrics.collectorMetadata" . | fromJson }}
{{- $deploymentCollectors := list }}
{{- range .Values.config.enabledCollectors }}
  {{- $collectorMeta := index $metadata . }}
  {{- if $collectorMeta }}
    {{- if eq (index $collectorMeta "deploymentMode") "deployment" }}
      {{- $deploymentCollectors = append $deploymentCollectors . }}
    {{- end }}
  {{- end }}
{{- end }}
{{- $deploymentCollectors | toJson }}
{{- end }}

{{/*
Get collectors that require DaemonSet (no leader election)
*/}}
{{- define "state-metrics.daemonsetCollectors" -}}
{{- $metadata := include "state-metrics.collectorMetadata" . | fromJson }}
{{- $daemonsetCollectors := list }}
{{- range .Values.config.enabledCollectors }}
  {{- $collectorMeta := index $metadata . }}
  {{- if $collectorMeta }}
    {{- if eq (index $collectorMeta "deploymentMode") "daemonset" }}
      {{- $daemonsetCollectors = append $daemonsetCollectors . }}
    {{- end }}
  {{- end }}
{{- end }}
{{- $daemonsetCollectors | toJson }}
{{- end }}

{{/*
Check if Deployment should be created
*/}}
{{- define "state-metrics.deploymentEnabled" -}}
{{- $deploymentCollectors := include "state-metrics.deploymentCollectors" . | fromJson }}
{{- if gt (len $deploymentCollectors) 0 }}
{{- "true" }}
{{- else }}
{{- "false" }}
{{- end }}
{{- end }}

{{/*
Check if DaemonSet should be created
*/}}
{{- define "state-metrics.daemonsetEnabled" -}}
{{- $daemonsetCollectors := include "state-metrics.daemonsetCollectors" . | fromJson }}
{{- if gt (len $daemonsetCollectors) 0 }}
{{- "true" }}
{{- else }}
{{- "false" }}
{{- end }}
{{- end }}
