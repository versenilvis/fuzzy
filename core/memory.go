// Package core - Memory: Tracks selection history and applies time-decayed boosts to results
package core

import (
	"bytes"
	"encoding/gob"
	"math"
	"slices"
	"sort"
	"sync"
	"time"
)

// FileRecord stores selection history for an item
type FileRecord struct {
	SelectCount int      // Total number of times this file was selected
	LastAccess  int64    // Unix timestamp of the last selection
	Queries     []string // List of most recent queries that led to this selection (max 3)
}

// FileMemory handles frecency (frequency + recency) scoring
type FileMemory struct {
	mu            sync.RWMutex
	files         map[string]*FileRecord // Map of absolute file paths to records
	maxFiles      int                    // Limit on number of items remembered (default 500)
	decayHalfLife int64                  // Score decay half-life in seconds (e.g. 12 hours)
	/*
		decayHalfLife solves the following problem:
		File A: Opened 100 times, but last access was 1 month ago
		File B: Opened only 5 times, but all within today
		Without decay, File A would always rank higher than File B
		With decay, File B can eventually overtake File A as its score remains fresh
	*/
	boostBase int // Base points added for each selection
}

// MemoryConfig contains settings for FileMemory
type MemoryConfig struct {
	MaxFiles      int
	DecayHalfLife int64
	BoostBase     int
}

// NewFileMemory initializes a new FileMemory instance
func NewFileMemory(cfg *MemoryConfig) *FileMemory {
	if cfg == nil {
		cfg = &MemoryConfig{}
	}
	if cfg.MaxFiles <= 0 {
		cfg.MaxFiles = 500
	}
	if cfg.DecayHalfLife <= 0 {
		cfg.DecayHalfLife = 43200 // 12 hours
	}
	if cfg.BoostBase <= 0 {
		cfg.BoostBase = 5000
	}

	return &FileMemory{
		files:         make(map[string]*FileRecord),
		maxFiles:      cfg.MaxFiles,
		decayHalfLife: cfg.DecayHalfLife,
		boostBase:     cfg.BoostBase,
	}
}

/*
RecordSelection records user interaction with a file
  - query: The search term typed by the user
  - filePath: The item selected by the user
*/
func (fm *FileMemory) RecordSelection(query, filePath string) {
	if query == "" || filePath == "" {
		return
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	queryNorm := Normalize(query)
	now := time.Now().Unix()

	record, exists := fm.files[filePath]
	if !exists {
		// Check capacity limit before adding a new record
		if len(fm.files) >= fm.maxFiles {
			fm.evictLowestScore(now)
		}
		record = &FileRecord{
			Queries: make([]string, 0, 3),
		}
		fm.files[filePath] = record
	}

	if record.SelectCount < math.MaxInt {
		record.SelectCount++
	}
	record.LastAccess = now

	// Update query history using a ring-buffer approach
	if !slices.Contains(record.Queries, queryNorm) {
		if len(record.Queries) >= 3 {
			// Remove the oldest query
			record.Queries = record.Queries[1:]
		}
		record.Queries = append(record.Queries, queryNorm)
	}
}

/*
GetBoostScores returns a map of boost points for relevant files based on current query
  - Uses Levenshtein distance to match current query against historical queries for each file
*/
func (fm *FileMemory) GetBoostScores(query string) map[string]int {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make(map[string]int)
	if query == "" || len(fm.files) == 0 {
		return result
	}

	queryNorm := Normalize(query)
	now := time.Now().Unix()

	for path, record := range fm.files {
		// Find the best match between current query and file history
		bestSim := 0.0
		for _, q := range record.Queries {
			// Calculate similarity ratio: 1 - (distance / maxLen)
			dist := LevenshteinRatio(queryNorm, q)
			maxLen := max(len(queryNorm), len(q))

			sim := 1.0
			if maxLen > 0 {
				sim = 1.0 - float64(dist)/float64(maxLen)
			}

			if sim > bestSim {
				bestSim = sim
			}
		}

		/*
			Levenshtein is stricter than Jaro-Winkler on prefix matches
			e.g "main" vs "main server" scores only ~0.36 (dist: 7)
			compared to JW's 0.8+. Lowered the memory boost threshold to 0.3
			to keep prioritizing prefix suggestions
		*/
		if bestSim < 0.3 {
			continue
		}

		// Calculate decay factor: 2^(-elapsed / half-life)
		elapsed := max(0, now-record.LastAccess)
		decay := math.Pow(2, -float64(elapsed)/float64(fm.decayHalfLife))

		// Final boost depends on selection frequency, time decay, and query relevance
		boost := float64(fm.boostBase) * float64(record.SelectCount) * decay * bestSim

		if boost > 0 {
			result[path] = int(boost)
		}
	}

	return result
}

// evictLowestScore: Removes the item with lowest frecency score to free up space
func (fm *FileMemory) evictLowestScore(now int64) {
	minScore := math.MaxFloat64
	var victim string

	for path, record := range fm.files {
		elapsed := now - record.LastAccess
		decay := math.Pow(2, -float64(elapsed)/float64(fm.decayHalfLife))
		score := float64(record.SelectCount) * decay

		if score < minScore {
			minScore = score
			victim = path
		}
	}

	if victim != "" {
		delete(fm.files, victim)
	}
}

/*
Export - Serializes memory data to a byte slice
  - Useful for persisting history to disk or sending over network
*/
func (fm *FileMemory) Export() ([]byte, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(fm.files); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

/*
Import - Deserializes memory data from a byte slice
*/
func (fm *FileMemory) Import(data []byte) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var imported map[string]*FileRecord
	if err := dec.Decode(&imported); err != nil {
		return err
	}
	fm.files = imported
	return nil
}

// GetRecentFiles - Returns up to 'limit' most recently selected file paths
func (fm *FileMemory) GetRecentFiles(limit int) []string {
	if limit <= 0 {
		return nil
	}

	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if len(fm.files) == 0 {
		return nil
	}

	type fileAccess struct {
		path string
		time int64
	}
	records := make([]fileAccess, 0, len(fm.files))
	for path, r := range fm.files {
		records = append(records, fileAccess{path, r.LastAccess})
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].time > records[j].time
	})

	if limit > len(records) {
		limit = len(records)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = records[i].path
	}
	return result
}
