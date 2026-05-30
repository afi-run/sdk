package afi

import (
	"testing"
	"time"
)

func TestQuote_IsStale(t *testing.T) {
	now := time.Now().UnixMilli()

	cases := []struct {
		name      string
		createdAt int64
		maxAgeSec int64
		want      bool
	}{
		{"fresh quote", now, 60, false},
		{"older than max age", now - 120_000, 60, true},
		{"exactly within max age", now - 30_000, 60, false},
		{"zero createdAt is never stale (legacy compatibility)", 0, 1, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := &Quote{CreatedAt: c.createdAt}
			if got := q.IsStale(c.maxAgeSec); got != c.want {
				t.Errorf("IsStale(%d) = %v, want %v", c.maxAgeSec, got, c.want)
			}
		})
	}
}
