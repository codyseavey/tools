// pbpaste outputs the system clipboard contents to stdout.
// This is a Linux equivalent of macOS's pbpaste command.
//
// Usage:
//
//	pbpaste
//	pbpaste > file.txt
//	pbpaste | grep pattern
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/codyseavey/tools/clipboard/internal/clipboard"
)

func main() {
	if err := run(); err != nil {
		// Don't print error for empty clipboard (match macOS behavior)
		if !errors.Is(err, clipboard.ErrClipboardEmpty) {
			fmt.Fprintf(os.Stderr, "pbpaste: %v\n", err)
			os.Exit(1)
		}
	}
}

func run() error {
	// Initialize clipboard
	cb, err := clipboard.New()
	if err != nil {
		return err
	}

	// Get clipboard contents
	data, err := cb.Paste()
	if err != nil {
		return err
	}

	// Write to stdout
	_, err = os.Stdout.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}
