package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValidScenario(t *testing.T) {
	yaml := `
apiVersion: gw-bench/v1
kind: Scenario
metadata:
  name: test-scenario
spec:
  protocol: http
  loadGenerator: fortio
  duration: 60s
  warmup: 10s
  cooldown: 30s
  targetQPS: 5000
  connections: 100
  payload:
    method: GET
    path: /echo
    sizeBytes: 0
  gateways:
    - name: gateway-a
      url: http://gateway-a.internal/echo
    - name: gateway-b
      url: http://gateway-b.internal/echo
`
	s, err := ParseScenario([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if s.Metadata.Name != "test-scenario" {
		t.Errorf("expected name 'test-scenario', got %q", s.Metadata.Name)
	}
	if len(s.Spec.Gateways) != 2 {
		t.Errorf("expected 2 gateways, got %d", len(s.Spec.Gateways))
	}
}

func TestValidation_BadAPIVersion(t *testing.T) {
	s := validScenario()
	s.APIVersion = "wrong/v2"
	assertValidationError(t, s, "unsupported apiVersion")
}

func TestValidation_BadKind(t *testing.T) {
	s := validScenario()
	s.Kind = "NotScenario"
	assertValidationError(t, s, "unsupported kind")
}

func TestValidation_MissingName(t *testing.T) {
	s := validScenario()
	s.Metadata.Name = ""
	assertValidationError(t, s, "metadata.name is required")
}

func TestValidation_BadProtocol(t *testing.T) {
	s := validScenario()
	s.Spec.Protocol = "grpc"
	assertValidationError(t, s, "unsupported protocol")
}

func TestValidation_BadLoadGenerator(t *testing.T) {
	s := validScenario()
	s.Spec.LoadGenerator = "wrk"
	assertValidationError(t, s, "unsupported loadGenerator")
}

func TestValidation_BadDuration(t *testing.T) {
	s := validScenario()
	s.Spec.Duration = "not-a-duration"
	assertValidationError(t, s, "invalid duration")
}

func TestValidation_NegativeDuration(t *testing.T) {
	s := validScenario()
	s.Spec.Duration = "-10s"
	assertValidationError(t, s, "duration must be positive")
}

func TestValidation_DurationNotGreaterThanWarmup(t *testing.T) {
	s := validScenario()
	s.Spec.Duration = "10s"
	s.Spec.Warmup = "10s"
	assertValidationError(t, s, "duration")
}

func TestValidation_BadWarmup(t *testing.T) {
	s := validScenario()
	s.Spec.Warmup = "not-a-duration"
	assertValidationError(t, s, "invalid warmup")
}

func TestValidation_BothPayloadAndK6Script(t *testing.T) {
	s := validScenario()
	s.Spec.K6Script = "script.js"
	assertValidationError(t, s, "exactly one of payload or k6Script")
}

func TestValidation_NeitherPayloadNorK6Script(t *testing.T) {
	s := validScenario()
	s.Spec.Payload = nil
	assertValidationError(t, s, "exactly one of payload or k6Script")
}

func TestValidation_FortioRequiresPayload(t *testing.T) {
	s := validScenario()
	s.Spec.LoadGenerator = "fortio"
	s.Spec.Payload = nil
	s.Spec.K6Script = "script.js"
	assertValidationError(t, s, "loadGenerator 'fortio' requires payload")
}

func TestValidation_K6RequiresK6Script(t *testing.T) {
	s := validScenario()
	s.Spec.LoadGenerator = "k6"
	assertValidationError(t, s, "loadGenerator 'k6' requires k6Script")
}

func TestValidation_MCPRequiresK6(t *testing.T) {
	s := validScenario()
	s.Spec.Protocol = "mcp"
	s.Spec.LoadGenerator = "fortio"
	assertValidationError(t, s, "protocol 'mcp' requires loadGenerator 'k6'")
}

func TestValidation_NoGateways(t *testing.T) {
	s := validScenario()
	s.Spec.Gateways = nil
	assertValidationError(t, s, "at least one gateway is required")
}

func TestValidation_GatewayMissingName(t *testing.T) {
	s := validScenario()
	s.Spec.Gateways[0].Name = ""
	assertValidationError(t, s, "gateways[0].name is required")
}

func TestValidation_GatewayMissingURL(t *testing.T) {
	s := validScenario()
	s.Spec.Gateways[0].URL = ""
	assertValidationError(t, s, "gateways[0].url is required")
}

func TestValidation_SingleGateway(t *testing.T) {
	s := validScenario()
	s.Spec.Gateways = []Gateway{{Name: "solo", URL: "http://solo.internal/echo"}}
	if err := s.Validate(); err != nil {
		t.Fatalf("single gateway should be valid (profile mode), got: %v", err)
	}
}

func TestValidation_SSEProtocol(t *testing.T) {
	s := validScenario()
	s.Spec.Protocol = "sse"
	if err := s.Validate(); err != nil {
		t.Fatalf("sse protocol should be valid, got: %v", err)
	}
}

func TestValidation_MCPWithK6(t *testing.T) {
	s := validScenario()
	s.Spec.Protocol = "mcp"
	s.Spec.LoadGenerator = "k6"
	s.Spec.Payload = nil
	s.Spec.K6Script = "scripts/k6/mcp-burst.js"
	if err := s.Validate(); err != nil {
		t.Fatalf("mcp with k6 should be valid, got: %v", err)
	}
}

func TestValidation_BadCooldown(t *testing.T) {
	s := validScenario()
	s.Spec.Cooldown = "not-a-duration"
	assertValidationError(t, s, "invalid cooldown")
}

func TestDurationHelpers(t *testing.T) {
	s := validScenario()
	if s.Spec.DurationSeconds() != 60 {
		t.Errorf("expected 60s duration, got %f", s.Spec.DurationSeconds())
	}
	if s.Spec.WarmupSeconds() != 10 {
		t.Errorf("expected 10s warmup, got %f", s.Spec.WarmupSeconds())
	}
	if s.Spec.MeasurementDuration().Seconds() != 50 {
		t.Errorf("expected 50s measurement, got %f", s.Spec.MeasurementDuration().Seconds())
	}
	if s.Spec.CooldownDuration().Seconds() != 30 {
		t.Errorf("expected 30s cooldown, got %f", s.Spec.CooldownDuration().Seconds())
	}
}

func TestCooldownDefault(t *testing.T) {
	s := validScenario()
	s.Spec.Cooldown = ""
	if s.Spec.CooldownDuration().Seconds() != 30 {
		t.Errorf("expected default 30s cooldown, got %f", s.Spec.CooldownDuration().Seconds())
	}
}

func TestCanonicalScenarios(t *testing.T) {
	scenarioDir := filepath.Join("..", "..", "scenarios")
	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		t.Skipf("skipping canonical scenario tests: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(scenarioDir, entry.Name())
			s, err := LoadScenario(path)
			if err != nil {
				t.Fatalf("failed to load: %v", err)
			}
			if err := s.Validate(); err != nil {
				t.Fatalf("validation failed: %v", err)
			}
		})
	}
}

func validScenario() *Scenario {
	return &Scenario{
		APIVersion: "gw-bench/v1",
		Kind:       "Scenario",
		Metadata:   Metadata{Name: "test"},
		Spec: Spec{
			Protocol:      "http",
			LoadGenerator: "fortio",
			Duration:      "60s",
			Warmup:        "10s",
			Cooldown:      "30s",
			TargetQPS:     5000,
			Connections:   100,
			Payload: &Payload{
				Method: "GET",
				Path:   "/echo",
			},
			Gateways: []Gateway{
				{Name: "gw-a", URL: "http://a.internal/echo"},
				{Name: "gw-b", URL: "http://b.internal/echo"},
			},
		},
	}
}

func assertValidationError(t *testing.T, s *Scenario, substr string) {
	t.Helper()
	err := s.Validate()
	if err == nil {
		t.Fatalf("expected validation error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("expected error containing %q, got: %v", substr, err)
	}
}
