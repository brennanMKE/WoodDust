package contrib

import (
	"sync"
	"time"
)

// SubmitLimiter enforces the submission ceilings (PLAN §6): per client per
// hour, per client per day, and a site-wide daily ceiling. It is separate
// from the per-IP token bucket on API GETs. State is in memory (single host);
// the global window is seeded from the database at startup so a restart does
// not reset it. Keys are client IPs held in memory only, never stored.
type SubmitLimiter struct {
	mu        sync.Mutex
	perHour   int
	perDay    int
	globalDay int
	byKey     map[string][]time.Time
	global    []time.Time
	now       func() time.Time
}

// NewSubmitLimiter builds a limiter with the given ceilings.
func NewSubmitLimiter(perHour, perDay, globalDay int) *SubmitLimiter {
	return &SubmitLimiter{perHour: perHour, perDay: perDay, globalDay: globalDay,
		byKey: map[string][]time.Time{}, now: time.Now}
}

// Seed records past submission times toward the global ceiling.
func (l *SubmitLimiter) Seed(times []time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.global = append(l.global, times...)
}

func prune(ts []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(ts) && !ts[i].After(cutoff) {
		i++
	}
	return ts[i:]
}

// Check reports whether key may submit now and, if not, how long to wait.
// It does not record anything; call Record after the submission is stored.
func (l *SubmitLimiter) Check(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	day := now.Add(-24 * time.Hour)
	l.global = prune(l.global, day)
	ts := prune(l.byKey[key], day)
	l.byKey[key] = ts
	if len(l.global) >= l.globalDay {
		return false, l.global[0].Sub(day)
	}
	if len(ts) >= l.perDay {
		return false, ts[len(ts)-l.perDay].Sub(day)
	}
	hour := now.Add(-time.Hour)
	recent := prune(ts, hour)
	if len(recent) >= l.perHour {
		return false, recent[len(recent)-l.perHour].Sub(hour)
	}
	return true, 0
}

// Record counts one submission for key.
func (l *SubmitLimiter) Record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.byKey[key] = append(l.byKey[key], now)
	l.global = append(l.global, now)
}

// Sweep drops keys with nothing in the last day.
func (l *SubmitLimiter) Sweep() {
	l.mu.Lock()
	defer l.mu.Unlock()
	day := l.now().Add(-24 * time.Hour)
	for k, ts := range l.byKey {
		if len(prune(ts, day)) == 0 {
			delete(l.byKey, k)
		}
	}
}
