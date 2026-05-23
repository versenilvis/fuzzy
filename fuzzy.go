// Package fuzzy - Entry point for fuzzy searching
package fuzzy

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/versenilvis/fuzzy/core"
)

// SearchOptions provides advanced search configuration
type SearchOptions struct {
	ContextBoosts map[string]int // Optional boosts for specific items
	Limit         int            // Maximum number of results to return
}

// MatchResult represents a scored search result
type MatchResult struct {
	Str   string
	Score int
}

// Searcher is the main object for performing fuzzy searches
type Searcher struct {
	Originals  []string            // Original items
	Normalized [][]byte            // Normalized items for fast matching
	Memory     *core.FileMemory    // Frecency memory system
	Filter     *core.UnigramFilter // Bitset filter for candidates
	baseStarts []int               // Start indices of filenames in paths
	scorePool  *sync.Pool          // Pool for reusing score buffers
}

// NewPlainSearcher creates a searcher for plain text items
func NewPlainSearcher(items []string) *Searcher {
	numItems := len(items)
	originals := make([]string, numItems)
	normPaths := make([][]byte, numItems)
	baseStarts := make([]int, numItems)

	for i, item := range items {
		originals[i] = item
		normItem := core.Normalize(item)
		baseStarts[i] = len(normItem)
		normPaths[i] = []byte(normItem)
	}

	return &Searcher{
		Originals:  originals,
		Normalized: normPaths,
		Memory:     core.NewFileMemory(nil),
		Filter:     core.NewUnigramFilter(normPaths),
		baseStarts: baseStarts,
		scorePool: &sync.Pool{
			New: func() any {
				buf := make([]int, numItems)
				for i := range buf {
					buf[i] = math.MinInt
				}
				return &buf
			},
		},
	}
}

// NewSearcher creates a searcher optimized for file paths
func NewSearcher(items []string) *Searcher {
	numItems := len(items)
	originals := make([]string, numItems)
	normPaths := make([][]byte, numItems)
	baseStarts := make([]int, numItems)

	for i, item := range items {
		originals[i] = item

		bStart := -1
		for j := len(item) - 1; j >= 0; j-- {
			if item[j] == '/' || item[j] == '\\' {
				bStart = j + 1
				break
			}
		}

		if bStart != -1 && bStart < len(item) {
			filename := item[bStart:]
			normFilename := core.Normalize(filename)
			baseStarts[i] = len(normFilename)
			priorityString := filename + " " + item
			normPaths[i] = []byte(core.Normalize(priorityString))
		} else {
			normItem := core.Normalize(item)
			baseStarts[i] = len(normItem)
			normPaths[i] = []byte(normItem)
		}
	}

	return &Searcher{
		Originals:  originals,
		Normalized: normPaths,
		Memory:     core.NewFileMemory(nil),
		Filter:     core.NewUnigramFilter(normPaths),
		baseStarts: baseStarts,
		scorePool: &sync.Pool{
			New: func() any {
				buf := make([]int, numItems)
				for i := range buf {
					buf[i] = math.MinInt
				}
				return &buf
			},
		},
	}
}

// NewSearcherWithMemory creates a searcher with existing memory
func NewSearcherWithMemory(items []string, memory *core.FileMemory) *Searcher {
	s := NewSearcher(items)
	if memory != nil {
		s.Memory = memory
	}
	return s
}

// SearchDebug prints debug information for a query
func (s *Searcher) SearchDebug(query string) {
	fmt.Printf("DEBUG SEARCH Query: [%s]\n", query)
	queryNorm := core.Normalize(query)
	queryPattern := []byte(queryNorm)

	for i, item := range s.Originals {
		score, matched := core.FuzzyScoreGreedy(queryPattern, s.Normalized[i], s.baseStarts[i])
		fmt.Printf("Item: [%s] | Norm: [%s] | Score: %d | Matched: %v | baseStart: %d\n", item, string(s.Normalized[i]), score, matched, s.baseStarts[i])
	}
}

// SearchWithScores performs fuzzy search and returns scored results
func (s *Searcher) SearchWithScores(query string, opts ...*SearchOptions) []MatchResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	queryNorm := core.Normalize(query)
	queryPattern := []byte(queryNorm)

	resLimit := 20
	if len(opts) > 0 && opts[0] != nil && opts[0].Limit > 0 {
		resLimit = opts[0].Limit
	}

	memoryBoosts := s.Memory.GetBoostScores(query)

	var matches []core.FuzzyMatch
	candidates := s.Filter.Filter(queryPattern)

	if candidates != nil {
		matches = core.FuzzyFindFiltered(queryPattern, s.Normalized, candidates, s.baseStarts, resLimit)
	} else {
		matches = core.FuzzyFindParallel(queryPattern, s.Normalized, s.baseStarts, s.Filter.Bin, resLimit)
	}

	if len(matches) == 0 && len(queryNorm) >= 3 {
		matches = s.findButTypo(queryNorm)
	}

	if len(matches) == 0 {
		return nil
	}

	scoreBufPtr := s.scorePool.Get().(*[]int)
	scoreBuf := *scoreBufPtr
	defer func() {
		for i := range scoreBuf {
			scoreBuf[i] = math.MinInt
		}
		s.scorePool.Put(scoreBufPtr)
	}()

	for _, m := range matches {
		scoreBuf[m.Index] = m.Score
	}

	rankedResults := make([]MatchResult, 0, len(matches))
	for _, m := range matches {
		filePath := s.Originals[m.Index]
		finalScore := m.Score

		if boost, exists := memoryBoosts[filePath]; exists {
			finalScore += boost
		}

		if len(opts) > 0 && opts[0] != nil && opts[0].ContextBoosts != nil {
			if boost, exists := opts[0].ContextBoosts[filePath]; exists {
				finalScore += boost
			}
		}

		rankedResults = append(rankedResults, MatchResult{
			Str:   filePath,
			Score: finalScore,
		})
	}

	sort.Slice(rankedResults, func(i, j int) bool {
		if rankedResults[i].Score == rankedResults[j].Score {
			return rankedResults[i].Str < rankedResults[j].Str
		}
		return rankedResults[i].Score > rankedResults[j].Score
	})

	if len(rankedResults) < resLimit {
		resLimit = len(rankedResults)
	}

	return rankedResults[:resLimit]
}

// Search performs fuzzy search and returns matching strings
func (s *Searcher) Search(query string, opts ...*SearchOptions) []string {
	rankedResults := s.SearchWithScores(query, opts...)
	if rankedResults == nil {
		return nil
	}

	finalStrings := make([]string, len(rankedResults))
	for i, res := range rankedResults {
		finalStrings[i] = res.Str
	}

	return finalStrings
}

// findButTypo is a fallback search for typos using Levenshtein distance
func (s *Searcher) findButTypo(query string) []core.FuzzyMatch {
	numItems := len(s.Normalized)
	if numItems == 0 {
		return nil
	}

	// allow 1 typo for every 4 characters in the query
	threshold := max(len(query)/4, 1)

	numCPUs := runtime.GOMAXPROCS(0)
	chunkSize := (numItems + numCPUs - 1) / numCPUs

	var wg sync.WaitGroup
	resultChan := make(chan []core.FuzzyMatch, numCPUs)

	// split work across all available CPU cores
	for i := range numCPUs {
		start := i * chunkSize
		if start >= numItems {
			break
		}
		end := min(start+chunkSize, numItems)

		wg.Add(1)
		go func(s0, e int) {
			defer wg.Done()
			var local []core.FuzzyMatch
			for j := s0; j < e; j++ {
				// skip if the item is marked as deleted in the tombstone bin
				blockIdx := j / 64
				bitPos := uint64(1) << (j % 64)
				if s.Filter.Bin[blockIdx]&bitPos != 0 {
					continue
				}

				filename := string(s.Normalized[j][:s.baseStarts[j]])
				dist := core.LevenshteinRatio(query, filename)
				if dist <= threshold {
					local = append(local, core.FuzzyMatch{
						Index: j,
						Score: 100 - dist*10,
					})
				}
			}
			resultChan <- local
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var matches []core.FuzzyMatch
	for chunk := range resultChan {
		matches = append(matches, chunk...)
	}
	return matches
}

// RecordSelection records an item selection to update frecency
func (s *Searcher) RecordSelection(query, filePath string) {
	s.Memory.RecordSelection(query, filePath)
}

// ClearCache clears selection history
func (s *Searcher) ClearCache() {
	s.Memory = core.NewFileMemory(nil)
}

// Normalize exposes core normalization logic
func Normalize(s string) string {
	return core.Normalize(s)
}

// LevenshteinRatio exposes core Levenshtein distance logic
func LevenshteinRatio(s1, s2 string) int {
	return core.LevenshteinRatio(s1, s2)
}
