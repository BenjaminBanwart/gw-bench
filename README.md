# gw-bench

A Kubernetes-native A/B benchmark harness for comparing ingress / Gateway API implementations under identical workloads.

## Overview

gw-bench runs identical load patterns through two (or more) Kubernetes ingress implementations against a shared, deterministic backend, and emits structured JSON results to stdout for ingestion by an existing log pipeline (Elasticsearch / Kibana).

**Initial use case:** comparing `agentgateway` and `ingress-nginx` for production decision-making — but the design is gateway-agnostic.

## Features

- **Fair comparisons** — same backend, same node placement, same payload, same warmup discard
- **Cluster-native execution** — load generation runs as a pod, not from a laptop
- **GitOps-friendly** — Helm chart ships through ArgoCD Application
- **Logs-first results** — structured JSON to stdout, consumed via existing log pipeline
- **Gateway-pluggable** — add a new gateway with config only, no code changes
- **Protocol-aware** — HTTP, SSE, and MCP-over-streamable-HTTP

## Quick Start

### Prerequisites

- Kubernetes cluster with at least one ingress/gateway implementation
- Helm 3
- `kubectl`

### Install

#### Helm OCI (recommended)

```bash
# Pull the example values for reference
helm show values oci://registry-1.docker.io/benjaminbanwart/gw-bench > my-values.yaml
# Edit my-values.yaml with your gateway configuration

# Install
helm install gw-bench oci://registry-1.docker.io/benjaminbanwart/gw-bench \
  -n gw-bench --create-namespace \
  -f my-values.yaml
```

#### From source

```bash
git clone https://github.com/BenjaminBanwart/gw-bench.git
cd gw-bench
cp deploy/helm/gw-bench/values.example.yaml my-values.yaml
# Edit my-values.yaml

helm install gw-bench deploy/helm/gw-bench/ \
  -n gw-bench --create-namespace \
  -f my-values.yaml
```

### Run a benchmark

```bash
# Trigger a one-off run
kubectl create job gw-bench-run-001 \
  --from=cronjob/gw-bench-runner \
  -n gw-bench

# Watch results
kubectl logs -f job/gw-bench-run-001 -n gw-bench
```

### Generate a report

```bash
# Save job logs to a file
kubectl logs job/gw-bench-run-001 -n gw-bench > results.ndjson

# Generate markdown report
gw-bench report --from file results.ndjson

# Or filter by run ID
gw-bench report --from file --run-id <run-id> results.ndjson
```

### Validate a scenario

```bash
gw-bench validate scenarios/http-small-5k-qps.yaml
```

## Configuration

### Gateway Setup

Each gateway needs two things: a routing rule (HTTPRoute or Ingress) and a DNS hostname resolvable inside the cluster.

Add gateways in your `values.yaml`:

```yaml
gateways:
  # Gateway API implementation (e.g. Envoy Gateway, agentgateway)
  myGateway:
    enabled: true
    type: gatewayApi
    scheme: https
    parentRef:
      name: my-gateway
      namespace: my-gateway-ns
    hostname: test.mygateway.internal

  # Ingress implementation (e.g. ingress-nginx)
  myIngress:
    enabled: true
    type: ingress
    scheme: https
    ingressClassName: nginx
    hostname: test.myingress.internal
```

### DNS Resolution

Gateway hostnames (e.g. `test.mygateway.internal`) must resolve inside the cluster. Common options:

- **CoreDNS rewrite** — add a rewrite rule in the `coredns` ConfigMap to point the hostname at the gateway's Service ClusterIP
- **ExternalDNS** — if your gateways have external LoadBalancer IPs
- **Headless entries** — some CNIs support in-cluster DNS registration for Services

Verify from inside the cluster: `kubectl run dns-test --rm -it --image=busybox -- nslookup test.mygateway.internal`

### Prometheus (Optional)

Resource metrics (CPU/memory per gateway pod) require a reachable Prometheus instance:

```yaml
runner:
  prometheus:
    url: http://prometheus-operated.monitoring.svc:9090
```

If omitted or unreachable, benchmarks still run — resource metrics are skipped with a warning.

### Choosing Scenarios

Select which bundled scenarios to run:

```yaml
runner:
  scenarios:
    - http-small-5k-qps
    - http-medium-payload
    - http-sustained-1k-qps
```

See `values.example.yaml` for a fully annotated configuration.

### Profile a Single Gateway

To baseline one gateway without comparison, configure only one gateway in your scenario or values file. gw-bench runs in **profile mode** — it executes the scenario and skips the comparison event.

```bash
# Quick single-gateway profile
gw-bench run scenarios/http-small-5k-qps.yaml
```

## What Ships in the Runner Image

The runner container (`Dockerfile.runner`) bundles everything needed to execute all scenarios:

| Binary | Source | Purpose |
|---|---|---|
| `gw-bench` | Built from source | CLI orchestrator — reads scenarios, shells out to load generators, queries Prometheus, emits NDJSON |
| `fortio` | `fortio/fortio:1.75.1` | HTTP load generator for fortio-based scenarios |
| `k6` | `grafana/k6:2.0.0` | Scriptable load generator for k6-based scenarios (SSE, MCP, many-routes) |

## Building

| Target | Description |
|---|---|
| `make build` | Compile `gw-bench` and `test-backend` to `bin/` |
| `make test` | Run all tests with race detection |
| `make lint` | Run `golangci-lint` |
| `make docker` | Build runner and backend container images |
| `make helm-lint` | Lint and template-render the Helm chart |
| `make clean` | Remove `bin/` |

## Canonical Scenarios

| Scenario | Protocol | Load Gen | Description |
|---|---|---|---|
| `http-small-5k-qps` | HTTP | fortio | Small payload at 5k QPS, 100 connections |
| `http-medium-payload` | HTTP | fortio | 100KB payload at 500 QPS, 50 connections |
| `http-large-payload` | HTTP | fortio | 100MB payload at 10 QPS, 5 connections |
| `http-sustained-1k-qps` | HTTP | fortio | 5-minute sustained 1k QPS |
| `many-routes` | HTTP | k6 | Fan-out across 500 routes, 100 VUs |
| `tls-connection-churn` | HTTP/TLS | fortio | No keepalive — new TLS handshake per request |
| `sse-streaming` | SSE | k6 | SSE event delivery test |
| `mcp-tool-burst` | MCP | k6 | MCP tools/call burst test |

## CLI

```
gw-bench run <scenario-file> [scenario-file...]            # Execute benchmark(s)
gw-bench validate <scenario-file>                          # Validate scenario YAML
gw-bench report --from file <results.ndjson>               # Generate comparison report (markdown)
gw-bench report --from file --format json <results.ndjson> # Report as JSON
gw-bench report --from file --run-id <id> <results.ndjson> # Filter by run ID
gw-bench version                                           # Print version info
```

## Documentation

- [Architecture](docs/architecture.md)
- [Deployment Guide](docs/deployment.md)
- [Scenario Authoring](docs/scenarios.md)
- [Adding a Gateway](docs/adding-a-gateway.md)
- [Kibana Integration](docs/kibana.md)
- [Original Design Doc](docs/original-design-doc.md)

## License

Apache-2.0
