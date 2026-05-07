package main

import (
	"fmt"

	"github.com/BenjaminBanwart/gw-bench/internal/compare"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var (
		runID  string
		from   string
		format string
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a comparison report from benchmark results",
		RunE: func(cmd *cobra.Command, args []string) error {
			var report *compare.Report
			var err error

			switch from {
			case "file":
				if len(args) < 1 {
					return fmt.Errorf("file path required when using --from file")
				}
				report, err = compare.LoadFromFile(args[0], runID)
			default:
				return fmt.Errorf("unsupported source: %q (supported: file)", from)
			}

			if err != nil {
				return err
			}

			switch format {
			case "markdown":
				fmt.Print(compare.FormatMarkdown(report))
			case "json":
				out, err := compare.FormatJSON(report)
				if err != nil {
					return err
				}
				fmt.Println(out)
			default:
				return fmt.Errorf("unsupported format: %q (supported: markdown, json)", format)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&runID, "run-id", "", "Filter events by run ID")
	cmd.Flags().StringVar(&from, "from", "file", "Data source (file)")
	cmd.Flags().StringVar(&format, "format", "markdown", "Output format (markdown, json)")

	return cmd
}
