package cli

import (
	"os"
	"path/filepath"
)

// redirectStderr sends stderr to a log file so that stray output from
// dependencies cannot corrupt the stdio JSON-RPC stream.  The caller
// should defer f.Close() on the returned file.
//
// If ROADY_MCP_LOG is set it is used as the log path; otherwise stderr
// is redirected to os.DevNull.
func redirectStderr() (*os.File, error) {
	logPath := os.Getenv("ROADY_MCP_LOG")
	if logPath == "" {
		logPath = os.DevNull
	}

	// Ensure the parent directory exists when a real path is given.
	if logPath != os.DevNull {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return nil, err
		}
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	// Swap the file descriptor underlying os.Stderr so that anything
	// writing to fd 2 (e.g. Go runtime panics) also lands in the file.
	os.Stderr = f
	return f, nil
}
