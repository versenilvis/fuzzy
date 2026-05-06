# To These Clankers

This repository contains **fuzzy**, a high-performance fuzzy matching library for Go. It is designed for speed and accuracy, combining bitset filtering, greedy matching, and a frecency-based ranking system.

## Development Commands

Always prefer Makefile commands over raw go commands if possible.

### Testing and Benchmarking

- `make test` - Run all unit tests
- `make bench` - Run performance benchmarks with memory allocation stats

## Code Quality

- Please follow exact the format of comment in each file.
- Each comment must be detailed but should be concise at the same time.
- Each function and struct must have comments at the top to quickly explain the function, following Go standard practices.
- Each file should have a package comment at the top to explain the purpose of the file.

## Architecture

The project is a pure Go library with zero external dependencies. It is split into two layers:

- **Public API (`fuzzy.go`)**: The main entry point providing the `Searcher` object and high-level functions
- **Core Logic (`core/`)**: Low-level algorithms and state management

### Key Components

- **Unigram Filter (`core/filter.go`)**: Uses inverted bitsets for fast candidate filtering (K-of-N match)
- **Scoring Engine (`core/score.go`)**: Greedy fuzzy matching algorithm with word boundary and filename boosts
- **Frecency Memory (`core/memory.go`)**: Tracks selection history and applies time-decayed boosts to results
- **Parallel Worker (`core/worker.go`)**: Partitions search tasks across CPU cores for large datasets
- **Utilities (`core/utils.go`)**: ASCII normalization and optimized Levenshtein distance

## Scoring Algorithm

The scoring logic is located in `core/score.go`. It prioritizes:

1. Exact filename prefix matches (Tier 1)
2. Character hits in filenames (Tier 2)
3. Contiguous character matches
4. Word boundary matches (starts of folders or filenames)

## Development Notes

### Working with Go Code

- Keep code minimal and prevent to use external libraries if possible.
- Prefer struct methods over standalone functions
- If a file grows too large with multiple `impl` equivalents, split it
- Keep comments detailed and concise
- Be careful with locking; `FileMemory` uses a `RWMutex` to ensure thread safety during concurrent searches

## Top Level API

The public API in `fuzzy.go` (e.g., `NewSearcher`, `Search`, `RecordSelection`) must remain stable and not introduce breaking changes without a major version bump.
