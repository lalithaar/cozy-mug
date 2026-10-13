package auth

import (
	"sync"
	"time"
)

// Lockout tracks repeated authentication failures for a key (user ID, email,
// client IP, or a combination chosen by the app). It is an in-memory helper:
// multi-instance deployments should persist counts in their own storage.
type Lockout struct {
	maxFailures int
	window      time.Duration
	lockout     time.Duration

	mu      sync.Mutex
	records map[string]*lockoutRecord
}

type lockoutRecord struct {
	failures    []time.Time
	lockedUntil time.Time
}

// NewLockout returns a limiter that locks after maxFailures within window, for
// lockout. Zero or negative inputs use conservative defaults.
func NewLockout(maxFailures int, window, lockout time.Duration) *Lockout {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if lockout <= 0 {
		lockout = 15 * time.Minute
	}
	return &Lockout{
		maxFailures: maxFailures,
		window:      window,
		lockout:     lockout,
		records:     map[string]*lockoutRecord{},
	}
}

// Allow reports whether key may attempt login now.
func (l *Lockout) Allow(key string) bool {
	_, locked := l.Locked(key)
	return !locked
}

// Locked reports whether key is currently locked and, when it is, the time it
// becomes available again.
func (l *Lockout) Locked(key string) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec := l.records[key]
	if rec == nil {
		return time.Time{}, false
	}
	now := time.Now()
	if !rec.lockedUntil.IsZero() && now.Before(rec.lockedUntil) {
		return rec.lockedUntil, true
	}
	if !rec.lockedUntil.IsZero() {
		// Lock expired; clear record for fresh counting.
		delete(l.records, key)
	}
	return time.Time{}, false
}

// Fail records a failed attempt. It returns true when that attempt triggered a
// lock, along with the lock expiry time.
func (l *Lockout) Fail(key string) (bool, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	rec := l.records[key]
	if rec == nil {
		rec = &lockoutRecord{}
		l.records[key] = rec
	}

	// Drop failures outside the sliding window.
	cutoff := now.Add(-l.window)
	i := 0
	for i < len(rec.failures) && !rec.failures[i].After(cutoff) {
		i++
	}
	rec.failures = append([]time.Time{}, rec.failures[i:]...)
	rec.failures = append(rec.failures, now)

	if len(rec.failures) < l.maxFailures {
		return false, time.Time{}
	}
	rec.lockedUntil = now.Add(l.lockout)
	rec.failures = nil
	return true, rec.lockedUntil
}

// Success clears failure history after an accepted login.
func (l *Lockout) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, key)
}
