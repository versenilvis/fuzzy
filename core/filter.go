package core

import (
	"math/bits"
	"runtime"
	"sync"
)

// UnigramFilter provides fast K-of-N matching using inverted bitsets
type UnigramFilter struct {
	// Bitsets maps each ASCII character to a bitset of documents containing it
	Bitsets    [256][]uint64
	Bin        []uint64 // Tombstone markers for deleted items
	NumTargets int      // Total number of indexed items
}

// NewUnigramFilter builds inverted bitsets for the given targets
func NewUnigramFilter(targets [][]byte) *UnigramFilter {
	numTargets := len(targets)
	blocks := (numTargets + 63) / 64

	uf := &UnigramFilter{
		NumTargets: numTargets,
		Bin:        make([]uint64, blocks),
	}

	for i := range uf.Bitsets {
		uf.Bitsets[i] = make([]uint64, blocks)
	}

	numWorkers := runtime.GOMAXPROCS(0)
	numWorkers = min(numWorkers, 16)

	blocksPerWorker := (blocks + numWorkers - 1) / numWorkers
	if blocksPerWorker == 0 {
		blocksPerWorker = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		startBlock := i * blocksPerWorker
		if startBlock >= blocks {
			break
		}
		endBlock := min(startBlock+blocksPerWorker, blocks)

		startDoc := startBlock * 64
		endDoc := min(endBlock*64, numTargets)

		wg.Add(1)

		go func(start, end int) {
			defer wg.Done()
			for docID := start; docID < end; docID++ {
				target := targets[docID]
				blockIdx := docID / 64
				bitPos := uint64(1) << (docID % 64)

				for _, b := range target {
					uf.Bitsets[b][blockIdx] |= bitPos
				}
			}
		}(startDoc, endDoc)
	}
	wg.Wait()

	threshold85 := (numTargets * 85) / 100
	for i := range uf.Bitsets {
		popcount := 0
		for _, block := range uf.Bitsets[i] {
			popcount += bits.OnesCount64(block)
		}

		// discard character if it appears in >= 85% of files
		if popcount >= threshold85 {
			uf.Bitsets[i] = nil
		}
	}

	return uf
}

// DeleteFile marks an item as deleted in the filter
func (uf *UnigramFilter) DeleteFile(docID int) {
	if docID >= uf.NumTargets {
		return
	}
	blockIdx := docID / 64
	bitPos := uint64(1) << (docID % 64)
	uf.Bin[blockIdx] |= bitPos
}

// Filter returns IDs of items that pass the K/N match threshold
func (uf *UnigramFilter) Filter(pattern []byte) []int {
	uniqueChars := make([]byte, 0, len(pattern))
	var seen [4]uint64
	for _, b := range pattern {
		idx := b / 64
		bit := uint64(1) << (b % 64)
		if seen[idx]&bit == 0 {
			seen[idx] |= bit
			if uf.Bitsets[b] != nil {
				uniqueChars = append(uniqueChars, b)
			}
		}
	}

	n := len(uniqueChars)
	if n < 2 {
		return nil
	}

	// Calculate typo tolerance
	threshold := n
	if n >= 6 {
		threshold = n - 2
	} else if n >= 4 {
		threshold = n - 1
	}

	counts := make([]uint8, uf.NumTargets)

	for _, b := range uniqueChars {
		set := uf.Bitsets[b]
		for blockIdx, block := range set {
			for block != 0 {
				trailing := bits.TrailingZeros64(block)
				docID := blockIdx*64 + trailing

				counts[docID]++
				block &= (block - 1)
			}
		}
	}

	validIDs := make([]int, 0, 1000)
	thresh8 := uint8(threshold)
	for docID, count := range counts {
		if count >= thresh8 {
			blockIdx := docID / 64
			bitPos := uint64(1) << (docID % 64)
			if uf.Bin[blockIdx]&bitPos == 0 {
				validIDs = append(validIDs, docID)
			}
		}
	}

	return validIDs
}
