/*
Copyright © 2026 TAN KA-SHING<tankashing@icloud.com>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/orbulant/alpha-beta-sigma/internal/tiingo"
	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Download all S&P 500 data within Tiingo free-tier rate limits",
	Long: `Download daily OHLCV data for every S&P 500 constituent, automatically
respecting Tiingo's free-tier limits (50 req/hour, 1000 req/day).

Progress is saved to a state file after each symbol. Interrupted or failed
downloads are retried on the next run; successfully downloaded symbols are
skipped. Run the command again each day until all 500 symbols are complete.

Typical schedule:
  Day 1:  alpha-beta-sigma schedule   → downloads ~500 (hits daily cap)
  Day 2:  alpha-beta-sigma schedule   → downloads remaining symbols

To see current progress without downloading, use --status.
To retry previously failed symbols, use --retry-failed.

Set TIINGO_API_KEY in a .env file or environment, or pass --api-key.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load()

		outDir, _ := cmd.Flags().GetString("out")
		apiKey, _ := cmd.Flags().GetString("api-key")
		statePath, _ := cmd.Flags().GetString("state")
		showStatus, _ := cmd.Flags().GetBool("status")
		retryFailed, _ := cmd.Flags().GetBool("retry-failed")
		if apiKey == "" {
			apiKey = os.Getenv("TIINGO_API_KEY")
		}
		if !showStatus && apiKey == "" {
			fmt.Fprintln(os.Stderr, "error: Tiingo API key required — set TIINGO_API_KEY in .env or pass --api-key")
			os.Exit(1)
		}

		if statePath == "" {
			statePath = filepath.Join(outDir, ".schedule-state.json")
		}

		sched, err := tiingo.NewScheduler(apiKey, outDir, statePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading state: %v\n", err)
			os.Exit(1)
		}

		fmt.Print("Fetching current S&P 500 constituents from iShares IVV... ")
		symbols, err := tiingo.FetchSP500Symbols()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%d symbols\n", len(symbols))

		if showStatus {
			done, failed, pending := sched.Stats(symbols)
			fmt.Printf("State file: %s\n", statePath)
			fmt.Printf("  Done:    %d\n", done)
			fmt.Printf("  Failed:  %d\n", failed)
			fmt.Printf("  Pending: %d\n", pending)
			fmt.Printf("  Total:   %d\n", len(symbols))
			return
		}

		if retryFailed {
			sched.ResetFailed(symbols)
			fmt.Println("Failed symbols cleared — they will be retried this run.")
		}

		done0, _, _ := sched.Stats(symbols)
		fmt.Printf("State: %s\n", statePath)
		fmt.Printf("Output: %s\n", outDir)
		fmt.Printf("Symbols: %d total, %d already done\n\n", len(symbols), done0)

		runErr := sched.Run(symbols, func(sym string, n, done, total int, dlErr error) {
			if dlErr != nil {
				fmt.Printf("[%d/%d | done %d] FAIL  %s: %v\n", n, total, done, sym, dlErr)
			} else {
				fmt.Printf("[%d/%d | done %d] OK    %s\n", n, total, done, sym)
			}
		})

		done1, failed1, pending1 := sched.Stats(symbols)
		fmt.Printf("\nDone: %d  Failed: %d  Pending: %d  (of %d total)\n", done1, failed1, pending1, len(symbols))

		if errors.Is(runErr, tiingo.ErrDailyLimitReached) {
			fmt.Println("\nDaily request limit reached. Re-run tomorrow to continue.")
			os.Exit(2)
		}
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", runErr)
			os.Exit(1)
		}
		if failed1 > 0 {
			fmt.Printf("\n%d symbols failed. Re-run with --retry-failed to attempt them again.\n", failed1)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(scheduleCmd)

	scheduleCmd.Flags().StringP("out", "o", "data", "Output directory for downloaded JSON files")
	scheduleCmd.Flags().StringP("api-key", "k", "", "Tiingo API key (or set TIINGO_API_KEY in .env)")
	scheduleCmd.Flags().String("state", "", "Path to state file (default: <out>/.schedule-state.json)")
	scheduleCmd.Flags().Bool("status", false, "Print download progress and exit without downloading")
	scheduleCmd.Flags().Bool("retry-failed", false, "Clear previous failures so they are retried this run")
}
