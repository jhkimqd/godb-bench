package db

import (
	"context"
	"crypto/sha256"
	"fmt"

	triedb "github.com/base/triedb-go"
	"github.com/holiman/uint256"
	"github.com/magiconair/properties"
	"github.com/pingcap/go-ycsb/pkg/ycsb"
)

type trieDB struct {
	db      *triedb.Database
	account triedb.Address // Single account to use for all storage
}

func (t *trieDB) Close() error {
	return t.db.Close()
}

func (t *trieDB) InitThread(ctx context.Context, threadID int, threadCount int) context.Context {
	return ctx
}

func (t *trieDB) CleanupThread(ctx context.Context) {
}

// keyToSlot converts a string key to a 32-byte storage slot
func keyToSlot(key string) triedb.Hash {
	hash := sha256.Sum256([]byte(key))
	return hash
}

// bytesToHash converts a byte slice to a 32-byte hash
func bytesToHash(data []byte) triedb.Hash {
	var hash triedb.Hash
	if len(data) > 32 {
		copy(hash[:], data[:32])
	} else {
		copy(hash[:], data)
	}
	return hash
}

func (t *trieDB) Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error) {
	tx, err := t.db.BeginRO()
	if err != nil {
		return nil, fmt.Errorf("failed to begin read transaction: %w", err)
	}
	defer tx.Commit()

	slot := keyToSlot(key)
	value, err := tx.GetStorage(t.account, slot)
	if err != nil {
		return nil, fmt.Errorf("failed to read key %s: %w", key, err)
	}

	if value == nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	data := make(map[string][]byte)
	data[fields[0]] = value[:]
	return data, nil
}

func (t *trieDB) Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error) {
	return nil, fmt.Errorf("scan is not supported")
}

func (t *trieDB) Update(ctx context.Context, table string, key string, values map[string][]byte) error {
	return t.Insert(ctx, table, key, values)
}

func (t *trieDB) Insert(ctx context.Context, table string, key string, values map[string][]byte) error {
	tx, err := t.db.BeginRW()
	if err != nil {
		return fmt.Errorf("failed to begin write transaction: %w", err)
	}

	// In YCSB, there is only one field.
	for _, value := range values {
		slot := keyToSlot(key)
		hash := bytesToHash(value)

		if err := tx.SetStorage(t.account, slot, &hash); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to write key %s: %w", key, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		return nil
	}
	return nil
}

func (t *trieDB) Delete(ctx context.Context, table string, key string) error {
	tx, err := t.db.BeginRW()
	if err != nil {
		return fmt.Errorf("failed to begin write transaction: %w", err)
	}

	slot := keyToSlot(key)
	if err := tx.SetStorage(t.account, slot, nil); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

type triedbCreator struct{}

func (c triedbCreator) Create(p *properties.Properties) (ycsb.DB, error) {
	path := p.GetString("datadir", "/tmp/triedb")

	// Check if we should use an existing database or create new
	useExisting := p.GetBool("triedb.use_existing", true)

	var db *triedb.Database
	var err error

	if useExisting {
		// Try to open existing database first
		db, err = triedb.Open(path)
		if err != nil {
			// If opening fails, try to create a new one
			db, err = triedb.CreateNew(path)
			if err != nil {
				return nil, fmt.Errorf("failed to open or create database at %s: %w", path, err)
			}
		}
	} else {
		// Force create new database (will fail if exists)
		db, err = triedb.CreateNew(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create new database at %s (use triedb.use_existing=true to open existing): %w", path, err)
		}
	}

	// Use a fixed account address for all storage operations
	// This is a dummy account since YCSB is just key-value, not account-based
	var account triedb.Address
	copy(account[:], []byte("YCSB_BENCHMARK_ACCOUNT__"))

	// Ensure the account exists with initial values
	tx, err := db.BeginRW()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Check if account exists, if not create it
	existingAccount, err := tx.GetAccount(account)
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, fmt.Errorf("failed to check account: %w", err)
	}

	if existingAccount == nil {
		// Create account with initial values
		newAccount := &triedb.Account{
			Nonce:       0,
			Balance:     uint256.NewInt(0),
			StorageRoot: triedb.Hash{},
			CodeHash:    make([]byte, 32),
		}
		if err := tx.SetAccount(account, newAccount); err != nil {
			tx.Rollback()
			db.Close()
			return nil, fmt.Errorf("failed to create account: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to commit account creation: %w", err)
	}

	return &trieDB{db: db, account: account}, nil
}

func init() {
	ycsb.RegisterDBCreator("triedb", triedbCreator{})
}
