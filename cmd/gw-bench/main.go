package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gw-bench",
		Short: "Kubernetes-native A/B benchmark harness for ingress/Gateway API implementations",
	}

	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newReportCmd())
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gw-bench %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <scenario-file>",
		Short: "Validate a scenario YAML file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scenario, err := loadAndValidateScenario(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("scenario %q is valid\n", scenario.Metadata.Name)
			return nil
		},
	}
}

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <scenario-file> [scenario-file...]",
		Short: "Execute benchmark scenario(s) against configured gateways",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScenarios(args)
		},
	}
}
