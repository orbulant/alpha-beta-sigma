/*
Copyright © 2026 TAN KA-SHING<tankashing@icloud.com>
*/
package tiingo

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// baseURL is the Tiingo end-of-day daily prices endpoint.
// Docs: https://www.tiingo.com/documentation/end-of-day
const baseURL = "https://api.tiingo.com/tiingo/daily/%s/prices?startDate=%s&endDate=%s&resampleFreq=daily"

// ivvHoldingsURL is the iShares IVV ETF daily holdings CSV published by BlackRock.
// It reflects the current S&P 500 constituents and requires no API key.
const ivvHoldingsURL = "https://www.ishares.com/us/products/239726/ishares-core-sp-500-etf/1467271812596.ajax?fileType=csv&fileName=IVV_holdings&dataType=fund"

// ivvPreambleLines is the number of metadata lines before the CSV header in the IVV holdings file.
const ivvPreambleLines = 9

// FetchSP500Symbols downloads the current IVV ETF holdings from BlackRock and returns
// the ticker symbols for all equity constituents, normalising known ticker differences
// between iShares and Tiingo (e.g. BRKB → BRK.B).
func FetchSP500Symbols() ([]string, error) {
	resp, err := http.Get(ivvHoldingsURL)
	if err != nil {
		return nil, fmt.Errorf("fetching IVV holdings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching IVV holdings: unexpected status %d", resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// Skip preamble lines before the actual CSV header.
	for range ivvPreambleLines {
		if _, err := r.Read(); err != nil {
			return nil, fmt.Errorf("reading IVV preamble: %w", err)
		}
	}

	// Read and discard the header row.
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("reading IVV header: %w", err)
	}

	// iShares uses different ticker spellings for a handful of symbols.
	normalize := map[string]string{
		"BRKB": "BRK.B",
		"BFB":  "BF.B",
	}

	var symbols []string
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading IVV holdings row: %w", err)
		}
		// Columns: Ticker(0), Name(1), Sector(2), Asset Class(3), ...
		if len(row) < 4 {
			continue
		}
		ticker := strings.TrimSpace(row[0])
		assetClass := strings.TrimSpace(row[3])
		if ticker == "" || assetClass != "Equity" {
			continue
		}
		if canonical, ok := normalize[ticker]; ok {
			ticker = canonical
		}
		symbols = append(symbols, ticker)
	}

	if len(symbols) == 0 {
		return nil, fmt.Errorf("IVV holdings parsed but no equity symbols found — format may have changed")
	}

	return symbols, nil
}

// startDate requests the earliest practical history available on Tiingo (~30+ years).
const startDate = "1994-01-01"

// Free tier: 50 requests/hour, 1000 requests/day.
// At 500 symbols we need 2 runs to stay within the daily limit.
// DefaultDelay of 4s gives ~15 req/min (900/hour) — safely under the 50/hour cap
// if batching is done in one session; set to 0 to go as fast as possible.
const DefaultDelay = 4 * time.Second

type priceRow struct {
	Date        string  `json:"date"`
	Open        float64 `json:"open"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Close       float64 `json:"close"`
	Volume      float64 `json:"volume"`
	AdjOpen     float64 `json:"adjOpen"`
	AdjHigh     float64 `json:"adjHigh"`
	AdjLow      float64 `json:"adjLow"`
	AdjClose    float64 `json:"adjClose"`
	AdjVolume   float64 `json:"adjVolume"`
	DivCash     float64 `json:"divCash"`
	SplitFactor float64 `json:"splitFactor"`
}

// tiingoTicker sanitises a ticker symbol for use in Tiingo API URLs.
// Tiingo uses BRKB instead of BRK.B — dots are stripped.
func tiingoTicker(symbol string) string {
	return strings.ReplaceAll(symbol, ".", "")
}

// Download fetches the full available daily OHLCV history for a symbol from Tiingo
// and writes it as JSON to outDir/<SYMBOL>.json.
// apiKey is required — store it in TIINGO_API_KEY or pass via --api-key.
func Download(symbol, apiKey, outDir string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Tiingo API key is required (--api-key or TIINGO_API_KEY)")
	}

	today := time.Now().Format("2006-01-02")
	url := fmt.Sprintf(baseURL, tiingoTicker(symbol), startDate, today)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("building request for %s: %w", symbol, err)
	}
	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("symbol %s not found on Tiingo", symbol)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limit hit for %s — slow down requests or wait before retrying", symbol)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d for symbol %s", resp.StatusCode, symbol)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response for %s: %w", symbol, err)
	}

	bodyStr := strings.TrimSpace(string(body))
	if bodyStr == "" || bodyStr == "[]" {
		return "", fmt.Errorf("no data returned for %s (delisted or no history from %s)", symbol, startDate)
	}

	var rows []priceRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return "", fmt.Errorf("parsing response for %s: %w", symbol, err)
	}

	pretty, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding JSON for %s: %w", symbol, err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir %s: %w", outDir, err)
	}

	outPath := filepath.Join(outDir, symbol+".json")
	if err := os.WriteFile(outPath, pretty, 0o644); err != nil {
		return "", fmt.Errorf("writing file %s: %w", outPath, err)
	}

	return outPath, nil
}

// DownloadAll downloads data for every symbol, writing files to outDir.
// delay is inserted between requests to respect Tiingo's rate limits (use DefaultDelay for free tier).
// Reports per-symbol progress via the progress callback (may be nil).
// Returns a map of symbol → error; successful downloads have a nil value.
func DownloadAll(symbols []string, apiKey, outDir string, delay time.Duration, progress func(symbol string, n, total int, err error)) map[string]error {
	results := make(map[string]error, len(symbols))
	for i, sym := range symbols {
		_, err := Download(sym, apiKey, outDir)
		results[sym] = err
		if progress != nil {
			progress(sym, i+1, len(symbols), err)
		}
		if i < len(symbols)-1 && delay > 0 {
			time.Sleep(delay)
		}
	}
	return results
}

