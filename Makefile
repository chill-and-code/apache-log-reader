build:
	@echo "generating the log-reader binary"
	go build -o bin/log-reader cmd/log-reader/main.go
	@echo "generating the log-generator binary"
	go build -o bin/log-generator cmd/log-generator/main.go

test:
	@echo "running all tests"
	go test -count=1 -v ./...

bench:
	@echo "running all benchmarks"
	go test -bench . ./...
