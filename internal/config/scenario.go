package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Scenario represents a benchmark scenario configuration.
type Scenario struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       Spec     `yaml:"spec" json:"spec"`
}

// Metadata contains scenario identification.
type Metadata struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Spec contains the scenario specification.
type Spec struct {
	Protocol      string            `yaml:"protocol" json:"protocol"`
	LoadGenerator string            `yaml:"loadGenerator" json:"loadGenerator"`
	Duration      string            `yaml:"duration" json:"duration"`
	Warmup        string            `yaml:"warmup" json:"warmup"`
	Cooldown      string            `yaml:"cooldown,omitempty" json:"cooldown,omitempty"`
	TargetQPS     int               `yaml:"targetQPS" json:"targetQPS"`
	Connections   int               `yaml:"connections" json:"connections"`
	Payload       *Payload          `yaml:"payload,omitempty" json:"payload,omitempty"`
	K6Script      string            `yaml:"k6Script,omitempty" json:"k6Script,omitempty"`
	K6Args        map[string]string `yaml:"k6Args,omitempty" json:"k6Args,omitempty"`
	Gateways      []Gateway         `yaml:"gateways" json:"gateways"`
}

// Payload describes the HTTP request to send.
type Payload struct {
	Method    string            `yaml:"method" json:"method"`
	Path      string            `yaml:"path" json:"path"`
	SizeBytes int               `yaml:"sizeBytes,omitempty" json:"sizeBytes,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body      string            `yaml:"body,omitempty" json:"body,omitempty"`
}

// Gateway describes a gateway to test against.
type Gateway struct {
	Name            string `yaml:"name" json:"name"`
	URL             string `yaml:"url" json:"url"`
	PromPodSelector string `yaml:"promPodSelector,omitempty" json:"promPodSelector,omitempty"`
}

// LoadScenario reads and parses a scenario YAML file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading scenario file: %w", err)
	}
	return ParseScenario(data)
}

// ParseScenario parses scenario YAML from bytes.
func ParseScenario(data []byte) (*Scenario, error) {
	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing scenario YAML: %w", err)
	}
	return &s, nil
}

// Validate checks all scenario invariants.
func (s *Scenario) Validate() error {
	if s.APIVersion != "gw-bench/v1" {
		return fmt.Errorf("unsupported apiVersion: %q (expected gw-bench/v1)", s.APIVersion)
	}
	if s.Kind != "Scenario" {
		return fmt.Errorf("unsupported kind: %q (expected Scenario)", s.Kind)
	}
	if s.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	// Protocol validation
	switch s.Spec.Protocol {
	case "http", "sse", "mcp":
	default:
		return fmt.Errorf("unsupported protocol: %q (must be http, sse, or mcp)", s.Spec.Protocol)
	}

	// Load generator validation
	switch s.Spec.LoadGenerator {
	case "fortio", "k6":
	default:
		return fmt.Errorf("unsupported loadGenerator: %q (must be fortio or k6)", s.Spec.LoadGenerator)
	}

	// Duration parsing and validation
	dur, err := time.ParseDuration(s.Spec.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s.Spec.Duration, err)
	}
	if dur <= 0 {
		return fmt.Errorf("duration must be positive")
	}

	warmup, err := time.ParseDuration(s.Spec.Warmup)
	if err != nil {
		return fmt.Errorf("invalid warmup %q: %w", s.Spec.Warmup, err)
	}
	if warmup < 0 {
		return fmt.Errorf("warmup must be non-negative")
	}
	if dur <= warmup {
		return fmt.Errorf("duration (%s) must be greater than warmup (%s)", s.Spec.Duration, s.Spec.Warmup)
	}

	// Cooldown validation
	if s.Spec.Cooldown != "" {
		cd, err := time.ParseDuration(s.Spec.Cooldown)
		if err != nil {
			return fmt.Errorf("invalid cooldown %q: %w", s.Spec.Cooldown, err)
		}
		if cd < 0 {
			return fmt.Errorf("cooldown must be non-negative")
		}
	}

	// Payload / k6Script mutual exclusivity
	hasPayload := s.Spec.Payload != nil
	hasK6Script := s.Spec.K6Script != ""
	if hasPayload && hasK6Script {
		return fmt.Errorf("exactly one of payload or k6Script must be set, not both")
	}
	if !hasPayload && !hasK6Script {
		return fmt.Errorf("exactly one of payload or k6Script must be set")
	}

	// Load generator / source compatibility
	if s.Spec.LoadGenerator == "fortio" && !hasPayload {
		return fmt.Errorf("loadGenerator 'fortio' requires payload to be set")
	}
	if s.Spec.LoadGenerator == "k6" && !hasK6Script {
		return fmt.Errorf("loadGenerator 'k6' requires k6Script to be set")
	}

	// Protocol constraints
	if s.Spec.Protocol == "mcp" && s.Spec.LoadGenerator != "k6" {
		return fmt.Errorf("protocol 'mcp' requires loadGenerator 'k6'")
	}

	// Gateways
	if len(s.Spec.Gateways) == 0 {
		return fmt.Errorf("at least one gateway is required")
	}
	for i, gw := range s.Spec.Gateways {
		if gw.Name == "" {
			return fmt.Errorf("gateways[%d].name is required", i)
		}
		if gw.URL == "" {
			return fmt.Errorf("gateways[%d].url is required", i)
		}
	}

	return nil
}

// DurationParsed returns the parsed duration.
func (s *Spec) DurationParsed() time.Duration {
	d, _ := time.ParseDuration(s.Duration)
	return d
}

// WarmupParsed returns the parsed warmup duration.
func (s *Spec) WarmupParsed() time.Duration {
	d, _ := time.ParseDuration(s.Warmup)
	return d
}

// CooldownDuration returns the cooldown duration, defaulting to 30s.
func (s *Spec) CooldownDuration() time.Duration {
	if s.Cooldown == "" {
		return 30 * time.Second
	}
	d, _ := time.ParseDuration(s.Cooldown)
	return d
}

// DurationSeconds returns the duration in seconds.
func (s *Spec) DurationSeconds() float64 {
	return s.DurationParsed().Seconds()
}

// WarmupSeconds returns the warmup in seconds.
func (s *Spec) WarmupSeconds() float64 {
	return s.WarmupParsed().Seconds()
}

// MeasurementDuration returns the measurement duration (total minus warmup).
func (s *Spec) MeasurementDuration() time.Duration {
	return s.DurationParsed() - s.WarmupParsed()
}
