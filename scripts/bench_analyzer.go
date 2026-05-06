package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type benchResult struct {
	name        string
	time        string
	mem         string
	description string
}

func main() {
	fmt.Fprintln(os.Stderr, "Running benchmarks, please wait...")

	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "./bench/...")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error running benchmarks: %v\n", err)
		return
	}

	results := parseBenchOutput(out.String())

	fmt.Println("\n================================================================================================")
	fmt.Printf("%-35s | %-12s | %-10s | %-30s\n", "BENCHMARK NAME", "TIME", "MEMORY", "DESCRIPTION")
	fmt.Println("================================================================================================")

	for _, r := range results {
		fmt.Printf("%-35s | %-12s | %-10s | %-30s\n", r.name, r.time, r.mem, r.description)
	}
	fmt.Println("================================================================================================")
}

func parseBenchOutput(output string) []benchResult {
	var results []benchResult
	scanner := bufio.NewScanner(strings.NewReader(output))

	// regex to parse: Name-Threads  Iterations  Time/op  Mem/op  Allocs/op
	re := regexp.MustCompile(`Benchmark([\w/]+)-?\d*\s+\d+\s+([\d\.]+)\sns/op\s+([\d\.]+)\sB/op`)

	descriptions := map[string]string{
		"Linux100k/ascii_short": "Real-world dataset (100K files)",
		"Linux100k/ascii_ext":   "Extended ASCII search",
		"Linux100k/path_like":   "Deep nesting (/usr/lib/...)",
		"Linux100k/typo":        "Typo-tolerant search",
		"Linux100k/long_query":  "Long query pattern",
		"Linux100k_NewSearcher": "Build index (~100K files)",
		"Search/1000_files":     "Standard workload (1K files)",
		"Search/10000_files":    "Medium workload (10K files)",
		"Normalize":             "String cleaning (ASCII)",
		"LevenshteinRatio":      "Zero allocation core distance",
		"SearchWithCache":       "Search with internal cache",
		"GetBoostScores":        "Ranking logic (Frecency)",
		"RecordSelection":       "Updating selection history",
	}

	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			name := matches[1]
			ns, _ := strconv.ParseFloat(matches[2], 64)
			bytes, _ := strconv.ParseFloat(matches[3], 64)

			results = append(results, benchResult{
				name:        name,
				time:        formatTime(ns),
				mem:         formatMem(bytes),
				description: descriptions[name],
			})
		}
	}
	return results
}

func formatTime(ns float64) string {
	if ns >= 1000000 {
		return fmt.Sprintf("%.2fms", ns/1000000)
	}
	if ns >= 1000 {
		return fmt.Sprintf("%.2fµs", ns/1000)
	}
	return fmt.Sprintf("%.1fns", ns)
}

func formatMem(b float64) string {
	if b >= 1024*1024 {
		return fmt.Sprintf("%.2fMB", b/(1024*1024))
	}
	if b >= 1024 {
		return fmt.Sprintf("%.2fKB", b/1024)
	}
	return fmt.Sprintf("%.0fB", b)
}
