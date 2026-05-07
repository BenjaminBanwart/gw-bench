package main

import (
	"fmt"

	"github.com/BenjaminBanwart/gw-bench/internal/config"
)

func loadAndValidateScenario(path string) (*config.Scenario, error) {
	scenario, err := config.LoadScenario(path)
	if err != nil {
		return nil, fmt.Errorf("loading scenario: %w", err)
	}
	if err := scenario.Validate(); err != nil {
		return nil, fmt.Errorf("validating scenario: %w", err)
	}
	return scenario, nil
}
