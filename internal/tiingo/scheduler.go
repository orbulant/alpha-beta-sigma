/*
Copyright © 2026 TAN KA-SHING<tankashing@icloud.com>
*/
package tiingo

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Free-tier hard limits. We stop slightly below the daily cap to leave headroom.
const (
	HourlyLimit = 50
	DailyLimit  = 1000
	dailySafe   = 990 // stop before hitting 1000 exactly
)

// SymbolState tracks the download result for a single ticker.
type SymbolState struct {
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"` // last error message, empty on success
}

// State is persisted to disk so downloads can resume across sessions.
type State struct {
	Symbols      map[string]*SymbolState `json:"symbols"`
	HourlyCount  int                     `json:"hourlyCount"`
	HourWindowAt time.Time               `json:"hourWindowAt"` // start of current 1-hour window
	DailyCount   int                     `json:"dailyCount"`
	DayWindowAt  time.Time               `json:"dayWindowAt"` // start of current calendar day (UTC)
}

// Scheduler coordinates rate-limited, resumable downloads for a symbol list.
type Scheduler struct {
	apiKey  string
	outDir  string
	state   *State
	statePath string
}

// NewScheduler creates a Scheduler backed by a JSON state file.
// If the file exists the download progress is resumed; otherwise a fresh state is created.
func NewScheduler(apiKey, outDir, statePath string) (*Scheduler, error) {
	state, err := loadState(statePath)
	if err != nil {
		return nil, err
	}
	return &Scheduler{
		apiKey:    apiKey,
		outDir:    outDir,
		state:     state,
		statePath: statePath,
	}, nil
}

// Run iterates through symbols, downloading any that are not yet marked done.
// It respects Tiingo's free-tier limits (50 req/hour, 1000 req/day).
// When the hourly budget is exhausted it sleeps until the hour window resets.
// When the daily budget is exhausted it returns ErrDailyLimitReached so the
// caller can exit cleanly and re-run the next day.
// progress is called after each attempt (may be nil).
func (s *Scheduler) Run(symbols []string, progress func(symbol string, n, done, total int, err error)) error {
	now := time.Now().UTC()
	s.state.resetWindowsIfNeeded(now)

	total := len(symbols)
	n := 0

	for _, sym := range symbols {
		st, exists := s.state.Symbols[sym]
		if !exists {
			st = &SymbolState{}
			s.state.Symbols[sym] = st
		}
		if st.Done {
			n++
			continue
		}

		// Check daily limit before attempting a request.
		if s.state.DailyCount >= dailySafe {
			_ = s.saveState()
			return ErrDailyLimitReached
		}

		// If hourly budget is exhausted, sleep until the hour window resets.
		if s.state.HourlyCount >= HourlyLimit {
			reset := s.state.HourWindowAt.Add(time.Hour)
			wait := time.Until(reset)
			if wait > 0 {
				fmt.Printf("\nHourly limit reached (%d req). Sleeping %s until %s UTC.\n",
					HourlyLimit, wait.Round(time.Second), reset.Format("15:04:05"))
				time.Sleep(wait)
			}
			// Open a new hour window.
			s.state.HourWindowAt = time.Now().UTC()
			s.state.HourlyCount = 0
		}

		_, dlErr := Download(sym, s.apiKey, s.outDir)
		s.state.HourlyCount++
		s.state.DailyCount++

		if dlErr != nil {
			st.Done = false
			st.Error = dlErr.Error()
		} else {
			st.Done = true
			st.Error = ""
		}

		n++
		done := s.countDone(symbols)
		if progress != nil {
			progress(sym, n, done, total, dlErr)
		}

		_ = s.saveState()

		// Inter-request delay to stay comfortably under the hourly cap.
		if n < total && DefaultDelay > 0 {
			time.Sleep(DefaultDelay)
		}
	}

	return nil
}

// Stats returns progress counts for the given symbol list based on current state.
func (s *Scheduler) Stats(symbols []string) (done, failed, pending int) {
	for _, sym := range symbols {
		st, exists := s.state.Symbols[sym]
		if !exists {
			pending++
			continue
		}
		if st.Done {
			done++
		} else if st.Error != "" {
			failed++
		} else {
			pending++
		}
	}
	return
}

// ResetFailed clears the error state for all failed symbols so they will be retried.
func (s *Scheduler) ResetFailed(symbols []string) {
	for _, sym := range symbols {
		if st, ok := s.state.Symbols[sym]; ok && !st.Done && st.Error != "" {
			st.Error = ""
		}
	}
	_ = s.saveState()
}

// ErrDailyLimitReached is returned when the daily request budget is exhausted.
var ErrDailyLimitReached = fmt.Errorf("daily request limit reached (%d/day) — re-run tomorrow", DailyLimit)

// countDone returns how many symbols in the list are marked done.
func (s *Scheduler) countDone(symbols []string) int {
	n := 0
	for _, sym := range symbols {
		if st, ok := s.state.Symbols[sym]; ok && st.Done {
			n++
		}
	}
	return n
}

func (s *Scheduler) saveState() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.statePath, data, 0o644)
}

func loadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &State{
			Symbols:      make(map[string]*SymbolState),
			HourWindowAt: time.Now().UTC(),
			DayWindowAt:  todayUTC(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state file %s: %w", path, err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parsing state file %s: %w", path, err)
	}
	if st.Symbols == nil {
		st.Symbols = make(map[string]*SymbolState)
	}
	return &st, nil
}

func (s *State) resetWindowsIfNeeded(now time.Time) {
	if now.After(s.HourWindowAt.Add(time.Hour)) {
		s.HourWindowAt = now
		s.HourlyCount = 0
	}
	today := todayUTC()
	if today.After(s.DayWindowAt) {
		s.DayWindowAt = today
		s.DailyCount = 0
	}
}

func todayUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
