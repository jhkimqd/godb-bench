package metrics

import (
	"context"
	"time"

	"github.com/pingcap/go-ycsb/pkg/ycsb"
)

// TrackedDB wraps a YCSB DB and tracks operation metrics
type TrackedDB struct {
	db        ycsb.DB
	collector *Collector
}

// NewTrackedDB creates a new metrics-tracking DB wrapper
func NewTrackedDB(db ycsb.DB, collector *Collector) *TrackedDB {
	return &TrackedDB{
		db:        db,
		collector: collector,
	}
}

// Close closes the underlying database
func (t *TrackedDB) Close() error {
	return t.db.Close()
}

// InitThread initializes a thread context
func (t *TrackedDB) InitThread(ctx context.Context, threadID int, threadCount int) context.Context {
	return t.db.InitThread(ctx, threadID, threadCount)
}

// CleanupThread cleans up a thread context
func (t *TrackedDB) CleanupThread(ctx context.Context) {
	t.db.CleanupThread(ctx)
}

// Read performs a read and tracks metrics
func (t *TrackedDB) Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error) {
	start := time.Now()
	result, err := t.db.Read(ctx, table, key, fields)
	t.collector.RecordRead(time.Since(start))
	return result, err
}

// Scan performs a scan and tracks metrics
func (t *TrackedDB) Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error) {
	start := time.Now()
	result, err := t.db.Scan(ctx, table, startKey, count, fields)
	t.collector.RecordScan(time.Since(start))
	return result, err
}

// Update performs an update and tracks metrics
func (t *TrackedDB) Update(ctx context.Context, table string, key string, values map[string][]byte) error {
	start := time.Now()
	err := t.db.Update(ctx, table, key, values)
	t.collector.RecordUpdate(time.Since(start))
	return err
}

// Insert performs an insert and tracks metrics
func (t *TrackedDB) Insert(ctx context.Context, table string, key string, values map[string][]byte) error {
	start := time.Now()
	err := t.db.Insert(ctx, table, key, values)
	t.collector.RecordInsert(time.Since(start))
	return err
}

// Delete performs a delete and tracks metrics
func (t *TrackedDB) Delete(ctx context.Context, table string, key string) error {
	start := time.Now()
	err := t.db.Delete(ctx, table, key)
	t.collector.RecordDelete(time.Since(start))
	return err
}

// GetCollector returns the underlying metrics collector
func (t *TrackedDB) GetCollector() *Collector {
	return t.collector
}

// GetUnderlyingDB returns the unwrapped database (for getting DB-specific metrics)
func (t *TrackedDB) GetUnderlyingDB() ycsb.DB {
	return t.db
}
