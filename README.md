# Apache Log Reader

Blazing fast log reader capable of working with giant log files (gigabytes) without too much of a spin.
Give it a try 🚀

## Prerequisites

- Make sure you have installed Go version >= `1.17`

## Resources

- [YouTube Video](https://youtube.com/c/SteveHook)
- [Log Generator Archive](https://raw.githubusercontent.com/chill-and-code/apache-log-reader/master/archives/log-generator.tar.gz)

## Build

```shell
# compiles and generates binaries for log-reader and log-generator inside the ./bin directory
make build
``` 

## Run

```shell
# run: "make build" first
# only run once to generate the test data, it may take a while (~5m)
./bin/log-generator
./bin/log-generator -dir <path/to/generated/logs>
# run the log-reader with the specified cli arguments
./bin/log-reader -d <path/to/log/files> -t <last_n_minutes>
# run the program directory without generating any binary
go run cmd/log-generator/main.go -dir <path/to/dir/testdata> -interval <interval_between_logs> lines-max <max_number_of_lines_per_log_file> lines-min <min_number_of_lines_per_log_file>
go run cmd/log-reader/main.go -d <path/to/log/files> -t <last_n_minutes>
# generate testdata in the current directory
./bin/log-generator
# adjust maximum/minimum number of logs per file and maximum number of log files
./bin/log-generator -lines-max 100000 -lines-min 50
go run cmd/log-generator/main.go -max-files=5 -max-lines=5 -min-lines=5
# display all logs from testdata directory that happened in the last 5 minutes
./bin/log-reader -d ./testdata -t 5
```

## Test

```shell
# runs all the tests present in test files
make test
# generate the benchmark data
make benchdata
# runs all the benchmarks present in test files
make bench
```

## Benchmarks

`M1 Max`

```text
goos: darwin
goarch: arm64
pkg: github.com/chill-and-code/apache-log-reader/logging
BenchmarkIndexTime-10    	    3340	    369069 ns/op	   84364 B/op	     145 allocs/op
PASS
ok  	github.com/chill-and-code/apache-log-reader/logging	2.230s
```

`Intel i9`

```text
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkIndexTime-16    	    1258	    800551 ns/op	   91046 B/op	     156 allocs/op
PASS
ok  	github.com/chill-and-code/apache-log-reader/logging	1.126s
```
