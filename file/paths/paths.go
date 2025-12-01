package paths

import (
	"dotxt/config"
	"dotxt/file/info"
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

/* file structure

[dotxt-config-dir]/
	[FileDirName]/
		todo
	[DoneDirName]/
		todo.done
	[BackupDirName]/
		todo.bak
		todo.done.bak
	[ArchiveDirName]/
		prev
		prev.done
		prev.bak
*/

const (
	DefaultFilename = "default"
	FileDirName     = "texts"
	DoneDirName     = "done"
	BackupDirName   = "backups"
	ArchiveDirName  = "archive"
)

func TextsDir() string {
	return filepath.Join(config.ConfigPath(), FileDirName)
}

func DoneDir() string {
	return filepath.Join(config.ConfigPath(), DoneDirName)
}

func BackupDir() string {
	return filepath.Join(config.ConfigPath(), BackupDirName)
}

func ArchiveDir() string {
	return filepath.Join(config.ConfigPath(), ArchiveDirName)
}

// represents the idea of a task list
type Path struct {
	Name     string
	Path     string
	Info     *info.Info
	DoneInfo *info.Info
}

func (p *Path) String() string {
	return p.Name
}
func (p *Path) Done() string {
	return filepath.Join(DoneDir(), p.Name+".done")
}
func (p *Path) Backup() string {
	return filepath.Join(BackupDir(), p.Name+".bak")
}
func (p *Path) DoneBackup() string {
	return filepath.Join(BackupDir(), p.Name+".done.bak")
}

func ParseFilepath(path string) (string, error) {
	path = norm.NFC.String(path)
	if strings.TrimSpace(path) == "" {
		return filepath.Join(TextsDir(), DefaultFilename), nil
	}
	if utils.RuneAt(path, utils.RuneCount(path)-1) == '/' {
		return "", fmt.Errorf("%w: path cannot end in a / '%s'", terrors.ErrParse, path)
	}
	if utils.RuneAt(path, 0) == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		path = strings.Replace(path, "~", homeDir, 1)
	}
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		if strings.HasPrefix(path, TextsDir()+"/") {
			return path, nil
		} else {
			return "", fmt.Errorf("%w: filepath not under '%s' '%s'", terrors.ErrParse, "/"+FileDirName, path)
		}
	}
	if tmpPath := filepath.Join(TextsDir(), path); filepath.IsAbs(tmpPath) {
		return tmpPath, nil
	}
	return "", fmt.Errorf("%w: filepath '%s'", terrors.ErrParse, path)
}

func NewPath(path string) (*Path, error) {
	path, err := ParseFilepath(path)
	if err != nil {
		return nil, err
	}
	out := Path{Path: path}
	out.Name = strings.Replace(path, TextsDir()+"/", "", 1)
	return &out, nil
}

func ParseDirpath(path string) (string, error) {
	path = norm.NFC.String(path)
	if strings.TrimSpace(path) == "" {
		return TextsDir(), nil
	}
	if utils.RuneAt(path, 0) == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		path = strings.Replace(path, "~", homeDir, 1)
	}
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		if strings.HasPrefix(path, TextsDir()+"/") || path == TextsDir() {
			return path, nil
		} else {
			return "", fmt.Errorf("%w: dirpath not under '%s' '%s'", terrors.ErrParse, "/"+FileDirName, path)
		}
	}
	if tmpPath := filepath.Join(TextsDir(), path); filepath.IsAbs(tmpPath) {
		return tmpPath, nil
	}
	return "", fmt.Errorf("%w: dirpath '%s'", terrors.ErrParse, path)
}

func NormalizePath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, utils.RuneSlice(path, 1))
	}
	return path, nil
}
