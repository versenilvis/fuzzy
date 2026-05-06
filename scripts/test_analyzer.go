//go:build ignore
// for my personal use, dont mind it
package main

import (
	"fmt"
	"github.com/versenilvis/fuzzy"
)

type scenario struct {
	name          string
	query         string
	files         []string
	expectedFirst string
}

func main() {
	scenarios := []scenario{
		{
			name:          "Greedy Trap (repeated chars)",
			query:         "ff",
			files:         []string{"f_static_ff.go", "flat_file_util.go"},
			expectedFirst: "f_static_ff.go",
		},
		{
			name:          "Word Boundaries (api)",
			query:         "api",
			files:         []string{"capillary_action.py", "src/internal/api_client.go", "apple_interface.go"},
			expectedFirst: "src/internal/api_client.go",
		},
		{
			name:          "Compactness (config)",
			query:         "config",
			files:         []string{"core/cfg/ob-navigation-git.json", "settings/config_backup.json"},
			expectedFirst: "settings/config_backup.json",
		},
		{
			name:          "Path Depth (main)",
			query:         "main",
			files:         []string{"src/internal/core/bootstrap/main.go", "src/main.go", "main.go"},
			expectedFirst: "main.go",
		},
		{
			name:          "Space Boundaries (this file)",
			query:         "tfhs",
			files:         []string{"test_file_has_system.log", "this file has space.pdf"},
			expectedFirst: "this file has space.pdf",
		},
	}

	fmt.Println("==================================================================================")
	fmt.Printf("%-35s | %-10s | %-10s | %-8s\n", "SCENARIO / FILE", "QUERY", "SCORE", "STATUS")
	fmt.Println("==================================================================================")

	overallPass := true
	for _, s := range scenarios {
		fmt.Printf("[%s]\n", s.name)
		searcher := fuzzy.NewSearcher(s.files)
		results := searcher.SearchWithScores(s.query)

		if len(results) == 0 {
			fmt.Printf("  %-33s | %-10s | %-10s | %-8s\n", "NO RESULTS", s.query, "-", "FAIL")
			overallPass = false
			continue
		}

		for i, res := range results {
			status := "   -"
			if i == 0 {
				if res.Str == s.expectedFirst {
					status = "PASS"
				} else {
					status = "FAIL"
					overallPass = false
				}
			}
			fmt.Printf("  %-33s | %-10s | %-10d | %-8s\n", res.Str, s.query, res.Score, status)
		}
		fmt.Println("----------------------------------------------------------------------------------")
	}

	fmt.Println()
	if overallPass {
		fmt.Println("FINAL VERDICT: [SUCCESS] All scoring scenarios match expectations!")
	} else {
		fmt.Println("FINAL VERDICT: [FAILED] Some scenarios did not return the expected top result.")
	}
}
