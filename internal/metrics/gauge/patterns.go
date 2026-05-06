package gauge

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// Pattern is the interface for all patterns.
type Pattern interface {
	Next(timestamp time.Time) float64
	Name() string
	isGaugePattern()
}

// Steady: gauge increases as defined by the given linear function.
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
func (s Steady) Name() string    { return "steady" }
func (s Steady) isGaugePattern() {}

// NewSteady creates a new steady pattern. Negative slopes are allowed since
// gauges can decrease over time (unlike counters).
func NewSteady(slope, offset float64) (Pattern, error) {
	return &Steady{Slope: slope, Offset: offset}, nil
}

// Random: gauge outputs a uniform random value in [Min, Max) at each sample.
type Random struct {
	Min, Max float64
}

func (r *Random) Next(_ time.Time) float64 {
	// #nosec G404
	return rand.Float64()*(r.Max-r.Min) + r.Min
}
func (r Random) Name() string    { return "random" }
func (r Random) isGaugePattern() {}

// NewRandom creates a new random pattern that emits uniform values in [min, max).
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

// Oscillating yields a never-ending oscillating sequence:
// hold A → ramp A→B → hold B → ramp B→A → repeat.
type Oscillating struct {
	PhaseAValue     float64
	PhaseACount     uint
	PhaseARampSteps uint
	PhaseBValue     float64
	PhaseBCount     uint
	PhaseBRampSteps uint

	// state
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
			if o.step < o.PhaseBRampSteps {
				o.step++
				t := float64(o.step) / float64(o.PhaseBRampSteps)
				return o.PhaseBValue + (o.PhaseAValue-o.PhaseBValue)*t
			}
			o.advance(phaseAHold)
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
func (o *Oscillating) Name() string    { return "oscillating" }
func (o *Oscillating) isGaugePattern() {}

// NewOscillating creates a new oscillating pattern.
func NewOscillating(phaseAValue, phaseBValue float64, phaseACount, phaseARampSteps, phaseBCount, phaseBRampSteps uint) (Pattern, error) {
	if phaseACount+phaseARampSteps+phaseBCount+phaseBRampSteps == 0 {
		return nil, fmt.Errorf("oscillating pattern must emit at least one sample")
	}
	return &Oscillating{PhaseAValue: phaseAValue, PhaseBValue: phaseBValue, PhaseACount: phaseACount, PhaseARampSteps: phaseARampSteps, PhaseBCount: phaseBCount, PhaseBRampSteps: phaseBRampSteps}, nil
}
