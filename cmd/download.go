/*
Copyright © 2026 TAN KA-SHING<tankashing@icloud.com>
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/orbulant/alpha-beta-sigma/internal/tiingo"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download S&P 500 daily OHLCV data from Tiingo",
	Long: `Download daily OHLCV CSV files for all S&P 500 constituents from Tiingo.

Files are saved as <SYMBOL>.csv inside the output directory.
Use --symbol to download a single ticker instead of the full index.

History starts from 1994-01-01 (~30+ years). Tiingo free tier allows
50 requests/hour and 1000 requests/day, so downloading all 500 symbols
requires at least 2 sessions. The default delay of 4s between requests
keeps well within the hourly cap.

Set TIINGO_API_KEY in a .env file or environment, or pass --api-key.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load .env if present; ignore error (file may not exist)
		_ = godotenv.Load()

		outDir, _ := cmd.Flags().GetString("out")
		symbol, _ := cmd.Flags().GetString("symbol")
		apiKey, _ := cmd.Flags().GetString("api-key")
		delayMs, _ := cmd.Flags().GetInt("delay-ms")

		if apiKey == "" {
			apiKey = os.Getenv("TIINGO_API_KEY")
		}
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "error: Tiingo API key required — set TIINGO_API_KEY in .env or pass --api-key")
			os.Exit(1)
		}

		delay := time.Duration(delayMs) * time.Millisecond

		if symbol != "" {
			path, err := tiingo.Download(symbol, apiKey, outDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Downloaded %s → %s\n", symbol, path)
			return
		}

		fmt.Print("Fetching current S&P 500 constituents from iShares IVV... ")
		symbols, err := tiingo.FetchSP500Symbols()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%d symbols\n\n", len(symbols))
		fmt.Printf("Downloading to %s (delay %s between requests)\n", outDir, delay)
		fmt.Printf("Note: Tiingo free tier = 50 req/hour, 1000 req/day. Full download requires 2 sessions.\n\n")

		results := tiingo.DownloadAll(symbols, apiKey, outDir, delay, func(sym string, n, total int, err error) {
			if err != nil {
				fmt.Printf("[%d/%d] FAIL  %s: %v\n", n, total, sym, err)
			} else {
				fmt.Printf("[%d/%d] OK    %s\n", n, total, sym)
			}
		})

		failed := 0
		for _, err := range results {
			if err != nil {
				failed++
			}
		}

		fmt.Printf("\nDone. %d succeeded, %d failed.\n", len(symbols)-failed, failed)
		if failed > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringP("out", "o", "data", "Output directory for CSV files")
	downloadCmd.Flags().StringP("symbol", "s", "", "Download a single symbol instead of the full S&P 500")
	downloadCmd.Flags().StringP("api-key", "k", "", "Tiingo API key (or set TIINGO_API_KEY in .env)")
	downloadCmd.Flags().Int("delay-ms", int(tiingo.DefaultDelay.Milliseconds()), "Delay between requests in milliseconds (free tier: keep ≥4000)")
}
