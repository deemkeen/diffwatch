package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/deemkeen/diffwatch/internal/ui"
	"github.com/deemkeen/diffwatch/internal/watcher"
)

func main() {
	// Parse command line arguments
	var watchPath string
	var recursive bool

	flag.StringVar(&watchPath, "path", ".", "")
	flag.StringVar(&watchPath, "p", ".", "")

	flag.BoolVar(&recursive, "recursive", false, "")
	flag.BoolVar(&recursive, "r", false, "")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -p, -path string\n")
		fmt.Fprintf(os.Stderr, "    \tPath to watch for changes (default: current directory)\n")
		fmt.Fprintf(os.Stderr, "  -r, -recursive\n")
		fmt.Fprintf(os.Stderr, "    \tWatch all subdirectories recursively\n")
	}

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
