# Kubernetes Helm Deployment

Welcome, colleague! In this integration guide, we're going to deploy Beluga AI services using Kubernetes Helm charts. Helm provides package management for Kubernetes, making deployment and configuration management easier.

## What you will build

You will create Helm charts for deploying Beluga AI services to Kubernetes, enabling easy deployment, scaling, and configuration management of your AI applications.

## Learning Objectives

- ✅ Create Helm charts for Beluga AI
- ✅ Deploy services to Kubernetes
- ✅ Configure service settings via Helm values
- ✅ Understand Helm best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Kubernetes cluster access
- Helm 3.x installed
- kubectl configured

## Step 1: Setup and Installation

Install Helm:
```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash


Verify installation:
helm version
```

## Step 2: Create Helm Chart Structure

Create Helm chart directory:
```
mkdir -p beluga-ai-chart/{templates,charts}
cd beluga-ai-chart

Create `Chart.yaml`:
apiVersion: v2
name: beluga-ai
description: Beluga AI Framework deployment
type: application
version: 1.0.0
appVersion: "1.0.0"
```

## Step 3: Create Deployment Template

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
```
      - name: beluga-ai
```text
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        ports:
```
        - name: http
```text
          containerPort: {{ .Values.service.port }}
          protocol: TCP
        env:
```
        - name: OPENAI_API_KEY
```
          valueFrom:
            secretKeyRef:
              name: {{ include "beluga-ai.fullname" . }}-secrets
              key: openai-api-key
        resources:
          {{- toYaml .Values.resources | nindent 10 }}

## Step 4: Create Service Template

Create `templates/service.yaml`:
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

## Step 5: Create Values File

Create `values.yaml`:
```yaml
replicaCount: 2
yaml
image:
  repository: beluga-ai
  tag: "latest"
  pullPolicy: IfNotPresent
yaml
service:
  type: ClusterIP
  port: 8080
yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi
yaml
config:
  llmProvider: "openai"
  embeddingProvider: "openai"


## Step 6: Deploy with Helm

Deploy the chart:
# Install
helm install beluga-ai ./beluga-ai-chart

# Upgrade
helm upgrade beluga-ai ./beluga-ai-chart

# Uninstall
helm uninstall beluga-ai
```

## Step 7: Complete Integration

Here's a complete Helm chart structure:
beluga-ai-chart/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   └── _helpers.tpl
└── charts/
```

Create `templates/_helpers.tpl`:
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

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `replicaCount` | Number of replicas | `2` | No |
| `image.repository` | Container image | `beluga-ai` | Yes |
| `image.tag` | Image tag | `latest` | No |
| `service.type` | Service type | `ClusterIP` | No |
| `service.port` | Service port | `8080` | No |

## Common Issues

### "Chart not found"

**Problem**: Chart path incorrect.

**Solution**: Verify chart directory:helm lint ./beluga-ai-chart
```

### "Image pull failed"

**Problem**: Image not accessible.

**Solution**: Ensure image is in registry or use image pull secrets.

## Production Considerations

When using Helm in production:

- **Versioning**: Pin chart and app versions
- **Secrets**: Use Kubernetes secrets for sensitive data
- **Resource limits**: Set appropriate limits
- **Health checks**: Configure liveness and readiness probes
- **Monitoring**: Add service monitors for Prometheus

## Next Steps

Congratulations! You've created Helm charts for Beluga AI. Next, learn how to:

- **[Auth0/JWT Authentication](./auth0-jwt-authentication.md)** - Secure APIs
- **[Server Package Documentation](../../api-docs/packages/server.md)** - Deep dive into server package
- **[Deployment Guide](../../getting-started/07-production-deployment.md)** - Production deployment

---

**Ready for more?** Check out the Integrations Index for more integration guides!
