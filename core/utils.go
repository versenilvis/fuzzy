package core

import (
	"strings"
)

// HasUpperCase checks if string contains any uppercase characters
func HasUpperCase(s string) bool {
	for i := range s {
		if s[i] >= 'A' && s[i] <= 'Z' {
			return true
		}
	}
	return false
}

// CountWordMatches counts how many words in query are present in target
func CountWordMatches(queryWords []string, target string) int {
	if len(target) < 2 {
		return 0
	}
	count := 0
	for _, word := range queryWords {
		if len(word) >= 2 && strings.Contains(target, word) {
			count++
		}
	}
	return count
}

// Normalize prepares string for fuzzy matching by converting to lowercase
func Normalize(s string) string {
	if s == "" {
		return ""
	}

	buf := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		char := s[i]
		if char >= 'A' && char <= 'Z' {
			buf = append(buf, char+32)
		} else if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '/' || char == '\\' || char == '_' || char == '-' || char == ' ' {
			buf = append(buf, char)
		}
	}
	return string(buf)
}

// FastSubstring returns a substring of first n characters efficiently
func FastSubstring(s string, n int) string {
	if len(s) <= n {
		return s
	}

	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}

	return s
}

/*
LevenshteinRatio calculates edit distance between two strings for typo suggestions
  - Objective: Transform string s1 into s2 with minimum cost
  - At each step, we have 3 options with cost +1
  - Delete a character from s1
  - Add a character to s1
  - Substitute:
    > No cost (+0) if characters are identical
    > Cost +1 if characters are different
  - Optimization: Uses a single column array instead of a full matrix to save memory
  - Performance: Uses stack-allocated buffer for short strings (<64 chars) to avoid heap allocation
*/
func LevenshteinRatio(s1, s2 string) int {
	r1 := []byte(s1)
	r2 := []byte(s2)
	s1Len := len(r1)
	s2Len := len(r2)

	/*
	   Base case: transforming s1 to an empty string
	   Example: s1 = "ABC", s2 = ""
	   "" to "" costs 0 (column[0] = 0)
	   "A" to "" costs 1 (column[1] = 1)
	   "AB" to "" costs 2 (column[2] = 2)
	   The column array will look like: [0, 1, 2, 3, ... len(s1)]
	*/
	if s1Len == 0 {
		return s2Len
	}
	if s2Len == 0 {
		return s1Len
	}

	// Ensure s1 is the shorter string to minimize memory usage
	if s1Len > s2Len {
		r1, r2 = r2, r1
		s1Len, s2Len = s2Len, s1Len
	}

	// Stack array for short strings (99% of filenames are < 64 chars)
	// Go compiler performs stack allocation for fixed-size arrays, zero heap alloc
	var stackBuf [64]int
	var column []int
	if s1Len+1 <= len(stackBuf) {
		column = stackBuf[:s1Len+1]
	} else {
		column = make([]int, s1Len+1)
	}

	for i := range column {
		column[i] = i
	}

	/*
	   Instead of a full matrix, we use 'column' as a stack and overwrite values as we go
	   Visualize the matrix:
	           "" |  A |  B |  C
	         ┌────┬────┬────┬────┐
	       ""│  0 │  1 │  2 │  3 │  <- Initial state: "" to A,B,C costs 1,2,3 steps
	         ├────┼────┼────┼────┤
	       A │  1 │  0 │  1 │  2 │  <- A=A (0 cost), others +1 based on previous steps
	         ├────┼────┼────┼────┤
	       X │  2 │  1 │  ? │  2 │  <- AX to AB: if X != B, cost = min(Top, Left, Diagonal) + 1
	         ├────┼────┼────┼────┤
	       C │  3 │  2 │  2 │  1 │  <- Result at (C,C) is 1 due to match (no cost increase)
	         └────┴────┴────┴────┘
	   We calculate current cell based on: min(Above, Left, Diagonal Above-Left) + cost
	*/

	for i := 1; i <= s2Len; i++ {
		column[0] = i    // Example: "" -> "A" (1 add), "" -> "AX" (2 adds)
		lastKey := i - 1 // Stores the Diagonal Above-Left value for the next cell
		for j := 1; j <= s1Len; j++ {
			/*
			   IMPORTANT: Save the current column[j] before it gets overwritten
			   It currently holds the "Above" value for the current cell,
			   and will become the "Diagonal" value for the next cell (j+1)
			*/
			oldKey := column[j]

			/*
			   Calculation logic:
			          	    (lastKey)    (old column[j])
			       		    DIAGONAL    |     ABOVE
			                   ↘        |      ↓
			                       ┌───────┐
			             LEFT    → │  ???  │ (Calculating...)
			         (column[j-1]) └───────┘
			*/
			incr := 0
			if r1[j-1] != r2[i-1] {
				incr = 1
			}

			// min(Above, Left, Diagonal)
			// minVal := column[j] + 1 // Delete
			// if column[j-1]+1 < minVal {
			// 	minVal = column[j-1] + 1 // Add
			// }
			// if lastKey+incr < minVal {
			// 	minVal = lastKey + incr // Substitute
			// }
			minVal := min(column[j]+1, column[j-1]+1, lastKey+incr)
			column[j] = minVal

			// The "Above" value of current cell becomes "Diagonal" for the next one
			lastKey = oldKey
		}
	}
	return column[s1Len]
}
