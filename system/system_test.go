package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nightlyone/lockfile"
)

func TestCheckAlreadyRunning(t *testing.T) {
	tmpDir := t.TempDir()

	lockFile := filepath.Join(tmpDir, "test.lock")

	lock, err := lockfile.New(lockFile)
	if err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			t.Fatalf("CheckAlreadyRunning returned an error: %v", err)
		}
	}()
	_, err = CheckAlreadyRunning(false, false)
	if err != nil {
		t.Fatalf("CheckAlreadyRunning returned an error: %v", err)
	}

	os.Remove(lockFile)
	_, err = CheckAlreadyRunning(false, false)
	if err != nil {
		t.Fatalf("CheckAlreadyRunning returned an error: %v", err)
	}
}
