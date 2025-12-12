package metrics

import (
	"os"
	"testing"
	"time"
)

func TestBenchmarkPlots(t *testing.T) {
	// Create a temporary directory for test plots
	tmpDir := t.TempDir()

	bp := NewBenchmarkPlots()

	// Add sample data for READ operations - simulating varying times
	for i := 0; i < 50; i++ {
		// Simulate varying operation times with some variance
		baseTime := 1000 // 1ms base
		variance := i * 10
		totalTime := time.Duration(baseTime+variance) * time.Microsecond
		bp.AddSample("READ", totalTime)
	}

	// Add sample data for UPDATE operations - slightly slower
	for i := 0; i < 50; i++ {
		baseTime := 1500 // 1.5ms base
		variance := i * 15
		totalTime := time.Duration(baseTime+variance) * time.Microsecond
		bp.AddSample("UPDATE", totalTime)
	}

	// Add sample data for INSERT operations - slowest
	for i := 0; i < 50; i++ {
		baseTime := 2000 // 2ms base
		variance := i * 20
		totalTime := time.Duration(baseTime+variance) * time.Microsecond
		bp.AddSample("INSERT", totalTime)
	}

	// Generate plots
	if err := bp.GeneratePlots(tmpDir); err != nil {
		t.Fatalf("Failed to generate plots: %v", err)
	}

	// Verify that plot files were created
	expectedFiles := []string{
		"READ_sample_times.png",
		"UPDATE_sample_times.png",
		"INSERT_sample_times.png",
	}

	for _, filename := range expectedFiles {
		filepath := tmpDir + "/" + filename
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected plot file not found: %s", filepath)
		} else {
			t.Logf("Successfully created plot: %s", filename)
		}
	}
}

func TestBenchmarkPlotsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	bp := NewBenchmarkPlots()

	// Generate plots with no data (should not error)
	if err := bp.GeneratePlots(tmpDir); err != nil {
		t.Fatalf("Failed to generate plots with no data: %v", err)
	}
}

func TestBenchmarkPlotsSingleSample(t *testing.T) {
	tmpDir := t.TempDir()
	bp := NewBenchmarkPlots()

	// Add single sample
	bp.AddSample("READ", 100*time.Millisecond)

	// Generate plots
	if err := bp.GeneratePlots(tmpDir); err != nil {
		t.Fatalf("Failed to generate plots with single sample: %v", err)
	}

	// Verify that plot file was created
	filepath := tmpDir + "/READ_sample_times.png"
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Errorf("Expected plot file not found: %s", filepath)
	}
}
