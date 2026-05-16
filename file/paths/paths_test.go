package paths

import (
	"dotxt/config"
	"dotxt/terrors"
	"dotxt/utils/testils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testils.EnsureTestDir()
	path := "/tmp/dotxt-testing/file-paths"
	if err := os.RemoveAll(path); err != nil {
		panic(err)
	}
	config.InitViper(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestDirStructure(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	assert.Equal(filepath.Join(config.ConfigPath(), "texts"), TextsDir())
	assert.Equal(filepath.Join(config.ConfigPath(), "done"), DoneDir())
	assert.Equal(filepath.Join(config.ConfigPath(), "backups"), BackupDir())
	assert.Equal(filepath.Join(config.ConfigPath(), "archive"), ArchiveDir())
}

func TestPath(t *testing.T) { // TODO
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	t.Run("NewPath", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			p, err := NewPath("")
			require.NoError(t, err)
			assert.Contains(p.Path, TextsDir())
			assert.Equal(DefaultFilename, p.Name)
		})
		t.Run("normal", func(t *testing.T) {
			for _, str := range []string{
				"dir/dir2/file", "file", filepath.Join(TextsDir(), "file"),
			} {
				p, err := NewPath(str)
				require.NoError(t, err)
				assert.Equal(strings.Replace(str, TextsDir()+"/", "", 1), p.Name)
				path, err := ParseFilepath(str)
				require.NoError(t, err)
				assert.Equal(path, p.Path)
			}
		})
	})
	t.Run("paths", func(t *testing.T) {
		name := "dir/dir2/file"
		p, err := NewPath(name)
		require.NoError(t, err)
		assert.Equal(name, p.Name)
		assert.Equal(filepath.Join(TextsDir(), name), p.Path)
		assert.Equal(filepath.Join(DoneDir(), name+".done"), p.Done())
		assert.Equal(filepath.Join(BackupDir(), name+".bak"), p.Backup())
		assert.Equal(filepath.Join(BackupDir(), name+".done.bak"), p.DoneBackup())
	})
}

func TestParseFilepath(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	helper := func(path string) string {
		out, err := ParseFilepath(path)
		assert.NoError(err)
		return out
	}
	t.Run("empty", func(t *testing.T) {
		assert.Equal(filepath.Join(TextsDir(), DefaultFilename), helper(""))
	})
	t.Run("absolute", func(t *testing.T) {
		assert.Equal(filepath.Join(TextsDir(), "file"), helper(filepath.Join(TextsDir(), "file")))
	})
	t.Run("error not under confpath/texts", func(t *testing.T) {
		_, err := ParseFilepath("/tmp/file")
		assert.Error(err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "filepath not under '/"+FileDirName+"'")
	})
	t.Run("basename", func(t *testing.T) {
		assert.Equal(filepath.Join(TextsDir(), "file"), helper("file"))
	})
	t.Run("nested basename", func(t *testing.T) {
		assert.Equal(filepath.Join(TextsDir(), "dir/file"), helper("dir/file"))
	})
	t.Run("error not ending in /", func(t *testing.T) {
		_, err := ParseFilepath("/tom/file/")
		assert.Error(err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "path cannot end in a /")
	})
	t.Run("home ~", func(t *testing.T) {
		require.NoError(t, err)
		assert.Equal(filepath.Join(TextsDir(), "dir/dir2/file"), helper("~/../.."+TextsDir()+"/dir/dir2/file"))
	})
}

func TestParseDirpath(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	helper := func(path string) string {
		out, err := ParseDirpath(path)
		assert.NoError(err)
		return out
	}
	t.Run("empty", func(t *testing.T) {
		assert.Equal(TextsDir(), helper(""))
	})
	t.Run("absolute", func(t *testing.T) {
		assert.Equal(filepath.Join(TextsDir(), "file"), helper(filepath.Join(TextsDir(), "file")))
	})
	t.Run("can end in /", func(t *testing.T) {
		assert.Equal(filepath.Join(TextsDir(), "file"), helper(filepath.Join(TextsDir(), "file")+"/"))
	})
	t.Run("error not under confpath/todos", func(t *testing.T) {
		_, err := ParseDirpath("/tmp/file")
		assert.Error(err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "dirpath not under '/"+FileDirName+"'")
	})
	t.Run("basename", func(t *testing.T) {
		assert.Equal(filepath.Join(TextsDir(), "file"), helper("file"))
	})
	t.Run("home ~", func(t *testing.T) {
		require.NoError(t, err)
		assert.Equal(filepath.Join(TextsDir(), "dir/dir2/dir3"), helper("~/../.."+TextsDir()+"/dir/dir2/dir3"))
	})
}

func TestDiagnosePathLoc(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	t.Run("in texts", func(t *testing.T) {
		addr := filepath.Join(TextsDir(), "file")
		assert.Equal(InTexts, DiagnosePathLoc(addr))
		assert.Equal(InTexts, DiagnosePathLoc(TextsDir()))
	})
	t.Run("in done", func(t *testing.T) {
		addr := filepath.Join(DoneDir(), "file")
		assert.Equal(InDone, DiagnosePathLoc(addr))
		assert.Equal(InDone, DiagnosePathLoc(DoneDir()))
	})
	t.Run("in backup", func(t *testing.T) {
		addr := filepath.Join(BackupDir(), "file")
		assert.Equal(InBackups, DiagnosePathLoc(addr))
		assert.Equal(InBackups, DiagnosePathLoc(BackupDir()))
	})
	t.Run("in archive", func(t *testing.T) {
		addr := filepath.Join(ArchiveDir(), "file")
		assert.Equal(InArchives, DiagnosePathLoc(addr))
		assert.Equal(InArchives, DiagnosePathLoc(ArchiveDir()))
	})
	t.Run("unknown", func(t *testing.T) {
		assert.Equal(UnknownLoc, DiagnosePathLoc(""))
		assert.Equal(UnknownLoc, DiagnosePathLoc("."))
		assert.Equal(UnknownLoc, DiagnosePathLoc(".."))
		assert.Equal(UnknownLoc, DiagnosePathLoc("/home"))
		assert.Equal(UnknownLoc, DiagnosePathLoc(config.ConfigPath()))
		assert.Equal(UnknownLoc, DiagnosePathLoc(filepath.Dir(config.ConfigPath())))
	})
}
