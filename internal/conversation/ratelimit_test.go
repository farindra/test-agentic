package conversation

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToMax(t *testing.T) {
	rl := newRateLimiter(time.Minute, 3)
	for i := 0; i < 3; i++ {
		if !rl.allow("contact-1") {
			t.Fatalf("attempt %d harusnya masih diizinkan", i+1)
		}
	}
	if rl.allow("contact-1") {
		t.Fatalf("attempt ke-4 harusnya udah kena limit")
	}
}

func TestRateLimiterIsolatesByKey(t *testing.T) {
	rl := newRateLimiter(time.Minute, 1)
	if !rl.allow("contact-a") {
		t.Fatalf("contact-a attempt pertama harusnya diizinkan")
	}
	if !rl.allow("contact-b") {
		t.Fatalf("contact-b harusnya gak kepengaruh limit contact-a")
	}
	if rl.allow("contact-a") {
		t.Fatalf("contact-a attempt kedua harusnya udah kena limit")
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	rl := newRateLimiter(20*time.Millisecond, 1)
	if !rl.allow("contact-1") {
		t.Fatalf("attempt pertama harusnya diizinkan")
	}
	if rl.allow("contact-1") {
		t.Fatalf("attempt kedua langsung harusnya kena limit")
	}
	time.Sleep(30 * time.Millisecond)
	if !rl.allow("contact-1") {
		t.Fatalf("setelah window lewat, harusnya diizinkan lagi")
	}
}
