.PHONY: test bench check

test:
	@go test -v ./tests/...

bench:
	@go test -bench=. -benchmem ./bench/...

check:
	@go run ./scripts/test_analyzer.go
