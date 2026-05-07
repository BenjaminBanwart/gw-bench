package loadgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/BenjaminBanwart/gw-bench/internal/config"
)

// K6 implements LoadGenerator using the k6 binary with xk6-infobip-mcp.
type K6 struct{}

// NewK6 creates a new K6 load generator.
func NewK6() *K6 {
	return &K6{}
}

// k6Summary represents the JSON summary output from k6's handleSummary().
type k6Summary struct {
	Metrics map[string]k6MetricData `json:"metrics"`
}

type k6MetricData struct {
	Type     string             `json:"type"`
	Contains string             `json:"contains"`
	Values   map[string]float64 `json:"values"`
}

// Run executes a k6 load test against the specified gateway.
func (k *K6) Run(ctx context.Context, scenario *config.Scenario, target config.Gateway) (*Result, error) {
	timeout := scenario.Spec.DurationParsed() + 60*time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	summaryPath := filepath.Join(os.TempDir(), fmt.Sprintf("k6-summary-%d.json", time.Now().UnixNano()))
	defer os.Remove(summaryPath)

	args := []string{
		"run",
		"--env", fmt.Sprintf("TARGET_URL=%s", target.URL),
		"--env", fmt.Sprintf("K6_SUMMARY_PATH=%s", summaryPath),
	}

	if scenario.Spec.K6Args != nil {
		if vus, ok := scenario.Spec.K6Args["vus"]; ok {
			args = append(args, "--vus", vus)
		}
		if dur, ok := scenario.Spec.K6Args["duration"]; ok {
			args = append(args, "--duration", dur)
		}
	}

	args = append(args, scenario.Spec.K6Script)

	measureStart := time.Now()

	cmd := exec.CommandContext(ctx, "k6", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stderr // k6 progress goes to stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("k6 exited with error: %w\nstderr: %s", err, stderr.String())
	}

	measureEnd := time.Now()

	summaryData, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, fmt.Errorf("reading k6 summary: %w", err)
	}

	return parseK6Output(summaryData, measureStart, measureEnd)
}

func parseK6Output(data []byte, measureStart, measureEnd time.Time) (*Result, error) {
	var summary k6Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parsing k6 summary JSON: %w", err)
	}

	result := &Result{
		LoadGenVersion:   "k6",
		MeasurementStart: measureStart,
		MeasurementEnd:   measureEnd,
	}

	// Parse http_req_duration metrics
	if dur, ok := summary.Metrics["http_req_duration"]; ok {
		result.P50Ms = dur.Values["p(50)"]
		result.P95Ms = dur.Values["p(95)"]
		result.P99Ms = dur.Values["p(99)"]
		result.P999Ms = dur.Values["p(99.9)"]
		result.MinMs = dur.Values["min"]
		result.MaxMs = dur.Values["max"]
		result.MeanMs = dur.Values["avg"]
	}

	// Parse request counts
	if reqs, ok := summary.Metrics["http_reqs"]; ok {
		result.TotalReqs = int64(reqs.Values["count"])
		result.ActualQPS = reqs.Values["rate"]
	}

	// Parse errors
	if fails, ok := summary.Metrics["http_req_failed"]; ok {
		result.Errors = int64(fails.Values["passes"]) // "passes" = count of failures in http_req_failed
		if result.TotalReqs > 0 {
			result.ErrorRate = float64(result.Errors) / float64(result.TotalReqs)
		}
	}

	// MCP-specific metrics (from custom k6 metrics)
	if calls, ok := summary.Metrics["mcp_tool_calls"]; ok {
		result.MCPToolCalls = int64(calls.Values["count"])
	}
	if setup, ok := summary.Metrics["mcp_session_setup"]; ok {
		result.MCPSessionSetupMs = setup.Values["avg"]
	}
	if ft, ok := summary.Metrics["mcp_first_token"]; ok {
		result.MCPStreamingFirstToken = ft.Values["avg"]
	}
	if sessions, ok := summary.Metrics["mcp_sessions"]; ok {
		result.MCPSessionsEstablished = int64(sessions.Values["count"])
	}

	return result, nil
}
