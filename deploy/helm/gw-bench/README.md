# gw-bench Helm Chart

A Helm chart for deploying the gw-bench benchmark harness — test backend, runner job, routing rules, and optional CronJob scheduling.

## Prerequisites

- Kubernetes 1.27+
- Helm 3
- At least one ingress or Gateway API implementation installed in the cluster
- Gateway hostnames resolvable inside the cluster (see [DNS Resolution](#dns-resolution))

## Installation

### From OCI registry (recommended)

```bash
# Pull example values for reference
helm show values oci://registry-1.docker.io/benjaminbanwart/gw-bench > my-values.yaml
# Edit my-values.yaml — see comments in values.example.yaml for guidance

helm install gw-bench oci://registry-1.docker.io/benjaminbanwart/gw-bench \
  -n gw-bench --create-namespace \
  -f my-values.yaml
```

### From source

```bash
helm install gw-bench deploy/helm/gw-bench/ -n gw-bench --create-namespace -f my-values.yaml
```

## Upgrading

```bash
helm upgrade gw-bench . -n gw-bench -f my-values.yaml
```

## What Gets Deployed

| Resource | Description |
|---|---|
| `Deployment/gw-bench-backend` | Deterministic HTTP server with `/echo`, `/sse`, `/mcp` endpoints |
| `Service/gw-bench-backend` | ClusterIP service for the backend |
| `HTTPRoute` / `Ingress` | One per enabled gateway, routing the test hostname to the backend |
| `CronJob/gw-bench-runner` | Runner job template (opt-in schedule via `runner.cronjob`) |
| `ConfigMap/gw-bench-scenarios` | Bundled scenario YAML files mounted into the runner pod |
| `ServiceAccount` + `RBAC` | Minimal permissions for the runner pod |
| `NetworkPolicy` | (Optional) Restricts traffic to/from the namespace |
| `ServiceMonitor` | (Optional) Prometheus scrape config for the backend |

## Running a Benchmark

```bash
# One-off run from the CronJob template
kubectl create job gw-bench-run-001 --from=cronjob/gw-bench-runner -n gw-bench

# Watch results (structured NDJSON)
kubectl logs -f job/gw-bench-run-001 -n gw-bench
```

## Key Values

| Value | Default | Description |
|---|---|---|
| `backend.enabled` | `true` | Deploy the test backend |
| `backend.replicas` | `2` | Backend pod replicas |
| `runner.scenarios` | `[http-small-5k-qps, ...]` | List of bundled scenario names to run |
| `runner.cronjob.enabled` | `false` | Enable scheduled runs |
| `runner.cronjob.schedule` | `"0 2 * * 0"` | Cron schedule (default: weekly Sunday 2 AM) |
| `runner.prometheus.url` | `""` | Prometheus URL for resource metrics (leave empty to skip) |
| `runner.clusterName` | `""` | Cluster identifier in emitted events |
| `gateways.<name>.enabled` | — | Enable this gateway for benchmarking |
| `gateways.<name>.type` | — | `gatewayApi` (HTTPRoute) or `ingress` (Ingress) |
| `gateways.<name>.hostname` | — | Test hostname routed to the backend |
| `scenarios.bundled` | `true` | Mount built-in scenarios from `scenarios/` |
| `networkPolicy.enabled` | `false` | Create a NetworkPolicy for the namespace |
| `serviceMonitor.enabled` | `false` | Create a Prometheus ServiceMonitor |

See `values.yaml` for the full set of defaults and `values.example.yaml` for an annotated working example.

## DNS Resolution

Gateway hostnames (e.g. `test.mygateway.internal`) must resolve inside the cluster. Options include CoreDNS rewrite rules, ExternalDNS, or Service-level DNS entries. See the [project README](../../../README.md#dns-resolution) for details.

## Uninstalling

```bash
helm uninstall gw-bench -n gw-bench
kubectl delete namespace gw-bench
```
