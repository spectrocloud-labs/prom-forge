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

// Random produces a monotonic counter whose per-second growth rate is drawn
// uniformly from [Min, Max) at each sample. The rate is integrated over
// wall-clock time between samples, so rate(counter[…]) over a sliding window
// resembles a uniform distribution in [Min, Max).
type Random struct {
	Min, Max float64
	accum    float64
	lastTime time.Time
	started  bool
}

// Next returns the next accumulated counter value.
func (r *Random) Next(t time.Time) float64 {
	if !r.started {
		r.lastTime = t
		r.started = true
		return r.accum
	}
	dt := t.Sub(r.lastTime).Seconds()
	if dt < 0 {
		dt = 0
	}
	// #nosec G404
	rate := rand.Float64()*(r.Max-r.Min) + r.Min
	r.accum += rate * dt
	r.lastTime = t
	return r.accum
}

func (r Random) Name() string { return "random" }

// NewRandom creates a counter pattern whose per-second growth rate is drawn
// uniformly from [min, max) at each sample. min must be >= 0 to keep the
// counter monotonic.
func NewRandom(min, max float64) (Pattern, error) {
	if min < 0 {
		return nil, fmt.Errorf("min must be >= 0 to keep counter rate non-negative")
	}
	if min >= max {
		return nil, fmt.Errorf("min must be less than max")
	}
	return &Random{Min: min, Max: max}, nil
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
