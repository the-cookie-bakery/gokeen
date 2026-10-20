package core

import "testing"

func TestClockFixedSteps(t *testing.T) {
	c := NewClock(120)
	steps, alpha := c.Advance(1.0 / 60.0)
	if steps != 2 {
		t.Fatalf("steps: %d", steps)
	}
	if alpha > 0.001 {
		t.Fatalf("alpha: %v", alpha)
	}
	c.Reset()
	steps, alpha = c.Advance(1.0 / 240.0)
	if steps != 0 {
		t.Fatalf("steps: %d", steps)
	}
	if alpha < 0.49 || alpha > 0.51 {
		t.Fatalf("alpha: %v", alpha)
	}
}

func TestClockClampsLargeDelta(t *testing.T) {
	c := NewClock(10)
	steps, _ := c.Advance(10)
	if steps != 2 {
		t.Fatalf("steps: %d", steps)
	}
}

func TestClockAccumulates(t *testing.T) {
	c := NewClock(100)
	total := 0
	for i := 0; i < 100; i++ {
		steps, _ := c.Advance(0.001)
		total += steps
	}
	if total != 10 {
		t.Fatalf("total steps: %d", total)
	}
}
