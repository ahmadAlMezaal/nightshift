package pipeline

import (
	"testing"
	"time"
)

func TestDispatchCapReached(t *testing.T) {
	cases := []struct {
		max, count int
		want       bool
	}{
		{40, 39, false},
		{40, 40, true},
		{40, 41, true},
		{0, 0, false},
		{0, 1_000_000, false},
		{-1, 5, false},
		{1, 1, true},
	}
	for _, c := range cases {
		if got := dispatchCapReached(c.max, c.count); got != c.want {
			t.Errorf("dispatchCapReached(%d, %d) = %v, want %v", c.max, c.count, got, c.want)
		}
	}
}

func TestRollDispatchWindow_ResetsAtUTCMidnight(t *testing.T) {
	day1 := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)

	p := &Pipeline{}

	p.rollDispatchWindow(day1)
	p.totalDispatches = 7
	p.dispatchCapped = true
	p.dailySuccessCount = 5
	p.dailyFailCount = 2

	p.rollDispatchWindow(day1.Add(8 * time.Hour))
	if p.totalDispatches != 7 || !p.dispatchCapped {
		t.Fatalf("same-day roll reset state: got count=%d capped=%v, want 7/true",
			p.totalDispatches, p.dispatchCapped)
	}
	if p.dailySuccessCount != 5 || p.dailyFailCount != 2 {
		t.Fatalf("same-day roll reset daily counters: got succ=%d fail=%d, want 5/2",
			p.dailySuccessCount, p.dailyFailCount)
	}

	p.rollDispatchWindow(day2)
	if p.totalDispatches != 0 {
		t.Fatalf("counter not reset at new day: got %d, want 0", p.totalDispatches)
	}
	if p.dispatchCapped {
		t.Fatal("cap alert flag not cleared at new day")
	}
	if p.dailySuccessCount != 0 || p.dailyFailCount != 0 {
		t.Fatalf("daily counters not reset at new day: got succ=%d fail=%d, want 0/0",
			p.dailySuccessCount, p.dailyFailCount)
	}
}

func TestBumpSuccessFeedsDailyCounter(t *testing.T) {
	p := &Pipeline{
		active:         map[string]struct{}{},
		failedAttempts: map[string]int{},
		dispatchWindow: time.Now().UTC().Truncate(24 * time.Hour),
	}

	p.bumpSuccess()
	p.bumpSuccess()
	if p.dailySuccessCount != 2 {
		t.Fatalf("bumpSuccess did not feed daily counter: got %d, want 2", p.dailySuccessCount)
	}
	if p.successCount != 2 {
		t.Fatalf("bumpSuccess did not feed session counter: got %d, want 2", p.successCount)
	}

	p.bumpFailed("ENG-1")
	if p.dailyFailCount != 1 || p.failCount != 1 {
		t.Fatalf("bumpFailed counters: got daily=%d session=%d, want 1/1", p.dailyFailCount, p.failCount)
	}

	p.rollDispatchWindow(p.dispatchWindow.Add(24 * time.Hour))
	if p.dailySuccessCount != 0 || p.dailyFailCount != 0 {
		t.Fatalf("daily counters survived window roll: got succ=%d fail=%d, want 0/0",
			p.dailySuccessCount, p.dailyFailCount)
	}
	if p.successCount != 2 || p.failCount != 1 {
		t.Fatalf("session counters reset by window roll: got succ=%d fail=%d, want 2/1",
			p.successCount, p.failCount)
	}
}
