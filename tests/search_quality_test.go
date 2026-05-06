package fuzzy_test

import (
	"testing"
	"github.com/versenilvis/fuzzy"
)

// TestSearchQuality examines the engine's ability to prioritize high-quality matches
// over noisy or deeply nested paths, and identifies limitations in greedy matching
func TestSearchQuality(t *testing.T) {
	
	t.Run("The Greedy Trap (repeated characters)", func(t *testing.T) {
		// Searching for 'ff'
		items := []string{
			"f_static_ff.go",      // Optimal match is the 'ff' at the end
			"flat_file_util.go",   // 'f' and 'f' are boundaries but not contiguous
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("ff")

		t.Logf("results for 'ff':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}

		// If greedy takes the first 'f' in 'f_static_ff.go', it misses the contiguous +200 bonus
	})

	t.Run("Scattered vs Structured (buffer)", func(t *testing.T) {
		items := []string{
			"b_u_f_f_e_r.go",
			"src/internal/core/buffer_service.go",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("buffer")

		t.Logf("results for 'buffer':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}

		if results[0].Str != "src/internal/core/buffer_service.go" {
			t.Logf("Warning: Scattered boundaries might be outranking contiguous deep matches")
		}
	})

	t.Run("Path depth preference (main)", func(t *testing.T) {
		items := []string{
			"main.go",
			"src/main.go",
			"src/internal/core/bootstrap/main.go",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("main")

		t.Logf("results for 'main':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}
	})

	t.Run("Space boundary quality (this file)", func(t *testing.T) {
		items := []string{
			"this file has space.pdf",
			"test_file_has_system.log",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("tfhs")

		t.Logf("results for 'tfhs':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}
	})
}
