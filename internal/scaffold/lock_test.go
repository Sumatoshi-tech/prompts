package scaffold_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Sumatoshi-tech/promptkit/internal/scaffold"
)

func TestAcquireLock_Success(t *testing.T) {
	dir := t.TempDir()

	if err := scaffold.AcquireLock(dir); err != nil {
		t.Fatalf("AcquireLock() error: %v", err)
	}
	defer scaffold.ReleaseLock(dir)

	// Lock file should exist with our PID.
	data, err := os.ReadFile(filepath.Join(dir, ".promptkit", "lock"))
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Fatalf("parsing PID from lock file: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("lock PID = %d, want %d", pid, os.Getpid())
	}
}

func TestAcquireLock_Conflict(t *testing.T) {
	dir := t.TempDir()

	// Acquire the lock (our own PID — alive).
	if err := scaffold.AcquireLock(dir); err != nil {
		t.Fatalf("first AcquireLock() error: %v", err)
	}
	defer scaffold.ReleaseLock(dir)

	// Second acquire should fail because our PID is alive.
	err := scaffold.AcquireLock(dir)
	if err == nil {
		t.Fatal("expected error for conflicting lock, got nil")
	}

	if !strings.Contains(err.Error(), "another promptkit process") {
		t.Errorf("error = %q, want to contain 'another promptkit process'", err.Error())
	}
}

func TestAcquireLock_StaleLock(t *testing.T) {
	dir := t.TempDir()
	lockDir := filepath.Join(dir, ".promptkit")

	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a lock with a PID that (almost certainly) doesn't exist.
	// PID 2147483647 is the max PID on Linux and extremely unlikely to be in use.
	stalePID := "2147483647"
	if err := os.WriteFile(filepath.Join(lockDir, "lock"), []byte(stalePID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should succeed by reclaiming the stale lock.
	if err := scaffold.AcquireLock(dir); err != nil {
		t.Fatalf("AcquireLock() should reclaim stale lock, got error: %v", err)
	}
	defer scaffold.ReleaseLock(dir)

	// Verify our PID is now in the lock.
	data, err := os.ReadFile(filepath.Join(lockDir, "lock"))
	if err != nil {
		t.Fatal(err)
	}

	pidStr := strings.TrimSpace(string(data))
	if pidStr != strconv.Itoa(os.Getpid()) {
		t.Errorf("lock PID = %s, want %d", pidStr, os.Getpid())
	}
}

func TestReleaseLock_RemovesFile(t *testing.T) {
	dir := t.TempDir()

	if err := scaffold.AcquireLock(dir); err != nil {
		t.Fatalf("AcquireLock() error: %v", err)
	}

	scaffold.ReleaseLock(dir)

	lockPath := filepath.Join(dir, ".promptkit", "lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should be removed after ReleaseLock")
	}
}

func TestReleaseLock_NoopWhenMissing(t *testing.T) {
	dir := t.TempDir()

	// Should not panic or error when no lock exists.
	scaffold.ReleaseLock(dir)
}
