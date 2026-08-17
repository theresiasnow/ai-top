package usage

import (
	"os"
	"path/filepath"
	"time"
)

// OpenClawSource reports OpenClaw's presence without inventing usage figures.
//
// OpenClaw keeps no usage log: a sweep of ~/.openclaw on 2026-08-17 found token
// fields only inside crash dumps under logs/stability. So this source reports
// Available=false even when OpenClaw is installed, which renders the row as
// "—" across the token and quota columns.
//
// An empty cell is the honest output here. Showing 0 would read as "OpenClaw
// used nothing", which is a different and false claim. If OpenClaw ever gains
// a usage log, this is the one file that needs to change.
type OpenClawSource struct {
	root string
}

// NewOpenClawSource creates a source for the default OpenClaw directory.
func NewOpenClawSource() *OpenClawSource {
	return &OpenClawSource{root: DefaultOpenClawRoot()}
}

// DefaultOpenClawRoot returns the standard OpenClaw directory.
func DefaultOpenClawRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".openclaw")
}

func (o *OpenClawSource) Name() string { return "OpenClaw" }

// Installed reports whether OpenClaw is present on this machine. The usage view
// uses this to tell "installed but reports nothing" apart from "not installed".
func (o *OpenClawSource) Installed() bool {
	if o.root == "" {
		return false
	}
	_, err := os.Stat(o.root)
	return err == nil
}

// Collect always reports unavailable usage: there is no usage log to read.
func (o *OpenClawSource) Collect(since time.Time) (HarnessUsage, error) {
	return HarnessUsage{
		Harness:     o.Name(),
		Available:   false,
		QuotaSource: QuotaNone,
	}, nil
}
