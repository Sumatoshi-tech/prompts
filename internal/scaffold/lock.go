package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const lockDir = ".promptkit"
const lockFile = "lock"

// AcquireLock creates a lock file in dir/.promptkit/lock containing the
// current PID. If a lock already exists and the owning process is still
// alive, it returns an error. Stale locks (dead PID) are reclaimed.
func AcquireLock(dir string) error {
	lockPath := filepath.Join(dir, lockDir, lockFile)

	if err := os.MkdirAll(filepath.Join(dir, lockDir), 0o755); err != nil {
		return fmt.Errorf("creating lock directory: %w", err)
	}

	// Check existing lock.
	data, err := os.ReadFile(lockPath)
	// Stale lock — reclaim by overwriting.
	if err == nil {
		pidStr := strings.TrimSpace(string(data))
		if pid, err := strconv.Atoi(pidStr); err == nil {
			if processAlive(pid) {
				return fmt.Errorf("another promptkit process (PID %d) is running in %s; remove %s if this is stale", pid, dir, lockPath)
			}
		}
	}

	// Write our PID.
	pid := os.Getpid()
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	return nil
}

// ReleaseLock removes the lock file from dir/.promptkit/lock.
func ReleaseLock(dir string) {
	lockPath := filepath.Join(dir, lockDir, lockFile)
	os.Remove(lockPath)
}

// processAlive checks whether a process with the given PID is running.
func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if the process exists without actually sending a signal.
	err = process.Signal(syscall.Signal(0))

	return err == nil
}
