package compare

import (
	"strings"
	"testing"

	"github.com/BenjaminBanwart/gw-bench/internal/events"
)

const sampleNDJSON = `{"event":"run_start","timestamp":"2026-05-06T14:00:00Z","run_id":"TEST001","scenario":"http-small-5k-qps","gateways":["gateway-a","gateway-b"],"gw_bench_version":"0.1.0"}
{"event":"scenario_start","timestamp":"2026-05-06T14:00:01Z","run_id":"TEST001","scenario":"http-small-5k-qps","gateway":"gateway-a","route":"http://a.internal/echo","load_generator":"fortio","gw_bench_version":"0.1.0"}
{"event":"scenario_complete","timestamp":"2026-05-06T14:01:01Z","run_id":"TEST001","scenario":"http-small-5k-qps","gateway":"gateway-a","route":"http://a.internal/echo","duration_s":60,"warmup_s":10,"target_qps":5000,"actual_qps":4823.4,"connections":100,"metrics":{"p50_ms":1.2,"p95_ms":3.8,"p99_ms":12.4,"p99_9_ms":45.2,"min_ms":0.4,"max_ms":124.0,"mean_ms":1.9,"errors":12,"error_rate":0.0001,"total_requests":289404},"gateway_resources":{"cpu_avg_cores":0.42,"cpu_max_cores":0.71,"mem_avg_mb":145.0,"mem_max_mb":162.0,"samples":12},"load_generator":"fortio","load_generator_version":"1.69.6","gw_bench_version":"0.1.0"}
{"event":"scenario_complete","timestamp":"2026-05-06T14:02:31Z","run_id":"TEST001","scenario":"http-small-5k-qps","gateway":"gateway-b","route":"http://b.internal/echo","duration_s":60,"warmup_s":10,"target_qps":5000,"actual_qps":4590.1,"connections":100,"metrics":{"p50_ms":1.3,"p95_ms":4.5,"p99_ms":16.2,"p99_9_ms":55.0,"min_ms":0.5,"max_ms":150.0,"mean_ms":2.3,"errors":25,"error_rate":0.0002,"total_requests":275406},"gateway_resources":{"cpu_avg_cores":0.30,"cpu_max_cores":0.52,"mem_avg_mb":165.0,"mem_max_mb":180.0,"samples":12},"load_generator":"fortio","load_generator_version":"1.69.6","gw_bench_version":"0.1.0"}
{"event":"comparison","timestamp":"2026-05-06T14:02:32Z","run_id":"TEST001","scenario":"http-small-5k-qps","winner_p99":"gateway-a","delta":{"p50_pct":-7.7,"p95_pct":-15.6,"p99_pct":-23.5,"throughput_pct":5.1,"cpu_avg_pct":40.0,"mem_avg_pct":-12.1},"a":"gateway-a","b":"gateway-b","gw_bench_version":"0.1.0"}
{"event":"run_complete","timestamp":"2026-05-06T14:02:35Z","run_id":"TEST001","duration_s":155,"exit_code":0,"gw_bench_version":"0.1.0"}
`

func TestLoadFromReader(t *testing.T) {
	report, err := LoadFromReader(strings.NewReader(sampleNDJSON), "TEST001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.RunID != "TEST001" {
		t.Errorf("expected run_id TEST001, got %q", report.RunID)
	}
	if report.Scenario != "http-small-5k-qps" {
		t.Errorf("expected scenario http-small-5k-qps, got %q", report.Scenario)
	}
	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(report.Results))
	}
	if report.Comparison == nil {
		t.Fatal("expected comparison event")
	}
	if report.Comparison.WinnerP99 != "gateway-a" {
		t.Errorf("expected winner gateway-a, got %q", report.Comparison.WinnerP99)
	}
}

func TestLoadFromReader_FilterByRunID(t *testing.T) {
	_, err := LoadFromReader(strings.NewReader(sampleNDJSON), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent run_id")
	}
}

func TestFormatMarkdown(t *testing.T) {
	report, err := LoadFromReader(strings.NewReader(sampleNDJSON), "TEST001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md := FormatMarkdown(report)

	// Check essential content
	checks := []string{
		"Benchmark Report",
		"TEST001",
		"gateway-a",
		"gateway-b",
		"P50",
		"P95",
		"P99",
		"Actual QPS",
		"CPU Avg",
		"Winner",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("markdown should contain %q", check)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	report := &Report{
		RunID:    "TEST001",
		Scenario: "test",
		Results: []events.ScenarioCompleteEvent{
			{
				BaseEvent: events.BaseEvent{Event: "scenario_complete", RunID: "TEST001"},
				Gateway:   "gw-a",
				ActualQPS: 5000,
			},
		},
	}

	jsonStr, err := FormatJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(jsonStr, "TEST001") {
		t.Error("JSON should contain run_id")
	}
}

func TestFormatMarkdown_SingleGateway(t *testing.T) {
	report := &Report{
		RunID:    "SINGLE001",
		Scenario: "profile-test",
		Results: []events.ScenarioCompleteEvent{
			{
				BaseEvent: events.BaseEvent{Event: "scenario_complete", RunID: "SINGLE001"},
				Gateway:   "solo-gw",
				ActualQPS: 3000,
				Metrics: events.Metrics{
					P50Ms:     1.0,
					P99Ms:     10.0,
					TotalReqs: 150000,
				},
			},
		},
	}

	md := FormatMarkdown(report)
	if !strings.Contains(md, "solo-gw") {
		t.Error("should contain gateway name")
	}
	if strings.Contains(md, "Delta") {
		t.Error("should not contain delta column for single gateway")
	}
}
