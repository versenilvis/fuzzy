//go:build linux

package bench_test

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/versenilvis/fuzzy"
)

// scanLinuxFiles scans Linux filesystem to get real file paths
func scanLinuxFiles(limit int) []string {
	roots := []string{"/usr", "/etc", "/var", "/opt"}
	var files []string

	for _, root := range roots {
		if len(files) >= limit {
			break
		}
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}
			if len(files) >= limit {
				return filepath.SkipAll
			}
			if !d.IsDir() {
				files = append(files, path)
			}
			return nil
		})
	}

	return files
}

func BenchmarkLinux100k(b *testing.B) {
	limit := 100_000
	files := scanLinuxFiles(limit)
	
	if len(files) < limit {
		b.Skipf("skipping: only found %d files, need %d", len(files), limit)
	}

	searcher := fuzzy.NewSearcher(files)

	queries := []struct {
		name  string
		query string
	}{
		{"ascii_short", "main"},
		{"ascii_ext", "config.yaml"},
		{"path_like", "lib/python"},
		{"typo", "mian"},
		{"long_query", "application.properties"},
	}

	for _, q := range queries {
		b.Run(q.name, func(b *testing.B) {
			for b.Loop() {
				searcher.Search(q.query)
			}
		})
	}
}

func BenchmarkLinux100k_NewSearcher(b *testing.B) {
	files := scanLinuxFiles(100_000)
	if len(files) < 50_000 {
		b.Skipf("need at least 50k files, found %d", len(files))
	}

	b.ResetTimer()
	for b.Loop() {
		fuzzy.NewSearcher(files)
	}
}

// PrintLinuxSummary prints real-world performance on Linux
func PrintLinuxSummary() {
	limit := 100_000
	files := scanLinuxFiles(limit)
	if len(files) < limit {
		return
	}

	fmt.Println("\nREAL-WORLD LINUX PERFORMANCE (100k files)")
	fmt.Println("------------------------------------------")
	
	searcher := fuzzy.NewSearcher(files)
	queries := []string{"main", "config.yaml", "lib/python"}
	
	for _, q := range queries {
		// Warm up
		searcher.Search(q)
		
		start := time.Now()
		iterations := 100
		for i := 0; i < iterations; i++ {
			searcher.Search(q)
		}
		duration := time.Since(start)
		msPerOp := float64(duration.Nanoseconds()) / float64(iterations) / 1000000.0
		
		fmt.Printf("Query: '%-12s' | %8.4f ms/op\n", q, msPerOp)
	}
	fmt.Println("------------------------------------------")
}
