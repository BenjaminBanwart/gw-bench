# Kibana Integration

## Index Pattern

Create an index pattern matching your log pipeline's index for gw-bench events (e.g., `logs-gw-bench-*`).

## Useful KQL Queries

### All events for a specific run
```
run_id: "01JX3K7M0000PITTSBURG00001"
```

### Comparison events only
```
event: "comparison"
```

### Scenario results for a specific gateway
```
event: "scenario_complete" AND gateway: "agentgateway"
```

### Error events
```
event: "error"
```

### P99 latency above threshold
```
event: "scenario_complete" AND metrics.p99_ms > 50
```

### Side-by-side comparison for a scenario
```
event: "scenario_complete" AND scenario: "http-small-5k-qps"
```

## Importing Saved Searches

Import the saved search and dashboard NDJSON files from `examples/kibana/`:

```bash
# Via Kibana UI: Management → Saved Objects → Import
# Or via API:
curl -X POST "https://kibana.internal/api/saved_objects/_import" \
  -H "kbn-xsrf: true" \
  --form file=@examples/kibana/saved-search-comparison.ndjson

curl -X POST "https://kibana.internal/api/saved_objects/_import" \
  -H "kbn-xsrf: true" \
  --form file=@examples/kibana/dashboard.ndjson
```

## Recommended Visualizations

1. **P99 Latency by Gateway** — Bar chart with `gateway` on X-axis, `metrics.p99_ms` on Y-axis
2. **Throughput Comparison** — Line chart with `timestamp` on X-axis, `actual_qps` split by `gateway`
3. **Error Rate Trend** — Line chart with `metrics.error_rate` over time
4. **Resource Usage** — Dual-axis chart with CPU and memory by gateway
