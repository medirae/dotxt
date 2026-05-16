package info

// TODO: write a wrapper for os.Stat and os.Lstat in rpc.shared and in them use a sync.Pool to limit concurrent kernel calls

import (
	"dotxt/terrors"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// finds the final target of a link
// equipped with a cycle detector
// if the target does not exist, or is not a regular file or directory, it will error
func EvalSymlink(path string) (string, os.FileInfo, error) {
	seen := make(map[string]bool) // detect cylces
	for {
		if seen[path] {
			return "", nil, fmt.Errorf("%w: symlink loop detected for '%q'", terrors.ErrPath, path)
		}
		seen[path] = true

		info, err := os.Lstat(path)
		if err != nil {
			return "", nil, fmt.Errorf("%w: could not Lstat path '%q': %w", terrors.ErrPath, path, err)
		}
		if mode := info.Mode(); mode&os.ModeSymlink == 0 && (mode.IsDir() || mode.IsRegular()) {
			return path, info, nil
		} else if mode&os.ModeSymlink == 0 {
			return "", nil, fmt.Errorf("%w: link target '%s' is not a regular file or directory", terrors.ErrFile, path)
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", nil, fmt.Errorf("%w: could not read symlink target of path '%q': %w", terrors.ErrPath, path, err)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
}

// checks whether the running process *should* have
// read and write access to path, based on file mode bits.
// It avoids writing/opening for write and does not create temp files.
//
// AI Note: Because of ACLs, SELinux, or capabilities, the result may be wrong.
// So you should still be prepared to handle EACCES/EPERM when actually reading/writing.
func HasPerms(path string, info os.FileInfo) (bool, bool, os.FileInfo, error) {
	var err error
	if info == nil {
		info, err = os.Lstat(path)
		if err != nil {
			return false, false, nil, fmt.Errorf("%w: failed to lstat path '%s' to check symlink: %w",
				terrors.ErrFile, path, err)
		}
	}

	if info.Mode()&os.ModeSymlink != 0 {
		_, err = os.Readlink(path)
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return false, false, info, nil // if symlink cannot be read, writing matters not
			} else {
				return false, false, nil, fmt.Errorf("%w: failed to read target of symlink '%s' for read access: %w",
					terrors.ErrFile, path, err)
			}
		}

		return true, true, info, nil // actual write permission on symlinks matters not
	}

	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		// fallback: can’t check ownership → trust mode bits loosely
		mode := info.Mode().Perm()
		return mode&0400 != 0, mode&0200 != 0, info, nil
	}

	// TODO: set these globally
	uid := os.Geteuid()
	gid := os.Getegid()

	mode := info.Mode().Perm()
	var r, w bool

	if sys.Uid == uint32(uid) { // owner
		r = mode&0400 != 0
		w = mode&0200 != 0
	}
	if sys.Gid == uint32(gid) { // group
		r = r || mode&0040 != 0
		w = w || mode&0020 != 0
	}
	{ // others
		r = r || mode&0004 != 0
		w = w || mode&0002 != 0
	}

	return r, w, info, nil
}

/*
represents a snapshot of filesystem path

- if p.IsSymlink:
  - DestAddr -> non-empty
  - IsFile -> link target isFile
  - Exists -> link
  - DestExists -> link target
  - HasRead -> Exists && link has read && DestExists && target has read
  - HasWrite -> Exists && link has read && DestExists && target has write
*/
type Info struct {
	Addr string

	DestAddr  string
	IsSymlink bool

	Exists     bool
	DestExists bool

	HasRead  bool
	HasWrite bool

	Mode    fs.FileMode
	Size    int64
	ModTime time.Time

	lastProbed time.Time // last probe dt
	// lock       *sync.RWMutex // this must be used by the user in concurrent parts of code
}

func (i *Info) String() string {
	if i.IsSymlink {
		return fmt.Sprintf("%s -> %s", i.Addr, i.DestAddr)
	}
	return i.Addr
}

// construct info and force a probe
func Identify(path string) (*Info, error) {
	info := NewInfo(path)
	if err := info.Refresh(true); err != nil {
		return info, err
	}
	return info, nil
}

// create shallow info object from path string
func NewInfo(path string) *Info {
	return &Info{Addr: filepath.Clean(path)}
}

// attempts to ensure there will be a probed info object returned
func EnsureInfo(i *Info, path string, force bool) (*Info, error) {
	path = filepath.Clean(path)
	if i == nil {
		i = NewInfo(path)
	}
	if i.Addr == "." && path == "." {
		return i, fmt.Errorf("%w: invalid address", terrors.ErrArg)
	} else if path != "." && (i.Addr == "." || i.Addr != path) {
		i.Addr = path
		i.lastProbed = time.Time{}
	}
	if err := i.Refresh(force); err != nil {
		return i, err
	}
	return i, nil
}

func (i *Info) AddrAccessible() bool {
	return i.Exists && ((i.IsSymlink && i.DestExists) || !i.IsSymlink) && i.HasRead
}

func (i *Info) NeedsRefresh() bool {
	if i.lastProbed.IsZero() {
		return true
	}
	// TODO (concurrency): review duration
	if i.lastProbed.After(time.Now().Add(-time.Second)) {
		return false
	}
	info, err := os.Lstat(i.Addr)
	if err != nil {
		return true
	}
	if i.Mode == info.Mode() &&
		i.Size == info.Size() &&
		i.ModTime.Equal(info.ModTime()) {
		return false
	}
	return true
}

func (i *Info) Refresh(force bool) error {
	addr := i.Addr
	if addr == "" || addr == "." {
		return fmt.Errorf("%w: empty path for Info", terrors.ErrPath)
	}
	if force || i.NeedsRefresh() {
		return i.probe()
	}
	return nil
}

func (i *Info) probe() error {
	addr := filepath.Clean(i.Addr)
	if addr == "" {
		return fmt.Errorf("%w: empty path for Info", terrors.ErrPath)
	}

	// reset fields that are derived
	*i = *NewInfo(addr)
	i.lastProbed = time.Now()

	linfo, err := os.Lstat(addr)
	if err != nil {
		i.Mode = 0
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("%w: could not Lstat path '%s': %w", terrors.ErrPath, i.Addr, err)
	}
	i.Exists = true
	i.Mode = linfo.Mode()
	var info os.FileInfo
	if i.Mode&os.ModeSymlink != 0 {
		i.IsSymlink = true
		destAddr, _info, err := EvalSymlink(addr)
		if err != nil {
			return fmt.Errorf("%w: could not eval symlink '%s': %w", terrors.ErrPath, i.Addr, err)
		}
		i.DestExists = true
		i.DestAddr = destAddr
		info = _info
	} else {
		info = linfo
	}

	i.Size = info.Size()
	i.ModTime = info.ModTime()

	reprAddr := addr
	if i.IsSymlink {
		reprAddr = i.DestAddr
	}
	lnr, lnw, _, err := HasPerms(reprAddr, info)
	if err != nil {
		return fmt.Errorf("%w: could not eval permissions of '%s': %w", terrors.ErrFile, reprAddr, err)
	}
	symOk := !i.IsSymlink || (i.IsSymlink && i.DestExists)
	i.HasRead = i.Exists && symOk && lnr
	i.HasWrite = i.Exists && symOk && lnw

	return nil
}
