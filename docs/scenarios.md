# Scenario Authoring Guide

Scenarios are YAML files that define a benchmark configuration. They specify the protocol, load generator, traffic pattern, and gateways to test.

## Schema

```yaml
apiVersion: gw-bench/v1
kind: Scenario
metadata:
  name: <unique-name>
  description: |
    Human-readable description of what this scenario tests.
spec:
  protocol: http | sse | mcp
  loadGenerator: fortio | k6
  duration: <go-duration>        # Total run time including warmup
  warmup: <go-duration>          # Discarded from metrics
  cooldown: <go-duration>        # Pause between gateways (default: 30s)
  targetQPS: <int>               # 0 = saturate
  connections: <int>
  noKeepAlive: <bool>            # Disable keepalive (force new conn per request)
  payload:                       # Required for fortio
    method: GET | POST
    path: /echo
    sizeBytes: 0
    headers:
      Key: Value
    body: ""
  k6Script: <path>               # Required for k6
  k6Args:
    vus: "50"
    duration: "60s"
  gateways:
    - name: <gateway-name>
      url: <full-url>
      promPodSelector: '<label-selector>'
```

## Validation Rules

- `apiVersion` must be `gw-bench/v1`
- `kind` must be `Scenario`
- `metadata.name` is required
- `protocol` must be `http`, `sse`, or `mcp`
- `loadGenerator` must be `fortio` or `k6`
- `duration` must be greater than `warmup`
- Exactly one of `payload` or `k6Script` must be set
- `fortio` requires `payload`; `k6` requires `k6Script`
- `mcp` protocol requires `k6` load generator
- At least one gateway is required

## Canonical Scenarios

The following scenarios ship in the `scenarios/` directory.

### HTTP / Fortio

| Scenario | QPS | Connections | Payload | Duration | Notes |
|---|---|---|---|---|---|
| `http-small-5k-qps` | 5,000 | 100 | 0 B | 60 s | Baseline latency test with no payload |
| `http-medium-payload` | 500 | 50 | 100 KB | 60 s | Throughput and buffering at moderate bandwidth |
| `http-large-payload` | 10 | 5 | 100 MB | 60 s | Throughput and buffering at high bandwidth |
| `http-sustained-1k-qps` | 1,000 | 50 | 0 B | 300 s | 5-minute steady-state stability test |
| `tls-connection-churn` | 500 | 50 | 0 B | 60 s | `noKeepAlive: true` — new TCP+TLS handshake per request; measures TLS termination overhead |

### K6

| Scenario | Protocol | VUs | Duration | Notes |
|---|---|---|---|---|
| `many-routes` | HTTP | 100 | 60 s | Fan-out across 500 `/route/{N}` paths; stresses routing table lookup performance |
| `sse-streaming` | SSE | 50 | 60 s | Opens SSE streams (10 events at 100 ms intervals) and validates event delivery |
| `mcp-tool-burst` | MCP | 50 | 60 s | MCP Streamable HTTP — initialize → tools/list → tools/call burst with idle gaps |

## Tips

### Warmup

Always set a non-zero warmup. The first 10–30 seconds of any test include connection establishment, JIT effects, and cache warming that aren't representative of steady-state performance.

### Connection Reuse

The `connections` parameter controls connection pooling. Be explicit about this — the howardjohn benchmark found nginx 5–20x slower than alternatives purely from connection pooling differences.

### Target QPS

- Set `targetQPS: 0` to saturate (find max throughput)
- Set a specific value for sustained-load testing

### Profile Mode

If you specify only one gateway, gw-bench runs in **profile mode**: it executes the scenario against the single gateway and skips the comparison event. Useful for baselining.

### First-Run Effects

Run a "throwaway" scenario before the real one if results look unstable. JIT compilation, DNS caching, and connection pool initialization can all affect early results.
