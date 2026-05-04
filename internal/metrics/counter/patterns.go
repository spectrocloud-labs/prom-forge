// metrics/counter/patterns.go
package counter

import (
	"fmt"
	"math/rand"
	"time"
)

// Pattern is the interface for all patterns.
type Pattern interface {
	Next(timestamp time.Time) float64
	Name() string
}

// Steady: counter holds a fixed value.
type Steady struct {
	Slope, Offset float64
	startTime     time.Time
}

func (s *Steady) Next(t time.Time) float64 {
	if s.startTime.IsZero() {
		s.startTime = t
	}
	timeElasped := t.Sub(s.startTime).Seconds()
	return s.Offset + s.Slope*float64(timeElasped)
}
func (s Steady) Name() string { return "steady" }

// NewSteady creates a new steady pattern.
func NewSteady(slope, offset float64) (Pattern, error) {
	if slope < 0 {
		return nil, fmt.Errorf("slope must be greater than or equal to 0 for a counter metric")
	}
	return &Steady{Slope: slope, Offset: offset}, nil
}

// Random: counter jumps to a random value in [Min, Max] each iteration always increasing.
type Random struct {
	Min, Max  float64
	lastValue *float64
}

func (r *Random) Next(_ time.Time) float64 {
	var value, min float64
	min = r.Min
	if r.lastValue != nil {
		min = *r.lastValue
	}
	value = min + rand.Float64()*(r.Max-min)
	r.lastValue = &value
	return value
}

func (r Random) Name() string { return "random" }

// NewRandom creates a new random pattern.
func NewRandom(min, max float64) (Pattern, error) {
	if min >= max {
		return nil, fmt.Errorf("min must be less than max")
	}
	return &Random{Min: min, Max: max}, nil
}

type oscPhase uint8

const (
	phaseAHold oscPhase = iota
	phaseARamp
	phaseBHold
	phaseBRamp
)

// Oscillating drives a monotonic counter staircase: hold at A, ramp A→B,
// hold at B, then ramp B→B′ where B′ = B+(B−A). After that ramp the series
// must not drop, so the next cycle uses A←B′ and B←B′+Δ (Δ = B−A for the
// cycle that just finished), not A←B (unlike the gauge oscillation).
type Oscillating struct {
	PhaseAValue     float64
	PhaseACount     uint
	PhaseARampSteps uint
	PhaseBValue     float64
	PhaseBCount     uint
	PhaseBRampSteps uint

	phase oscPhase
	step  uint
}

// Next returns the next value in the oscillation.
func (o *Oscillating) Next(_ time.Time) float64 {
	for {
		switch o.phase {
		case phaseAHold:
			if o.step < o.PhaseACount {
				o.step++
				return o.PhaseAValue
			}
			o.advance(phaseARamp)

		case phaseARamp:
			if o.PhaseARampSteps == 0 {
				o.advance(phaseBHold)
				return o.PhaseBValue
			}
			if o.step < o.PhaseARampSteps {
				o.step++
				t := float64(o.step) / float64(o.PhaseARampSteps)
				return o.PhaseAValue + (o.PhaseBValue-o.PhaseAValue)*t
			}
			o.advance(phaseBHold)

		case phaseBHold:
			if o.step < o.PhaseBCount {
				o.step++
				return o.PhaseBValue
			}
			o.advance(phaseBRamp)

		case phaseBRamp:
			delta := o.PhaseBValue - o.PhaseAValue
			nextB := o.PhaseBValue + delta
			if o.PhaseBRampSteps == 0 {
				o.PhaseAValue = nextB
				o.PhaseBValue = nextB + delta
				o.advance(phaseAHold)
				return o.PhaseAValue
			}
			if o.step < o.PhaseBRampSteps {
				o.step++
				t := float64(o.step) / float64(o.PhaseBRampSteps)
				v := o.PhaseBValue + (nextB-o.PhaseBValue)*t
				if o.step == o.PhaseBRampSteps {
					o.PhaseAValue = nextB
					o.PhaseBValue = nextB + delta
					o.advance(phaseAHold)
				}
				return v
			}
		}
	}
}

func (o *Oscillating) advance(next oscPhase) {
	o.phase = next
	o.step = 0
}

// Reset returns the iterator to the beginning of phase A.
func (o *Oscillating) Reset() {
	o.phase = phaseAHold
	o.step = 0
}
func (o *Oscillating) Name() string { return "oscillating" }

// NewOscillating creates a new oscillating staircase pattern for counters.
func NewOscillating(phaseAValue, phaseBValue float64, phaseACount, phaseARampSteps, phaseBCount, phaseBRampSteps uint) (Pattern, error) {
	if phaseAValue > phaseBValue {
		return nil, fmt.Errorf("phase B value must be greater than or equal to phase A value")
	}
	if phaseACount+phaseARampSteps+phaseBCount+phaseBRampSteps == 0 {
		return nil, fmt.Errorf("oscillating pattern must emit at least one sample")
	}
	return &Oscillating{
		PhaseAValue: phaseAValue, PhaseBValue: phaseBValue,
		PhaseACount: phaseACount, PhaseARampSteps: phaseARampSteps,
		PhaseBCount: phaseBCount, PhaseBRampSteps: phaseBRampSteps,
	}, nil
}
