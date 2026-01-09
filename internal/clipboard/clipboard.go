// Package clipboard provides cross-platform clipboard access for Linux,
// supporting both X11 and Wayland display servers.
package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	// ErrNoClipboardTool is returned when no supported clipboard tool is found
	ErrNoClipboardTool = errors.New("no supported clipboard tool found (install xclip, xsel, or wl-clipboard)")
	// ErrClipboardEmpty is returned when the clipboard is empty
	ErrClipboardEmpty = errors.New("clipboard is empty")
)

// Backend represents a clipboard backend
type Backend interface {
	Copy(data []byte) error
	Paste() ([]byte, error)
	Available() bool
}

// Clipboard provides clipboard operations
type Clipboard struct {
	backend Backend
}

// New creates a new Clipboard instance, auto-detecting the appropriate backend
func New() (*Clipboard, error) {
	backend := detectBackend()
	if backend == nil {
		return nil, ErrNoClipboardTool
	}
	return &Clipboard{backend: backend}, nil
}

// Copy copies data to the clipboard
func (c *Clipboard) Copy(data []byte) error {
	return c.backend.Copy(data)
}

// Paste retrieves data from the clipboard
func (c *Clipboard) Paste() ([]byte, error) {
	return c.backend.Paste()
}

// detectBackend finds an available clipboard backend
func detectBackend() Backend {
	// Check for Wayland first (if WAYLAND_DISPLAY is set)
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		wl := &WaylandBackend{}
		if wl.Available() {
			return wl
		}
	}

	// Fall back to X11 backends
	xclip := &XclipBackend{}
	if xclip.Available() {
		return xclip
	}

	xsel := &XselBackend{}
	if xsel.Available() {
		return xsel
	}

	// Try Wayland even without WAYLAND_DISPLAY (some setups)
	wl := &WaylandBackend{}
	if wl.Available() {
		return wl
	}

	return nil
}

// WaylandBackend implements clipboard for Wayland using wl-copy/wl-paste
type WaylandBackend struct{}

// Available checks if wl-clipboard tools are installed
func (w *WaylandBackend) Available() bool {
	_, errCopy := exec.LookPath("wl-copy")
	_, errPaste := exec.LookPath("wl-paste")
	return errCopy == nil && errPaste == nil
}

// Copy copies data to the Wayland clipboard
func (w *WaylandBackend) Copy(data []byte) error {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy failed: %w", err)
	}
	return nil
}

// Paste retrieves data from the Wayland clipboard
func (w *WaylandBackend) Paste() ([]byte, error) {
	cmd := exec.Command("wl-paste", "-n")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// wl-paste returns error when clipboard is empty
		if out.Len() == 0 {
			return nil, ErrClipboardEmpty
		}
		return nil, fmt.Errorf("wl-paste failed: %w", err)
	}
	return out.Bytes(), nil
}

// XclipBackend implements clipboard for X11 using xclip
type XclipBackend struct{}

// Available checks if xclip is installed
func (x *XclipBackend) Available() bool {
	_, err := exec.LookPath("xclip")
	return err == nil
}

// Copy copies data to the X11 clipboard using xclip
func (x *XclipBackend) Copy(data []byte) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip failed: %w", err)
	}
	return nil
}

// Paste retrieves data from the X11 clipboard using xclip
func (x *XclipBackend) Paste() ([]byte, error) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if out.Len() == 0 {
			return nil, ErrClipboardEmpty
		}
		return nil, fmt.Errorf("xclip failed: %w", err)
	}
	return out.Bytes(), nil
}

// XselBackend implements clipboard for X11 using xsel
type XselBackend struct{}

// Available checks if xsel is installed
func (x *XselBackend) Available() bool {
	_, err := exec.LookPath("xsel")
	return err == nil
}

// Copy copies data to the X11 clipboard using xsel
func (x *XselBackend) Copy(data []byte) error {
	cmd := exec.Command("xsel", "--clipboard", "--input")
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xsel failed: %w", err)
	}
	return nil
}

// Paste retrieves data from the X11 clipboard using xsel
func (x *XselBackend) Paste() ([]byte, error) {
	cmd := exec.Command("xsel", "--clipboard", "--output")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if out.Len() == 0 {
			return nil, ErrClipboardEmpty
		}
		return nil, fmt.Errorf("xsel failed: %w", err)
	}
	return out.Bytes(), nil
}
