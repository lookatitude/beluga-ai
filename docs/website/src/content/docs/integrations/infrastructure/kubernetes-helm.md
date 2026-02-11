---
title: Kubernetes Helm Deployment
description: Deploy Beluga AI services to Kubernetes using Helm charts for repeatable, configurable deployments with scaling, secrets management, and production-ready defaults.
---

## Overview

Deploying AI agents to production requires repeatable, auditable infrastructure. Helm charts let you package your Beluga AI services as versioned, configurable Kubernetes releases that can be deployed consistently across development, staging, and production environments. This eliminates manual deployment steps, enforces resource limits to prevent runaway costs, and provides a standard upgrade path for your team. This guide walks through creating and deploying Helm charts for Beluga AI services on Kubernetes. By the end, you will have a production-ready Helm chart that deploys a Beluga AI REST server with configurable replicas, resource limits, secrets, and service exposure.

## Prerequisites

- A running Kubernetes cluster (local or cloud-managed)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) configured for your cluster
- [Helm 3.x](https://helm.sh/docs/intro/install/) installed
- A container image of your Beluga AI application pushed to a registry
- Familiarity with Kubernetes Deployment and Service resources

## Installation

Install Helm if not already available:

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

Verify your tools:

```bash
helm version
kubectl cluster-info
```

## Configuration

### Chart Structure

A complete Beluga AI Helm chart follows this layout:

```text
beluga-ai-chart/
  Chart.yaml
  values.yaml
  templates/
    _helpers.tpl
    deployment.yaml
    service.yaml
    configmap.yaml
    secret.yaml
  charts/
```

### Chart.yaml

Define chart metadata:

```yaml
apiVersion: v2
name: beluga-ai
description: Beluga AI Framework deployment
type: application
version: 1.0.0
appVersion: "1.0.0"
```

### values.yaml

Default configuration values:

```yaml
replicaCount: 2

image:
  repository: beluga-ai
  tag: "latest"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 8080

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

config:
  llmProvider: "openai"
  embeddingProvider: "openai"
```

| Value | Description | Default |
|---|---|---|
| `replicaCount` | Number of pod replicas | `2` |
| `image.repository` | Container image name | `beluga-ai` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Kubernetes pull policy | `IfNotPresent` |
| `service.type` | Service type (`ClusterIP`, `NodePort`, `LoadBalancer`) | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `250m` |
| `resources.requests.memory` | Memory request | `256Mi` |

## Usage

### Template Helpers

Create `templates/_helpers.tpl` with standard label and name helpers:

```yaml
{{- define "beluga-ai.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "beluga-ai.fullname" -}}
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

{{- define "beluga-ai.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "beluga-ai.labels" -}}
helm.sh/chart: {{ include "beluga-ai.chart" . }}
{{ include "beluga-ai.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "beluga-ai.selectorLabels" -}}
app.kubernetes.io/name: {{ include "beluga-ai.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
```

### Deployment Template

Create `templates/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "beluga-ai.fullname" . }}
  labels:
    {{- include "beluga-ai.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "beluga-ai.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "beluga-ai.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: beluga-ai
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.service.port }}
              protocol: TCP
          env:
            - name: OPENAI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ include "beluga-ai.fullname" . }}-secrets
                  key: openai-api-key
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
```

### Service Template

Create `templates/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "beluga-ai.fullname" . }}
  labels:
    {{- include "beluga-ai.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "beluga-ai.selectorLabels" . | nindent 4 }}
```

### Deploy

Validate your chart, then install:

```bash
# Lint the chart
helm lint ./beluga-ai-chart

# Install to the cluster
helm install beluga-ai ./beluga-ai-chart

# Verify the deployment
kubectl get pods -l app.kubernetes.io/name=beluga-ai
```

Upgrade or uninstall:

```bash
# Upgrade with new values
helm upgrade beluga-ai ./beluga-ai-chart --set replicaCount=3

# Uninstall
helm uninstall beluga-ai
```

## Advanced Topics

### Secrets Management

Create a Kubernetes secret for API keys before deploying:

```bash
kubectl create secret generic beluga-ai-secrets \
  --from-literal=openai-api-key="sk-..." \
  --from-literal=anthropic-api-key="sk-ant-..."
```

Reference secrets in your Deployment template via `secretKeyRef` as shown above. Never store API keys in `values.yaml` or check them into version control.

### Health Checks

Add liveness and readiness probes to detect and recover from unhealthy pods:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 10
  periodSeconds: 15
readinessProbe:
  httpGet:
    path: /readyz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Horizontal Pod Autoscaling

Scale based on CPU utilization:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "beluga-ai.fullname" . }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "beluga-ai.fullname" . }}
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### Monitoring with Prometheus

Add a `ServiceMonitor` for Prometheus scraping if you expose a `/metrics` endpoint:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ include "beluga-ai.fullname" . }}
spec:
  selector:
    matchLabels:
      {{- include "beluga-ai.selectorLabels" . | nindent 6 }}
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

### Pinning Versions

For production deployments, always pin both chart and image versions:

```bash
helm install beluga-ai ./beluga-ai-chart \
  --version 1.0.0 \
  --set image.tag=v1.2.3
```

## Troubleshooting

### Chart not found

**Cause**: Incorrect chart path or missing `Chart.yaml`.

**Resolution**: Run `helm lint ./beluga-ai-chart` to validate the chart structure. Ensure `Chart.yaml` exists at the chart root.

### Image pull failed

**Cause**: The container image is not accessible from the cluster.

**Resolution**: Verify the image exists in your registry. If using a private registry, configure `imagePullSecrets` in the Deployment spec:

```yaml
spec:
  imagePullSecrets:
    - name: registry-credentials
```

### Pod CrashLoopBackOff

**Cause**: The application fails to start, often due to missing environment variables or configuration.

**Resolution**: Check pod logs with `kubectl logs <pod-name>`. Verify all required secrets and config maps are created before deployment.

## Related Resources

- [Auth0 JWT Authentication](/integrations/auth0-jwt) -- Secure API endpoints
- [Server Package Guide](/api-reference/server) -- Beluga AI server configuration
- [Observability Guide](/guides/observability) -- Monitoring and tracing setup
