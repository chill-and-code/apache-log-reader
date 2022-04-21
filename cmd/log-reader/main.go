package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/steevehook/weblog-analytics/logging"
)

func main() {
	quit := make(chan os.Signal, 1)
	directoryFlag := flag.String("d", ".", "the directory where all the logs are stored")
	minutesFlag := flag.Int("t", 1, "last n minutes of worth of logs to read")

	flag.Parse()
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	cfg := logging.LogsConfig{
		Directory:    *directoryFlag,
		LastNMinutes: *minutesFlag,
	}
	logs, err := logging.NewLogs(cfg)
	if err != nil {
		log.Fatalf("could not create log reader: %v", err)
	}

	go func() {
		err := logs.Print(ctx, os.Stdout)
		if err != nil {
			log.Fatalf("could not print logs: %v", err)
		}

		quit <- os.Interrupt
	}()

	<-quit
	cancel()
}
