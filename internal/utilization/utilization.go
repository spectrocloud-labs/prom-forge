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
	y1 := args.Y1
	y1Count := args.Y1Count
	y2 := args.Y2
	y2Count := args.Y2Count
	y1y2StepCount := args.Y1Y2StepCount
	y2y1StepCount := args.Y2Y1StepCount
	return func(yield func(float64) bool) {
		for {
			// sustain y1 for y1Count data points
			for range y1Count {
				if !yield(y1) {
					return
				}
			}

			// raise or fall from y1 to y2 at the y1y2StepCount
			for step := 1; step <= y1y2StepCount; step++ {
				t := float64(step) / float64(y1y2StepCount)
				if !yield(y1 + (y2-y1)*t) {
					return
				}
			}

			// sustain y2 for y2Count data points
			for range y2Count {
				if !yield(y2) {
					return
				}
			}

			// raise or fall from y2 to y1 at the y2y1StepCount
			for step := 1; step <= y2y1StepCount; step++ {
				t := float64(step) / float64(y2y1StepCount)
				if !yield(y2 + (y1-y2)*t) {
					return
				}
			}
		}
	}
}
