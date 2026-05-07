# Adding a Gateway

gw-bench is designed to be gateway-agnostic. To add a new gateway to your benchmark:

## 1. Install the gateway

Deploy the gateway in your cluster using its standard installation method.

## 2. Add routing in Helm values

Add an entry under `gateways` in your `values.yaml`:

### For Gateway API implementations

```yaml
gateways:
  myGateway:
    enabled: true
    type: gatewayApi
    parentRef:
      name: my-gateway          # Name of the Gateway resource
      namespace: my-gateway-ns  # Namespace of the Gateway resource
    hostname: test.mygateway.internal
```

### For Ingress implementations

```yaml
gateways:
  myIngress:
    enabled: true
    type: ingress
    ingressClassName: my-ingress-class
    hostname: test.myingress.internal
```

## 3. Add gateway to scenario files

Add the new gateway to your scenario YAML under `gateways`:

```yaml
gateways:
  - name: myGateway
    url: http://test.mygateway.internal/echo
    promPodSelector: 'app.kubernetes.io/name="my-gateway"'
```

## 4. Configure DNS

Ensure the gateway hostname resolves within the cluster. Options:

- CoreDNS rewrite rules
- In-cluster DNS entries
- `/etc/hosts` on the runner pod (not recommended)

## 5. Verify

```bash
# Deploy the updated chart
helm upgrade gw-bench deploy/helm/gw-bench/ -n gw-bench -f values.yaml

# Check routes are created
kubectl get httproute,ingress -n gw-bench

# Run a quick validation
kubectl create job gw-bench-test --from=cronjob/gw-bench-runner -n gw-bench
kubectl logs job/gw-bench-test -n gw-bench
```

## Notes

- The `promPodSelector` is used to query Prometheus for resource metrics. It should match the labels on the gateway pods. Use standard Prometheus label matchers.
- If your gateway doesn't expose standard `container_cpu_usage_seconds_total` / `container_memory_working_set_bytes` metrics, resource usage will be reported as empty (the benchmark still runs).
- For fair comparison, ensure all gateways are running on the same node class with similar resource limits.
