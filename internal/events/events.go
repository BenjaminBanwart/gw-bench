package events

import "time"

// BaseEvent contains fields common to all events.
type BaseEvent struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	RunID     string    `json:"run_id"`
	Cluster   string    `json:"cluster,omitempty"`
	Version   string    `json:"gw_bench_version"`
}

// RunStartEvent is emitted once at the top of each Job run.
type RunStartEvent struct {
	BaseEvent
	Scenario string   `json:"scenario"`
	Gateways []string `json:"gateways"`
}

// ScenarioStartEvent is emitted once per gateway before load generation.
type ScenarioStartEvent struct {
	BaseEvent
	Scenario      string `json:"scenario"`
	Gateway       string `json:"gateway"`
	Route         string `json:"route"`
	LoadGenerator string `json:"load_generator"`
}

// ScenarioProgressEvent is emitted periodically during long runs.
type ScenarioProgressEvent struct {
	BaseEvent
	Gateway    string  `json:"gateway"`
	ElapsedS   int     `json:"elapsed_s"`
	CurrentQPS float64 `json:"current_qps"`
}

// Metrics contains latency and throughput measurements.
type Metrics struct {
	P50Ms     float64 `json:"p50_ms"`
	P95Ms     float64 `json:"p95_ms"`
	P99Ms     float64 `json:"p99_ms"`
	P999Ms    float64 `json:"p99_9_ms"`
	MinMs     float64 `json:"min_ms"`
	MaxMs     float64 `json:"max_ms"`
	MeanMs    float64 `json:"mean_ms"`
	Errors    int64   `json:"errors"`
	ErrorRate float64 `json:"error_rate"`
	TotalReqs int64   `json:"total_requests"`

	// SSE-specific
	SSEEventsReceived   int64   `json:"sse_events_received,omitempty"`
	SSEStreamDurationMs float64 `json:"sse_stream_duration_ms,omitempty"`
	SSEReconnections    int64   `json:"sse_reconnections,omitempty"`

	// MCP-specific
	MCPToolCalls           int64   `json:"mcp_tool_calls,omitempty"`
	MCPSessionSetupMs      float64 `json:"mcp_session_setup_ms,omitempty"`
	MCPStreamingFirstToken float64 `json:"mcp_streaming_first_token_ms,omitempty"`
	MCPSessionsEstablished int64   `json:"mcp_sessions_established,omitempty"`
}

// GatewayResources contains resource usage data for a gateway during the test.
type GatewayResources struct {
	CPUAvgCores float64 `json:"cpu_avg_cores"`
	CPUMaxCores float64 `json:"cpu_max_cores"`
	MemAvgMB    float64 `json:"mem_avg_mb"`
	MemMaxMB    float64 `json:"mem_max_mb"`
	Samples     int     `json:"samples"`
}

// ScenarioCompleteEvent is emitted once per gateway after load generation.
type ScenarioCompleteEvent struct {
	BaseEvent
	Scenario         string            `json:"scenario"`
	Gateway          string            `json:"gateway"`
	Route            string            `json:"route"`
	DurationS        float64           `json:"duration_s"`
	WarmupS          float64           `json:"warmup_s"`
	TargetQPS        int               `json:"target_qps"`
	ActualQPS        float64           `json:"actual_qps"`
	Connections      int               `json:"connections"`
	Metrics          Metrics           `json:"metrics"`
	GatewayResources *GatewayResources `json:"gateway_resources,omitempty"`
	LoadGenerator    string            `json:"load_generator"`
	LoadGenVersion   string            `json:"load_generator_version"`
}

// ComparisonDelta holds the percentage differences between two gateways.
type ComparisonDelta struct {
	P50Pct        float64 `json:"p50_pct"`
	P95Pct        float64 `json:"p95_pct"`
	P99Pct        float64 `json:"p99_pct"`
	ThroughputPct float64 `json:"throughput_pct"`
	CPUAvgPct     float64 `json:"cpu_avg_pct"`
	MemAvgPct     float64 `json:"mem_avg_pct"`
}

// ComparisonEvent is emitted once after both gateways complete.
type ComparisonEvent struct {
	BaseEvent
	Scenario  string          `json:"scenario"`
	WinnerP99 string          `json:"winner_p99"`
	Delta     ComparisonDelta `json:"delta"`
	A         string          `json:"a"`
	B         string          `json:"b"`
}

// RunCompleteEvent is the final event emitted.
type RunCompleteEvent struct {
	BaseEvent
	DurationS float64 `json:"duration_s"`
	ExitCode  int     `json:"exit_code"`
}

// ErrorEvent is emitted on any failure.
type ErrorEvent struct {
	BaseEvent
	Phase   string `json:"phase"`
	Gateway string `json:"gateway,omitempty"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
