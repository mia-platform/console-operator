{{/*
Expand the name of the chart.
*/}}
{{- define "operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "operator.fullname" -}}
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
Create version used to select the Console Operator version to be installed .
*/}}
{{- define "operator.version" -}}
{{- if .Values.tag }}
{{- .Values.tag }}
{{- else if .Chart.AppVersion }}
{{- .Chart.AppVersion }}
{{- else }}
{{- fail "At least one between .Values.tag and .Chart.AppVersion should be set" }}
{{- end }}
{{- end }}

{{/*
The suffix added to the Console Operator images, to identify CI builds.
*/}}
{{- define "operator.suffix" -}}
{{/* https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string */}}
{{- $semverregex := "^v(?P<major>0|[1-9]\\d*)\\.(?P<minor>0|[1-9]\\d*)\\.(?P<patch>0|[1-9]\\d*)(?:-(?P<prerelease>(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$" }}
{{- if or (eq .Values.tag "") (mustRegexMatch $semverregex .Values.tag) }}
{{- print "" }}
{{- else }}
{{- print "-ci" }}
{{- end }}
{{- end }}

{{/*
Create a name prefixed with the chart name, it accepts a dict which contains the field "name".
*/}}
{{- define "operator.prefixedName" -}}
{{- printf "%s-%s" (include "operator.name" .) .name }}
{{- end }}

{{/*
Create the file name of a role starting from a prefix, it accepts a dict which contains the field "prefix".
*/}}
{{- define "operator.role-filename" -}}
{{- printf "files/%s-%s" .prefix "Role.yaml" }}
{{- end }}

{{/*
Create the file name of a cluster role starting from a prefix, it accepts a dict which contains the field "prefix".
*/}}
{{- define "operator.cluster-role-filename" -}}
{{- printf "files/%s-%s" .prefix "ClusterRole.yaml" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "operator.labels" -}}
{{ include "operator.selectorLabels" . }}
helm.sh/chart: {{ include "operator.chart" . }}
app.kubernetes.io/version: {{ quote (include "operator.version" .) }}
app.kubernetes.io/managed-by: {{ quote .Release.Service }}
{{- end }}

{{/*
Selector labels, it accepts a dict which contains fields "name" and "module"
*/}}
{{- define "operator.selectorLabels" -}}
app.kubernetes.io/name: {{ quote .name }}
app.kubernetes.io/instance: {{ quote (printf "%s-%s" .Release.Name .name) }}
app.kubernetes.io/component: {{ quote .module }}
app.kubernetes.io/part-of: {{ quote (include "operator.name" .) }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the Pod security context
*/}}
{{- define "operator.podSecurityContext" -}}
runAsNonRoot: true
runAsUser: 1000
runAsGroup: 1000
fsGroup: 1000
{{- end -}}

{{/*
Get the Container security context
*/}}
{{- define "operator.containerSecurityContext" -}}
allowPrivilegeEscalation: false
{{- end -}}