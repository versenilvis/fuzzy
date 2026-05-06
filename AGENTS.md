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
- **Scoring Engine (`core/score.go`)**: Greedy fuzzy matching with **Right-to-Left** alignment to prioritize filenames
- **Frecency Memory (`core/memory.go`)**: Thread-safe history tracking with `RWMutex` and time-decayed boosts
- **Parallel Worker (`core/worker.go`)**: Smart partitioning of search tasks optimized for high-core CI/CD environments
- **Utilities (`core/utils.go`)**: ASCII-optimized normalization and zero-allocation Levenshtein distance

## Scoring Algorithm

The engine uses a multi-tier scoring strategy located in `core/score.go`:

1.  **Tier 1 (Prefix Match)**: Bonus for exact prefix matches on the filename (highest priority)
2.  **Tier 2 (Density Match)**: Boost for character hits specifically within the filename (ignoring path noise)
3.  **Tier 3 (Right-to-Left Greedy)**: Aligns query characters from right to left to ensure the last query character matches a deep path component
4.  **Tier 4 (Structural Boost)**: Extra points for word boundaries (e.g., `_`, `-`, `/`, or spaces)

## Development Notes

### Working with Go Code

- Keep code minimal and prevent to use external libraries if possible.
- Prefer struct methods over standalone functions
- If a file grows too large with multiple `impl` equivalents, split it
- Keep comments detailed and concise
- Be careful with locking; `FileMemory` uses a `RWMutex` to ensure thread safety during concurrent searches
- Use `any` type for generics to avoid type casting
- Use ` for := range n` instead of `for i` when possible
- Use `min` and `max` functions to avoid branching

## Top Level API

The public API in `fuzzy.go` (e.g., `NewSearcher`, `Search`, `RecordSelection`) must remain stable and not introduce breaking changes without a major version bump.
