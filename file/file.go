package file

import (
	"dotxt/file/info"
	"dotxt/file/paths"
	"dotxt/terrors"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func MkDir(path string, all bool) error {
	var err error
	if all {
		err = os.MkdirAll(path, 0755)
	} else {
		err = os.Mkdir(path, 0755)
	}
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func MkStructure() error {
	err := MkDir(paths.TextsDir(), true)
	if err != nil {
		return err
	}
	err = MkDir(paths.DoneDir(), false)
	if err != nil {
		return err
	}
	err = MkDir(paths.BackupDir(), false)
	if err != nil {
		return err
	}
	err = MkDir(paths.ArchiveDir(), false)
	if err != nil {
		return err
	}
	return nil
}

func Walk(root string, fn func(string, os.FileInfo, error)) {
	stack := []string{root}
	for len(stack) > 0 {
		path := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		i, err := os.Lstat(path)
		if err != nil {
			fn(path, i, err)
			continue
		}
		if i.Mode()&os.ModeSymlink != 0 {
			targetInfo, err := os.Stat(path)
			if err != nil {
				fn(path, i, err)
				continue
			}
			i = targetInfo
		}
		if i.Mode().IsDir() {
			entries, err := os.ReadDir(path)
			fn(path, i, err)
			if err != nil {
				continue
			}
			for _, e := range entries {
				stack = append(stack, filepath.Join(path, e.Name()))
			}
		} else if i.Mode().IsRegular() {
			fn(path, i, nil)
		}
	}
}

// WARN: does not validate or modify path
// ensures file existence: if it doesn't exist, it's created; and if it exists, it's untouched.
func Create(path string) error {
	if err := MkDir(filepath.Dir(path), true); err != nil {
		return err
	}
	fd, err := os.OpenFile(path, os.O_CREATE, 0644)
	if fd != nil {
		defer fd.Close()
	}
	return err
}

// WARN: does not validate or modify path
func Delete(path string) error {
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// WARN: does not validate or modify path
func Read(path string) ([]string, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return []string{}, err
	}
	return strings.Split(string(bs), "\n"), nil
}

// WARN: does not validate or modify path
func Write(path, text string) error {
	return os.WriteFile(path, []byte(text), 0644)
}

// WARN: does not validate or modify path
func Append(path, text string) error {
	data, err := Read(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		text = strings.Join(data, "\n") + "\n" + text
	}
	return Write(path, text)
}

// the archive paths are untouched
// renames (copy&remove) the related paths to new related paths
// if any of the new paths are occupied the operation never occurs
// if any of the paths errors, the operation is reverted
func RenamePath(old, new *paths.Path) error {
	if old == nil || new == nil {
		return fmt.Errorf("%w: nil value not allowed", terrors.ErrArg)
	}

	pathOccupied := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil || !os.IsNotExist(err)
	}
	avails := map[string]string{
		old.Path:         new.Path,
		old.Done():       new.Done(),
		old.Backup():     new.Backup(),
		old.DoneBackup(): new.DoneBackup(),
	}
	for op := range avails {
		i, err := info.Identify(op)
		if err != nil && !os.IsNotExist(err) {
			continue
		}
		if !i.Exists {
			delete(avails, op)
			continue
		}
		np := avails[op]
		if pathOccupied(np) {
			return fmt.Errorf("%w: %w: destination path was occupied: '%s'",
				terrors.ErrPath, terrors.ErrFile, np)
		}
	}

	var errs []error
	var stack []func() error
	for op, np := range avails {
		if err := os.Rename(op, np); err != nil {
			errs = append(errs, err)
			for ndx := len(stack) - 1; ndx >= 0; ndx-- {
				fn := stack[ndx]
				if err := fn(); err != nil {
					errs = append(errs, err)
				}
			}
			break
		}
		stack = append(stack, func() error { return os.Rename(np, op) })
	}
	return errors.Join(errs...)
}

// WARN: does not validate or modify path
// streams content of src to dest
// applies file mode bits
func Copy(src, dest string) error {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sfi.Mode())
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// if any of the paths are available they will be backed up
// if any files are present at the backup destinations they will be overwritten
func Backup(path *paths.Path) error {
	if path == nil {
		return fmt.Errorf("%w: nil value not allowed", terrors.ErrArg)
	}
	if err := Copy(path.Path, path.Backup()); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := Copy(path.Done(), path.DoneBackup()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func Archive(path *paths.Path) error {
	if path == nil {
		return fmt.Errorf("%w: nil value not allowed", terrors.ErrArg)
	}

	timePrefix := time.Now().Format("2006-01-02t15-04-05") + "--"
	name := path.Name
	if strings.ContainsRune(name, '/') {
		name = filepath.Join(filepath.Dir(name), timePrefix+filepath.Base(name))
	} else {
		name = timePrefix + name
	}

	var errs []error
	j := func(in string) string { return filepath.Join(paths.ArchiveDir(), in) }
	for s, d := range map[string]string{
		path.Path:         j(name),
		path.Done():       j(name + ".done"),
		path.Backup():     j(name + ".bak"),
		path.DoneBackup(): j(name + ".done.bak"),
	} {
		if err := Copy(s, d); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// arg path must be beneath paths.TextsDir()
func List(path string) ([]*paths.Path, error) {
	if !strings.HasPrefix(path, paths.TextsDir()) {
		return nil, fmt.Errorf("%w: %w: path '%s' must be beneath paths.TextsDir()",
			terrors.ErrArg, terrors.ErrPath, path)
	}
	var errs []error
	var out []*paths.Path
	Walk(path, func(s string, fi os.FileInfo, err error) {
		if err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
			return
		}
		p, perr := paths.NewPath(path)
		if perr != nil {
			errs = append(errs, perr)
			return
		}
		out = append(out, p)
	})
	return out, errors.Join(errs...)
}

// TODO: (tui)
// func UnArchive(path string) error

// removeFromDoneFile(ids []int, p *paths.Path) ([]string, error)
// 	reads path's donefile
// 	extracts lines marked by ids and removes them from donefile
// 	rewrites back donefile
// 	supports symlinks
