package loadgen

import (
	"testing"
	"time"
)

func TestParseFortioOutput(t *testing.T) {
	// Captured fixture from a real fortio run (simplified)
	fixture := []byte(`{
  "StartTime": "2026-05-06T14:00:00Z",
  "RequestedQPS": "5000",
  "RequestedDuration": "50s",
  "ActualQPS": 4823.4,
  "ActualDuration": 50000000000,
  "NumThreads": 100,
  "Version": "1.69.6",
  "DurationHistogram": {
    "Count": 241170,
    "Min": 0.0004,
    "Max": 0.124,
    "Sum": 458.223,
    "Avg": 0.0019,
    "StdDev": 0.003,
    "Percentiles": [
      {"Percentile": 50, "Value": 0.0012},
      {"Percentile": 75, "Value": 0.0020},
      {"Percentile": 90, "Value": 0.0030},
      {"Percentile": 95, "Value": 0.0038},
      {"Percentile": 99, "Value": 0.0124},
      {"Percentile": 99.9, "Value": 0.0452}
    ]
  },
  "RetCodes": {
    "200": 241158,
    "503": 12
  },
  "Sizes": {
    "Count": 241170,
    "Min": 0,
    "Max": 256,
    "Sum": 61739520,
    "Avg": 256,
    "StdDev": 0,
    "Percentiles": []
  }
}`)

	start := time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC)
	end := start.Add(50 * time.Second)

	result, err := parseFortioOutput(fixture, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActualQPS != 4823.4 {
		t.Errorf("expected ActualQPS 4823.4, got %f", result.ActualQPS)
	}
	if result.LoadGenVersion != "1.69.6" {
		t.Errorf("expected version '1.69.6', got %q", result.LoadGenVersion)
	}

	// Latency checks (values in ms)
	assertFloat(t, "P50Ms", result.P50Ms, 1.2)
	assertFloat(t, "P95Ms", result.P95Ms, 3.8)
	assertFloat(t, "P99Ms", result.P99Ms, 12.4)
	assertFloat(t, "P999Ms", result.P999Ms, 45.2)
	assertFloat(t, "MinMs", result.MinMs, 0.4)
	assertFloat(t, "MaxMs", result.MaxMs, 124.0)
	assertFloat(t, "MeanMs", result.MeanMs, 1.9)

	// Request/error checks
	if result.TotalReqs != 241170 {
		t.Errorf("expected TotalReqs 241170, got %d", result.TotalReqs)
	}
	if result.Errors != 12 {
		t.Errorf("expected 12 errors, got %d", result.Errors)
	}
	expectedErrorRate := 12.0 / 241170.0
	if diff := result.ErrorRate - expectedErrorRate; diff > 0.00001 || diff < -0.00001 {
		t.Errorf("expected error rate ~%f, got %f", expectedErrorRate, result.ErrorRate)
	}

	// Measurement window
	if !result.MeasurementStart.Equal(start) {
		t.Errorf("expected MeasurementStart %v, got %v", start, result.MeasurementStart)
	}
	if !result.MeasurementEnd.Equal(end) {
		t.Errorf("expected MeasurementEnd %v, got %v", end, result.MeasurementEnd)
	}
}

func TestParseFortioOutput_AllSuccess(t *testing.T) {
	fixture := []byte(`{
  "ActualQPS": 1000,
  "Version": "1.69.6",
  "DurationHistogram": {
    "Count": 50000,
    "Min": 0.001,
    "Max": 0.050,
    "Sum": 75.0,
    "Avg": 0.0015,
    "StdDev": 0.001,
    "Percentiles": [
      {"Percentile": 50, "Value": 0.001},
      {"Percentile": 99, "Value": 0.010}
    ]
  },
  "RetCodes": {"200": 50000}
}`)

	result, err := parseFortioOutput(fixture, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}
	if result.ErrorRate != 0 {
		t.Errorf("expected 0 error rate, got %f", result.ErrorRate)
	}
}

func TestParseFortioOutput_InvalidJSON(t *testing.T) {
	_, err := parseFortioOutput([]byte("not json"), time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func assertFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	diff := got - want
	if diff > 0.01 || diff < -0.01 {
		t.Errorf("%s: expected %f, got %f", name, want, got)
	}
}
