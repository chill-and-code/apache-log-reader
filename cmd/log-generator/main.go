package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"
)

const (
	dataDir        = "testdata"
	dateTimeFormat = "02/Jan/2006:15:04:05 -0700"
)

// The process should take ~5 minutes, for an Intel i9.
// The results may vary depending on the CPU.
func main() {
	dirFlag := flag.String("dir", ".", "the directory to store all the testdata in")
	intervalFlag := flag.Duration("interval", 10*time.Second, "interval between each log line")
	maxFilesFlag := flag.Int("max-files", 10, "maximum number of log files")
	maxLinesFlag := flag.Int("max-lines", 50_000_000, "maximum number of lines per log file")
	minLinesFlag := flag.Int("min-lines", 100, "minimum number of lines per log file")
	flag.Parse()

	now := time.Now()
	defer func() {
		fmt.Println("elapsed", time.Since(now))
	}()

	err := os.MkdirAll(path.Join(*dirFlag, dataDir), 0777)
	if err != nil && os.IsNotExist(err) {
		log.Fatalf("could not create data directory: %v", err)
	}

	file, err := os.Create(path.Join(*dirFlag, dataDir, "http-1.log"))
	if err != nil {
		log.Fatalf("could not create the first file: %v", err)
	}

	log.Println("generating the giant log file")
	ticker := time.NewTicker(10 * time.Second)
	nowUTC := time.Now().UTC()
	timeRange := nowUTC
	interval := *intervalFlag
	max := *maxLinesFlag // ~5GB
	for i := 0; i < max; i++ {
		select {
		case <-ticker.C:
			log.Println("iteration:", i, "generating logs, waiting...")
		default:
			timeRange = nowUTC.Add(-time.Duration(max-i) * interval)
			logLine := fmt.Sprintf(
				"127.0.0.1 user-identifier frank [%v] \"GET /api/endpoint HTTP/1.0\" 500 123\n",
				timeRange.Format(dateTimeFormat),
			)

			_, err := file.WriteString(logLine)
			if err != nil {
				log.Fatalf("could not write log to file: %v", err)
			}
		}
	}
	err = os.Chtimes(path.Join(*dirFlag, dataDir, "http-1.log"), timeRange, timeRange)
	if err != nil {
		log.Fatalf("could set modified time for file: %s: %v", file.Name(), err)
	}

	log.Println("generating 10 other smaller log files")
	maxFiles := *maxFilesFlag
	maxLogsPerFile := *minLinesFlag
	timeRange = timeRange.Add(time.Duration(maxLogsPerFile)*interval + interval)
	for i := 1; i < maxFiles; i++ {
		f, err := os.Create(path.Join(*dirFlag, dataDir, fmt.Sprintf("http-%d.log", i+1)))
		if err != nil {
			log.Fatalf("could not create file %d: %v", i+1, err)
		}

		for j := 0; j < maxLogsPerFile; j++ {
			logLine := fmt.Sprintf(
				"127.0.0.1 user-identifier frank [%v] \"GET /api/endpoint HTTP/1.0\" 500 123\n",
				timeRange.Add(-time.Duration(maxLogsPerFile-j)*interval).Format(dateTimeFormat),
			)

			_, err := f.WriteString(logLine)
			if err != nil {
				log.Fatalf("could not write log to file: %v", err)
			}
		}

		err = os.Chtimes(path.Join(*dirFlag, dataDir, fmt.Sprintf("http-%d.log", i+1)), timeRange, timeRange)
		if err != nil {
			log.Fatalf("could set modified time for file: %s: %v", f.Name(), err)
		}
		timeRange = timeRange.Add(time.Duration(maxLogsPerFile) * interval)

		_ = f.Close()
	}
}
