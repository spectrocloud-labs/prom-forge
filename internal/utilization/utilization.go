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
			if !yield(float64(rand.IntN(int(args.Max-args.Min+1)) + int(args.Min))) {
				return
			}
		}
	}
}

// OscillatingUtilization is an iterator that oscillates between 2 values given the oscillating utilization pattern arguments
func OscillatingUtilization(args config.OscillatingUtilizationPattern) iter.Seq[float64] {
	sustainValue := args.SustainValue
	minValue := args.MinValue
	minCount := args.MinCount
	sustainCount := args.SustainCount
	dropSteps := args.DropSteps
	riseSteps := args.RiseSteps
	return func(yield func(float64) bool) {
		for {
			// sustain
			for range sustainCount {
				if !yield(sustainValue) {
					return
				}
			}

			// drop
			for step := range dropSteps + 1 {
				t := float64(step) / float64(dropSteps)
				if !yield(sustainValue + (minValue-sustainValue)*t) {
					return
				}
			}

			// min sustain
			for range minCount {
				if !yield(minValue) {
					return
				}
			}

			// rise
			for step := 1; step <= riseSteps; step++ {
				t := float64(step) / float64(riseSteps)
				if !yield(minValue + (sustainValue-minValue)*t) {
					return
				}
			}
		}
	}
}
