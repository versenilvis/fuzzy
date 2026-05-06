# Fuzzy scoring logic

This document breaks down exactly how the search engine decides which results show up first. We don't just look for character matches; we use a multi-tiered scoring system to make sure the file you're actually looking for is right at the top

Here is a deep dive into the scoring mechanics:

## 1. Optimal alignment algorithm

Instead of a simple greedy approach, we use a two-phase optimal alignment strategy. The engine first identifies all possible match positions and then performs a right-to-left scan to find the "best" path. This ensures that the engine maximizes contiguous bonuses by anchoring to the rightmost word segments first.

Example: searching for `ff` in `f_static_ff.go` will correctly ignore the first isolated `f` and instead pick the contiguous `ff` at the end to maximize the score

## 2. Priority tiers

To make the search "smart," we categorize results into four priority levels:

### Tier 1: Perfect filename prefix
If you type `main` and there is a file named `main.go`, it immediately gets a 1000000 point boost. This is our highest priority because if you type the start of a filename exactly, that is almost certainly what you want

Example: query `api` will rank `src/api.go` much higher than `src/_api.go`. This is because `api` is a perfect prefix of the filename `api.go` (Tier 1 boost), whereas in `_api.go`, the query starts after an underscore and misses that massive boost

### Tier 2: Short filename matches
If the filename is short (less than three times the query length) and contains all your query characters, it receives a 500000 point boost. This helps files like `init.py` or `app.js` float to the top when you type their core characters

Example: query `ip` will easily find `init.py` even if there are many other files containing those letters

### Tier 3: Hits in filename
Matches found within the actual filename part of a path have their score doubled. We always prioritize the filename itself over the parent directory names

Example: query `util` will rank `src/util.go` higher than `util/main.go` because the match is in the filename itself

### Tier 4: Hits in path
If your query is only found in the directory structure (like typing `src` and finding `src/utils.go`), the result is kept but given a lower priority compared to direct filename matches

## 3. Score boosts

Beyond the tiers, we add points based on the "quality" of the match:

- Contiguous matches (+200): If your characters appear right next to each other (like `abc` matching `abc`), the score jumps significantly
  - Example: query `api` will score much higher against `api_client.go` (where `a`, `p`, `i` are adjacent) than against `application_internal.go` (where `a`, `p`, `i` are scattered far apart)

- Word boundaries (+80): If a character matches at the very beginning of the string (index 0) or immediately after a separator like `/`, `_`, `.`, `-`, or a space, it gets a boost. This treats the start of the entire path as a natural word boundary
  - Example: query `src` matching `src/main.go` gets a boost because `s` is at the very beginning. Similarly, query `api` matching `src/_api.go` gets a boost because `a` follows an underscore

- Compactness: The shorter the overall range of the match (fewer skipped characters), the higher the base score. Example: query `main` will score much higher against `main.go` than against `module_archive_internal.go`

## 4. Penalties

- Path length: Deeper files with longer paths get a small penalty. This ensures that if two files have identical fuzzy matches, the one closer to the root directory wins
- Sparsity: If the matched characters are too scattered across a long path, the base score drops and we might even reject the result if it looks like noise

## 5. Frecency (history)

Example: if you have `abc/main.go` (frequently opened) and `xyz/mai.go` (never opened), searching for `mai` will rank `main.go` first. Even though `mai.go` is a closer physical match, the library applies a massive boost based on your history to prioritize the file you likely actually want. This boost slowly decays over time if you stop accessing that file
