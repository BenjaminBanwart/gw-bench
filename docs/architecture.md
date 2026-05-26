# Architecture

See [original-design-doc.md](original-design-doc.md) for the full architectural design document.

## Key Components

- **Runner** (`cmd/gw-bench`): CLI that orchestrates benchmark scenarios, shells out to fortio/k6, queries Prometheus, and emits structured JSON events.
- **Test Backend** (`cmd/test-backend`): Deterministic HTTP server with `/echo`, `/sse`, and `/mcp` endpoints.
- **Helm Chart** (`deploy/helm/gw-bench`): Deploys both components plus routing (HTTPRoute/Ingress) for each gateway.

## Data Flow

```
Runner Pod → Gateway → Test Backend
     ↓
  stdout (NDJSON)
     ↓
  Log Pipeline → Elasticsearch → Kibana
```

## Event Types

| Event | Description |
|---|---|
| `run_start` | Emitted once at the start of a benchmark job |
| `scenario_start` | Emitted per gateway before load generation |
| `scenario_complete` | Emitted per gateway with full metrics |
| `comparison` | Emitted once comparing two gateways |
| `run_complete` | Final event with total duration |
| `error` | Emitted on any failure |
