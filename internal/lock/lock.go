// Package lock keeps a second taskmoth from opening the same database
// (decision G8.4).
package lock

import "errors"

// FileName is the lock file inside the data directory.
const FileName = "taskmoth.lock"

// ErrLocked means another process holds the lock.
var ErrLocked = errors.New("already locked by another process")

// Lock is a held lock; Release frees it. The operating system also frees it
// if the process exits or crashes.
type Lock struct{ release func() error }

// Release frees the lock.
func (l *Lock) Release() error { return l.release() }
