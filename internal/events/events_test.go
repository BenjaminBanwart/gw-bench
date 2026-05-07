package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBaseEventMarshal(t *testing.T) {
	evt := RunStartEvent{
		BaseEvent: BaseEvent{
			Event:     "run_start",
			Timestamp: time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
			RunID:     "TEST001",
			Cluster:   "cluster-a",
			Version:   "0.1.0",
		},
		Scenario: "http-small-5k-qps",
		Gateways: []string{"gateway-a", "gateway-b"},
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded["event"] != "run_start" {
		t.Errorf("expected event 'run_start', got %v", decoded["event"])
	}
	if decoded["run_id"] != "TEST001" {
		t.Errorf("expected run_id 'TEST001', got %v", decoded["run_id"])
	}
	if decoded["cluster"] != "cluster-a" {
		t.Errorf("expected cluster 'cluster-a', got %v", decoded["cluster"])
	}
	if decoded["scenario"] != "http-small-5k-qps" {
		t.Errorf("expected scenario name, got %v", decoded["scenario"])
	}
}

func TestScenarioCompleteEventMarshal(t *testing.T) {
	evt := ScenarioCompleteEvent{
		BaseEvent: BaseEvent{
			Event: "scenario_complete",
			RunID: "TEST001",
		},
		Scenario:    "test",
		Gateway:     "gw-a",
		ActualQPS:   4823.4,
		Connections: 100,
		Metrics: Metrics{
			P50Ms:     1.2,
			P99Ms:     12.4,
			TotalReqs: 289404,
			Errors:    12,
			ErrorRate: 0.0001,
		},
		GatewayResources: &GatewayResources{
			CPUAvgCores: 0.42,
			MemAvgMB:    145.0,
			Samples:     12,
		},
		LoadGenerator:  "fortio",
		LoadGenVersion: "1.69.6",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Verify round-trip
	var decoded ScenarioCompleteEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ActualQPS != 4823.4 {
		t.Errorf("expected ActualQPS 4823.4, got %f", decoded.ActualQPS)
	}
	if decoded.Metrics.P99Ms != 12.4 {
		t.Errorf("expected P99Ms 12.4, got %f", decoded.Metrics.P99Ms)
	}
	if decoded.GatewayResources == nil {
		t.Fatal("expected gateway_resources to be present")
	}
	if decoded.GatewayResources.CPUAvgCores != 0.42 {
		t.Errorf("expected CPU 0.42, got %f", decoded.GatewayResources.CPUAvgCores)
	}
}

func TestErrorEventMarshal(t *testing.T) {
	evt := ErrorEvent{
		BaseEvent: BaseEvent{
			Event: "error",
			RunID: "TEST001",
		},
		Phase:   "loadgen",
		Gateway: "gw-a",
		Message: "fortio exited with code 1",
		Details: "connection refused",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded["phase"] != "loadgen" {
		t.Errorf("expected phase 'loadgen', got %v", decoded["phase"])
	}
	if decoded["message"] != "fortio exited with code 1" {
		t.Errorf("unexpected message: %v", decoded["message"])
	}
}

func TestComparisonEventMarshal(t *testing.T) {
	evt := ComparisonEvent{
		BaseEvent: BaseEvent{
			Event: "comparison",
			RunID: "TEST001",
		},
		Scenario:  "test",
		WinnerP99: "gw-a",
		Delta: ComparisonDelta{
			P50Pct:        -8.3,
			P99Pct:        -23.4,
			ThroughputPct: 5.2,
			CPUAvgPct:     40.0,
		},
		A: "gw-a",
		B: "gw-b",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ComparisonEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.WinnerP99 != "gw-a" {
		t.Errorf("expected winner gw-a, got %q", decoded.WinnerP99)
	}
	if decoded.Delta.P99Pct != -23.4 {
		t.Errorf("expected P99 delta -23.4, got %f", decoded.Delta.P99Pct)
	}
}

func TestMCPMetricsOmitEmpty(t *testing.T) {
	evt := ScenarioCompleteEvent{
		BaseEvent: BaseEvent{Event: "scenario_complete"},
		Metrics: Metrics{
			P50Ms:     1.0,
			TotalReqs: 100,
		},
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	str := string(data)
	// MCP fields with zero values should be omitted
	if contains(str, "mcp_tool_calls") {
		t.Error("zero MCP fields should be omitted")
	}
	if contains(str, "sse_events_received") {
		t.Error("zero SSE fields should be omitted")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && jsonContains(s, substr)
}

func jsonContains(s, key string) bool {
	for i := 0; i <= len(s)-len(key); i++ {
		if s[i:i+len(key)] == key {
			return true
		}
	}
	return false
}
