# gw-bench

A Kubernetes-native A/B benchmark harness for comparing ingress / Gateway API implementations under identical workloads.

## Overview

gw-bench runs identical load patterns through two (or more) Kubernetes ingress implementations against a shared, deterministic backend, and emits structured JSON results to stdout for ingestion by an existing log pipeline (Elasticsearch / Kibana).

**Initial use case:** comparing `agentgateway` and `ingress-nginx` for production decision-making — but the design is gateway-agnostic.

## Features

- **Fair comparisons** — same backend, same node placement, same payload, same warmup discard
- **Cluster-native execution** — load generation runs as a pod, not from a laptop
- **GitOps-friendly** — Helm chart ships through ArgoCD ApplicationSet
- **Logs-first results** — structured JSON to stdout, consumed via existing log pipeline
- **Gateway-pluggable** — add a new gateway with config only, no code changes
- **Protocol-aware** — HTTP, SSE, and MCP-over-streamable-HTTP

## Quick Start

### Prerequisites

- Kubernetes cluster with at least one ingress/gateway implementation
- Helm 3
- `kubectl`

### Install

```bash
# Clone the repo
git clone https://github.com/BenjaminBanwart/gw-bench.git
cd gw-bench

# Copy and edit values
cp deploy/helm/gw-bench/values.example.yaml my-values.yaml
# Edit my-values.yaml with your gateway configuration

# Install
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

## Canonical Scenarios

| Scenario | Protocol | Load Gen | Description |
|---|---|---|---|
| `http-small-5k-qps` | HTTP | fortio | Small payload at 5k QPS, 100 connections |
| `http-large-payload` | HTTP | fortio | 100KB payload at 500 QPS |
| `http-sustained-1k-qps` | HTTP | fortio | 5-minute sustained 1k QPS |
| `sse-streaming` | SSE | k6 | SSE event delivery test |
| `mcp-tool-burst` | MCP | k6 | MCP tools/call burst test |

## CLI

```
gw-bench run <scenario-file> [scenario-file...]   # Execute benchmark(s)
gw-bench validate <scenario-file>                   # Validate scenario YAML
gw-bench report --from file <results.ndjson>        # Generate comparison report
gw-bench version                                    # Print version info
```

## Documentation

- [Architecture](docs/architecture.md)
- [Deployment Guide](docs/deployment.md)
- [Scenario Authoring](docs/scenarios.md)
- [Adding a Gateway](docs/adding-a-gateway.md)
- [Kibana Integration](docs/kibana.md)

## License

Apache-2.0
