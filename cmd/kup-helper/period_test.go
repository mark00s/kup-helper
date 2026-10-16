package main

import (
	"testing"
	"time"
)

func TestStartFromDayOfMonth(t *testing.T) {
	tests := []struct {
		now  string
		day  int
		want string
	}{
		{"2026-10-15", 20, "2026-09-20"},
		{"2026-10-15", 31, "2026-09-30"},
		{"2026-03-10", 30, "2026-02-28"},
		{"2028-03-10", 30, "2028-02-29"},
		{"2026-01-05", 15, "2025-12-15"},
	}
	for _, tt := range tests {
		now, _ := time.Parse("2006-01-02", tt.now)
		got := startFromDayOfMonth(now, tt.day).Format("2006-01-02")
		if got != tt.want {
			t.Errorf("now=%s day=%d: got %s, want %s", tt.now, tt.day, got, tt.want)
		}
	}
}
