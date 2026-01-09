// pbcopy copies stdin to the system clipboard.
// This is a Linux equivalent of macOS's pbcopy command.
//
// Usage:
//
//	echo "hello" | pbcopy
//	cat file.txt | pbcopy
//	pbcopy < file.txt
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/codyseavey/tools/clipboard/internal/clipboard"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "pbcopy: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Read all input from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Initialize clipboard
	cb, err := clipboard.New()
	if err != nil {
		return err
	}

	// Copy to clipboard
	if err := cb.Copy(data); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	return nil
}
