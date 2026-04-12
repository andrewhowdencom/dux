package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// Workspace defines an isolated location where active state can be persisted
// and shared between multiple execution modes in a single session.
type Workspace struct {
	sessionID string
	dir       string
}

// New creates or connects to a persistent workspace in the XDG data directory tightly bound to the session.
func New(sessionID string) (*Workspace, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	dir := filepath.Join(xdg.DataHome, "dux", "sessions", sessionID, "workspace")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session workspace: %w", err)
	}

	return &Workspace{
		sessionID: sessionID,
		dir:       dir,
	}, nil
}

// DirPath returns the absolute filesystem path for this session's workspace.
func (w *Workspace) DirPath() string {
	return w.dir
}
