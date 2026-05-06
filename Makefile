.PHONY: test bench check-test check-bench

test:
	@go test -v ./tests/...

bench:
	@go test -bench=. -benchmem ./bench/...

ct: # check-test
	@go run ./scripts/test_analyzer.go

cb: # check-bench
	@go run ./scripts/bench_analyzer.go