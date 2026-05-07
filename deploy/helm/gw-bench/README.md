# gw-bench Helm Chart

A Helm chart for deploying the gw-bench benchmark harness.

## Installation

```bash
helm install gw-bench deploy/helm/gw-bench/ -n gw-bench --create-namespace -f values.yaml
```

## Running a benchmark

```bash
kubectl create job gw-bench-run-001 --from=cronjob/gw-bench-runner -n gw-bench
```

## Values

See `values.yaml` for all configurable values and `values.example.yaml` for a ready-to-use example.
