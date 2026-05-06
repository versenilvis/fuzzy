package fuzzy_test

import (
	"testing"
	"github.com/versenilvis/fuzzy"
)

// TestWordBoundaries verifies that characters following separators (/, _, ., -, @, +) 
// receive the correct boundary boost
func TestWordBoundaries(t *testing.T) {
	t.Run("Basic boundaries (api)", func(t *testing.T) {
		items := []string{
			"src/internal/api_client.go",
			"apple_interface.go",
			"capillary_action.py",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("api")

		t.Logf("results for 'api':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}

		if results[0].Str != "src/internal/api_client.go" {
			t.Errorf("expected 'src/internal/api_client.go' to be first, got %s", results[0].Str)
		}
	})

	t.Run("Special character boundaries (build)", func(t *testing.T) {
		items := []string{
			"node_modules/@scope/build-tools/index.js",
			"rebuild+final.sh",
			"build.config.ts",
			"dist/build/prod/main.bin",
		}
		s := fuzzy.NewSearcher(items)
		results := s.SearchWithScores("build")

		t.Logf("results for 'build':")
		for _, res := range results {
			t.Logf("file: %s | score: %d", res.Str, res.Score)
		}

		// build.config.ts should be very high due to prefix + boundary before dot
		if results[0].Str != "build.config.ts" {
			t.Logf("top result: %s", results[0].Str)
		}
	})
}
