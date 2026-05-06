package fuzzy_test

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/versenilvis/fuzzy"
	"github.com/versenilvis/fuzzy/core"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"Python", "python"},
		{"path/to/file.go", "path/to/file.go"},
		{"UPPER_CASE", "upper_case"},
		{"", ""},
	}

	for _, tt := range tests {
		result := fuzzy.Normalize(tt.input)
		if result != tt.expected {
			t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLevenshteinRatio(t *testing.T) {
	tests := []struct {
		s1, s2   string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"main", "mian", 2},
		{"kitten", "sitting", 3},
		{"hello", "hallo", 1},
	}

	for _, tt := range tests {
		result := fuzzy.LevenshteinRatio(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("LevenshteinRatio(%q, %q) = %d, want %d", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestNewSearcher(t *testing.T) {
	files := []string{
		"/home/user/main.go",
		"/home/user/config.yaml",
		"/home/user/README.md",
	}

	searcher := fuzzy.NewSearcher(files)

	if len(searcher.Originals) != 3 {
		t.Errorf("Originals has %d elements, want 3", len(searcher.Originals))
	}
	if len(searcher.Normalized) != 3 {
		t.Errorf("Normalized has %d elements, want 3", len(searcher.Normalized))
	}
	if searcher.Memory == nil {
		t.Error("Memory not initialized")
	}
}

func TestSearcher_Search_Basic(t *testing.T) {
	files := []string{
		"/project/main.go",
		"/project/main_test.go",
		"/project/config.yaml",
		"/project/README.md",
	}

	searcher := fuzzy.NewSearcher(files)

	results := searcher.Search("main")
	if len(results) < 2 {
		t.Errorf("Search('main') returned %d results, want at least 2", len(results))
	}
	if !slices.Contains(results, "/project/main.go") {
		t.Error("Search('main') did not find /project/main.go")
	}
}

func TestSearcher_Search_Typo(t *testing.T) {
	files := []string{
		"/project/main.go",
		"/project/config.yaml",
	}

	searcher := fuzzy.NewSearcher(files)

	results := searcher.Search("mian")
	if !slices.Contains(results, "/project/main.go") {
		t.Error("Search('mian') should find main.go (typo tolerance)")
	}
}

func TestFileMemory_RecordSelection(t *testing.T) {
	mem := core.NewFileMemory(nil)

	mem.RecordSelection("main", "/project/main.go")
	boosts := mem.GetBoostScores("main")
	if boosts["/project/main.go"] == 0 {
		t.Error("GetBoostScores should return score for recorded file")
	}

	mem.RecordSelection("main", "/project/main.go")
	boosts2 := mem.GetBoostScores("main")
	if boosts2["/project/main.go"] <= boosts["/project/main.go"] {
		t.Error("Score should increase when recorded again")
	}
}

func TestFileMemory_GetBoostScores_SimilarQuery(t *testing.T) {
	mem := core.NewFileMemory(nil)
	mem.RecordSelection("main server", "/project/main_server.go")

	tests := []string{"main", "main ser"}
	for _, query := range tests {
		scores := mem.GetBoostScores(query)
		if scores["/project/main_server.go"] == 0 {
			t.Errorf("GetBoostScores(%q) should return score for similar query", query)
		}
	}
}

func TestSearcher_RecordSelection_BoostsResults(t *testing.T) {
	files := []string{
		"/project/main.go",
		"/project/main_server.go",
		"/project/main_test.go",
		"/project/config.yaml",
	}

	searcher := fuzzy.NewSearcher(files)

	searcher.RecordSelection("main", "/project/main_test.go")
	searcher.RecordSelection("main", "/project/main_test.go")
	searcher.RecordSelection("main", "/project/main_test.go")

	results := searcher.Search("main")
	if len(results) == 0 {
		t.Fatal("Search returned no results")
	}
	if results[0] != "/project/main_test.go" {
		t.Errorf("Frequently selected file should be at top, got %q", results[0])
	}
}

func TestSearcher_ContextBoosts(t *testing.T) {
	files := []string{
		"auth/user.go",
		"auth/user_test.go",
		"models/user.go",
	}
	searcher := fuzzy.NewSearcher(files)

	opts := &fuzzy.SearchOptions{
		ContextBoosts: map[string]int{
			"auth/user_test.go": 10000,
		},
	}
	results := searcher.Search("user", opts)
	if len(results) == 0 || results[0] != "auth/user_test.go" {
		t.Errorf("Expected auth/user_test.go at top due to ContextBoost, got %v", results)
	}
}

func TestSearcher_EdgeCases(t *testing.T) {
	t.Run("Empty or space query", func(t *testing.T) {
		s := fuzzy.NewSearcher([]string{"a.go", "b.go"})
		if s.Search("") != nil {
			t.Error("Empty query should return nil")
		}
		if len(s.Search("   ")) > 0 {
			t.Error("Whitespace query should return no results")
		}
	})

	t.Run("Empty input data", func(t *testing.T) {
		s := fuzzy.NewSearcher([]string{})
		if s.Search("anything") != nil {
			t.Error("Search on empty list should return nil")
		}
	})

	t.Run("Deterministic Sorting", func(t *testing.T) {
		files := []string{"/b/utils.go", "/a/utils.go"}
		s := fuzzy.NewSearcher(files)
		res1 := s.Search("utils")
		if res1[0] != "/a/utils.go" {
			t.Errorf("Sort not deterministic, got %s first", res1[0])
		}
	})

	t.Run("Concurrency & Race Condition", func(t *testing.T) {
		files := make([]string, 100)
		for i := range 100 {
			files[i] = strings.Repeat("a", i) + ".go"
		}
		s := fuzzy.NewSearcher(files)
		var wg sync.WaitGroup

		for range 20 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 50 {
					s.Search("aaa")
				}
			}()
		}

		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 20 {
					s.RecordSelection("aaa", files[0])
				}
			}()
		}

		wg.Wait()
	})
}
