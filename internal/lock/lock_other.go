//go:build !unix

package lock

// Acquire is a no-op where flock isn't available; taskmoth supports macOS
// and Linux (PLAN.md G0).
func Acquire(string) (*Lock, error) { return &Lock{release: func() error { return nil }}, nil }
