package shared

import (
<<<<<<< Updated upstream
	"os"
	"path/filepath"
	"sync"
	"time"
)

var internalEventTimeout time.Duration

func init() {
	registerStaticLock(ConfigLockName, ConfigLock)
	registerStaticLock(TaskLockName, TaskLock)
	registerStaticLock(SymLinksLockName, SymLinksLock)
	registerStaticLock(TrackedLockName, TrackedLock)
	registerStaticLock(InternalEventsLockName, InternalEventsLock)
}

var ( // resources
	ConfigLock     = new(sync.RWMutex)
	ConfigLockName = "3-config"

	// used to lock the task package itself so that write operations such as
	//  removing, adding, or updating the task.Lists or other such variables
	//  of the task package, are concurrent-safe.
	TaskLock     = NewRWXMutex(300 * time.Millisecond)
	TaskLockName = "4-task"

	// used to keep track of symlink->file/dir
	// note that the link is not necessarily direct: symlink->...symlink...->target
	SymLinks         = make(map[string]string)
	SymLinksLock     = new(sync.RWMutex)
	SymLinksLockName = "1-symlinks"
	// used to keep track of actual files/dirs that were to be tracked
	// instead of the parent path that was submitted to fsnotify
	Tracked         = make(map[string]bool) // TODO: make use of
	TrackedLock     = new(sync.RWMutex)
	TrackedLockName = "2-tracked"

	// used to mark an internal event,
	//  so upcoming events in a time-window can be dropped.
	InternalEvents         = make(map[string]*InternalEvent)
	InternalEventsLock     = new(sync.RWMutex)
	InternalEventsLockName = "0-internal-events"
)

var Socket string = filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "dotxt-rpc.socket")
||||||| Stash base
=======
	"time"
)

var internalEventTimeout time.Duration

func init() {
	registerStaticLock(ConfigLockName, ConfigLock)
	registerStaticLock(TaskLockName, TaskLock)
	registerStaticLock(SymLinksLockName, SymLinksLock)
	registerStaticLock(TrackedLockName, TrackedLock)
	registerStaticLock(InternalEventsLockName, InternalEventsLock)
}

var ( // resources
	ConfigLock     = NewRWMutex()
	ConfigLockName = "3-config"

	// used to lock the task package itself so that write operations such as
	//  removing, adding, or updating the task.Lists or other such variables
	//  of the task package, are concurrent-safe.
	TaskLock     = NewRWXMutex(300 * time.Millisecond)
	TaskLockName = "4-task"

	// used to keep track of symlink->file/dir
	// note that the link is not necessarily direct: symlink->...symlink...->target
	SymLinks         = make(map[string]string)
	SymLinksLock     = NewRWMutex()
	SymLinksLockName = "1-symlinks"
	// used to keep track of actual files/dirs that were to be tracked
	// instead of the parent path that was submitted to fsnotify
	Tracked         = make(map[string]bool) // TODO: make use of
	TrackedLock     = NewRWMutex()
	TrackedLockName = "2-tracked"

	// used to mark an internal event,
	//  so upcoming events in a time-window can be dropped.
	InternalEvents         = make(map[string]*InternalEvent)
	InternalEventsLock     = NewRWMutex()
	InternalEventsLockName = "0-internal-events"
)
>>>>>>> Stashed changes
