# Deployment Guide

## Prerequisites

- Kubernetes cluster with at least two ingress/gateway implementations installed
- Helm 3
- `kubectl` configured for the target cluster
- (Optional) Prometheus for resource metrics collection
- (Optional) ArgoCD for GitOps deployment

## Quick Start

### 1. Install with Helm

```bash
# Create namespace
kubectl create namespace gw-bench

# Install the chart
helm install gw-bench deploy/helm/gw-bench/ \
  -n gw-bench \
  -f deploy/helm/gw-bench/values.example.yaml
```

### 2. Verify deployment

```bash
kubectl get pods -n gw-bench
# Should show: gw-bench-backend-xxx (2 replicas)
```

### 3. Run a benchmark

```bash
kubectl create job gw-bench-run-001 \
  --from=cronjob/gw-bench-runner \
  -n gw-bench
```

### 4. Check results

```bash
kubectl logs job/gw-bench-run-001 -n gw-bench
```

## Networking Requirements

- The runner pod must be able to reach gateway URLs (HTTP)
- The runner pod must be able to reach Prometheus (if configured)
- The backend pods must be reachable from the gateway pods
- DNS resolution must work for gateway hostnames

## Prometheus Configuration

For resource metrics collection, set `runner.prometheus.url` to your Prometheus instance:

```yaml
runner:
  prometheus:
    url: http://prometheus-operated.monitoring.svc:9090
```

Requirements for accurate resource metrics:
- cAdvisor-sourced `container_*` metrics must be available
- `scrape_interval` ≤ 15s on gateway targets
- Gateway pods must have standard container labels

If Prometheus is unreachable, benchmarks will still run — resource metrics will be omitted with a warning.

## First Run Checklist

1. Verify backend pods are healthy: `kubectl get pods -n gw-bench`
2. Verify gateway routes are configured: `kubectl get httproute,ingress -n gw-bench`
3. Test backend reachability through each gateway manually
4. Run a short scenario first to validate end-to-end flow
5. Check NDJSON output in job logs for valid events
