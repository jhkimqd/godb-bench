package metrics

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// SampleData represents a single benchmark sample
type SampleData struct {
	SampleIndex int64         // The sequential sample number for this operation
	TotalTime   time.Duration // Time taken for this sample
}

// BenchmarkPlots contains data for generating criterion-style plots
type BenchmarkPlots struct {
	samples        map[string][]SampleData // operation -> samples
	sampleCounters map[string]int64        // operation -> current sample count
}

// NewBenchmarkPlots creates a new BenchmarkPlots instance
func NewBenchmarkPlots() *BenchmarkPlots {
	return &BenchmarkPlots{
		samples:        make(map[string][]SampleData),
		sampleCounters: make(map[string]int64),
	}
}

// AddSample records a sample for an operation
// The sample index is automatically incremented for each operation
func (bp *BenchmarkPlots) AddSample(operation string, totalTime time.Duration) {
	bp.sampleCounters[operation]++
	bp.samples[operation] = append(bp.samples[operation], SampleData{
		SampleIndex: bp.sampleCounters[operation],
		TotalTime:   totalTime,
	})
}

// GeneratePlots creates scatter plots for all operations showing progression over time
// outputDir is the directory where plots will be saved
func (bp *BenchmarkPlots) GeneratePlots(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for operation, samples := range bp.samples {
		if len(samples) == 0 {
			continue
		}

		// Generate sample times plot
		if err := bp.generateSampleTimesPlot(operation, samples, outputDir); err != nil {
			fmt.Printf("Warning: failed to generate plot for %s: %v\n", operation, err)
		}
	}

	return nil
}

// generateSampleTimesPlot creates a scatter plot of sample time vs sample index
// Each point represents one sample, showing the progression of operation times
func (bp *BenchmarkPlots) generateSampleTimesPlot(operation string, samples []SampleData, outputDir string) error {
	fmt.Printf("DEBUG: Generating plot for operation '%s' with %d samples.\n", operation, len(samples))
	p, err := plot.New()
	if err != nil {
		return fmt.Errorf("failed to create plot: %w", err)
	}

	p.Title.Text = fmt.Sprintf("%s: Sample Times", operation)
	p.X.Label.Text = "Sample Index"
	p.Y.Label.Text = "Time (µs)"

	// Create scatter plot data
	pts := make(plotter.XYs, len(samples))
	for i, sample := range samples {
		pts[i].X = float64(sample.SampleIndex)
		pts[i].Y = float64(sample.TotalTime.Microseconds())
	}

	// Create scatter plot
	scatter, err := plotter.NewScatter(pts)
	if err != nil {
		return fmt.Errorf("failed to create scatter plot: %w", err)
	}

	// Customize appearance
	scatter.GlyphStyle.Color = color.RGBA{R: 70, G: 130, B: 180, A: 255} // Steel blue
	scatter.GlyphStyle.Radius = vg.Points(1)

	p.Add(scatter)

	// Add grid
	p.Add(plotter.NewGrid())

	// Save the plot
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(outputDir, fmt.Sprintf("%s_%s_sample_times.png", operation, timestamp))
	if err := p.Save(8*vg.Inch, 6*vg.Inch, filename); err != nil {
		return fmt.Errorf("failed to save plot: %w", err)
	}

	fmt.Printf("Generated plot: %s\n", filename)
	return nil
}
