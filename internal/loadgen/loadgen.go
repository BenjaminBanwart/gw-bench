package loadgen

import (
	"context"
	"time"

	"github.com/BenjaminBanwart/gw-bench/internal/config"
)

// LoadGenerator runs a load test scenario against a target gateway.
type LoadGenerator interface {
	Run(ctx context.Context, scenario *config.Scenario, target config.Gateway) (*Result, error)
}

// Result contains the parsed output from a load generator run.
type Result struct {
	ActualQPS        float64
	LoadGenVersion   string
	MeasurementStart time.Time
	MeasurementEnd   time.Time

	// Latency metrics
	P50Ms  float64
	P95Ms  float64
	P99Ms  float64
	P999Ms float64
	MinMs  float64
	MaxMs  float64
	MeanMs float64

	// Request metrics
	Errors    int64
	ErrorRate float64
	TotalReqs int64

	// SSE-specific
	SSEEventsReceived   int64
	SSEStreamDurationMs float64
	SSEReconnections    int64

	// MCP-specific
	MCPToolCalls           int64
	MCPSessionSetupMs      float64
	MCPStreamingFirstToken float64
	MCPSessionsEstablished int64
}
