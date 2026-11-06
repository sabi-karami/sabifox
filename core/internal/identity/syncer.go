package identity

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/sabi-karami/sabifox/core/internal/securefile"
)

// State is the time zone sync's saved state (identity.json): the user's
// choice, and the zone to restore, kept on disk so that a crash or a
// power cut while connected still restores it at the next start.
type State struct {
	SyncTimezone bool `json:"sync_timezone"`
	// OriginalZone is the system zone before the sync changed it, "" when
	// the sync changed nothing.
	OriginalZone string `json:"original_zone,omitempty"`
	// AppliedZone is the zone the sync set.
	AppliedZone string `json:"applied_zone,omitempty"`
	// BlockTrackers blocks advertising, tracking, malware, phishing and
	// crypto-mining sites in the proxy (trackers.go).
	BlockTrackers bool `json:"block_trackers,omitempty"`
}

// Syncer keeps the system time zone on the exit's while enabled and
// connected, and restores the user's zone afterwards.
type Syncer struct {
	mu    sync.Mutex
	path  string
	sys   System
	state State
	// failed is the zone whose setting last failed: not retried until the
	// target changes or the sync is switched again (a Linux password
	// prompt must not come back every few seconds).
	failed    string
	lastError string
	// overridden: the user changed the zone the sync set; it is left
	// alone until the exit changes.
	overridden bool
	changedAt  time.Time
}

// NewSyncer loads the state from path (none yet: disabled).
func NewSyncer(path string, sys System) *Syncer {
	s := &Syncer{path: path, sys: sys}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s.state)
	}
	return s
}

func (s *Syncer) save() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	return securefile.WritePrivate(s.path, append(data, '\n'))
}

// Status is the sync's state for the report.
type Status struct {
	Supported    bool      `json:"supported"`
	Enabled      bool      `json:"enabled"`
	Applied      bool      `json:"applied"`
	AppliedZone  string    `json:"applied_zone,omitempty"`
	OriginalZone string    `json:"original_zone,omitempty"`
	Overridden   bool      `json:"overridden,omitempty"`
	Error        string    `json:"error,omitempty"`
	ChangedAt    time.Time `json:"changed_at,omitzero"`
}

// Status reports the sync's state.
func (s *Syncer) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Status{
		Supported:    s.sys.Supported(),
		Enabled:      s.state.SyncTimezone,
		Applied:      s.state.OriginalZone != "",
		AppliedZone:  s.state.AppliedZone,
		OriginalZone: s.state.OriginalZone,
		Overridden:   s.overridden,
		Error:        s.lastError,
		ChangedAt:    s.changedAt,
	}
}

// Enabled reports the user's choice.
func (s *Syncer) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.SyncTimezone
}

// SetEnabled saves the user's choice; switching it clears a failure.
func (s *Syncer) SetEnabled(on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on && !s.sys.Supported() {
		return ErrUnsupported
	}
	s.state.SyncTimezone = on
	s.failed, s.lastError, s.overridden = "", "", false
	return s.save()
}

// Apply sets the system zone to the exit's IANA zone; with target "" it
// restores the user's zone (disconnected, or the sync disabled).
func (s *Syncer) Apply(target string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.SyncTimezone || target == "" {
		s.restoreLocked()
		return
	}
	if !s.sys.Supported() {
		return
	}
	name := s.sys.Name(target)
	if name == "" {
		s.lastError = "no system time zone for " + target
		return
	}
	if s.failed != "" && SameZone(s.failed, name) {
		return
	}
	current, err := s.sys.Zone()
	if err != nil {
		s.lastError = err.Error()
		return
	}
	if s.state.AppliedZone != "" && SameZone(s.state.AppliedZone, name) {
		// Already applied: a different zone now is the user's own change.
		s.overridden = !SameZone(current, name)
		return
	}
	s.overridden = false
	if SameZone(current, name) {
		return
	}
	saved := s.state
	if s.state.OriginalZone == "" {
		s.state.OriginalZone = current
	}
	s.state.AppliedZone = name
	// The zone to restore is on disk before the system changes.
	if err := s.save(); err != nil {
		s.state = saved
		s.lastError = err.Error()
		return
	}
	if err := s.sys.SetZone(name); err != nil {
		s.state = saved
		_ = s.save()
		s.failed, s.lastError = name, err.Error()
		return
	}
	s.failed, s.lastError, s.changedAt = "", "", time.Now()
}

// Restore puts the user's zone back if the sync changed it.
func (s *Syncer) Restore() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restoreLocked()
}

func (s *Syncer) restoreLocked() {
	s.failed, s.overridden = "", false
	if s.state.OriginalZone == "" {
		return
	}
	current, err := s.sys.Zone()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		s.lastError = err.Error()
		return
	}
	// A zone the user chose meanwhile is theirs to keep.
	if err == nil && SameZone(current, s.state.AppliedZone) && !SameZone(current, s.state.OriginalZone) {
		if err := s.sys.SetZone(s.state.OriginalZone); err != nil {
			s.lastError = err.Error()
			return
		}
		s.changedAt = time.Now()
	}
	s.state.OriginalZone, s.state.AppliedZone = "", ""
	s.lastError = ""
	_ = s.save()
}
