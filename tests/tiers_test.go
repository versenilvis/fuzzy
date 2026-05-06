package fuzzy_test

import (
	"testing"
	"github.com/versenilvis/fuzzy"
)

// TestPriorityTiers verifies that prefix matches and filename hits are 
// correctly categorized into higher scoring tiers
func TestPriorityTiers(t *testing.T) {
	t.Run("Prefix priority (auth)", func(t *testing.T) {
		items := []string{
			"authentication_service.go",
			"lib/auth.go",
			"author_metadata.json",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("auth")

		t.Logf("results for 'auth':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}

		// authentication_service.go and author_metadata.json both have prefix 'auth'
		// lib/auth.go has 'auth' in filename but not as prefix
		if results[0].Score <= results[2].Score {
			t.Errorf("tiers not working correctly for prefix matches")
		}
	})
}
