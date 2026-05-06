package fuzzy_test

import (
	"testing"
	"github.com/versenilvis/fuzzy"
)

// TestCompactness verifies that contiguous matches win over sparse ones, 
// even if the sparse path is shorter
func TestCompactness(t *testing.T) {
	t.Run("Contiguous vs Sparsity (model)", func(t *testing.T) {
		items := []string{
			"app/core/database/v1/user_model.go",
			"manual_over_delivery.txt",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("model")

		t.Logf("results for 'model':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}

		if results[0].Str != "app/core/database/v1/user_model.go" {
			t.Errorf("expected 'user_model.go' to win due to contiguous match, got %s", results[0].Str)
		}
	})

	t.Run("Match range compactness (config)", func(t *testing.T) {
		items := []string{
			"core/cfg/ob-navigation-git.json",
			"settings/config_backup.json",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("config")

		t.Logf("results for 'config':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}

		if results[0].Str != "settings/config_backup.json" {
			t.Errorf("expected 'config_backup.json' to win due to compactness, got %s", results[0].Str)
		}
	})
}
