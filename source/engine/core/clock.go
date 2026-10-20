package core

import "time"

const (
	DefaultTickRate = 120
	DefaultMaxFrame = 0.25
)

type Clock struct {
	TickRate      int
	FixedDelta    float64
	MaxFrameDelta float64
	accumulator   float64
	totalTime     float64
}

func NewClock(tickRate int) *Clock {
	if tickRate <= 0 {
		tickRate = DefaultTickRate
	}
	return &Clock{
		TickRate:      tickRate,
		FixedDelta:    1 / float64(tickRate),
		MaxFrameDelta: DefaultMaxFrame,
	}
}

// Advance consumes a frame delta and returns the number of fixed steps to run
// and the interpolation alpha for rendering.
func (c *Clock) Advance(delta float64) (steps int, alpha float32) {
	if delta < 0 {
		delta = 0
	}
	if delta > c.MaxFrameDelta {
		delta = c.MaxFrameDelta
	}
	c.accumulator += delta
	c.totalTime += delta
	for c.accumulator >= c.FixedDelta {
		c.accumulator -= c.FixedDelta
		steps++
	}
	alpha = float32(c.accumulator / c.FixedDelta)
	return steps, alpha
}

func (c *Clock) TotalTime() float64 {
	return c.totalTime
}

func (c *Clock) Reset() {
	c.accumulator = 0
	c.totalTime = 0
}

type WallClock struct {
	last time.Time
}

func NewWallClock() *WallClock {
	return &WallClock{last: time.Now()}
}

func (w *WallClock) Delta() float64 {
	now := time.Now()
	d := now.Sub(w.last).Seconds()
	w.last = now
	return d
}
