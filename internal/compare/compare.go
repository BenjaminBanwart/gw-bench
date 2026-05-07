package compare

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BenjaminBanwart/gw-bench/internal/events"
)

// Report contains the data needed for a comparison report.
type Report struct {
	RunID      string
	Scenario   string
	Results    []events.ScenarioCompleteEvent
	Comparison *events.ComparisonEvent
}

// LoadFromFile reads NDJSON events from a file and builds a Report.
func LoadFromFile(path string, runID string) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer func() { _ = f.Close() }()
	return LoadFromReader(f, runID)
}

// LoadFromReader reads NDJSON events and builds a Report.
func LoadFromReader(r io.Reader, runID string) (*Report, error) {
	report := &Report{RunID: runID}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		var base events.BaseEvent
		if err := json.Unmarshal(line, &base); err != nil {
			continue
		}
		if runID != "" && base.RunID != runID {
			continue
		}
		if report.RunID == "" {
			report.RunID = base.RunID
		}

		switch base.Event {
		case "scenario_complete":
			var evt events.ScenarioCompleteEvent
			if err := json.Unmarshal(line, &evt); err != nil {
				continue
			}
			report.Results = append(report.Results, evt)
			report.Scenario = evt.Scenario
		case "comparison":
			var evt events.ComparisonEvent
			if err := json.Unmarshal(line, &evt); err != nil {
				continue
			}
			report.Comparison = &evt
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading events: %w", err)
	}

	if len(report.Results) == 0 {
		return nil, fmt.Errorf("no scenario_complete events found for run_id %q", runID)
	}

	return report, nil
}

// FormatMarkdown renders a comparison report as a markdown table.
func FormatMarkdown(report *Report) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Benchmark Report: %s\n\n", report.Scenario))
	sb.WriteString(fmt.Sprintf("**Run ID:** `%s`\n\n", report.RunID))

	if len(report.Results) == 0 {
		sb.WriteString("No results available.\n")
		return sb.String()
	}

	// Header
	sb.WriteString("| Metric |")
	for _, r := range report.Results {
		sb.WriteString(fmt.Sprintf(" %s |", r.Gateway))
	}
	if report.Comparison != nil {
		sb.WriteString(" Delta (%) |")
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|---|")
	for range report.Results {
		sb.WriteString("---|")
	}
	if report.Comparison != nil {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// Rows
	writeRow := func(label string, vals []string, delta string) {
		sb.WriteString(fmt.Sprintf("| %s |", label))
		for _, v := range vals {
			sb.WriteString(fmt.Sprintf(" %s |", v))
		}
		if report.Comparison != nil {
			sb.WriteString(fmt.Sprintf(" %s |", delta))
		}
		sb.WriteString("\n")
	}

	vals := func(f func(r events.ScenarioCompleteEvent) string) []string {
		result := make([]string, len(report.Results))
		for i, r := range report.Results {
			result[i] = f(r)
		}
		return result
	}

	var d events.ComparisonDelta
	if report.Comparison != nil {
		d = report.Comparison.Delta
	}

	writeRow("Actual QPS", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.1f", r.ActualQPS)
	}), fmtPct(d.ThroughputPct))

	writeRow("P50 (ms)", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.2f", r.Metrics.P50Ms)
	}), fmtPct(d.P50Pct))

	writeRow("P95 (ms)", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.2f", r.Metrics.P95Ms)
	}), fmtPct(d.P95Pct))

	writeRow("P99 (ms)", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.2f", r.Metrics.P99Ms)
	}), fmtPct(d.P99Pct))

	writeRow("P99.9 (ms)", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.2f", r.Metrics.P999Ms)
	}), "")

	writeRow("Min (ms)", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.2f", r.Metrics.MinMs)
	}), "")

	writeRow("Max (ms)", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.2f", r.Metrics.MaxMs)
	}), "")

	writeRow("Mean (ms)", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.2f", r.Metrics.MeanMs)
	}), "")

	writeRow("Total Requests", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%d", r.Metrics.TotalReqs)
	}), "")

	writeRow("Errors", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%d", r.Metrics.Errors)
	}), "")

	writeRow("Error Rate", vals(func(r events.ScenarioCompleteEvent) string {
		return fmt.Sprintf("%.4f%%", r.Metrics.ErrorRate*100)
	}), "")

	// Resource usage
	hasResources := false
	for _, r := range report.Results {
		if r.GatewayResources != nil {
			hasResources = true
			break
		}
	}

	if hasResources {
		sb.WriteString("\n### Resource Usage\n\n")
		sb.WriteString("| Metric |")
		for _, r := range report.Results {
			sb.WriteString(fmt.Sprintf(" %s |", r.Gateway))
		}
		if report.Comparison != nil {
			sb.WriteString(" Delta (%) |")
		}
		sb.WriteString("\n|---|")
		for range report.Results {
			sb.WriteString("---|")
		}
		if report.Comparison != nil {
			sb.WriteString("---|")
		}
		sb.WriteString("\n")

		writeRow("CPU Avg (cores)", vals(func(r events.ScenarioCompleteEvent) string {
			if r.GatewayResources == nil {
				return "N/A"
			}
			return fmt.Sprintf("%.3f", r.GatewayResources.CPUAvgCores)
		}), fmtPct(d.CPUAvgPct))

		writeRow("CPU Max (cores)", vals(func(r events.ScenarioCompleteEvent) string {
			if r.GatewayResources == nil {
				return "N/A"
			}
			return fmt.Sprintf("%.3f", r.GatewayResources.CPUMaxCores)
		}), "")

		writeRow("Mem Avg (MB)", vals(func(r events.ScenarioCompleteEvent) string {
			if r.GatewayResources == nil {
				return "N/A"
			}
			return fmt.Sprintf("%.1f", r.GatewayResources.MemAvgMB)
		}), fmtPct(d.MemAvgPct))

		writeRow("Mem Max (MB)", vals(func(r events.ScenarioCompleteEvent) string {
			if r.GatewayResources == nil {
				return "N/A"
			}
			return fmt.Sprintf("%.1f", r.GatewayResources.MemMaxMB)
		}), "")
	}

	// Winner
	if report.Comparison != nil {
		sb.WriteString(fmt.Sprintf("\n**P99 Winner:** %s\n", report.Comparison.WinnerP99))
	}

	return sb.String()
}

// FormatJSON renders the report as JSON.
func FormatJSON(report *Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func fmtPct(v float64) string {
	if v == 0 {
		return "-"
	}
	sign := ""
	if v > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.1f%%", sign, v)
}
