/*
Copyright © 2026 TAN KA-SHING<tankashing@icloud.com>
*/
package cmd

import (
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"

	"github.com/orbulant/alpha-beta-sigma/internal/ma"
	"github.com/spf13/cobra"
)

var ma200wCmd = &cobra.Command{
	Use:   "ma200w",
	Short: "200-week moving average scan across all downloaded stock data",
	Long: `Scans all JSON files in the data directory concurrently and computes the
200-week moving average (SMA of the last closing price of each ISO week) for
every ticker with sufficient history.

Prints each symbol, its latest adjusted close, the 200-week MA value, and how
far the current price sits above or below that MA as a percentage.`,
	Run: func(cmd *cobra.Command, args []string) {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		workers, _ := cmd.Flags().GetInt("workers")
		belowOnly, _ := cmd.Flags().GetBool("below-only")

		results, errs := ma.ScanDir(dataDir, workers)

		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "warn: %v\n", e)
		}

		if len(results) == 0 {
			fmt.Fprintln(os.Stderr, "no results — check --data-dir path")
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SYMBOL\tDATE\tPRICE\t200W MA\tVS MA\tSTATUS")
		fmt.Fprintln(w, "------\t----\t-----\t-------\t------\t------")

		for _, r := range results {
			if belowOnly && !r.BelowMA {
				continue
			}
			status := "ABOVE"
			if r.BelowMA {
				status = "BELOW"
			}
			fmt.Fprintf(w, "%s\t%s\t%.4f\t%.4f\t%+.2f%%\t%s\n",
				r.Symbol, r.LatestDate, r.LatestPrice, r.MA200W, r.PctFromMA, status)
		}
		w.Flush()

		below := 0
		for _, r := range results {
			if r.BelowMA {
				below++
			}
		}
		fmt.Printf("\n%d tickers scanned — %d below 200-week MA (%.1f%%)\n",
			len(results), below, float64(below)/float64(len(results))*100)
	},
}

func init() {
	rootCmd.AddCommand(ma200wCmd)

	ma200wCmd.Flags().StringP("data-dir", "d", "data", "Directory containing the JSON stock data files")
	ma200wCmd.Flags().IntP("workers", "w", runtime.NumCPU(), "Number of concurrent goroutines for scanning")
	ma200wCmd.Flags().BoolP("below-only", "b", false, "Only show tickers currently trading below their 200-week MA")
}
