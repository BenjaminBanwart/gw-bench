# gw-bench: Architectural and Implementation Plan

A Kubernetes-native A/B benchmark harness for comparing ingress / Gateway API implementations under identical workloads. Initial use case: comparing `agentgateway` and `ingress-nginx` for production decision-making, but the design is gateway-agnostic from day one.

---

## 1. Project overview

### What it is

`gw-bench` is a Helm-deployable test harness that runs identical workloads through two (or more) Kubernetes ingress implementations against a shared, deterministic backend, and emits structured JSON results to stdout for ingestion by an existing log pipeline (Elasticsearch / Kibana).

### Why it exists

There is no turnkey tool for "compare two Kubernetes ingresses fairly under realistic and AI-native traffic patterns." The closest existing prior art:

- `gateway-api-bench` (howardjohn) — fortio-based comparison of Gateway API implementations. Doesn't include agentgateway, doesn't speak MCP, isn't designed for repeated A/B runs.
- `xk6-infobip-mcp` — k6 extension for MCP protocol load testing. Solves the protocol layer, not the comparison framing.
- Vendor-published benchmarks (Solo.io, etc.) — useful sanity checks but produce numbers from someone else's environment, not yours.

`gw-bench` is the glue: it wraps fortio and k6, deploys a deterministic test backend, runs identical scenarios through each gateway, and produces correlated structured-log output for side-by-side comparison.

### Goals

- **Fair comparisons.** Hold every variable constant except the gateway under test: same backend, same node placement (or at minimum same node class), same payload, same client behavior, same warmup discard.
- **Cluster-native execution.** Load generation runs as a pod in the cluster, not from a developer laptop. Eliminates network variability between client and gateway.
- **GitOps-friendly.** Everything declarative. Helm chart ships through ArgoCD ApplicationSet. Scenarios committed as YAML.
- **Logs-first results.** Structured JSON to stdout. Consume via existing log pipeline. No bespoke results store.
- **Gateway-pluggable.** Add a new gateway by adding an HTTPRoute/Ingress and a config entry. No code changes required.
- **Protocol-aware.** Handle plain HTTP, SSE, and MCP-over-streamable-HTTP from the start. WebSocket and gRPC are Phase 3+.

### Non-goals

- Not a continuous performance regression system (Phase 4+ at earliest).
- Not a UI or dashboard product. Kibana saved searches and a small CLI report subcommand are sufficient.
- Not a synthetic-monitoring tool. This runs on demand, not continuously.
- Not an auth/security testing harness. Functional/protocol correctness is the load generator's concern.

---

## 2. Tech stack

| Concern | Choice | Rationale |
|---|---|---|
| Runner language | Go (1.22+) | K8s ecosystem default, single static binary, strong client libraries. |
| HTTP load generator | fortio (binary) | Designed for service-mesh comparison, low overhead, JSON output. |
| MCP/SSE load generator | k6 (binary) + xk6-infobip-mcp | Protocol-aware, scriptable, established extension. |
| Test backend language | Go | Trivial echo + minimal MCP server, easy to ship as scratch image. |
| Packaging | Helm 3 | Matches existing platform deployment pattern. |
| GitOps | ArgoCD ApplicationSet | Matches existing pattern. |
| Secrets | ESO + 1Password (Phase 3+) | Matches platform standard. Not needed in Phase 1. |
| CI | GitHub Actions reusable workflows | Matches existing org pattern. |
| Container registry | ghcr.io | Default for GitHub-hosted projects. |
| Result transport | stdout (structured JSON) | Logs already flow to Elastic. Zero new infrastructure. |
| Resource metric collection | Prometheus query (existing instance) | Already deployed in the clusters. |

---

## 3. Repository layout

```
gw-bench/
├── README.md
├── LICENSE                              # Apache-2.0
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml
├── .github/
│   └── workflows/
│       ├── ci.yaml                      # build, test, lint on PR
│       ├── release.yaml                 # tag → image + chart push
│       └── helm-lint.yaml
├── cmd/
│   ├── gw-bench/
│   │   └── main.go                      # CLI entry point
│   └── test-backend/
│       └── main.go                      # echo + minimal MCP server
├── internal/
│   ├── config/                          # scenario YAML parsing, validation
│   │   ├── scenario.go
│   │   └── scenario_test.go
│   ├── runner/                          # orchestration
│   │   ├── runner.go
│   │   └── runner_test.go
│   ├── loadgen/
│   │   ├── loadgen.go                   # interface
│   │   ├── fortio.go                    # subprocess + JSON parse
│   │   ├── fortio_test.go
│   │   ├── k6.go
│   │   └── k6_test.go
│   ├── metrics/                         # Prometheus query layer
│   │   ├── prometheus.go
│   │   └── prometheus_test.go
│   ├── events/                          # structured log event types
│   │   ├── events.go
│   │   └── emit.go
│   └── compare/                         # report generation
│       ├── compare.go
│       └── compare_test.go
├── scenarios/                           # canonical scenario library
│   ├── http-small-5k-qps.yaml
│   ├── http-large-payload.yaml
│   ├── http-sustained-1k-qps.yaml
│   ├── sse-streaming.yaml
│   └── mcp-tool-burst.yaml
├── scripts/
│   └── k6/
│       ├── mcp-burst.js                 # k6 + xk6-infobip-mcp scripts
│       └── sse-streaming.js
├── deploy/
│   ├── helm/
│   │   └── gw-bench/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       ├── values.example.yaml      # ready-to-use example with two gateways
│   │       ├── README.md
│   │       └── templates/
│   │           ├── _helpers.tpl
│   │           ├── namespace.yaml
│   │           ├── backend-deployment.yaml
│   │           ├── backend-service.yaml
│   │           ├── backend-httproute.yaml      # one per gateway, ranged
│   │           ├── backend-ingress.yaml        # for non-Gateway-API gateways
│   │           ├── runner-configmap.yaml       # mounts scenarios
│   │           ├── runner-job.yaml             # template for `kubectl create job --from=...`
│   │           ├── runner-cronjob.yaml         # optional, opt-in via values
│   │           ├── runner-rbac.yaml            # SA + minimal RBAC
│   │           ├── networkpolicy.yaml          # Cilium-compatible
│   │           └── servicemonitor.yaml         # optional
│   └── argocd/
│       ├── applicationset.yaml          # example for two-cluster rollout
│       └── README.md
├── docs/
│   ├── architecture.md                  # this document, trimmed
│   ├── scenarios.md                     # how to author a scenario
│   ├── deployment.md                    # ArgoCD wiring, networking
│   ├── adding-a-gateway.md
│   └── kibana.md                        # saved searches, sample queries
└── examples/
    └── kibana/
        ├── saved-search-comparison.ndjson
        └── dashboard.ndjson
```

---

## 4. Architecture

### Topology in-cluster

```
┌─────────────────────────────────────────────────────────────────┐
│  cluster: pittsburg (or columbia, or lab)                       │
│                                                                 │
│   namespace: gw-bench                                           │
│                                                                 │
│   ┌─────────────────┐       ┌──────────────────────────────┐    │
│   │ runner Job      │──────▶│ test-backend (Deployment)    │    │
│   │  - reads        │  via  │  - /echo (HTTP)              │    │
│   │    scenario     │  one  │  - /sse (Server-Sent Events) │    │
│   │  - shells out   │  of   │  - /mcp (Streamable HTTP)    │    │
│   │    fortio/k6    │  two  │                              │    │
│   │  - emits JSON   │ paths └──────────────────────────────┘    │
│   │  - exits        │                                           │
│   └────────┬────────┘                                           │
│            │                                                    │
│            ▼ (path A: through agentgateway)                     │
│   ┌────────────────┐         ┌────────────────┐                 │
│   │ agentgateway   │────────▶│ test-backend   │                 │
│   └────────────────┘         └────────────────┘                 │
│                                                                 │
│            ▼ (path B: through ingress-nginx)                    │
│   ┌────────────────┐         ┌────────────────┐                 │
│   │ ingress-nginx  │────────▶│ test-backend   │                 │
│   └────────────────┘         └────────────────┘                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
              stdout (JSON)   │
                              ▼
                  ┌──────────────────────┐
                  │ existing log pipeline│
                  │   → Elasticsearch    │
                  │   → Kibana / Cerebro │
                  └──────────────────────┘
```

The runner Job hits the **gateway URL**, not the backend service directly. This means the same Pod-to-Pod path: runner → CNI → gateway → CNI → backend. The only thing that changes between scenario halves is which gateway is in the path.

### Execution flow for a single run

1. User commits/edits a scenario YAML, syncs ArgoCD (or runs `helm upgrade` directly in a lab cluster).
2. User triggers a run: `kubectl create job gw-bench-run-<id> --from=cronjob/gw-bench-runner -n gw-bench` (or via a CLI wrapper that does this).
3. Runner pod starts, reads `SCENARIO` env var, loads scenario YAML from mounted ConfigMap.
4. Runner emits `run_start` event with a generated `run_id`.
5. Runner emits `scenario_start` event for gateway A.
6. Runner shells out to fortio (or k6) targeting gateway A's URL for the **warmup duration** (results discarded). Then immediately runs a second invocation for the **measurement duration**. Captures load generator JSON output from the second run only.
7. Runner queries Prometheus for gateway A's pod CPU/memory over the measurement window (excluding warmup).
8. Runner emits `scenario_complete` event for gateway A with metrics + resource usage.
9. Cooldown sleep (configurable, default 30s).
10. Repeat 5–8 for gateway B.
11. If ≥ 2 gateways, runner emits `comparison` event correlating A and B. If only 1 gateway (profile mode), this step is skipped.
12. Runner emits `run_complete` event.
13. Runner sleeps 3 seconds (log pipeline flush) and exits 0.

Failure handling: any subprocess failure or timeout emits an `error` event with full context, then exits non-zero. The Job's `restartPolicy: Never` and `backoffLimit: 0` keep it idempotent.

---

## 5. Component specifications

### 5.1 Test backend (`cmd/test-backend`)

A single Go HTTP server. Tiny image (scratch + static binary).

**Endpoints:**

| Path | Method | Behavior |
|---|---|---|
| `/healthz` | GET | 200 OK |
| `/echo` | GET, POST | Returns request body or `?size=N` bytes of static payload. Optional `?delay_ms=N` to simulate backend latency. |
| `/sse` | GET | Server-Sent Events. Streams `?count=N` events at `?interval_ms=M` cadence then closes. |
| `/mcp` | POST | Minimal MCP server speaking Streamable HTTP. Supports `tools/list` (returns two deterministic tools: `echo` and `delay`) and `tools/call`. |

**Configuration:** entirely via env vars and query parameters. No config file. The backend should never be the bottleneck — it must be fast enough that any latency you measure comes from the gateway path.

**Resource recommendations (in chart values):** request 500m / 512Mi, limit 2 / 2Gi, replicas 2, anti-affinity on hostname so they spread.

### 5.2 Runner (`cmd/gw-bench`)

Single binary, two primary subcommands:

- `gw-bench run <scenario-file>` — execute one scenario through all gateways listed in it, emit events.
- `gw-bench report --run-id <id> [--from elastic|file]` — Phase 2+: query Elastic (or a local result file) and render a markdown comparison table.

Optionally:

- `gw-bench validate <scenario-file>` — schema-check a scenario.
- `gw-bench version` — version + build info.

**Inputs:**
- Scenario YAML (path or stdin).
- Env vars: `RUN_ID` (auto-generated as a ULID if absent — provides lexicographic sortability and embedded timestamp without collision risk), `CLUSTER_NAME`, `PROMETHEUS_URL`, `LOG_LEVEL`.

**Outputs:**
- Structured JSON events to stdout, one event per line (NDJSON-compatible).
- Exit 0 on success, non-zero on any error event.

**Subprocess management:**
- Uses `os/exec` with explicit timeout context (scenario duration + 60s buffer).
- Captures stdout/stderr to in-memory buffers.
- For fortio: invokes `fortio load -qps -t -c -json -` and parses the JSON result. Warmup is handled by running fortio twice: once for the warmup duration (output discarded), then again for the measurement duration (output captured and parsed).
- For k6: invokes `k6 run script.js` with a `handleSummary()` export function in the script that writes JSON summary to a known path. The runner reads and parses this file after k6 exits. (`--summary-export` is deprecated in modern k6 — use `handleSummary()` instead.)

### 5.3 Scenario schema

```yaml
apiVersion: gw-bench/v1
kind: Scenario
metadata:
  name: http-small-5k-qps
  description: |
    Small JSON payload at sustained 5k QPS over 100 connections.
    Tests connection-reuse efficiency at moderate load.
spec:
  protocol: http              # http | sse | mcp
  loadGenerator: fortio       # fortio | k6
  duration: 60s
  warmup: 10s                 # discarded from final metrics
  cooldown: 30s               # between gateways
  targetQPS: 5000             # 0 = saturate
  connections: 100
  payload:
    method: GET
    path: /echo
    sizeBytes: 0
    headers:
      X-Test-Run: gw-bench
  # For k6/MCP scenarios, use these instead of payload:
  # k6Script: scripts/k6/mcp-burst.js
  # k6Args:
  #   vus: 50
  gateways:
    - name: agentgateway
      url: http://test.agentgateway.internal/echo
      promPodSelector: 'app.kubernetes.io/name="agentgateway"'
    - name: ingress-nginx
      url: http://test.nginx.internal/echo
      promPodSelector: 'app.kubernetes.io/name="ingress-nginx"'
```

**Validation rules:**
- `duration` ≥ `warmup`.
- Exactly one of `payload` or `k6Script` present.
- `loadGenerator: fortio` requires `payload`. `loadGenerator: k6` requires `k6Script`.
- At least 1 entry in `gateways`. If only 1 gateway is present, the runner operates in **profile mode** (`--no-compare`): it runs the scenario against the single gateway, emits `scenario_complete`, but skips the `comparison` event. This is useful for baselining a new gateway before a comparison target exists.
- `protocol: mcp` requires `loadGenerator: k6`.

### 5.4 Event schema

All events are NDJSON, one per line, written to stdout. Common fields:

```go
type BaseEvent struct {
    Event     string    `json:"event"`
    Timestamp time.Time `json:"timestamp"` // RFC3339, UTC
    RunID     string    `json:"run_id"`
    Cluster   string    `json:"cluster,omitempty"`
    Version   string    `json:"gw_bench_version"`
}
```

**Event types:**

```jsonc
// run_start — emitted once at the top of each Job run
{
  "event": "run_start",
  "timestamp": "2026-05-06T14:00:00Z",
  "run_id": "01JX3K7M0000PITTSBURG00001",
  "cluster": "pittsburg",
  "scenario": "http-small-5k-qps",
  "gateways": ["agentgateway", "ingress-nginx"],
  "gw_bench_version": "0.1.0"
}

// scenario_start — once per gateway
{
  "event": "scenario_start",
  "timestamp": "...",
  "run_id": "...",
  "scenario": "http-small-5k-qps",
  "gateway": "agentgateway",
  "route": "http://test.agentgateway.internal/echo",
  "load_generator": "fortio"
}

// scenario_progress — emitted every 15s during long runs
{
  "event": "scenario_progress",
  "timestamp": "...",
  "run_id": "...",
  "gateway": "agentgateway",
  "elapsed_s": 30,
  "current_qps": 4912.3
}

// scenario_complete — once per gateway, the meat of the dataset
{
  "event": "scenario_complete",
  "timestamp": "...",
  "run_id": "...",
  "scenario": "http-small-5k-qps",
  "gateway": "agentgateway",
  "route": "...",
  "duration_s": 60,
  "warmup_s": 10,
  "target_qps": 5000,
  "actual_qps": 4823.4,
  "connections": 100,
  "metrics": {
    "p50_ms": 1.2,
    "p95_ms": 3.8,
    "p99_ms": 12.4,
    "p99_9_ms": 45.2,
    "min_ms": 0.4,
    "max_ms": 124.0,
    "mean_ms": 1.9,
    "errors": 12,
    "error_rate": 0.0001,
    "total_requests": 289404,
    // --- SSE-specific fields (present only when protocol=sse) ---
    "sse_events_received": 0,       // total SSE events across all connections
    "sse_stream_duration_ms": 0,    // avg time from first to last event per stream
    "sse_reconnections": 0,         // total reconnection attempts
    // --- MCP-specific fields (present only when protocol=mcp) ---
    "mcp_tool_calls": 0,            // total tools/call invocations
    "mcp_session_setup_ms": 0,      // avg initialize→initialized handshake time
    "mcp_streaming_first_token_ms": 0, // avg time to first streamed token in tool response
    "mcp_sessions_established": 0   // total MCP sessions opened
  },
  "gateway_resources": {
    "cpu_avg_cores": 0.42,
    "cpu_max_cores": 0.71,
    "mem_avg_mb": 145.0,
    "mem_max_mb": 162.0,
    "samples": 12
  },
  "load_generator": "fortio",
  "load_generator_version": "1.69.6"
}

// comparison — emitted once after both gateways complete
{
  "event": "comparison",
  "timestamp": "...",
  "run_id": "...",
  "scenario": "http-small-5k-qps",
  "winner_p99": "agentgateway",
  "delta": {
    "p50_pct": -8.3,           // negative = A faster than B
    "p95_pct": -15.2,
    "p99_pct": -23.4,
    "throughput_pct": 5.2,
    "cpu_avg_pct": 40.0,       // positive = A used more CPU
    "mem_avg_pct": -12.5
  },
  "a": "agentgateway",
  "b": "ingress-nginx"
}

// run_complete — final event
{
  "event": "run_complete",
  "timestamp": "...",
  "run_id": "...",
  "duration_s": 245,
  "exit_code": 0
}

// error — emitted on any failure, exits non-zero
{
  "event": "error",
  "timestamp": "...",
  "run_id": "...",
  "phase": "loadgen|prometheus|parse|setup",
  "gateway": "agentgateway",
  "message": "fortio exited with code 1: ...",
  "details": "..."
}
```

### 5.5 Helm chart

Chart at `deploy/helm/gw-bench/`. Top-level `values.yaml` keys:

```yaml
backend:
  enabled: true
  image:
    repository: ghcr.io/<owner>/gw-bench-backend
    tag: ""              # defaults to chart appVersion
  replicas: 2
  resources:
    requests: { cpu: 500m, memory: 512Mi }
    limits:   { cpu: 2,    memory: 2Gi   }
  affinity: {}           # populated by chart for anti-affinity by default

runner:
  image:
    repository: ghcr.io/<owner>/gw-bench
    tag: ""
  resources:
    requests: { cpu: 1, memory: 1Gi }
    limits:   { cpu: 4, memory: 4Gi }
  cronjob:
    enabled: false       # opt-in
    schedule: "0 2 * * 0" # weekly Sunday 02:00
  defaultScenario: http-small-5k-qps
  prometheus:
    url: http://prometheus-operated.monitoring.svc:9090

gateways:
  # The chart renders one HTTPRoute (or Ingress) per entry.
  # The runner reads gateway URLs from the scenario YAML, not these,
  # but this is what creates the routes for the backend.
  agentgateway:
    enabled: true
    type: gatewayApi      # gatewayApi | ingress
    parentRef:
      name: agentgateway
      namespace: agentgateway-system
    hostname: test.agentgateway.internal
  ingressNginx:
    enabled: true
    type: ingress
    ingressClassName: nginx
    hostname: test.nginx.internal

scenarios:
  # ConfigMap-mounted into the runner. Default ships with the canonical library.
  bundled: true
  custom: {}             # name -> YAML string

networkPolicy:
  enabled: true
  # Cilium-compatible; only allows runner -> gateways -> backend
```

**RBAC:** the runner ServiceAccount needs only:
- `get`, `list` on `Pods` in the gateway namespaces. This is used to resolve `promPodSelector` labels to actual pod names for Prometheus query scoping (e.g., building `pod=~"agentgateway-.*"` matchers). Not used for direct metrics collection.
- `get`, `list` on the chart's own ConfigMap (for scenario loading; covered by automounted SA).
- No write permissions anywhere.

If Prometheus is queried via the platform's central instance, no in-cluster RBAC is needed beyond the ServiceAccount token if Prometheus uses K8s SA auth; otherwise use a token from ESO.

**NetworkPolicy** (Cilium-compatible CRD or core NetworkPolicy):
- Runner egress: DNS, gateway services only, Prometheus URL, log pipeline.
- Backend ingress: only from the configured gateway namespaces (label-selected).
- Backend egress: DNS only.

---

## 6. Phased implementation plan

Each phase is a complete, mergeable, demoable unit. The agent should not begin a phase before the previous one is on `main` and lint/test green.

### Phase 0 — Skeleton (target: 1 PR)

- `go.mod` with module path `github.com/<owner>/gw-bench`.
- `Makefile` with targets: `build`, `test`, `lint`, `docker`, `helm-lint`.
- `cmd/gw-bench/main.go` with `cobra` CLI scaffolding: `run`, `validate`, `version`.
- `internal/events/events.go` with all event type structs and a `Emit(event)` function that marshals to JSON and writes to stdout with a flush.
- Basic GitHub Actions: `ci.yaml` runs `go test`, `golangci-lint`, `go build` on PR.
- README stub.

**Acceptance:** `gw-bench version` prints version, `gw-bench validate <missing-file>` returns a clear error, CI green.

### Phase 1 — Scenario loading and validation (target: 1 PR)

- `internal/config/scenario.go` with full struct definitions matching the schema in §5.3.
- `LoadScenario(path string) (*Scenario, error)` with YAML parsing.
- `Validate() error` enforcing all rules listed in §5.3.
- Unit tests covering: every validation rule, the canonical scenarios in `scenarios/`.
- Commit the canonical scenario library at `scenarios/`.

**Acceptance:** `gw-bench validate scenarios/http-small-5k-qps.yaml` exits 0 for all bundled scenarios, exits 1 with helpful errors for malformed input.

### Phase 2 — Fortio wrapper (target: 1 PR)

- `internal/loadgen/loadgen.go` with the `LoadGenerator` interface:
  ```go
  type LoadGenerator interface {
      Run(ctx context.Context, scenario *config.Scenario, target config.Gateway) (*Result, error)
  }
  ```
- `internal/loadgen/fortio.go` shells out to `fortio load`, parses JSON output into `Result`.
- Dockerfile for the runner image: `FROM fortio/fortio:latest AS fortio` then COPY the binary into a thin Go base image so both binaries are present.
- Integration test: spin up `httpbin` in a container, run a 5-second fortio scenario against it, verify a `Result` is parsed correctly.

**Acceptance:** `gw-bench run scenarios/http-small-5k-qps.yaml` produces valid `scenario_start` and `scenario_complete` events for both gateways when targets are reachable. (Use a local httpbin or echo for first verification.)

### Phase 3 — Test backend (target: 1 PR)

- `cmd/test-backend/main.go` with `/healthz`, `/echo`, `/sse` endpoints. (MCP comes in Phase 6.)
- Dockerfile, multi-arch build (amd64 + arm64).
- Image push to ghcr.io via release workflow.

**Acceptance:** backend image runs locally, all endpoints respond correctly, image < 30MB.

### Phase 4 — Helm chart (target: 1 PR)

- Full chart at `deploy/helm/gw-bench/` per §5.5.
- `helm-lint.yaml` workflow runs `helm lint` and `helm template` against `values.example.yaml`.
- `deploy/argocd/applicationset.yaml` example for two clusters (Pittsburg, Columbia) with cluster-specific overrides.
- `docs/deployment.md` covering ArgoCD wiring.

**Acceptance:** `helm install` against a kind cluster (with ingress-nginx and a stub agentgateway) deploys cleanly. `kubectl create job --from=...` runs the canonical scenario end-to-end and produces valid NDJSON in the Job logs.

### Phase 5 — Prometheus integration (target: 1 PR)

- `internal/metrics/prometheus.go` queries `container_cpu_usage_seconds_total` and `container_memory_working_set_bytes` for the gateway pods over the measurement window (excluding warmup).
- Uses `rate()` over the measurement window and lets Prometheus handle interpolation, rather than assuming a specific sample resolution. The actual number of samples depends on the target's `scrape_interval`; the runner records the sample count in `gateway_resources.samples` for transparency.
- Populate `gateway_resources` field in `scenario_complete` events.
- Graceful degradation: if Prometheus is unreachable, or if the expected metrics are not present (e.g., non-standard cgroup configurations, gateway pods running outside standard container runtimes), emit a warning event with the specific error but don't fail the run. Document in `docs/deployment.md` that accurate resource metrics require cAdvisor-sourced `container_*` metrics and a `scrape_interval` ≤ 15s on gateway targets.

**Acceptance:** running against a cluster with Prometheus produces non-zero `gateway_resources` numbers. Running without Prometheus configured produces empty `gateway_resources` and a logged warning.

### Phase 6 — k6 + MCP (target: 1 PR)

- `internal/loadgen/k6.go` shells out to k6. Scripts must export a `handleSummary()` function that writes JSON to a runner-specified path (env var `K6_SUMMARY_PATH`). The runner reads this file after k6 exits.
- Dockerfile updated to install k6 + xk6-infobip-mcp (or equivalent MCP extension) into the runner image. May require building k6 with `xk6 build`.
- `scripts/k6/mcp-burst.js` — drives MCP `tools/call` against the test backend's `/mcp` endpoint with realistic VU ramp and idle-time-between-calls. Exports `handleSummary()` for JSON result capture.
- Add MCP endpoint to the test backend (`cmd/test-backend/main.go`).
- New canonical scenario: `scenarios/mcp-tool-burst.yaml`.

**Acceptance:** `gw-bench run scenarios/mcp-tool-burst.yaml` produces `scenario_complete` events with MCP-specific metrics. K6 summary fields are correctly mapped into the standard `metrics` block.

### Phase 7 — Comparison report CLI (target: 1 PR)

- `gw-bench report --run-id <id>` subcommand.
- Two backends: `--from elastic` (queries Elastic for events with that `run_id`) and `--from file` (reads NDJSON from a path).
- Output formats: `--format markdown` (default), `--format json`.
- Sample Kibana saved searches and dashboard ndjson committed to `examples/kibana/`.
- `docs/kibana.md` with example KQL queries.

**Acceptance:** given a known `run_id` with two `scenario_complete` events in Elastic, `gw-bench report --run-id <id>` produces a side-by-side markdown table.

### Phase 8 — Hardening and polish (target: 1 PR)

- CronJob template for scheduled regression runs (opt-in via values).
- Multi-scenario runs: `gw-bench run scenarios/*.yaml` runs them sequentially with shared `run_id`.
- `docs/adding-a-gateway.md`.
- `docs/scenarios.md`.
- README polish, install instructions, screenshots of a Kibana view.
- Tag v0.1.0.

---

## 7. Testing strategy

### Unit
- All `internal/config` validation rules.
- All event struct marshaling produces stable JSON (golden files).
- `internal/loadgen` parsers against captured fortio/k6 JSON fixtures.
- `internal/metrics` query construction (mock Prometheus client).

### Integration
- A `make integration-test` target spins up `kind`, installs ingress-nginx via Helm, deploys a stub agentgateway (or a second nginx instance labeled differently), runs the canonical scenario, asserts NDJSON output shape.
- This runs in CI on every PR but is skippable with a label for fast iteration.

### Manual / smoke
- Lab cluster smoke test before any version tag: run all canonical scenarios end-to-end, eyeball the Kibana view.
- Document a "first run on a new cluster" checklist in `docs/deployment.md`.

### What we deliberately don't test
- Actual gateway performance correctness — we're not validating that the gateways do their jobs, we're measuring overhead.
- Long-running stability — Phase 8+ concern, not v0.1.

---

## 8. CI/CD

### `.github/workflows/ci.yaml` (PR + push to main)
- `golangci-lint` with the project's `.golangci.yml`.
- `go test ./...` with race detector.
- `go build ./...`.
- `helm lint deploy/helm/gw-bench/`.
- `helm template deploy/helm/gw-bench/ -f deploy/helm/gw-bench/values.example.yaml | kubeconform -strict`.

### `.github/workflows/release.yaml` (on git tag `v*`)
- Build multi-arch images (`amd64`, `arm64`) for `gw-bench` and `gw-bench-backend`, push to ghcr.io.
- Package the Helm chart, push to a `gh-pages` branch as a chart repo.
- Create a GitHub Release with auto-generated notes.

**Tag sanitization:** GitHub Releases API rejects forward slashes in tag names even though git accepts them. Keep tag format `vX.Y.Z` or `vX.Y.Z-rcN` only — no slashes.

---

## 9. Operational notes

### First deployment
- Lab cluster only. Don't run this in Pittsburg or Columbia until a full pass on lab is clean.
- Two replicas of the test backend with hostname anti-affinity.
- Schedule the runner Job on a node *different* from the gateway pods to avoid measuring noisy-neighbor effects from the load generator on the gateway itself.

### Resource sizing
- Runner Job: 1 CPU / 1Gi requests, 4 CPU / 4Gi limits is enough for fortio at 10k QPS. K6 with many VUs may need more.
- For target QPS > 20k, expect to need a beefier runner or distribute load across multiple Jobs (Phase 9+ concern).

### Things that will bite you
- **Connection reuse.** Make this an explicit scenario knob. The howardjohn benchmark found nginx 5–20× slower than alternatives purely from connection pooling differences.
- **Warm-up.** First 10–30s of any test is not representative. Always discard.
- **First-run JIT / cache effects.** Run a "throwaway" scenario before the real one if results look unstable. Document this in `docs/scenarios.md`.
- **Cilium TCP-establishment overhead.** If a scenario forces new TCP connections per request, you'll see Cilium CPU spike. That's accurate, but document it so results aren't misread.
- **Log shipping flush.** The 3-second sleep before exit is intentional. Don't remove it.

### Scope guards
If during implementation the agent finds itself reaching for any of these, stop and check with the human first:
- A custom results store (database, S3 layout, etc.) — out of scope, logs handle it.
- A web UI — Phase 4+, not now.
- Any persistent state in the runner — runner is stateless and idempotent.
- Authenticated MCP / LLM passthrough scenarios — Phase 5+, requires ESO + 1Password wiring.
- Multi-cluster orchestration from a single controller — out of scope; ArgoCD handles per-cluster deployment.

---

## 10. Definition of done for v0.1.0

- All Phase 0–8 PRs merged.
- A Helm install in a lab cluster, with two real gateways, produces valid NDJSON for all canonical scenarios.
- A `gw-bench report` against a real Elasticsearch index produces a readable markdown comparison.
- README has a copy-pasteable install + first-run section.
- `docs/architecture.md`, `docs/deployment.md`, `docs/scenarios.md`, `docs/adding-a-gateway.md`, `docs/kibana.md` exist and are not stubs.
- Tag `v0.1.0` cut, images on ghcr.io, chart on gh-pages.