package main

import (
	"flag"
	"log"
	"os"

	"github.com/steevehook/weblog-analytics/logging"
)

func main() {
	directoryFlag := flag.String("d", ".", "the directory where all the logs are stored")
	minutesFlag := flag.Int("t", 1, "last n minutes worth of logs to read")

	flag.Parse()

	cfg := logging.LogsConfig{
		Directory:    *directoryFlag,
		LastNMinutes: *minutesFlag,
	}
	logs, err := logging.NewLogs(cfg)
	if err != nil {
		log.Fatalf("could not create logs: %v", err)
	}

	err = logs.Print(os.Stdout)
	if err != nil {
		log.Fatalf("could not print logs: %v", err)
	}
}
