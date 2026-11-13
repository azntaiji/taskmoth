//go:build unix

package lock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Acquire takes an exclusive advisory lock (flock) on the file at path,
// creating it if needed. It returns ErrLocked if another holder has it.
func Acquire(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close() // already returning the lock error
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("locking %s: %w", path, err)
	}
	return &Lock{release: f.Close}, nil // closing the file drops the flock
}
