// Package core - Worker: Partitions search tasks across CPU cores for large datasets
package core

import (
	"container/heap"
	"runtime"
	"sync"
)

// FuzzyMatch represents a single search result with its score
type FuzzyMatch struct {
	Index int
	Score int
}

/*
minHeap: Used to efficiently extract Top-K results
  - Partial sort O(N log K) is much faster than full sort O(N log N)
  - The smallest score is always at index 0, allowing quick replacement
*/
type minHeap []FuzzyMatch

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Score < h[j].Score }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(FuzzyMatch)) }
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

const defaultTopK = 20

/*
FuzzyFindFiltered performs fuzzy search only on pre-filtered candidate IDs
  - If candidates are few (< 500), runs in a single thread to avoid overhead
  - For large candidate sets, partitions work across all available CPU cores
*/
func FuzzyFindFiltered(query []byte, items [][]byte, candidates []int, baseStarts []int, limit int) []FuzzyMatch {
	if limit <= 0 {
		limit = defaultTopK
	}
	n := len(candidates)
	if n == 0 {
		return nil
	}

	// Fast path: Single-threaded for small sets
	if n < 500 {
		h := &minHeap{}
		heap.Init(h)
		for _, idx := range candidates {
			if score, matched := FuzzyScoreGreedy(query, items[idx], baseStarts[idx]); matched {
				if h.Len() < limit {
					heap.Push(h, FuzzyMatch{Index: idx, Score: score})
				} else if score > (*h)[0].Score {
					(*h)[0] = FuzzyMatch{Index: idx, Score: score}
					heap.Fix(h, 0)
				}
			}
		}
		return heapToSorted(h)
	}

	// Parallel path: Split candidates into chunks based on CPU count
	numCPUs := runtime.GOMAXPROCS(0)
	chunkSize := (n + numCPUs - 1) / numCPUs

	var wg sync.WaitGroup
	resultChan := make(chan []FuzzyMatch, numCPUs)

	for i := range numCPUs {
		start := i * chunkSize
		if start >= n {
			break
		}
		end := min(start+chunkSize, n)

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			h := &minHeap{}
			heap.Init(h)
			for _, idx := range candidates[s:e] {
				if score, matched := FuzzyScoreGreedy(query, items[idx], baseStarts[idx]); matched {
					if h.Len() < limit {
						heap.Push(h, FuzzyMatch{Index: idx, Score: score})
					} else if score > (*h)[0].Score {
						(*h)[0] = FuzzyMatch{Index: idx, Score: score}
						heap.Fix(h, 0)
					}
				}
			}
			resultChan <- heapToSorted(h)
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Merge Top-K results from all workers into a final min-heap
	finalHeap := &minHeap{}
	heap.Init(finalHeap)
	for matches := range resultChan {
		for _, m := range matches {
			if finalHeap.Len() < limit {
				heap.Push(finalHeap, m)
			} else if m.Score > (*finalHeap)[0].Score {
				(*finalHeap)[0] = m
				heap.Fix(finalHeap, 0)
			}
		}
	}

	return heapToSorted(finalHeap)
}

/*
FuzzyFindParallel scans all items in parallel when filtering is not available
  - This is a fallback for very short queries where bitset filter is disabled
*/
func FuzzyFindParallel(query []byte, items [][]byte, baseStarts []int, bin []uint64, limit int) []FuzzyMatch {
	if limit <= 0 {
		limit = defaultTopK
	}
	numItems := len(items)
	if numItems == 0 {
		return nil
	}

	numCPUs := runtime.GOMAXPROCS(0)
	chunkSize := (numItems + numCPUs - 1) / numCPUs

	var wg sync.WaitGroup
	resultChan := make(chan []FuzzyMatch, numCPUs)

	for i := range numCPUs {
		start := i * chunkSize
		if start >= numItems {
			break
		}
		end := min(start+chunkSize, numItems)

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			h := &minHeap{}
			heap.Init(h)
			for j := s; j < e; j++ {
				// skip if the item is marked as deleted in the tombstone bin
				if bin != nil {
					blockIdx := j / 64
					bitPos := uint64(1) << (j % 64)
					if bin[blockIdx]&bitPos != 0 {
						continue
					}
				}

				if score, matched := FuzzyScoreGreedy(query, items[j], baseStarts[j]); matched {
					if h.Len() < limit {
						heap.Push(h, FuzzyMatch{Index: j, Score: score})
					} else if score > (*h)[0].Score {
						(*h)[0] = FuzzyMatch{Index: j, Score: score}
						heap.Fix(h, 0)
					}
				}
			}
			resultChan <- heapToSorted(h)
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	finalHeap := &minHeap{}
	heap.Init(finalHeap)
	for matches := range resultChan {
		for _, m := range matches {
			if finalHeap.Len() < limit {
				heap.Push(finalHeap, m)
			} else if m.Score > (*finalHeap)[0].Score {
				(*finalHeap)[0] = m
				heap.Fix(finalHeap, 0)
			}
		}
	}

	return heapToSorted(finalHeap)
}

// heapToSorted converts min-heap into a descending sorted slice
func heapToSorted(h *minHeap) []FuzzyMatch {
	n := h.Len()
	result := make([]FuzzyMatch, n)
	for i := n - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(FuzzyMatch)
	}
	return result
}
