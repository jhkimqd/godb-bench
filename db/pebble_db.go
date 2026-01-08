package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/magiconair/properties"
	"github.com/pingcap/go-ycsb/pkg/ycsb"
)

type pebbleDB struct {
	db *pebble.DB
}

func (p *pebbleDB) Close() error {
	return p.db.Close()
}

func (p *pebbleDB) InitThread(ctx context.Context, threadID int, threadCount int) context.Context {
	return ctx
}

func (p *pebbleDB) CleanupThread(ctx context.Context) {
}

func (p *pebbleDB) Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error) {
	value, closer, err := p.db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	data := make(map[string][]byte)
	data[fields[0]] = value
	return data, nil
}

func (p *pebbleDB) Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error) {
	return nil, fmt.Errorf("scan is not supported")
}

func (p *pebbleDB) Update(ctx context.Context, table string, key string, values map[string][]byte) error {
	return p.Insert(ctx, table, key, values)
}

func (p *pebbleDB) Insert(ctx context.Context, table string, key string, values map[string][]byte) error {
	// In YCSB, there is only one field.
	for _, value := range values {
		return p.db.Set([]byte(key), value, pebble.Sync)
	}
	return nil
}

func (p *pebbleDB) Delete(ctx context.Context, table string, key string) error {
	return p.db.Delete([]byte(key), pebble.Sync)
}

// BatchInsert inserts multiple records in a single batch
func (p *pebbleDB) BatchInsert(ctx context.Context, table string, keys []string, values []map[string][]byte) error {
	if len(keys) == 0 {
		return nil
	}

	batch := p.db.NewBatch()
	defer batch.Close()

	for i, key := range keys {
		// In YCSB, there is only one field per record
		for _, value := range values[i] {
			if err := batch.Set([]byte(key), value, pebble.Sync); err != nil {
				return fmt.Errorf("failed to add key %s to batch: %w", key, err)
			}
			break // Only one field in YCSB
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	return nil
}

// BatchUpdate updates multiple records in a single batch
func (p *pebbleDB) BatchUpdate(ctx context.Context, table string, keys []string, values []map[string][]byte) error {
	return p.BatchInsert(ctx, table, keys, values)
}

// BatchRead reads multiple records
func (p *pebbleDB) BatchRead(ctx context.Context, table string, keys []string, fields []string) ([]map[string][]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	results := make([]map[string][]byte, len(keys))
	for i, key := range keys {
		value, closer, err := p.db.Get([]byte(key))
		if err != nil {
			return nil, fmt.Errorf("failed to read key %s in batch: %w", key, err)
		}

		data := make(map[string][]byte)
		data[fields[0]] = value
		results[i] = data
		closer.Close()
	}

	return results, nil
}

// BatchDelete deletes multiple records in a single batch
func (p *pebbleDB) BatchDelete(ctx context.Context, table string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	batch := p.db.NewBatch()
	defer batch.Close()

	for _, key := range keys {
		if err := batch.Delete([]byte(key), pebble.Sync); err != nil {
			return fmt.Errorf("failed to add key %s to delete batch: %w", key, err)
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit delete batch: %w", err)
	}
	return nil
}

// Metrics returns the PebbleDB metrics
func (p *pebbleDB) Metrics() *pebble.Metrics {
	return p.db.Metrics()
}

type pebbleCreator struct{}

func (c pebbleCreator) Create(p *properties.Properties) (ycsb.DB, error) {
	path := p.GetString("datadir", "/tmp/pebble")

	// Check if we should use an existing database or create new
	useExisting := p.GetBool("pebble.use_existing", true)

	// Check if a config file is specified for custom options
	configFile := p.GetString("pebble.config", "")
	var opts *pebble.Options

	if configFile != "" {
		// Load custom Pebble options from JSON file
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read pebble config file %s: %w", configFile, err)
		}

		opts = &pebble.Options{}
		if err := json.Unmarshal(data, opts); err != nil {
			return nil, fmt.Errorf("failed to parse pebble config file %s: %w", configFile, err)
		}
	} else {
		// Use default options
		opts = &pebble.Options{}
	}

	// Allow override of cache size via property
	if p.GetString("pebble.cache_size", "") != "" {
		cacheSize := p.GetInt64("pebble.cache_size", 8<<20) // default 8MB
		opts.Cache = pebble.NewCache(cacheSize)
		defer opts.Cache.Unref()
	}

	// Allow override of write buffer size
	if p.GetString("pebble.memtable_size", "") != "" {
		opts.MemTableSize = uint64(p.GetInt64("pebble.memtable_size", 4<<20)) // default 4MB
	}

	// Allow override of max open files
	if p.GetString("pebble.max_open_files", "") != "" {
		opts.MaxOpenFiles = int(p.GetInt("pebble.max_open_files", 1000))
	}

	var db *pebble.DB
	var err error

	if useExisting {
		// Check if the database directory exists before trying to open it.
		_, statErr := os.Stat(path)
		dbExists := !os.IsNotExist(statErr)

		db, err = pebble.Open(path, opts)
		if err != nil {
			// If opening fails, it might be corrupted. Clean and recreate.
			fmt.Printf("Failed to open database at %s (%v), recreating...\n", path, err)
			if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to clean database directory at %s: %w", path, err)
			}
			db, err = pebble.Open(path, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to create database at %s: %w", path, err)
			}
			fmt.Printf("Created new database at %s\n", path)
		} else {
			if dbExists {
				fmt.Printf("Using existing database at %s\n", path)
			} else {
				fmt.Printf("Created new database at %s\n", path)
			}
		}
	} else {
		// Force create new database - clean directory first
		fmt.Printf("Creating new database at %s\n", path)
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to clean database directory at %s: %w", path, err)
		}
		db, err = pebble.Open(path, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create new database at %s (use pebble.use_existing=true to open existing): %w", path, err)
		}
	}

	return &pebbleDB{db: db}, nil
}

func init() {
	ycsb.RegisterDBCreator("pebble", pebbleCreator{})
}
