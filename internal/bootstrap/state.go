// Package bootstrap holds process-wide mutable state initialized during app startup.
package bootstrap

import (
	"sync"
	"sync/atomic"
)

// State mirrors the intent of bootstrap/state.ts: session, cwd, project root,
// cost/meter placeholders. Safe for concurrent access from async prefetch hooks.
type State struct {
	mu sync.RWMutex

	sessionID   string
	cwd         string
	projectRoot string

	totalCost atomic.Uint64
	// durationMs is wall time placeholder (e.g. session duration), milliseconds.
	durationMs atomic.Uint64
	// meterEvents counts lightweight metering ticks (placeholder).
	meterEvents atomic.Uint64

	usageMu   sync.Mutex
	lastUsage TokenUsage
}

// TokenUsage holds token counts from the last Messages API usage block (align cost-tracker / usage.ts).
type TokenUsage struct {
	InputTokens              int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
	OutputTokens             int64
}

// RecordTokenUsage updates the last snapshot and bumps TotalCost with a coarse per-1k-token placeholder.
func (s *State) RecordTokenUsage(u TokenUsage) {
	s.usageMu.Lock()
	s.lastUsage = u
	s.usageMu.Unlock()
	combined := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens + u.OutputTokens
	if combined > 0 {
		s.AddTotalCost(uint64((combined + 999) / 1000))
	}
}

// LastTokenUsage returns the last recorded usage.
func (s *State) LastTokenUsage() TokenUsage {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	return s.lastUsage
}

// NewState returns an empty State.
func NewState() *State {
	return &State{}
}

// SetSessionID sets the active session identifier (UUID-like hex from app layer).
func (s *State) SetSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = id
}

// SessionID returns the session id or "".
func (s *State) SessionID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionID
}

// SetCwd sets the process working directory at bootstrap (original cwd).
func (s *State) SetCwd(cwd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cwd = cwd
}

// Cwd returns cwd.
func (s *State) Cwd() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cwd
}

// SetProjectRoot sets resolved project root (may equal cwd).
func (s *State) SetProjectRoot(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projectRoot = root
}

// ProjectRoot returns project root.
func (s *State) ProjectRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projectRoot
}

// AddTotalCost adds delta to the cost accumulator (placeholder, e.g. micro-dollars).
func (s *State) AddTotalCost(delta uint64) {
	s.totalCost.Add(delta)
}

// TotalCost returns accumulated cost placeholder.
func (s *State) TotalCost() uint64 {
	return s.totalCost.Load()
}

// SetDurationMs sets duration placeholder.
func (s *State) SetDurationMs(ms uint64) {
	s.durationMs.Store(ms)
}

// DurationMs returns duration placeholder.
func (s *State) DurationMs() uint64 {
	return s.durationMs.Load()
}

// IncMeter increments metering event counter.
func (s *State) IncMeter() {
	s.meterEvents.Add(1)
}

// MeterEvents returns meter tick count.
func (s *State) MeterEvents() uint64 {
	return s.meterEvents.Load()
}
