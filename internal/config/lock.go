package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// ErrLocked is returned when another dojo process holds the lock.
var ErrLocked = errors.New("another dojo command is already running")

// Lock is an advisory whole-CLI lock. Only one lab or exam may be active at a
// time, and only one process may mutate an environment.
type Lock struct{ f *os.File }

// Acquire takes the lock, failing fast rather than blocking so the user gets a
// clear message instead of a hang.
func Acquire() (*Lock, error) {
	home, err := EnsureHome()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(home, "lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		holder, _ := os.ReadFile(f.Name())
		f.Close()
		if len(holder) > 0 {
			return nil, fmt.Errorf("%w (pid %s)", ErrLocked, string(holder))
		}
		return nil, ErrLocked
	}
	if err := f.Truncate(0); err == nil {
		f.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0)
	}
	return &Lock{f: f}, nil
}

// Release drops the lock.
func (l *Lock) Release() {
	if l == nil || l.f == nil {
		return
	}
	syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	l.f.Truncate(0)
	l.f.Close()
}
