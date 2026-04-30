/*
Copyright © 2026 TAN KA-SHING<tankashing@icloud.com>
*/
package ma

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// weeksRequired is the number of weekly closes needed for a 200-week MA.
const weeksRequired = 200

type priceRow struct {
	Date     string  `json:"date"`
	AdjClose float64 `json:"adjClose"`
}

// Result holds the 200-week MA analysis for a single ticker.
type Result struct {
	Symbol      string
	LatestDate  string
	LatestPrice float64
	MA200W      float64
	// PctFromMA is positive when price is above MA, negative when below.
	// e.g. -15.3 means the price is 15.3% below the 200-week MA.
	PctFromMA float64
	BelowMA   bool
}

// ScanDir reads all *.json files in dataDir concurrently and returns 200-week MA
// results for every ticker that has enough history. Files with fewer than
// weeksRequired weekly closes are silently skipped.
// workers controls the goroutine pool size (0 → one goroutine per file).
func ScanDir(dataDir string, workers int) ([]Result, []error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, []error{fmt.Errorf("reading data dir %s: %w", dataDir, err)}
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ".json" {
			files = append(files, filepath.Join(dataDir, e.Name()))
		}
	}

	if workers <= 0 || workers > len(files) {
		workers = len(files)
	}

	type job struct{ path string }
	type outcome struct {
		result Result
		err    error
		skip   bool
	}

	jobs := make(chan job, len(files))
	outcomes := make(chan outcome, len(files))

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				r, skip, e := analyseFile(j.path)
				outcomes <- outcome{result: r, err: e, skip: skip}
			}
		}()
	}

	for _, f := range files {
		jobs <- job{path: f}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(outcomes)
	}()

	var results []Result
	var errs []error
	for o := range outcomes {
		if o.err != nil {
			errs = append(errs, o.err)
			continue
		}
		if !o.skip {
			results = append(results, o.result)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Symbol < results[j].Symbol
	})

	return results, errs
}

// analyseFile parses one JSON file and computes the 200-week MA.
// skip=true when the file has insufficient history (not an error).
func analyseFile(path string) (Result, bool, error) {
	symbol := strings.TrimSuffix(filepath.Base(path), ".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, false, fmt.Errorf("%s: reading file: %w", symbol, err)
	}

	// Skip non-array JSON files (e.g. state files stored in the same directory).
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return Result{}, true, nil
	}

	var rows []priceRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return Result{}, false, fmt.Errorf("%s: parsing JSON: %w", symbol, err)
	}

	// Keep only rows with a valid adjusted close.
	valid := rows[:0]
	for _, r := range rows {
		if r.AdjClose > 0 {
			valid = append(valid, r)
		}
	}
	rows = valid

	if len(rows) == 0 {
		return Result{}, true, nil
	}

	// Rows are already chronological from Tiingo; take the last close of each ISO week.
	weekly := weeklyCloses(rows)
	if len(weekly) < weeksRequired {
		return Result{}, true, nil
	}

	ma := mean(weekly[len(weekly)-weeksRequired:])
	latest := weekly[len(weekly)-1]
	pct := (latest.AdjClose - ma) / ma * 100

	return Result{
		Symbol:      symbol,
		LatestDate:  latest.Date,
		LatestPrice: latest.AdjClose,
		MA200W:      math.Round(ma*100) / 100,
		PctFromMA:   math.Round(pct*100) / 100,
		BelowMA:     latest.AdjClose < ma,
	}, false, nil
}

// isoWeek returns a stable key for the ISO week that date belongs to.
func isoWeek(date string) string {
	// date is "2006-01-02T..." — take only the date portion.
	d := date
	if i := strings.IndexByte(date, 'T'); i > 0 {
		d = date[:i]
	}
	var year, month, day int
	fmt.Sscanf(d, "%d-%d-%d", &year, &month, &day)
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	y, w := t.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}

func weeklyCloses(rows []priceRow) []priceRow {
	seen := make(map[string]priceRow, len(rows)/5)
	order := make([]string, 0, len(rows)/5)

	for _, r := range rows {
		key := isoWeek(r.Date)
		if _, exists := seen[key]; !exists {
			order = append(order, key)
		}
		// Always overwrite so we keep the last trading day of the week.
		seen[key] = r
	}

	out := make([]priceRow, len(order))
	for i, k := range order {
		out[i] = seen[k]
	}
	return out
}

func mean(rows []priceRow) float64 {
	var sum float64
	for _, r := range rows {
		sum += r.AdjClose
	}
	return sum / float64(len(rows))
}
