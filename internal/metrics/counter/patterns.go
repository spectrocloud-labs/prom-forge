// metrics/counter/patterns.go
package counter

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/spectrocloud-labs/prom-forge/internal/metrics/gauge"
)

// Pattern is the interface for all patterns.
type Pattern interface {
	Next(timestamp time.Time) float64
	Name() string
}

// Steady: counter increases as defined by the given linear function.
type Steady struct {
	Slope, Offset float64
	startTime     time.Time
}

func (s *Steady) Next(t time.Time) float64 {
	if s.startTime.IsZero() {
		s.startTime = t
	}
	timeElapsed := t.Sub(s.startTime).Seconds()
	return s.Offset + s.Slope*timeElapsed
}
func (s Steady) Name() string { return "steady" }

// NewSteady creates a new steady pattern.
func NewSteady(slope, offset float64) (Pattern, error) {
	if slope < 0 {
		return nil, fmt.Errorf("slope must be greater than or equal to 0 for a counter metric")
	}
	return &Steady{Slope: slope, Offset: offset}, nil
}

// Random: counter increases as defined by the given linear function and adds random noise in the range [Min, Max) at each point.
type Random struct {
	Slope, Offset float64
	Min, Max      float64
	startTime     time.Time
	lastY         float64
	started       bool
}

func (r *Random) Next(t time.Time) float64 {
	if r.startTime.IsZero() {
		r.startTime = t
	}
	timeElapsed := t.Sub(r.startTime).Seconds()
	// #nosec G404
	noise := rand.Float64()*(r.Max-r.Min) + r.Min
	y := r.Slope*timeElapsed + r.Offset + noise

	// Force monotonic increase
	if r.started && y <= r.lastY {
		y = r.lastY + noise + 1e-9
	}

	r.lastY = y
	r.started = true
	return y
}

func (r Random) Name() string { return "random" }

// NewRandom creates a new random pattern.
func NewRandom(slope, offset, min, max float64) (Pattern, error) {
	if slope < 0 {
		return nil, fmt.Errorf("slope must be greater than or equal to 0 for a counter metric")
	}
	if min >= max {
		return nil, fmt.Errorf("min must be less than max")
	}
	return &Random{Slope: slope, Offset: offset, Min: min, Max: max}, nil
}

// Oscillating produces a monotonic counter whose per-second growth rate
// follows a gauge-style oscillation between PhaseAValue and PhaseBValue.
// The pattern values are interpreted as rate (units/second), and integrated
// across wall-clock time between samples. As a result, rate(counter[…])
// over a sliding window will resemble the underlying gauge oscillation.
type Oscillating struct {
	inner    gauge.Pattern
	accum    float64
	lastTime time.Time
	started  bool
}

// Next returns the next accumulated counter value.
func (o *Oscillating) Next(t time.Time) float64 {
	if !o.started {
		o.lastTime = t
		o.started = true
		return o.accum
	}
	rate := o.inner.Next(t)
	dt := t.Sub(o.lastTime).Seconds()
	if dt < 0 {
		dt = 0
	}
	o.accum += rate * dt
	o.lastTime = t
	return o.accum
}
func (o *Oscillating) Name() string { return "oscillating" }

// NewOscillating creates a counter pattern whose growth rate oscillates
// between phaseAValue and phaseBValue (interpreted as units per second).
func NewOscillating(phaseAValue, phaseBValue float64, phaseACount, phaseARampSteps, phaseBCount, phaseBRampSteps uint) (Pattern, error) {
	if phaseAValue < 0 || phaseBValue < 0 {
		return nil, fmt.Errorf("oscillating phase values must be >= 0 to keep counter monotonic")
	}
	inner, err := gauge.NewOscillating(phaseAValue, phaseBValue, phaseACount, phaseARampSteps, phaseBCount, phaseBRampSteps)
	if err != nil {
		return nil, err
	}
	return &Oscillating{inner: inner}, nil
}
