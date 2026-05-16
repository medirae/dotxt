package server

import (
	"context"
	"dotxt/file/info"
	"dotxt/file/paths"
	"dotxt/logging"
	"dotxt/rpc/shared"
	"dotxt/utils"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

<<<<<<< Updated upstream
func trackPath(watcher *fsnotify.Watcher, addr string, pinfo *info.Info) {
	pinfo, err := info.EnsureInfo(pinfo, addr, false)
	if err != nil {
		logging.Logger.Warnf("trackPath: skipping path '%q': %w", addr, err)
		return
	}
	if !pinfo.AddrAccessible() {
		logging.Logger.Warnf("trackPath: skipping inaccessible path '%q'", addr)
		return
	}
	if pinfo.IsInArchive() || path.IsInEtc() {
||||||| Stash base
=======
// TODO: the use of file.info.Info is extremely inefficient, the os.Stat and os.Lstat are hit multiple times unnecessarily

func trackPath(watcher *fsnotify.Watcher, pathAddr string) {
	path, err := info.Identify(pathAddr)
	if err != nil {
		logging.Logger.Warnf("trackPath: skipping path '%q': %w", path, err)
		return
	}
	if !path.DoesExist() {
		logging.Logger.Warnf("trackPath: skipping non-existent path '%q'", path)
		return
	}
	if !path.HasRead {
		logging.Logger.Warnf("trackPath: skipping path '%q' with no read permission", path)
		return
	}
	if path.IsInArchive() || path.IsInEtc() {
>>>>>>> Stashed changes
		return
	}
	if path.IsFile && !path.HasWrite && !path.IsConfigFile() {
		logging.Logger.Warnf("trackPath: skipping file '%q' with no write permission", path)
		return
	}

	selectedPath := path.Addr
	if path.IsSymlink {
		selectedPath = path.DestAddr
		{ // track symlinks
			err := watcher.Add(filepath.Dir(path.Addr))
			if err != nil {
				logging.Logger.Warnf("skipping path '%q', failed adding to watcher: %w", path, err)
				return
			}
			shared.SymLinksLock.Lock()
			shared.SymLinks[path.Addr] = path.DestAddr
			shared.SymLinksLock.Unlock()
		}
	}

	{ // track directories and files
		err := watcher.Add(filepath.Dir(selectedPath))
		if err != nil {
			logging.Logger.Warnf("skipping path '%q', failed adding to watcher: %w", selectedPath, err)
			return
		}
	}

	if !path.IsFile { // recurse: track entries of directories
		entries, err := os.ReadDir(selectedPath)
		if err != nil {
			logging.Logger.Warnf("skipping path '%q', failed reading directory entries of: %w", selectedPath, err)
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				trackPath(watcher, e.Name())
			}
		}
	}
}

type eventWeb struct {
	ops   fsnotify.Op
	timer *time.Timer
}

var (
	// a web of events to capture consecutive events and react to them all at once
	eventWebs     = make(map[string]*eventWeb)
	eventWebsLock = new(sync.RWMutex)
)

func (ew *eventWeb) process(w *fsnotify.Watcher, path info.Info) {
	var mk bool
	var makeUp, tearDown bool
	var makeUpTarget, tearDownTarget string

	if path.IsSymlink {
		if ew.ops.Has(fsnotify.Create) {
			trackPath(w, path.Addr)
			mk, makeUp, makeUpTarget = true, true, path.DestAddr
		} else if ew.ops.Has(fsnotify.Write) {
			newTargetPath, err := info.EvalSymlink(path.Addr)
			if err != nil {
				logging.Logger.Warnf("RPC server: Watcher: eventWeb.process: skipping event, failed to evaluate symlink path '%q': %w", path.Addr, err)
				return
			}
			if path.DestAddr != newTargetPath {
				err = w.Remove(path.DestAddr)
				if err != nil {
					logging.Logger.Warnf("RPC server: Watcher: eventWeb.process: skipping event, failed to remove target path '%q': %w", path.DestAddr, err)
					return
				}
				shared.SymLinksLock.Lock()
				shared.SymLinks[path.Addr] = newTargetPath
				shared.SymLinksLock.Unlock()
				trackPath(w, path.Addr)
				mk, makeUp, makeUpTarget = true, true, newTargetPath
				tearDown, tearDownTarget = true, path.DestAddr
				path.DestAddr = newTargetPath
			}
		}
	} else if !path.IsFile && ew.ops.Has(fsnotify.Create) {
		trackPath(w, path.Addr)
	} else if ew.ops.Has(fsnotify.Write) || ew.ops.Has(fsnotify.Create) {
		mk, makeUp, makeUpTarget = true, true, path.Addr
	}

	if !mk {
		return
	}

	if path.IsConfigFile() {
		label := fmt.Sprintf("watcher-config-%s-%s", ew.ops, path.ReprAddr())
		_, _, err := MakeRequest(label, nil, func(ctx context.Context) error {
			// TODO: utilize config functions to read and write the config file
			return nil
		}, []shared.NamedLock{{Name: shared.ConfigLockName, Mode: shared.Write}})
		if err != nil {
			logging.Logger.Errorf("Watcher.eventWeb.process: failed to make request '%s': %w", label, err)
		}
	} else {
		exists := shared.PathLockExists(path.ReprAddr())
		taskMode := shared.Read
		if (tearDown && exists) || (makeUp && !exists) {
			taskMode = shared.Write
		}
		resources := []shared.NamedLock{{Name: shared.TaskLockName, Mode: taskMode}}
		resources = append(resources, shared.NamedLock{Name: shared.PrefixPath(path.ReprAddr()), Mode: shared.Write})
		label := fmt.Sprintf("watcher-task-%s-%s", ew.ops, path.ReprAddr())
		_, _, err := MakeRequest(
			label, nil,
			func(ctx context.Context) error {
				if tearDown {
					fmt.Println(tearDownTarget, makeUpTarget)
					// TODO: teardown target
				}
				if makeUp {
					// TODO: makeup target
				}
				return nil
			},
			resources,
		)
		if err != nil {
			logging.Logger.Errorf("Watcher.eventWeb.process: failed to make request '%s': %w", label, err)
		}
	}
}

func MakeWatcher() (*fsnotify.Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	trackPath(w, paths.TextsDir())
	return w, nil
}

// uses fsnotify to watch filesystem events
// must be run as a goroutine
func Watch(wg *sync.WaitGroup, done context.Context, w *fsnotify.Watcher) {
	defer wg.Done()

	for {
		select {
		case <-done.Done():
			return
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			logging.Logger.Error(err) // TODO: find out what kind of errors fsnotify can make
		case e, ok := <-w.Events:
			if !ok {
				return
			}

			if e.Has(fsnotify.Remove) || e.Has(fsnotify.Rename) {
				err := w.Remove(e.Name)
				if err != nil {
					logging.Logger.Warnf("RPC server: Watcher: eventWeb.process: skipping event, failed to remove path '%q': %w", e.Name, err)
					return
				}

				shared.SymLinksLock.RLock()
				destAddr, isSymlink := shared.SymLinks[e.Name]
				shared.SymLinksLock.RUnlock()
				if isSymlink {
					err = w.Remove(destAddr)
					if err != nil {
						logging.Logger.Warnf("RPC server: Watcher: eventWeb.process: skipping event, failed to remove target of symlink at path '%q': %w", destAddr, err)
						return
					}
					shared.SymLinksLock.Lock()
					delete(shared.SymLinks, e.Name)
					shared.SymLinksLock.Unlock()
				}
				// TODO: make request: teardown
				continue
			}

			path, err := info.Identify(e.Name)
			if err != nil {
				logging.Logger.Warnf("skipping event '%s', failed to identify path '%q': %w", e, path, err)
				continue
			}
			if !path.DoesExist() {
				logging.Logger.Warnf("RPC server: Watcher: skipping event '%s', non-existent path '%q'", e, path)
				continue
			}
			if !path.HasRead {
				logging.Logger.Warnf("RPC server: Watcher: skipping event '%s', path '%q' with no read permission", e, path)
				continue
			}
			if path.IsInArchive() || path.IsInEtc() {
				continue
			}
			if path.IsFile && !path.HasWrite && !path.IsConfigFile() {
				logging.Logger.Warnf("RPC server: Watcher: skipping event '%s', file '%q' with no write permission", e, path)
				continue
			}

			key := utils.GetFileCanonicalKey(path.ReprAddr())
			shared.InternalEventsLock.Lock()
			// TODO: maybe extend this with web of events?
			ie, ok := shared.InternalEvents[key]
			if ok {
				if !ie.IsValid() || ie.HasChanged(path.ReprAddr()) {
					delete(shared.InternalEvents, key)
					ie, ok = nil, false
				} else { // coalesce
					shared.InternalEventsLock.Unlock()
					continue
				}
			}
			shared.InternalEventsLock.Unlock()

			eventWebsLock.Lock()
			web, ok := eventWebs[path.Addr]
			if !ok {
				eventWebs[path.Addr] = &eventWeb{
					timer: time.AfterFunc(externalEventTimeout, func() {
						eventWebsLock.Lock()
						delete(eventWebs, path.Addr)
						eventWebsLock.Unlock()
						web.process(w, path)
					}),
					ops: e.Op,
				}
			} else {
				web.timer.Stop()
				web.ops = web.ops | e.Op
				web.timer.Reset(externalEventTimeout)
			}
			eventWebsLock.Unlock()
			continue
		}
	}
}

/*
default watcher behavior:
	- track only *parent directories* of files and directories
	- RENAME:
		- file:	tear down internal memory (if there's a create let the system recreate the memory)
			- unfortunately the time window is undeterministic, otherwise a time window of 10ms to
				catch the CREATE and then instead of tearing down, renaming internally would be more optimized.
		- dir: if any change to files is necessary, the event will have also been sent internally as well.
			and the previous path will be removed from the watcher automatically. so just ignore.
	- CHMOD:
		- file: create a timer to send an internal signal to watcher after say 100ms, to check the
			whether the process has read&write perms for path.
			- the timer is to make sure the perm check is not a premature one since CHMOD maybe
				followed by some other events as well.
		- dir: ignore.
	- REMOVE:
		- file: tear down internal memory. remove from watcher.
		- dir: ignore.
	- WRITE:
		- file: wait out every 100ms until they're all done, then reload internal memory
	- CREATE:
		- file: set a timer to wait out upcoming WRITE and CHMOD events; then create internal memory. &track
		- dir: track

	- create a web of events:
		- init: CREATE, WRITE, and CHMOD start a timer and a queue.
			- then any event that comes will be captured and stored in a queue.
		- CREATE: store
		- WRITE: reset timer
		- CHMOD: store
		- destroy: RENAME, REMOVE, and timer end are terminators.
			- stop timer, remove the web, and make the rpc request.

- develop a expectation system for CREATE and WRITE on top of the event web system:
	type exp struct {
		WriteID uint64
		Size int64
		ModTime time.Time
		TTL time.Time
	}
	expectMu sync.Mutex
	expects map[string]exp

	- use atomic write-pattern to reduce the number of events per internal op: write to /tmp then move
	- for bursty events, wait; and then after awhile after the last one, process to reduce unnecessary processing.
	- validate heavily before actiong
	- store metadata as a form of checksum to validate state and uniqueness to remove redundant processing; maybe mtime+size or something...

- REMEMBER!!: there are two actions:
	1. make-up:: read data from disk into lists - whether they already exist or not
	2. tear-down:: remove data from lists - if they exist
*/
/*
	ops for config file:
		- Create: read new config
		- Write: read new config
		- Remove: set default
		- Rename: set default
		- Chmod: check permissions
*/
