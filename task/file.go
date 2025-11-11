package task

import (
	"dotxt/config"
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/text/unicode/norm"
)

/* file structure

dotxt-config-dir/
	todos/
		todo
		_etc/
			todo.done
			todo.bak
		_archive/
			prev
			prev.done
			prev.bak

*/

const DefaultTodo = "todo"

func todosDir() string {
	return filepath.Join(config.ConfigPath(), "todos")
}

func etcDir() string {
	return filepath.Join(todosDir(), "_etc")
}

func parseFilepath(path string) (string, error) {
	path = norm.NFC.String(path)
	if strings.TrimSpace(path) == "" {
		return filepath.Join(todosDir(), "todo"), nil
	}
	if utils.RuneAt(path, utils.RuneCount(path)-1) == '/' {
		return "", fmt.Errorf("%w: path cannot end in a / '%s'", terrors.ErrParse, path)
	}
	if filepath.IsAbs(path) {
		if strings.HasPrefix(path, filepath.Join(todosDir())+"/") {
			return path, nil
		} else {
			return "", fmt.Errorf("%w: filepath not under /todos '%s'", terrors.ErrParse, path)
		}
	}
	if tmpPath := filepath.Join(todosDir(), path); filepath.IsAbs(tmpPath) {
		return tmpPath, nil
	}
	return "", fmt.Errorf("%w: filepath '%s'", terrors.ErrParse, path)
}

func parseDirpath(path string) (string, error) {
	path = norm.NFC.String(path)
	if strings.TrimSpace(path) == "" {
		return filepath.Join(todosDir(), "todo"), nil
	}
	if filepath.IsAbs(path) {
		if strings.HasPrefix(path, filepath.Join(todosDir())) {
			return path, nil
		} else {
			return "", fmt.Errorf("%w: dirpath not under /todos '%s'", terrors.ErrParse, path)
		}
	}
	if tmpPath := filepath.Join(todosDir(), path); filepath.IsAbs(tmpPath) {
		return tmpPath, nil
	}
	return "", fmt.Errorf("%w: dirpath '%s'", terrors.ErrParse, path)
}

func mkDirs(path string) error {
	mkdir := func(path string, all bool) error {
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

	err := mkdir(todosDir(), true)
	if err != nil {
		return err
	}
	err = mkdir(filepath.Join(todosDir(), "_etc"), false)
	if err != nil {
		return err
	}
	err = mkdir(filepath.Join(todosDir(), "_archive"), false)
	if err != nil {
		return err
	}

	if strings.TrimSpace(path) != "" {
		path, err := parseDirpath(path)
		if err != nil {
			return err
		}
		postPath := strings.TrimPrefix(path, todosDir())
		if utils.RuneCount(postPath) > 0 {
			err = mkdir(path, true)
			if err != nil {
				return err
			}
			err = mkdir(filepath.Join(todosDir(), "_etc", postPath), true)
			if err != nil {
				return err
			}
			err = mkdir(filepath.Join(todosDir(), "_archive", postPath), true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func prepFileTaskFromPath(path string) (string, error) {
	path, err := parseFilepath(path)
	if err != nil {
		return "", err
	}
	if !Lists.Exists(path) {
		return "", fmt.Errorf("%w: %s", terrors.ErrListNotInMemory, path)
	}
	return path, nil
}

func CheckFileExistence(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	if utils.FileExists(path) {
		return nil
	}
	return fmt.Errorf("%w: '%s'", os.ErrNotExist, path)
}

func locateFiles() error {
	todoPath := todosDir()
	err := mkDirs("")
	if err != nil {
		return err
	}

	var walk func(string)
	walk = func(path string) {
		info, err := os.Lstat(path)
		if err != nil {
			return
		}
		var isDir bool
		var isValidFile bool
		if info.Mode()&os.ModeSymlink != 0 {
			targetInfo, err := os.Stat(path)
			if err != nil {
				return
			}
			if targetInfo.IsDir() {
				isDir = true
			} else if targetInfo.Mode().IsRegular() {
				isValidFile = true
			}
		}
		isDir = isDir || info.IsDir()
		isValidFile = isValidFile || info.Mode().IsRegular()
		if isDir {
			entries, err := os.ReadDir(path)
			if err != nil {
				return
			}
			for _, e := range entries {
				walk(filepath.Join(path, e.Name()))
			}
			return
		} else if isValidFile {
			Lists.Init(path)
		}
	}
	walk(todoPath)
	return nil
}

func resolveSymlinkPath(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil && os.IsNotExist(err) {
		return path, nil
	} else if err != nil {
		return "", fmt.Errorf("%w: could not lstat '%q'", err, path)
	}
	if info.Mode().IsRegular() {
		return path, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		targetPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", fmt.Errorf("%w: could not resolve symlink '%q'", err, path)
		}
		targetInfo, err := os.Stat(targetPath)
		if err != nil {
			return "", fmt.Errorf("%w: could not stat target '%q'", err, targetPath)
		}
		if targetInfo.Mode().IsRegular() {
			return targetPath, nil
		}
		return "", fmt.Errorf("symlink '%q' points to non-regular file '%q'", path, targetPath)
	}
	return "", fmt.Errorf("'%q' is neither a regular file nor a symlink to one", path)
}

func appendToDoneFile(text, path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	if err = mkDirs(filepath.Dir(path)); err != nil {
		return err
	}
	path = strings.TrimPrefix(path, filepath.Join(config.ConfigPath(), "todos/"))
	path = filepath.Join(todosDir(), "_etc", path+".done")
	tpath, err := resolveSymlinkPath(path)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(tpath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o655)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	if size > 0 {
		buf := make([]byte, 1)
		_, err = file.ReadAt(buf, size-1)
		if err != nil {
			return err
		}
		if buf[0] != '\n' {
			text = fmt.Sprintf("\n%s", text)
		}
	}
	_, err = file.Write([]byte(text))
	return err
}

func removeFromDoneFile(ids []int, path string) ([]string, error) {
	var tasks []string
	if len(ids) < 1 {
		return tasks, fmt.Errorf("%w: empty array ids", terrors.ErrValue)
	}
	path, err := parseFilepath(path)
	if err != nil {
		return tasks, err
	}
	if err = mkDirs(filepath.Dir(path)); err != nil {
		return tasks, err
	}
	path = strings.TrimPrefix(path, filepath.Join(config.ConfigPath(), "todos/"))
	path = filepath.Join(todosDir(), "_etc", path+".done")
	tpath, err := resolveSymlinkPath(path)
	if err != nil {
		return tasks, err
	}
	data, err := os.ReadFile(tpath)
	if err != nil {
		return tasks, err
	}
	lines := strings.Split(string(data), "\n")
	sort.Sort(sort.Reverse(sort.IntSlice(ids)))
	for _, id := range ids {
		if len(lines)-1 < id {
			return tasks, fmt.Errorf("%w: id '%d' exceeds number of lines in done file %s", terrors.ErrValue, id, path)
		}
		text := lines[id]
		lines = slices.Delete(lines, id, id+1)
		if validateEmptyText(text) == nil {
			tasks = append(tasks, text)
		}
	}
	err = os.WriteFile(tpath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return tasks, err
	}
	return tasks, nil
}

func CreateFile(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	if utils.FileExists(path) {
		return nil
	}
	if err = mkDirs(filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		return err
	}
	locateFiles()
	return nil
}

func LoadFile(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	err = locateFiles()
	if err != nil {
		return err
	}
	if !Lists.Exists(path) || !utils.FileExists(path) {
		return os.ErrNotExist
	}
	fileTasks, err := ParseTasks(path) // TODO: support symlinks
	if err != nil {
		return err
	}
	Lists[path].Tasks = fileTasks
	CleanupRelations(path)
	return nil
}

func ReloadFiles() error {
	for path := range Lists {
		if err := LoadFile(path); err != nil {
			return err
		}
	}
	return nil
}

func LoadOrCreateFile(path string) error {
	err := LoadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err != nil && os.IsNotExist(err) {
		err = CreateFile(path)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO: centralize the usages of filepath.Join... e.g. bakdir -> backupFile(path) (string, error)

func BackupFile(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	bakPath := strings.TrimPrefix(path, todosDir()+"/")
	bakPath = filepath.Join(etcDir(), bakPath+".bak")
	data, err := os.ReadFile(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(bakPath, data, 0644)
}

func StoreFile(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	fileTasks, ok := Lists.Tasks(path)
	if !ok {
		return fmt.Errorf("%w: '%s'", terrors.ErrListNotInMemory, path)
	}
	var lines []string
	for _, task := range fileTasks {
		lines = append(lines, task.Raw())
	}
	if err = mkDirs(filepath.Dir(path)); err != nil {
		return err
	}
	tpath, err := resolveSymlinkPath(path)
	if err != nil {
		return err
	}
	err = BackupFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tpath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}
	return nil
}

func StoreFiles() error {
	for path := range Lists {
		if err := StoreFile(path); err != nil {
			return err
		}
	}
	return nil
}

func taskifyRandomFile(path string) ([]Task, error) {
	path, err := resolveSymlinkPath(path)
	if err != nil {
		return []Task{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []Task{}, err
	}
	var tasks []Task
	for line := range strings.SplitSeq(string(data), "\n") {
		task, err := ParseTask(nil, line)
		if err != nil {
			continue
		}
		tasks = append(tasks, *task)
	}
	return tasks, nil
}

func LsFiles() ([]string, error) {
	var out []string
	if err := mkDirs(""); err != nil {
		return out, err
	}
	rootDir := filepath.Join(todosDir())
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return out, err
	}

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), "_") {
			return fs.SkipDir
		}
		if !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
