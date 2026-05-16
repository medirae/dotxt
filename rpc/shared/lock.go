package shared

import (
	"fmt"
	"sort"
	"sync"
)

type RLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

// LockMode indicates read or write intent
type LockMode int

const (
	Read LockMode = iota
	Write
)

<<<<<<< Updated upstream
var (
	// holds named locks.
	registry   = make(map[string]RLocker)
	registryMu sync.RWMutex
	// holds statically defined locks, since there's no variability, there's no need for a mutex.
	registryStatic = make(map[string]RLocker)
)

// registers a lock under a canonical name.
// If a lock with that name already exists, it is overwritten.
func registerLock(name string, l RLocker) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = l
}

// registers locks in the static map; must only be used during init
func registerStaticLock(name string, l RLocker) {
	registryStatic[name] = l
}

// returns a lock if registered (statically or otherwise)
func getLock(name string) (RLocker, bool) {
	l, ok := registryStatic[name]
	if !ok {
		registryMu.RLock()
		l, ok = registry[name]
		registryMu.RUnlock()
	}
	return l, ok
}

func PrefixPath(path string) string {
	return "path:" + path
}

// checks whether the corresponding path lock exists in registry
func PathLockExists(path string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := registry[PrefixPath(path)]
	return ok
}

// ensures a *sync.RWMutex exists for path and registers an adapter
// under the canonical name "path:<path>". returns the RLocker adapter.
// does not overwrite existing pathname
func RegisterPathLock(path string) RLocker {
	key := PrefixPath(path)

	if l, ok := getLock(key); ok {
		return l
	}
	mu := new(sync.RWMutex)
	registerLock(key, mu)
	return mu
}

// builds NamedLocks for a slice of paths
func NamedLocksForPaths(paths []string, mode LockMode) []NamedLock {
	out := make([]NamedLock, 0, len(paths))
	for _, p := range paths {
		out = append(out, NamedLock{Name: PrefixPath(p), Mode: mode})
	}
	return out
}

// builds NamedLocks for a slice of keys
func NamedLocks(keys []string, mode LockMode) []NamedLock {
	out := make([]NamedLock, 0, len(keys))
	for _, p := range keys {
		out = append(out, NamedLock{Name: p, Mode: mode})
	}
	return out
}

type NamedLock struct {
	Name string
	Mode LockMode
}

// takes a slice of required named locks (can include "path:<p>" entries or global locks),
// sorts & deduplicates by name (Write wins against Read in dedupe), then acquires them in sorted order.
||||||| Stash base
=======
func NewMutex() *sync.Mutex     { return new(sync.Mutex) }
func NewRWMutex() *sync.RWMutex { return new(sync.RWMutex) }

var (
	// holds named locks.
	registry   = make(map[string]RLocker)
	registryMu sync.RWMutex
	// holds statically defined locks, since there's no variability, there's no need for a mutex.
	registryStatic = make(map[string]RLocker)
)

// registers a lock under a canonical name.
// If a lock with that name already exists, it is overwritten.
func registerLock(name string, l RLocker) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = l
}

// registers locks in the static map; must only be used during init
func registerStaticLock(name string, l RLocker) {
	registryStatic[name] = l
}

// returns a lock if registered (statically or otherwise)
func getLock(name string) (RLocker, bool) {
	l, ok := registryStatic[name]
	if !ok {
		registryMu.RLock()
		l, ok = registry[name]
		registryMu.RUnlock()
	}
	return l, ok
}

func PrefixPath(path string) string {
	return "path:" + path
}

// checks whether the corresponding path lock exists in registry
func PathLockExists(path string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := registry[PrefixPath(path)]
	return ok
}

// ensures a *sync.RWMutex exists for path and registers an adapter
// under the canonical name "path:<path>". returns the RLocker adapter.
// does not overwrite existing pathname
func RegisterPathLock(path string) RLocker {
	key := PrefixPath(path)

	if l, ok := getLock(key); ok {
		return l
	}
	mu := new(sync.RWMutex)
	registerLock(key, mu)
	return mu
}

// builds NamedLocks for a slice of paths
func NamedLocksForPaths(paths []string, mode LockMode) []NamedLock {
	out := make([]NamedLock, 0, len(paths))
	for _, p := range paths {
		out = append(out, NamedLock{Name: PrefixPath(p), Mode: mode})
	}
	return out
}

// builds NamedLocks for a slice of keys
func NamedLocks(keys []string, mode LockMode) []NamedLock {
	out := make([]NamedLock, 0, len(keys))
	for _, p := range keys {
		out = append(out, NamedLock{Name: p, Mode: mode})
	}
	return out
}

type NamedLock struct {
	Name string
	Mode LockMode
}

// takes a slice of required named locks (can include "path:<p>" entries or global locks),
// sorts & deduplicates by name (Write wins), then acquires them in sorted order.
>>>>>>> Stashed changes
// returns an unlock func which will release the locks in reverse order.
func AcquireLocks(req []NamedLock) (func(), error) {
	// dedupe into map[name]Mode where Write overrides Read
	winner := make(map[string]LockMode, len(req))
	for _, n := range req {
		if existing, ok := winner[n.Name]; ok {
			if existing == Write || n.Mode == Read {
				// keep existing
				continue
			}
		}
		// either not present or new is Write (upgrade)
		winner[n.Name] = n.Mode
	}

	// collect & sort names deterministically
	names := make([]string, 0, len(winner))
	for nm := range winner {
		names = append(names, nm)
	}
	sort.Strings(names)

	locks := make([]RLocker, 0, len(names))
	modes := make([]LockMode, 0, len(names))
	// gather lock objects (ensuring path locks on demand)
	for _, nm := range names {
		l, ok := getLock(nm)
		if !ok {
			return nil, fmt.Errorf("AcquireLocks: lock name '%s' was not in registry", nm)
		}
		locks = append(locks, l)
		modes = append(modes, winner[nm])
	}

	// acquire in order.
	for i, l := range locks {
		if modes[i] == Write {
			l.Lock()
		} else {
			l.RLock()
		}
	}

	// unlock function: reverse order, matching modes
	return func() {
		for i := len(locks) - 1; i >= 0; i-- {
			if modes[i] == Write {
				locks[i].Unlock()
			} else {
				locks[i].RUnlock()
			}
		}
	}, nil
}
