package metrics

import (
	"context"
	"testing"
	"time"
)

// mockDB is a simple mock database for testing
type mockDB struct{}

func (m *mockDB) Close() error { return nil }
func (m *mockDB) InitThread(ctx context.Context, threadID int, threadCount int) context.Context {
	return ctx
}
func (m *mockDB) CleanupThread(ctx context.Context) {}
func (m *mockDB) Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error) {
	time.Sleep(1 * time.Millisecond) // Simulate work
	return map[string][]byte{"field": []byte("value")}, nil
}
func (m *mockDB) Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error) {
	time.Sleep(2 * time.Millisecond) // Simulate work
	return nil, nil
}
func (m *mockDB) Update(ctx context.Context, table string, key string, values map[string][]byte) error {
	time.Sleep(1 * time.Millisecond) // Simulate work
	return nil
}
func (m *mockDB) Insert(ctx context.Context, table string, key string, values map[string][]byte) error {
	time.Sleep(1 * time.Millisecond) // Simulate work
	return nil
}
func (m *mockDB) Delete(ctx context.Context, table string, key string) error {
	time.Sleep(1 * time.Millisecond) // Simulate work
	return nil
}

func TestCollectorBasic(t *testing.T) {
	collector := NewCollector()

	// Record some operations
	collector.RecordRead(1 * time.Millisecond)
	collector.RecordRead(2 * time.Millisecond)
	collector.RecordUpdate(3 * time.Millisecond)
	collector.RecordInsert(4 * time.Millisecond)

	// Check counts
	if collector.readCount.Load() != 2 {
		t.Errorf("Expected 2 reads, got %d", collector.readCount.Load())
	}
	if collector.updateCount.Load() != 1 {
		t.Errorf("Expected 1 update, got %d", collector.updateCount.Load())
	}
	if collector.insertCount.Load() != 1 {
		t.Errorf("Expected 1 insert, got %d", collector.insertCount.Load())
	}
}

func TestTrackedDB(t *testing.T) {
	mock := &mockDB{}
	collector := NewCollector()
	tracked := NewTrackedDB(mock, collector)

	ctx := context.Background()

	// Perform operations
	_, _ = tracked.Read(ctx, "table", "key", []string{"field"})
	_ = tracked.Update(ctx, "table", "key", map[string][]byte{"field": []byte("value")})
	_ = tracked.Insert(ctx, "table", "key", map[string][]byte{"field": []byte("value")})

	// Check counts
	if collector.readCount.Load() != 1 {
		t.Errorf("Expected 1 read, got %d", collector.readCount.Load())
	}
	if collector.updateCount.Load() != 1 {
		t.Errorf("Expected 1 update, got %d", collector.updateCount.Load())
	}
	if collector.insertCount.Load() != 1 {
		t.Errorf("Expected 1 insert, got %d", collector.insertCount.Load())
	}

	// Check that latencies were recorded (histograms should have values)
	if collector.readLatency.TotalCount() != 1 {
		t.Errorf("Expected 1 read latency, got %d", collector.readLatency.TotalCount())
	}
}

func TestReadAmplification(t *testing.T) {
	collector := NewCollector()

	// Record reads with amplification
	collector.RecordReadWithAmp(1*time.Millisecond, 3)
	collector.RecordReadWithAmp(2*time.Millisecond, 5)
	collector.RecordReadWithAmp(3*time.Millisecond, 4)

	// Check amplification tracking
	if collector.readAmpCount.Load() != 3 {
		t.Errorf("Expected 3 read amp samples, got %d", collector.readAmpCount.Load())
	}

	expectedSum := uint64(3 + 5 + 4)
	if collector.readAmpSum.Load() != expectedSum {
		t.Errorf("Expected read amp sum %d, got %d", expectedSum, collector.readAmpSum.Load())
	}

	// Average should be 4.0
	avg := float64(collector.readAmpSum.Load()) / float64(collector.readAmpCount.Load())
	if avg != 4.0 {
		t.Errorf("Expected avg read amp 4.0, got %.2f", avg)
	}
}
