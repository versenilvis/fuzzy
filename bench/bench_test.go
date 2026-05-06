package bench_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/versenilvis/fuzzy"
	"github.com/versenilvis/fuzzy/core"
)

func TestMain(m *testing.M) {
	code := m.Run()
	printPerformanceSummary()
	PrintLinuxSummary()
	os.Exit(code)
}

func generateTestFiles(n int) []string {
	files := make([]string, n)
	for i := 0; i < n; i++ {
		files[i] = fmt.Sprintf("/home/user/project/module_%d/sub_%d/file_%d.go", i/100, i/10, i)
	}
	return files
}

func BenchmarkSearch(b *testing.B) {
	b.Run("1000 files", func(b *testing.B) {
		files := generateTestFiles(1000)
		searcher := fuzzy.NewSearcher(files)
		b.ResetTimer()
		for b.Loop() {
			searcher.Search("file_500")
		}
	})

	b.Run("10000 files", func(b *testing.B) {
		files := generateTestFiles(10000)
		searcher := fuzzy.NewSearcher(files)
		b.ResetTimer()
		for b.Loop() {
			searcher.Search("module_5")
		}
	})
}

func BenchmarkNormalize(b *testing.B) {
	str := "this/is/a/path/with/main.go"
	for b.Loop() {
		core.Normalize(str)
	}
}

func BenchmarkLevenshteinRatio(b *testing.B) {
	s1 := "authentication_service.go"
	s2 := "authenticator_service.go"
	for b.Loop() {
		core.LevenshteinRatio(s1, s2)
	}
}

func printPerformanceSummary() {
	sizes := []int{1000, 10000, 100000}
	
	fmt.Println("\nSEARCH PERFORMANCE SUMMARY")
	fmt.Println("------------------------------------------")
	
	for _, size := range sizes {
		files := generateTestFiles(size)
		searcher := fuzzy.NewSearcher(files)
		query := "file_500"
		if size > 10000 {
			query = "module_50"
		}
		
		searcher.Search(query)
		
		start := time.Now()
		iterations := 200
		for i := 0; i < iterations; i++ {
			searcher.Search(query)
		}
		duration := time.Since(start)
		msPerOp := float64(duration.Nanoseconds()) / float64(iterations) / 1000000.0
		
		fmt.Printf("Dataset: %7d files | %8.4f ms/op\n", size, msPerOp)
	}
	fmt.Println("------------------------------------------")
}
