package loadgen

import (
	"testing"
	"time"
)

func TestParseK6Output(t *testing.T) {
	fixture := []byte(`{
  "metrics": {
    "http_req_duration": {
      "type": "trend",
      "contains": "time",
      "values": {
        "avg": 2.5,
        "min": 0.5,
        "med": 2.0,
        "max": 150.0,
        "p(50)": 2.0,
        "p(90)": 5.0,
        "p(95)": 8.0,
        "p(99)": 25.0,
        "p(99.9)": 120.0
      }
    },
    "http_reqs": {
      "type": "counter",
      "contains": "default",
      "values": {
        "count": 150000,
        "rate": 3000.5
      }
    },
    "http_req_failed": {
      "type": "rate",
      "contains": "default",
      "values": {
        "rate": 0.001,
        "passes": 150
      }
    },
    "mcp_tool_calls": {
      "type": "counter",
      "contains": "default",
      "values": {
        "count": 75000,
        "rate": 1500
      }
    },
    "mcp_session_setup": {
      "type": "trend",
      "contains": "time",
      "values": {
        "avg": 15.5,
        "min": 8.0,
        "max": 45.0
      }
    },
    "mcp_first_token": {
      "type": "trend",
      "contains": "time",
      "values": {
        "avg": 3.2,
        "min": 1.0,
        "max": 20.0
      }
    },
    "mcp_sessions": {
      "type": "counter",
      "contains": "default",
      "values": {
        "count": 500,
        "rate": 10
      }
    }
  }
}`)

	start := time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC)
	end := start.Add(50 * time.Second)

	result, err := parseK6Output(fixture, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActualQPS != 3000.5 {
		t.Errorf("expected ActualQPS 3000.5, got %f", result.ActualQPS)
	}

	// Latency checks
	assertFloat(t, "P50Ms", result.P50Ms, 2.0)
	assertFloat(t, "P95Ms", result.P95Ms, 8.0)
	assertFloat(t, "P99Ms", result.P99Ms, 25.0)
	assertFloat(t, "P999Ms", result.P999Ms, 120.0)
	assertFloat(t, "MinMs", result.MinMs, 0.5)
	assertFloat(t, "MaxMs", result.MaxMs, 150.0)
	assertFloat(t, "MeanMs", result.MeanMs, 2.5)

	// Request counts
	if result.TotalReqs != 150000 {
		t.Errorf("expected TotalReqs 150000, got %d", result.TotalReqs)
	}
	if result.Errors != 150 {
		t.Errorf("expected 150 errors, got %d", result.Errors)
	}

	// MCP-specific
	if result.MCPToolCalls != 75000 {
		t.Errorf("expected MCPToolCalls 75000, got %d", result.MCPToolCalls)
	}
	assertFloat(t, "MCPSessionSetupMs", result.MCPSessionSetupMs, 15.5)
	assertFloat(t, "MCPStreamingFirstToken", result.MCPStreamingFirstToken, 3.2)
	if result.MCPSessionsEstablished != 500 {
		t.Errorf("expected MCPSessionsEstablished 500, got %d", result.MCPSessionsEstablished)
	}
}

func TestParseK6Output_MinimalHTTP(t *testing.T) {
	fixture := []byte(`{
  "metrics": {
    "http_req_duration": {
      "type": "trend",
      "contains": "time",
      "values": {
        "avg": 1.0,
        "min": 0.5,
        "max": 5.0,
        "p(50)": 0.8,
        "p(95)": 2.0,
        "p(99)": 4.0
      }
    },
    "http_reqs": {
      "type": "counter",
      "contains": "default",
      "values": {
        "count": 10000,
        "rate": 500
      }
    }
  }
}`)

	result, err := parseK6Output(fixture, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalReqs != 10000 {
		t.Errorf("expected 10000 reqs, got %d", result.TotalReqs)
	}
	if result.MCPToolCalls != 0 {
		t.Errorf("expected 0 MCP calls for HTTP-only, got %d", result.MCPToolCalls)
	}
}

func TestParseK6Output_InvalidJSON(t *testing.T) {
	_, err := parseK6Output([]byte("not json"), time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
