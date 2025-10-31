package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hv11297/diffwatch/internal/ui"
	"github.com/hv11297/diffwatch/internal/watcher"
)

func main() {
	// Parse command line arguments
	var watchPath string
	var recursive bool

	flag.StringVar(&watchPath, "path", ".", "Path to watch for changes")
	flag.StringVar(&watchPath, "p", ".", "Path to watch for changes (shorthand)")

	flag.BoolVar(&recursive, "recursive", false, "Watch all subdirectories recursively")
	flag.BoolVar(&recursive, "r", false, "Watch all subdirectories recursively (shorthand)")

	flag.Parse()

	// Validate path
	if _, err := os.Stat(watchPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: path does not exist: %s\n", watchPath)
		os.Exit(1)
	}

	// Create file watcher
	fw, err := watcher.New(watchPath, recursive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer fw.Close()

	// Create UI
	program := ui.New(fw)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		program.Quit()
	}()

	// Start the program
	if err := program.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
