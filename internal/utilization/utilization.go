package utilization

import (
	"iter"
	"math/rand/v2"

	"github.com/spectrocloud-labs/prom-forge/internal/config"
)

// SteadyUtilization is an iterator that generates a steady value given the steady utilization pattern arguments
func SteadyUtilization(args config.SteadyUtilizationPattern) iter.Seq[float64] {
	return func(yield func(float64) bool) {
		for {
			if !yield(args.Value) {
				return
			}
		}
	}
}

// RandomUtilization is an iterator that generates a random value between the min and max values given the random utilization pattern arguments
func RandomUtilization(args config.RandomUtilizationPattern) iter.Seq[float64] {
	return func(yield func(float64) bool) {
		for {
			// #nosec G404
			if !yield(args.Min + rand.Float64()*(args.Max-args.Min)) {
				return
			}
		}
	}
}

// OscillatingUtilization is an iterator that oscillates between 2 values given the oscillating utilization pattern arguments.
// phaseA is 0-180 degrees or [0, π], phaseB is 180-360 degrees or [π, 2π].
func OscillatingUtilization(args config.OscillatingUtilizationPattern) iter.Seq[float64] {
	phaseAValue := args.PhaseA.Value
	phaseACount := args.PhaseA.HoldCount
	phaseARampSteps := args.PhaseA.RampSteps
	phaseBValue := args.PhaseB.Value
	phaseBCount := args.PhaseB.HoldCount
	phaseBRampSteps := args.PhaseB.RampSteps

	return func(yield func(float64) bool) {
		for {
			// sustain phaseAValue for phaseACount data points
			for range phaseACount {
				if !yield(phaseAValue) {
					return
				}
			}

			// raise or fall from phaseAValue to phaseBValue at the phaseARampSteps
			for step := 1; step <= phaseARampSteps; step++ {
				t := float64(step) / float64(phaseARampSteps)
				if !yield(phaseAValue + (phaseBValue-phaseAValue)*t) {
					return
				}
			}

			// sustain phaseBValue for phaseBCount data points
			for range phaseBCount {
				if !yield(phaseBValue) {
					return
				}
			}

			// raise or fall from phaseBValue to phaseAValue at the phaseBRampSteps
			for step := 1; step <= phaseBRampSteps; step++ {
				t := float64(step) / float64(phaseBRampSteps)
				if !yield(phaseBValue + (phaseAValue-phaseBValue)*t) {
					return
				}
			}
		}
	}
}
