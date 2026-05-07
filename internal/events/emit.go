package events

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/BenjaminBanwart/gw-bench/internal/config"
)

// Emitter writes structured JSON events to stdout.
type Emitter struct {
	runID   string
	cluster string
	version string
	start   time.Time
}

// NewEmitter creates a new event emitter.
func NewEmitter(runID, cluster, version string) *Emitter {
	return &Emitter{
		runID:   runID,
		cluster: cluster,
		version: version,
		start:   time.Now(),
	}
}

func (e *Emitter) base(event string) BaseEvent {
	return BaseEvent{
		Event:     event,
		Timestamp: time.Now().UTC(),
		RunID:     e.runID,
		Cluster:   e.cluster,
		Version:   e.version,
	}
}

// Emit marshals an event to JSON and writes it to stdout as a single line.
func Emit(event any) {
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw-bench: failed to marshal event: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(data))
}

// EmitRunStart emits a run_start event.
func (e *Emitter) EmitRunStart(scenario string, gateways []string) {
	Emit(RunStartEvent{
		BaseEvent: e.base("run_start"),
		Scenario:  scenario,
		Gateways:  gateways,
	})
}

// EmitScenarioStart emits a scenario_start event.
func (e *Emitter) EmitScenarioStart(scenario, gateway, route, loadGenerator string) {
	Emit(ScenarioStartEvent{
		BaseEvent:     e.base("scenario_start"),
		Scenario:      scenario,
		Gateway:       gateway,
		Route:         route,
		LoadGenerator: loadGenerator,
	})
}

// LoadGenResult is the subset of load generator results needed by the emitter.
// This avoids an import cycle between events and loadgen.
type LoadGenResult struct {
	ActualQPS        float64
	LoadGenVersion   string
	MeasurementStart time.Time
	MeasurementEnd   time.Time
	Metrics          Metrics
}

// EmitScenarioComplete emits a scenario_complete event and returns it.
func (e *Emitter) EmitScenarioComplete(scenario *config.Scenario, gw config.Gateway, result *LoadGenResult, resources *GatewayResources) ScenarioCompleteEvent {
	evt := ScenarioCompleteEvent{
		BaseEvent:        e.base("scenario_complete"),
		Scenario:         scenario.Metadata.Name,
		Gateway:          gw.Name,
		Route:            gw.URL,
		DurationS:        scenario.Spec.DurationSeconds(),
		WarmupS:          scenario.Spec.WarmupSeconds(),
		TargetQPS:        scenario.Spec.TargetQPS,
		ActualQPS:        result.ActualQPS,
		Connections:      scenario.Spec.Connections,
		Metrics:          result.Metrics,
		GatewayResources: resources,
		LoadGenerator:    scenario.Spec.LoadGenerator,
		LoadGenVersion:   result.LoadGenVersion,
	}
	Emit(evt)
	return evt
}

// EmitComparison emits a comparison event between two gateway results.
func (e *Emitter) EmitComparison(scenario string, a, b ScenarioCompleteEvent) {
	delta := ComparisonDelta{}
	if b.Metrics.P50Ms != 0 {
		delta.P50Pct = pctDiff(a.Metrics.P50Ms, b.Metrics.P50Ms)
	}
	if b.Metrics.P95Ms != 0 {
		delta.P95Pct = pctDiff(a.Metrics.P95Ms, b.Metrics.P95Ms)
	}
	if b.Metrics.P99Ms != 0 {
		delta.P99Pct = pctDiff(a.Metrics.P99Ms, b.Metrics.P99Ms)
	}
	if b.ActualQPS != 0 {
		delta.ThroughputPct = pctDiff(a.ActualQPS, b.ActualQPS)
	}
	if a.GatewayResources != nil && b.GatewayResources != nil {
		if b.GatewayResources.CPUAvgCores != 0 {
			delta.CPUAvgPct = pctDiff(a.GatewayResources.CPUAvgCores, b.GatewayResources.CPUAvgCores)
		}
		if b.GatewayResources.MemAvgMB != 0 {
			delta.MemAvgPct = pctDiff(a.GatewayResources.MemAvgMB, b.GatewayResources.MemAvgMB)
		}
	}

	winnerP99 := a.Gateway
	if b.Metrics.P99Ms < a.Metrics.P99Ms {
		winnerP99 = b.Gateway
	}

	Emit(ComparisonEvent{
		BaseEvent: e.base("comparison"),
		Scenario:  scenario,
		WinnerP99: winnerP99,
		Delta:     delta,
		A:         a.Gateway,
		B:         b.Gateway,
	})
}

// EmitRunComplete emits a run_complete event.
func (e *Emitter) EmitRunComplete(exitCode int) {
	Emit(RunCompleteEvent{
		BaseEvent: e.base("run_complete"),
		DurationS: time.Since(e.start).Seconds(),
		ExitCode:  exitCode,
	})
}

// EmitError emits an error event.
func (e *Emitter) EmitError(phase, gateway, message, details string) {
	Emit(ErrorEvent{
		BaseEvent: e.base("error"),
		Phase:     phase,
		Gateway:   gateway,
		Message:   message,
		Details:   details,
	})
}

// pctDiff computes (a - b) / b * 100
func pctDiff(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return (a - b) / b * 100
}
