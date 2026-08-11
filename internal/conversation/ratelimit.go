package conversation

import (
	"sync"
	"time"
)

// rateLimiter adalah sliding-window counter in-memory per kontak — cukup buat
// nahan spam/flood dari satu nomor/chat biar gak ngeburu-buru habisin kuota
// biaya API provider AI. In-memory doang (reset kalau restart) itu udah
// cukup buat scope single-instance aplikasi ini; gak butuh Redis atau
// storage terpisah.
type rateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	max    int
	events map[string][]time.Time
}

func newRateLimiter(window time.Duration, max int) *rateLimiter {
	return &rateLimiter{window: window, max: max, events: make(map[string][]time.Time)}
}

// allow balikin false kalau key ini udah ngelewatin batas pesan dalam
// window waktu terakhir.
func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	kept := r.events[key][:0]
	for _, t := range r.events[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= r.max {
		r.events[key] = kept
		return false
	}
	r.events[key] = append(kept, now)
	return true
}
