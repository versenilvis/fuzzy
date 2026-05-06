<!-- <div align="center">
 <img width="20%" width="1920" height="1920" alt="gopher-min" src="https://github.com/user-attachments/assets/a7f7729e-2e34-4ecc-8866-c8c85d93f233" />

  <h1>Fuzzy</h1>

  [![License: 0BSD](https://img.shields.io/badge/License-0BSD-blue?style=for-the-badge&logo=github&logoColor=white)](./LICENSE)
  [![Status](https://img.shields.io/badge/status-beta-yellow?style=for-the-badge&logo=github&logoColor=white)]()
  [![Documentation](https://img.shields.io/badge/docs-available-brightgreen?style=for-the-badge&logo=github&logoColor=white)](./fuzzy.go)
</div> -->

# Fuzzy

<p><b>Fuzzy is a fast and accurate fuzzy matching library for file search. It combines multiple search algorithms with a smart frecency-based ranking system to provide relevant results quickly.</b></p>

> [!IMPORTANT]
> **Fuzzy focuses on accuracy and typo tolerance while maintaining high performance.**  
> It is optimized for file path searching rather than just single strings.
> This package is intended for local use or side projects.

<br>

## Features

- **Typo tolerance**: Handles common typing errors using Levenshtein distance
- **Multi-algorithm**: Uses bitset filtering and optimal alignment fuzzy matching
- **Frecency ranking**: Learns from user behavior to prioritize frequently and recently used files
- **Parallel processing**: Scalable performance for large datasets
- **Thread-safe**: Safe for concurrent use in multi-threaded applications

## Installation

```bash
go get github.com/versenilvis/fuzzy
```

**Requirements**: Go 1.21+

## Usage

<details open>
  <summary><b>Basic Example</b></summary>
<br>

```go
package main

import (
	"fmt"
	"github.com/versenilvis/fuzzy"
)

func main() {
	// 1. Create Searcher
	files := []string{
		"/home/user/Documents/report.pdf",
		"/home/user/Documents/contract.docx",
		"/home/user/Music/song.mp3",
		"/home/user/Code/main.go",
		"/home/user/Code/utils.go",
	}

	searcher := fuzzy.NewSearcher(files)

	// 2. Basic Search
	fmt.Println("--- Searching 'report' ---")
	results := searcher.Search("report")
	for _, path := range results {
		fmt.Println("  ->", path)
	}

	// 3. Typo Tolerance
	fmt.Println("\n--- Searching 'maiin' (typo) ---")
	results = searcher.Search("maiin")
	for _, path := range results {
		fmt.Println("  ->", path)
	}

	// 4. Learning from User behavior
	searcher.RecordSelection("main", "/home/user/Code/main.go")
	
	// Subsequent searches will prioritize this file
	results = searcher.Search("mai")
}
```
</details>

## API Reference

### `NewSearcher(items []string) *Searcher`
Creates a new searcher optimized for file paths.

### `Search(query string, opts ...SearchOption) []string`
Returns the top matching results for a query, automatically applying frecency boosts.

### `RecordSelection(query, selected string)`
Records a selection to increase the priority (frecency) of a file for future searches.

## License

This project is licensed under the [0BSD License](LICENSE). Meaning you can do whatever you want with it.
