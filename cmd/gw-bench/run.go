package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/BenjaminBanwart/gw-bench/internal/config"
	"github.com/BenjaminBanwart/gw-bench/internal/events"
	"github.com/BenjaminBanwart/gw-bench/internal/loadgen"
	"github.com/BenjaminBanwart/gw-bench/internal/metrics"
	"github.com/oklog/ulid/v2"
)

func runScenarios(paths []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	runID := os.Getenv("RUN_ID")
	if runID == "" {
		runID = ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
	}
	cluster := os.Getenv("CLUSTER_NAME")
	promURL := os.Getenv("PROMETHEUS_URL")

	emitter := events.NewEmitter(runID, cluster, version)

	for _, path := range paths {
		scenario, err := loadAndValidateScenario(path)
		if err != nil {
			emitter.EmitError("setup", "", err.Error(), "")
			return err
		}

		if err := runSingleScenario(ctx, emitter, scenario, promURL); err != nil {
			return err
		}
	}

	return nil
}

func runSingleScenario(ctx context.Context, emitter *events.Emitter, scenario *config.Scenario, promURL string) error {
	gwNames := make([]string, len(scenario.Spec.Gateways))
	for i, gw := range scenario.Spec.Gateways {
		gwNames[i] = gw.Name
	}

	emitter.EmitRunStart(scenario.Metadata.Name, gwNames)

	var completedEvents []events.ScenarioCompleteEvent
	var lg loadgen.LoadGenerator

	switch scenario.Spec.LoadGenerator {
	case "fortio":
		lg = loadgen.NewFortio()
	case "k6":
		lg = loadgen.NewK6()
	default:
		err := fmt.Errorf("unsupported load generator: %s", scenario.Spec.LoadGenerator)
		emitter.EmitError("setup", "", err.Error(), "")
		return err
	}

	for i, gw := range scenario.Spec.Gateways {
		emitter.EmitScenarioStart(scenario.Metadata.Name, gw.Name, gw.URL, scenario.Spec.LoadGenerator)

		result, err := lg.Run(ctx, scenario, gw)
		if err != nil {
			emitter.EmitError("loadgen", gw.Name, err.Error(), "")
			return fmt.Errorf("load generation for gateway %q: %w", gw.Name, err)
		}

		// Convert loadgen.Result to events.LoadGenResult to avoid import cycle
		lgResult := &events.LoadGenResult{
			ActualQPS:        result.ActualQPS,
			LoadGenVersion:   result.LoadGenVersion,
			MeasurementStart: result.MeasurementStart,
			MeasurementEnd:   result.MeasurementEnd,
			Metrics: events.Metrics{
				P50Ms:                  result.P50Ms,
				P95Ms:                  result.P95Ms,
				P99Ms:                  result.P99Ms,
				P999Ms:                 result.P999Ms,
				MinMs:                  result.MinMs,
				MaxMs:                  result.MaxMs,
				MeanMs:                 result.MeanMs,
				Errors:                 result.Errors,
				ErrorRate:              result.ErrorRate,
				TotalReqs:              result.TotalReqs,
				SSEEventsReceived:      result.SSEEventsReceived,
				SSEStreamDurationMs:    result.SSEStreamDurationMs,
				SSEReconnections:       result.SSEReconnections,
				MCPToolCalls:           result.MCPToolCalls,
				MCPSessionSetupMs:      result.MCPSessionSetupMs,
				MCPStreamingFirstToken: result.MCPStreamingFirstToken,
				MCPSessionsEstablished: result.MCPSessionsEstablished,
			},
		}

		var resources *events.GatewayResources
		if promURL != "" && gw.PromPodSelector != "" {
			promClient := metrics.NewPrometheusClient(promURL)
			res, err := promClient.QueryGatewayResources(ctx, gw.PromPodSelector, lgResult.MeasurementStart, lgResult.MeasurementEnd)
			if err != nil {
				emitter.EmitError("prometheus", gw.Name, err.Error(), "metrics collection failed, continuing without resource data")
			} else {
				resources = res
			}
		}

		completeEvt := emitter.EmitScenarioComplete(scenario, gw, lgResult, resources)
		completedEvents = append(completedEvents, completeEvt)

		// Cooldown between gateways (skip after last)
		if i < len(scenario.Spec.Gateways)-1 {
			cooldown := scenario.Spec.CooldownDuration()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cooldown):
			}
		}
	}

	// Emit comparison if ≥ 2 gateways
	if len(completedEvents) >= 2 {
		emitter.EmitComparison(scenario.Metadata.Name, completedEvents[0], completedEvents[1])
	}

	// Log pipeline flush
	time.Sleep(3 * time.Second)

	emitter.EmitRunComplete(0)

	return nil
}
