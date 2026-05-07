package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/BenjaminBanwart/gw-bench/internal/events"
)

// PrometheusClient queries Prometheus for gateway resource metrics.
type PrometheusClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPrometheusClient creates a new Prometheus client.
func NewPrometheusClient(baseURL string) *PrometheusClient {
	return &PrometheusClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type promQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string   `json:"metric"`
			Value  [2]json.RawMessage  `json:"value"`
			Values [][]json.RawMessage `json:"values"`
		} `json:"result"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

// QueryGatewayResources queries Prometheus for CPU and memory usage of gateway pods
// during the measurement window.
func (c *PrometheusClient) QueryGatewayResources(ctx context.Context, podSelector string, start, end time.Time) (*events.GatewayResources, error) {
	resources := &events.GatewayResources{}

	// Query CPU usage rate
	cpuQuery := fmt.Sprintf(`rate(container_cpu_usage_seconds_total{%s,container!="POD",container!=""}[1m])`, podSelector)
	cpuAvg, cpuMax, cpuSamples, err := c.queryRangeAggregates(ctx, cpuQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("querying CPU metrics: %w", err)
	}
	resources.CPUAvgCores = cpuAvg
	resources.CPUMaxCores = cpuMax
	resources.Samples = cpuSamples

	// Query memory usage
	memQuery := fmt.Sprintf(`container_memory_working_set_bytes{%s,container!="POD",container!=""}`, podSelector)
	memAvg, memMax, _, err := c.queryRangeAggregates(ctx, memQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("querying memory metrics: %w", err)
	}
	resources.MemAvgMB = memAvg / 1024 / 1024
	resources.MemMaxMB = memMax / 1024 / 1024

	return resources, nil
}

func (c *PrometheusClient) queryRangeAggregates(ctx context.Context, query string, start, end time.Time) (avg, max float64, samples int, err error) {
	step := "15s"
	params := url.Values{
		"query": {query},
		"start": {fmt.Sprintf("%d", start.Unix())},
		"end":   {fmt.Sprintf("%d", end.Unix())},
		"step":  {step},
	}

	reqURL := fmt.Sprintf("%s/api/v1/query_range?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result promQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, 0, fmt.Errorf("decoding response: %w", err)
	}
	if result.Status != "success" {
		return 0, 0, 0, fmt.Errorf("prometheus query failed: %s", result.Error)
	}

	var sum float64
	var count int
	var maxVal float64

	for _, series := range result.Data.Result {
		for _, pair := range series.Values {
			if len(pair) < 2 {
				continue
			}
			var val float64
			var valStr string
			if err := json.Unmarshal(pair[1], &valStr); err != nil {
				continue
			}
			if _, err := fmt.Sscanf(valStr, "%f", &val); err != nil {
				continue
			}
			sum += val
			count++
			if val > maxVal {
				maxVal = val
			}
		}
	}

	if count == 0 {
		return 0, 0, 0, nil
	}

	return sum / float64(count), maxVal, count, nil
}
