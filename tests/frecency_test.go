package fuzzy_test

import (
	"testing"

	"github.com/versenilvis/fuzzy"
)

// TestFrecencyOverPerfectMatch verifies that a frequently selected item
// will eventually outrank a perfect character match that hasn't been selected
func TestFrecencyOverPerfectMatch(t *testing.T) {
	items := []string{
		"abc/main.go",
		"xyz/mai.go",
	}

	searcher := fuzzy.NewSearcher(items)

	// query 'mai' matches 'mai.go' perfectly (index 0)
	// it also matches 'main.go' as a prefix (index 0)
	// without history, 'mai.go' would win due to shorter length

	// simulate opening abc/main.go 10 times to build frecency
	for range 10 {
		searcher.RecordSelection("mai", "abc/main.go")
	}

	results := searcher.SearchWithScores("mai")

	t.Logf("search results for 'mai':")
	for _, res := range results {
		t.Logf("file: %s | score: %d", res.Str, res.Score)
	}

	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// abc/main.go should be the top result due to the massive frecency boost
	if results[0].Str != "abc/main.go" {
		t.Errorf("expected top result to be 'abc/main.go', got '%s'", results[0].Str)
	}

	if results[0].Score <= results[1].Score {
		t.Errorf("expected frequently opened file to have higher score (%d) than perfect match (%d)", results[0].Score, results[1].Score)
	}
}
