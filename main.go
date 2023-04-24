package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"main/healthcheck"

	"gopkg.in/yaml.v3"
)

func main() {

	// define constants
	const pingTimeout = 500 * time.Millisecond
	const pingInterval = 15 * time.Second

	// create cancelable context
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// parse args
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <endpoints_path>\n", os.Args[0])
		os.Exit(1)
	}
	endpointsPath := os.Args[1]

	// open endpoints file
	endpointsFile, err := os.Open(endpointsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer endpointsFile.Close()

	// parse endpoints file
	var endpoints []healthcheck.Endpoint
	err = yaml.NewDecoder(endpointsFile).Decode(&endpoints)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// catch SIGINT
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		<-sigs
		ctxCancel()
	}()

	// create HTTP client
	client := &http.Client{
		Timeout: pingTimeout,
	}

	// start periodic pinging
	err = healthcheck.PeriodicHttpPing(ctx, client, endpoints, pingInterval, healthcheck.PrintStats)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
