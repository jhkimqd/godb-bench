package metrics

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// Statistics holds statistical metrics for a benchmark
type Statistics struct {
	Mean       float64
	StdDev     float64
	Median     float64
	MAD        float64 // Median Absolute Deviation
	Min        float64
	Max        float64
	Count      int64
	Throughput float64 // Operations per second
	R2         float64 // R-squared from linear regression
}

// ConfidenceInterval represents a confidence interval for a statistic
type ConfidenceInterval struct {
	LowerBound float64
	Estimate   float64
	UpperBound float64
}

const (
	bootstrapSamples = 100000 // Number of bootstrap resamples (same as criterion.rs default)
	confidenceLevel  = 0.95   // 95% confidence interval
)

// calculateStatistics computes statistical metrics from sample data
func calculateStatistics(samples []SampleData) Statistics {
	if len(samples) == 0 {
		return Statistics{}
	}

	// Extract times as float64 microseconds for calculations
	times := make([]float64, len(samples))
	var sum float64
	min := math.MaxFloat64
	max := 0.0

	for i, sample := range samples {
		timeUs := float64(sample.TotalTime.Microseconds())
		times[i] = timeUs
		sum += timeUs
		if timeUs < min {
			min = timeUs
		}
		if timeUs > max {
			max = timeUs
		}
	}

	// Calculate mean
	mean := sum / float64(len(samples))

	// Calculate standard deviation
	var varianceSum float64
	for _, t := range times {
		diff := t - mean
		varianceSum += diff * diff
	}
	stdDev := math.Sqrt(varianceSum / float64(len(samples)))

	// Calculate median and MAD
	sortedTimes := make([]float64, len(times))
	copy(sortedTimes, times)
	sort.Float64s(sortedTimes)

	median := calculateMedian(sortedTimes)
	mad := calculateMAD(sortedTimes, median)

	// Calculate throughput (ops/sec)
	meanSeconds := mean / 1_000_000 // Convert microseconds to seconds
	throughput := 1.0 / meanSeconds

	// Calculate R² (we don't do linear regression here since we're just tracking individual ops)
	// For individual operations, R² isn't as meaningful, but we can calculate it if needed
	r2 := calculateR2(samples)

	return Statistics{
		Mean:       mean,
		StdDev:     stdDev,
		Median:     median,
		MAD:        mad,
		Min:        min,
		Max:        max,
		Count:      int64(len(samples)),
		Throughput: throughput,
		R2:         r2,
	}
}

// calculateMedian returns the median of a sorted slice
func calculateMedian(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

// calculateMAD calculates the Median Absolute Deviation
func calculateMAD(sorted []float64, median float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	deviations := make([]float64, len(sorted))
	for i, val := range sorted {
		deviations[i] = math.Abs(val - median)
	}
	sort.Float64s(deviations)
	return calculateMedian(deviations)
}

// calculateR2 calculates R² for the samples using linear regression
// R² measures the goodness-of-fit: how well the data fits a linear model
// where X = sample index (iteration number) and Y = time
func calculateR2(samples []SampleData) float64 {
	if len(samples) < 2 {
		return 1.0
	}

	n := float64(len(samples))

	// Calculate sums for linear regression
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for i, sample := range samples {
		x := float64(i + 1) // Sample index (1-based)
		y := float64(sample.TotalTime.Microseconds())

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	// Calculate slope (beta) and intercept (alpha) for linear regression
	// beta = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	numerator := n*sumXY - sumX*sumY
	denominator := n*sumX2 - sumX*sumX

	if denominator == 0 {
		return 0.0 // No variance in X
	}

	// Calculate R² (coefficient of determination)
	// R² = (correlation coefficient)²
	// correlation = (n*sumXY - sumX*sumY) / sqrt((n*sumX2 - sumX²) * (n*sumY2 - sumY²))

	varX := n*sumX2 - sumX*sumX
	varY := n*sumY2 - sumY*sumY

	if varY == 0 {
		return 1.0 // No variance in Y means perfect fit
	}

	correlation := numerator / math.Sqrt(varX*varY)
	r2 := correlation * correlation

	// Ensure R² is between 0 and 1
	if r2 < 0 {
		r2 = 0
	}
	if r2 > 1 {
		r2 = 1
	}

	return r2
}

// bootstrapResample performs bootstrap resampling to calculate confidence intervals
func bootstrapResample(samples []SampleData, statFunc func([]float64) float64, numResamples int) ConfidenceInterval {
	if len(samples) == 0 {
		return ConfidenceInterval{}
	}

	times := make([]float64, len(samples))
	for i, sample := range samples {
		times[i] = float64(sample.TotalTime.Microseconds())
	}

	// Calculate the actual statistic from the original sample
	estimate := statFunc(times)

	// Perform bootstrap resampling
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	bootstrapStats := make([]float64, numResamples)

	for i := 0; i < numResamples; i++ {
		// Resample with replacement
		resample := make([]float64, len(times))
		for j := 0; j < len(times); j++ {
			idx := rng.Intn(len(times))
			resample[j] = times[idx]
		}
		bootstrapStats[i] = statFunc(resample)
	}

	// Sort bootstrap statistics
	sort.Float64s(bootstrapStats)

	// Calculate confidence interval bounds (95% CI)
	// Using percentile bootstrap method
	alpha := 1.0 - confidenceLevel
	lowerIdx := int(float64(numResamples) * (alpha / 2.0))
	upperIdx := int(float64(numResamples) * (1.0 - alpha/2.0))

	if lowerIdx < 0 {
		lowerIdx = 0
	}
	if upperIdx >= numResamples {
		upperIdx = numResamples - 1
	}

	return ConfidenceInterval{
		LowerBound: bootstrapStats[lowerIdx],
		Estimate:   estimate,
		UpperBound: bootstrapStats[upperIdx],
	}
}

// PrintStatistics outputs statistics in a criterion-style format
func (bp *BenchmarkPlots) PrintStatistics() {
	for operation, samples := range bp.samples {
		if len(samples) == 0 {
			continue
		}

		stats := calculateStatistics(samples)

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Printf("%s: Additional Statistics\n", operation)
		fmt.Println(strings.Repeat("=", 80))

		// Calculate all confidence intervals using bootstrap resampling

		// Throughput CI (inverted from time)
		throughputCI := bootstrapResample(samples, func(times []float64) float64 {
			sum := 0.0
			for _, t := range times {
				sum += t
			}
			meanTimeUs := sum / float64(len(times))
			meanTimeSec := meanTimeUs / 1_000_000.0
			return 1.0 / meanTimeSec // ops/sec
		}, bootstrapSamples)

		// Mean CI
		meanCI := bootstrapResample(samples, func(times []float64) float64 {
			sum := 0.0
			for _, t := range times {
				sum += t
			}
			return sum / float64(len(times))
		}, bootstrapSamples)

		// Std. Dev CI
		stdDevCI := bootstrapResample(samples, func(times []float64) float64 {
			mean := 0.0
			for _, t := range times {
				mean += t
			}
			mean /= float64(len(times))

			variance := 0.0
			for _, t := range times {
				diff := t - mean
				variance += diff * diff
			}
			return math.Sqrt(variance / float64(len(times)))
		}, bootstrapSamples)

		// Median CI
		medianCI := bootstrapResample(samples, func(times []float64) float64 {
			sorted := make([]float64, len(times))
			copy(sorted, times)
			sort.Float64s(sorted)
			return calculateMedian(sorted)
		}, bootstrapSamples)

		// MAD CI
		madCI := bootstrapResample(samples, func(times []float64) float64 {
			sorted := make([]float64, len(times))
			copy(sorted, times)
			sort.Float64s(sorted)
			median := calculateMedian(sorted)
			return calculateMAD(sorted, median)
		}, bootstrapSamples)

		// R² CI - need to calculate R² for bootstrapped samples
		r2CI := ConfidenceInterval{
			LowerBound: stats.R2,
			Estimate:   stats.R2,
			UpperBound: stats.R2,
		}

		// For R², we'd need to bootstrap the entire sample set with indices
		// This is more complex, so we'll use a simplified approach
		// In criterion.rs, they bootstrap the linear regression slopes
		if len(samples) > 10 {
			r2Samples := make([]float64, 1000) // Reduced for R² calculation
			rng := rand.New(rand.NewSource(time.Now().UnixNano()))

			for i := 0; i < 1000; i++ {
				// Resample samples (not just times)
				resampledData := make([]SampleData, len(samples))
				for j := 0; j < len(samples); j++ {
					idx := rng.Intn(len(samples))
					resampledData[j] = samples[idx]
				}
				r2Samples[i] = calculateR2(resampledData)
			}

			sort.Float64s(r2Samples)
			lowerIdx := int(float64(1000) * 0.025)
			upperIdx := int(float64(1000) * 0.975)
			if lowerIdx < 0 {
				lowerIdx = 0
			}
			if upperIdx >= 1000 {
				upperIdx = 999
			}

			r2CI.LowerBound = r2Samples[lowerIdx]
			r2CI.UpperBound = r2Samples[upperIdx]
		}

		// Print table header
		fmt.Printf("%-15s %15s %15s %15s\n", "", "Lower bound", "Estimate", "Upper bound")

		// Print each statistic
		fmt.Printf("%-15s %15s %15s %15s\n",
			"Throughput",
			formatThroughput(throughputCI.LowerBound),
			formatThroughput(throughputCI.Estimate),
			formatThroughput(throughputCI.UpperBound))

		fmt.Printf("%-15s %15.7f %15.7f %15.7f\n",
			"R²",
			r2CI.LowerBound,
			r2CI.Estimate,
			r2CI.UpperBound)

		fmt.Printf("%-15s %15s %15s %15s\n",
			"Mean",
			formatDuration(meanCI.LowerBound),
			formatDuration(meanCI.Estimate),
			formatDuration(meanCI.UpperBound))

		fmt.Printf("%-15s %15s %15s %15s\n",
			"Std. Dev.",
			formatDuration(stdDevCI.LowerBound),
			formatDuration(stdDevCI.Estimate),
			formatDuration(stdDevCI.UpperBound))

		fmt.Printf("%-15s %15s %15s %15s\n",
			"Median",
			formatDuration(medianCI.LowerBound),
			formatDuration(medianCI.Estimate),
			formatDuration(medianCI.UpperBound))

		fmt.Printf("%-15s %15s %15s %15s\n",
			"MAD",
			formatDuration(madCI.LowerBound),
			formatDuration(madCI.Estimate),
			formatDuration(madCI.UpperBound))
	}
}

// formatThroughput formats throughput in ops/sec
func formatThroughput(opsPerSec float64) string {
	if opsPerSec >= 1_000_000 {
		return fmt.Sprintf("%.3f Melem/s", opsPerSec/1_000_000)
	} else if opsPerSec >= 1_000 {
		return fmt.Sprintf("%.3f Kelem/s", opsPerSec/1_000)
	}
	return fmt.Sprintf("%.3f elem/s", opsPerSec)
}

// formatDuration formats duration in appropriate units
func formatDuration(microseconds float64) string {
	if microseconds >= 1_000_000 {
		return fmt.Sprintf("%.2f s", microseconds/1_000_000)
	} else if microseconds >= 1_000 {
		return fmt.Sprintf("%.2f ms", microseconds/1_000)
	}
	return fmt.Sprintf("%.2f µs", microseconds)
}
