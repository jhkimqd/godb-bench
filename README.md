# godb-bench - Database Benchmark Tool

A CLI tool to run YCSB benchmarks on PebbleDB and TrieDB with support for custom configurations.

## Quick Start

```bash
# Build
go build -o godb-bench

# Run PebbleDB benchmark
./godb-bench pebble ycsb --workload ./workloada.spec

# Run TrieDB benchmark
./godb-bench triedb ycsb --workload ./workloada.spec
```

## Available Commands

```bash
./godb-bench pebble ycsb    # YCSB benchmark for PebbleDB
./godb-bench triedb ycsb    # YCSB benchmark for TrieDB
./godb-bench triedb bench   # Basic TrieDB benchmark
```

## YCSB Workload File

Create a workload file (e.g., `workloada.spec`):

```properties
recordcount=1000
operationcount=1000
workload=core

readallfields=true

readproportion=0.5
updateproportion=0.5
scanproportion=0
insertproportion=0

requestdistribution=uniform
```

**Key Parameters:**
- `recordcount` - Number of records to load
- `operationcount` - Number of operations to perform
- `readproportion`, `updateproportion`, `insertproportion` - Operation mix (must sum to 1.0)
- `requestdistribution` - `uniform`, `zipfian` (hot keys), or `latest`

## Command-Line Options

### Common Flags
```bash
-w, --workload <file>         # Workload specification file (required)
-P, --property_file <file>    # Additional property file
-p, --prop <key>=<value>      # Override individual properties
```

### Override Properties
```bash
-p recordcount=10000          # Override record count
-p operationcount=10000       # Override operation count
-p datadir=/path/to/db        # Database location (default: /tmp/pebble or /tmp/triedb)
```

## PebbleDB Configuration

### Quick Configuration via Properties
```bash
./godb-bench pebble ycsb -w workload.spec \
  -p pebble.cache_size=134217728 \
  -p pebble.memtable_size=67108864 \
  -p pebble.max_open_files=2000
```

**Available Properties:**
- `pebble.cache_size` - Block cache size in bytes (default: 8MB)
- `pebble.memtable_size` - MemTable size in bytes (default: 4MB)
- `pebble.max_open_files` - Max open files (default: 1000)

### Advanced Configuration via JSON
Create a config file (e.g., `pebble-config.json`):
```json
{
  "MemTableSize": 67108864,
  "MaxOpenFiles": 2000,
  "L0CompactionThreshold": 4,
  "L0StopWritesThreshold": 12,
  "MaxConcurrentCompactions": 2,
  "DisableWAL": false
}
```

Use it:
```bash
./godb-bench pebble ycsb -w workload.spec \
  -p pebble.config=./pebble-config.json
```

**Key JSON Options:**
- `MemTableSize` - Size of each memtable (bytes)
- `L0CompactionThreshold` - L0 files before compaction starts
- `L0StopWritesThreshold` - L0 files before blocking writes
- `MaxConcurrentCompactions` - Parallel compaction threads
- `DisableWAL` - Disable write-ahead log (testing only)

See `pebble-config-example.json` for all available options.

## TrieDB Configuration

### Use Existing Database (Default)
```bash
./godb-bench triedb ycsb -w workload.spec \
  -p datadir=/path/to/existing/db
```

### Force Create New Database
```bash
./godb-bench triedb ycsb -w workload.spec \
  -p datadir=/path/to/new/db \
  -p triedb.use_existing=false
```

**Available Properties:**
- `triedb.use_existing` - Open existing DB or create new (default: true)

## Common Use Cases

### 1. Test with Production Configuration
```bash
# Test PebbleDB with production config
./godb-bench pebble ycsb -w workload.spec \
  -p datadir=/tmp/test-db \
  -p pebble.config=./prod-config.json \
  -p recordcount=100000
```

### 2. A/B Test Different Configurations
```bash
# Configuration A: Small cache
./godb-bench pebble ycsb -w workload.spec \
  -p datadir=/tmp/pebble-a \
  -p pebble.cache_size=67108864 \
  > results-a.log

# Configuration B: Large cache
./godb-bench pebble ycsb -w workload.spec \
  -p datadir=/tmp/pebble-b \
  -p pebble.cache_size=268435456 \
  > results-b.log
```

### 3. Compare PebbleDB vs TrieDB
```bash
# PebbleDB
./godb-bench pebble ycsb -w workload.spec \
  -p recordcount=100000 > pebble-results.log

# TrieDB
./godb-bench triedb ycsb -w workload.spec \
  -p recordcount=100000 > triedb-results.log
```

### 4. Test on Existing Database
```bash
# Copy production database
cp -r /prod/pebble /tmp/pebble-test

# Run benchmark
./godb-bench pebble ycsb -w workload.spec \
  -p datadir=/tmp/pebble-test
```

## Example Workloads

### Read-Heavy (95% reads)
```properties
recordcount=1000000
operationcount=1000000
workload=core
readproportion=0.95
updateproportion=0.05
requestdistribution=zipfian
```

### Write-Heavy (90% updates)
```properties
recordcount=1000000
operationcount=1000000
workload=core
readproportion=0.1
updateproportion=0.9
requestdistribution=uniform
```

### Mixed with Hot Keys
```properties
recordcount=1000000
operationcount=1000000
workload=core
readproportion=0.5
updateproportion=0.5
requestdistribution=zipfian
```

## Performance Tuning Tips

### PebbleDB

**For Read-Heavy Workloads:**
```bash
-p pebble.cache_size=268435456  # Larger cache (256MB)
```

**For Write-Heavy Workloads:**
```bash
-p pebble.memtable_size=134217728  # Larger memtable (128MB)
```

**For Maximum Throughput (testing only):**
```json
{
  "DisableWAL": true,
  "MaxConcurrentCompactions": 4
}
```

**For Memory-Constrained Systems:**
```bash
-p pebble.cache_size=8388608     # 8MB
-p pebble.memtable_size=4194304  # 4MB
-p pebble.max_open_files=500
```

### TrieDB

**Warm-Up Before Benchmark:**
```bash
# Warm-up phase
./godb-bench triedb ycsb -w workload.spec \
  -p recordcount=10000 -p operationcount=10000

# Actual benchmark
./godb-bench triedb ycsb -w workload.spec
```

## Common Size Values

- 8 MB = `8388608`
- 16 MB = `16777216`
- 32 MB = `33554432`
- 64 MB = `67108864`
- 128 MB = `134217728`
- 256 MB = `268435456`
- 512 MB = `536870912`
- 1 GB = `1073741824`

## Cleanup

```bash
# Remove test databases
rm -rf /tmp/pebble /tmp/triedb

# Remove specific test directories
rm -rf /tmp/pebble-a /tmp/pebble-b /tmp/test-*
```

## Implementation Notes

### PebbleDB Adapter
- Uses PebbleDB's native key-value interface
- Writes use `pebble.Sync` for durability
- Automatically opens existing databases

### TrieDB Adapter
- Uses TrieDB's account storage interface
- Keys hashed to 32-byte storage slots (SHA-256)
- Values padded/truncated to 32 bytes
- All operations use transactions (RO/RW)

### Limitations
- Scan operations not supported (returns error)
- TrieDB values limited to 32 bytes

## Troubleshooting

**"Please specify a workload file"**
- Ensure you use `-w` or `--workload` flag with a valid file

**"Failed to open database"**
- Check database path exists and has correct permissions
- For TrieDB, try `-p triedb.use_existing=false` to create fresh

**"DB creator not found"**
- Rebuild the project: `go build -o godb-bench`
- Ensure db package is imported correctly

## Examples

See example configuration files:
- `pebble-config-example.json` - Full PebbleDB configuration template
- `pebble-config-test.json` - Simple test configuration
- `workloada.spec` - Example YCSB workload

## Architecture

```
godb-bench/
├── main.go                    # Entry point
├── cmd/
│   ├── root.go               # Root command
│   ├── pebble.go             # PebbleDB parent command
│   ├── pebble_ycsb.go        # PebbleDB YCSB command
│   ├── triedb.go             # TrieDB parent command
│   ├── triedb_ycsb.go        # TrieDB YCSB command
│   └── triedb_bench.go       # TrieDB basic benchmark
└── db/
    ├── pebble_db.go          # PebbleDB YCSB adapter
    └── triedb_db.go          # TrieDB YCSB adapter
```

## License

See LICENSE file in repository root.
