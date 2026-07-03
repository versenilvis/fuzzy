// Package core - Scoring engine for fuzzy matching with optimal alignment search
package core

/*
FuzzyScoreGreedy calculates the fuzzy match score by finding the optimal character
alignment that maximizes the contiguous bonus, rather than simply taking the first
match found. It uses a right-to-left seam-finding strategy: it starts from the last
pattern character and walks backwards, greedily chaining consecutive positions to
build the longest possible contiguous run. This means that for a query like 'ff'
against 'f_static_ff.go', it correctly prefers the contiguous 'ff' at the end over
the isolated 'f' at the start.
*/
func FuzzyScoreGreedy(pattern []byte, target []byte, baseStart int) (int, bool) {
	lenP := len(pattern)
	lenT := len(target)

	if lenP == 0 || lenP > lenT {
		return 0, false
	}

	/*
	   Phase 1: verify a basic forward match exists and record the last valid position
	   for each pattern character. This is a single O(N+M) pass to confirm feasibility
	   before spending time on optimal alignment.
	*/
	patternIdx := 0
	for i := range lenT {
		if patternIdx < lenP && target[i] == pattern[patternIdx] {
			patternIdx++
		}
	}
	if patternIdx != lenP {
		return 0, false
	}

	/*
	   Phase 2: find the optimal alignment using a right-to-left pass. Starting from
	   the right side of the target, we find the rightmost occurrence of the last
	   pattern character, then walk left to find consecutive predecessors. This builds
	   the longest rightmost-anchored contiguous run automatically. Any remaining
	   unanchored characters are resolved with a final left-to-right greedy pass.

	   Why right-to-left: contiguous runs in filenames usually appear at meaningful
	   word segments like 'config', 'buffer', or 'api'. By anchoring to the rightmost
	   feasible chain first, we avoid greedily consuming isolated leading characters
	   that would prevent us from finding the better rightward chain.
	*/
	chosen := make([]int, lenP)
	for i := range lenP {
		chosen[i] = -1
	}

	// find the rightmost valid position for the last pattern character
	for ti := lenT - 1; ti >= 0; ti-- {
		if target[ti] == pattern[lenP-1] {
			chosen[lenP-1] = ti
			break
		}
	}
	if chosen[lenP-1] == -1 {
		return 0, false
	}

	// walk backwards through the pattern, chaining consecutive positions
	for pi := lenP - 2; pi >= 0; pi-- {
		anchor := chosen[pi+1]
		found := -1

		// prefer the position immediately before the next character (contiguous)
		if anchor > 0 && target[anchor-1] == pattern[pi] {
			found = anchor - 1
		} else {
			// fall back to the rightmost occurrence before the anchor
			for ti := anchor - 1; ti >= 0; ti-- {
				if target[ti] == pattern[pi] {
					found = ti
					break
				}
			}
		}

		if found == -1 {
			return 0, false
		}
		chosen[pi] = found
	}

	/*
	   Phase 3: score the optimal alignment
	*/
	totalScore := 0
	firstMatchIdx := chosen[0]
	lastMatchIdx := chosen[lenP-1]

	for pi, ti := range chosen {
		// consecutive match bonus: the core benefit of the optimal alignment
		if pi > 0 && ti == chosen[pi-1]+1 {
			totalScore += 200
		}

		// word boundary boost: a match at the start or after a separator indicates
		// the user typed the beginning of a meaningful word
		if ti == 0 {
			totalScore += 80
		} else {
			switch target[ti-1] {
			case '/', '\\', '_', '-', '.', ' ':
				totalScore += 80
			default:
				totalScore += 10
			}
		}

		// small bonus if the match falls in the filename rather than the directory path
		if ti < baseStart {
			totalScore += 15
		}
	}

	matchRange := lastMatchIdx - firstMatchIdx + 1
	matchRange = min(matchRange, lenP)
	baseScore := (lenP * 100) - (matchRange-lenP)*5

	if baseScore <= 0 {
		return 0, false
	}
	totalScore += baseScore

	lengthPenalty := lenT * 2
	totalScore -= lengthPenalty

	/*
	   Tier 1: perfect prefix match on the filename itself. This earns the largest
	   boost and returns immediately since no other result can compete.
	*/
	if baseStart >= lenP {
		isPerfectStart := true
		for i := range lenP {
			if target[i] != pattern[i] {
				isPerfectStart = false
				break
			}
		}
		if isPerfectStart {
			totalScore += 1000000
			return totalScore, true
		}
	}

	/*
	   Tier 2: short filename containing all query characters. We verify using a
	   character frequency bucket to avoid rescanning the pattern.
	*/
	if baseStart <= lenP*3 {
		var charBucket [256]int8
		for i := range baseStart {
			charBucket[target[i]]++
		}
		filenameHits := 0
		for _, b := range pattern {
			if charBucket[b] > 0 {
				charBucket[b]--
				filenameHits++
			}
		}
		if filenameHits == lenP {
			totalScore += 500000
			return totalScore, true
		}
	}

	/*
	   Tier 3 vs 4: double the score for matches in the filename, penalize matches
	   that only appear in the directory path.
	*/
	if firstMatchIdx < baseStart {
		totalScore += (totalScore * 200) / 100
	} else {
		totalScore -= (lenT / 3)
	}

	return totalScore, true
}
