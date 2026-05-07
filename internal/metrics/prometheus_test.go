package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryGatewayResources_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var result promQueryResult
		result.Status = "success"
		result.Data.ResultType = "matrix"

		if query != "" && len(query) > 4 && query[:4] == "rate" {
			// CPU query
			result.Data.Result = []struct {
				Metric map[string]string   `json:"metric"`
				Value  [2]json.RawMessage  `json:"value"`
				Values [][]json.RawMessage `json:"values"`
			}{
				{
					Metric: map[string]string{"pod": "gw-pod-1"},
					Values: [][]json.RawMessage{
						{json.RawMessage(`1609459200`), json.RawMessage(`"0.42"`)},
						{json.RawMessage(`1609459215`), json.RawMessage(`"0.55"`)},
						{json.RawMessage(`1609459230`), json.RawMessage(`"0.71"`)},
					},
				},
			}
		} else {
			// Memory query
			result.Data.Result = []struct {
				Metric map[string]string   `json:"metric"`
				Value  [2]json.RawMessage  `json:"value"`
				Values [][]json.RawMessage `json:"values"`
			}{
				{
					Metric: map[string]string{"pod": "gw-pod-1"},
					Values: [][]json.RawMessage{
						{json.RawMessage(`1609459200`), json.RawMessage(`"152043520"`)},
						{json.RawMessage(`1609459215`), json.RawMessage(`"156237824"`)},
						{json.RawMessage(`1609459230`), json.RawMessage(`"169869312"`)},
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := NewPrometheusClient(server.URL)
	start := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(60 * time.Second)

	resources, err := client.QueryGatewayResources(context.Background(), `app="test"`, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resources.Samples != 3 {
		t.Errorf("expected 3 samples, got %d", resources.Samples)
	}
	if resources.CPUAvgCores == 0 {
		t.Error("expected non-zero CPU avg")
	}
	if resources.CPUMaxCores == 0 {
		t.Error("expected non-zero CPU max")
	}
	if resources.MemAvgMB == 0 {
		t.Error("expected non-zero memory avg")
	}
	if resources.MemMaxMB == 0 {
		t.Error("expected non-zero memory max")
	}
}

func TestQueryGatewayResources_PrometheusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(promQueryResult{
			Status: "error",
			Error:  "bad query",
		})
	}))
	defer server.Close()

	client := NewPrometheusClient(server.URL)
	_, err := client.QueryGatewayResources(context.Background(), `app="test"`, time.Now(), time.Now().Add(time.Minute))
	if err == nil {
		t.Fatal("expected error for failed Prometheus query")
	}
}

func TestQueryGatewayResources_Unreachable(t *testing.T) {
	client := NewPrometheusClient("http://localhost:1")
	_, err := client.QueryGatewayResources(context.Background(), `app="test"`, time.Now(), time.Now().Add(time.Minute))
	if err == nil {
		t.Fatal("expected error for unreachable Prometheus")
	}
}

func TestQueryGatewayResources_EmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := promQueryResult{Status: "success"}
		result.Data.ResultType = "matrix"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := NewPrometheusClient(server.URL)
	resources, err := client.QueryGatewayResources(context.Background(), `app="test"`, time.Now(), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resources.CPUAvgCores != 0 {
		t.Errorf("expected 0 CPU avg for empty result, got %f", resources.CPUAvgCores)
	}
	if resources.Samples != 0 {
		t.Errorf("expected 0 samples for empty result, got %d", resources.Samples)
	}
}
