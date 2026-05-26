package loadgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/BenjaminBanwart/gw-bench/internal/config"
)

// Fortio implements LoadGenerator using the fortio binary.
type Fortio struct{}

// NewFortio creates a new Fortio load generator.
func NewFortio() *Fortio {
	return &Fortio{}
}

// fortioResult represents the JSON output from fortio.
type fortioResult struct {
	ActualQPS         float64          `json:"ActualQPS"`
	DurationSeconds   float64          `json:"ActualDuration"`
	RequestedQPS      string           `json:"RequestedQPS"`
	NumThreads        int              `json:"NumThreads"`
	Version           string           `json:"Version"`
	DurationHistogram fortioHistogram  `json:"DurationHistogram"`
	RetCodes          map[string]int64 `json:"RetCodes"`
	Sizes             fortioHistogram  `json:"Sizes"`
}

type fortioHistogram struct {
	Count       int64              `json:"Count"`
	Min         float64            `json:"Min"`
	Max         float64            `json:"Max"`
	Sum         float64            `json:"Sum"`
	Avg         float64            `json:"Avg"`
	StdDev      float64            `json:"StdDev"`
	Percentiles []fortioPercentile `json:"Percentiles"`
}

type fortioPercentile struct {
	Percentile float64 `json:"Percentile"`
	Value      float64 `json:"Value"`
}

// Run executes a fortio load test against the specified gateway.
func (f *Fortio) Run(ctx context.Context, scenario *config.Scenario, target config.Gateway) (*Result, error) {
	warmup := scenario.Spec.WarmupParsed()
	measurement := scenario.Spec.MeasurementDuration()
	timeout := scenario.Spec.DurationParsed() + 60*time.Second

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Warmup run (output discarded)
	if warmup > 0 {
		if err := f.runFortio(ctx, scenario, target, warmup, nil); err != nil {
			return nil, fmt.Errorf("warmup: %w", err)
		}
	}

	// Measurement run
	var stdout bytes.Buffer
	measureStart := time.Now()
	if err := f.runFortio(ctx, scenario, target, measurement, &stdout); err != nil {
		return nil, fmt.Errorf("measurement: %w", err)
	}
	measureEnd := time.Now()

	return parseFortioOutput(stdout.Bytes(), measureStart, measureEnd)
}

func (f *Fortio) runFortio(ctx context.Context, scenario *config.Scenario, target config.Gateway, duration time.Duration, stdout *bytes.Buffer) error {
	args := []string{
		"load",
		"-qps", fmt.Sprintf("%d", scenario.Spec.TargetQPS),
		"-t", duration.String(),
		"-c", fmt.Sprintf("%d", scenario.Spec.Connections),
		"-p", "50,95,99,99.9",
		"-json", "/dev/stdout",
	}

	if scenario.Spec.Payload != nil && scenario.Spec.Payload.Method != "" {
		args = append(args, "-X", scenario.Spec.Payload.Method)
	}
	if scenario.Spec.Payload != nil {
		for k, v := range scenario.Spec.Payload.Headers {
			args = append(args, "-H", fmt.Sprintf("%s: %s", k, v))
		}
		if scenario.Spec.Payload.Body != "" {
			args = append(args, "-payload", scenario.Spec.Payload.Body)
		}
	}

	// For large payloads (>1MB), increase Fortio's HTTP buffer and per-request timeout.
	const largePayloadThreshold = 1024 * 1024
	if scenario.Spec.Payload != nil && scenario.Spec.Payload.SizeBytes > largePayloadThreshold {
		bufKB := scenario.Spec.Payload.SizeBytes/1024 + 128
		args = append(args, "-httpbufferkb", fmt.Sprintf("%d", bufKB))
		// Per-request timeout proportional to payload size, minimum 15s.
		timeoutSec := scenario.Spec.Payload.SizeBytes/(5*1024*1024) + 15
		args = append(args, "-timeout", fmt.Sprintf("%ds", timeoutSec))
	}

	if scenario.Spec.NoKeepAlive {
		args = append(args, "-keepalive=false")
	}

	args = append(args, target.URL)

	cmd := exec.CommandContext(ctx, "fortio", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if stdout != nil {
		cmd.Stdout = stdout
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fortio exited with error: %w\nstderr: %s", err, stderr.String())
	}
	return nil
}

func parseFortioOutput(data []byte, measureStart, measureEnd time.Time) (*Result, error) {
	var fr fortioResult
	if err := json.Unmarshal(data, &fr); err != nil {
		return nil, fmt.Errorf("parsing fortio JSON output: %w", err)
	}

	result := &Result{
		ActualQPS:        fr.ActualQPS,
		LoadGenVersion:   fr.Version,
		MeasurementStart: measureStart,
		MeasurementEnd:   measureEnd,
	}

	hist := fr.DurationHistogram
	result.MinMs = hist.Min * 1000
	result.MaxMs = hist.Max * 1000
	result.MeanMs = hist.Avg * 1000
	result.TotalReqs = hist.Count

	// Extract percentiles
	for _, p := range hist.Percentiles {
		valMs := p.Value * 1000
		switch {
		case p.Percentile == 50:
			result.P50Ms = valMs
		case p.Percentile == 95:
			result.P95Ms = valMs
		case p.Percentile == 99:
			result.P99Ms = valMs
		case p.Percentile == 99.9:
			result.P999Ms = valMs
		}
	}

	// Calculate errors
	var totalOK, totalErr int64
	for code, count := range fr.RetCodes {
		if code == "200" || code == "201" || code == "204" {
			totalOK += count
		} else {
			totalErr += count
		}
	}
	result.Errors = totalErr
	total := totalOK + totalErr
	if total > 0 {
		result.ErrorRate = float64(totalErr) / float64(total)
	}

	return result, nil
}
