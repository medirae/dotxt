package file

import (
	"dotxt/config"
	"dotxt/file/paths"
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
	path := "/tmp/dotxt-testing/file"
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

func TestMkDir(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("single directory", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "single")
		require.NoError(t, MkDir(path, false))
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(info.IsDir())
	})
	t.Run("nested directories with all=true", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "a", "b", "c")
		require.NoError(t, MkDir(path, true))
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(info.IsDir())
	})
	t.Run("existing directory should not error", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "exists")
		require.NoError(t, os.Mkdir(path, 0755))
		assert.NoError(MkDir(path, false))
	})
}

func TestMkStructure(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	assert.DirExists(tmpDir)
	assert.NoDirExists(paths.TextsDir())
	assert.NoDirExists(paths.DoneDir())
	assert.NoDirExists(paths.BackupDir())
	assert.NoDirExists(paths.ArchiveDir())
	err = MkStructure()
	require.NoError(t, err)
	assert.DirExists(paths.TextsDir())
	assert.DirExists(paths.DoneDir())
	assert.DirExists(paths.BackupDir())
	assert.DirExists(paths.ArchiveDir())
}

func TestWalk(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	list := make(map[string]bool)
	mkf := func(name string) {
		p := filepath.Join(paths.TextsDir(), name)
		list[p] = false
		require.NoError(t, os.WriteFile(p, []byte(""), 0644))
	}
	mkd := func(name string) {
		p := filepath.Join(paths.TextsDir(), name)
		list[p] = false
		require.NoError(t, MkDir(p, true))
	}
	mks := func(old, new string) {
		old = filepath.Join(paths.TextsDir(), old)
		new = filepath.Join(paths.TextsDir(), new)
		os.Symlink(old, new)
	}
	mkf("f1")
	mkf("f2")
	mkd("d1")
	mkf("d1/f3")
	mkd("d2/d3/d4")
	mkf("d2/d3/d4/f4")
	mkf("d2/d3/f5")
	mks("f1", "s1")
	mks("f2", "s2")
	mks("d2/d3/f5", "s3")
	Walk(paths.TextsDir(), func(s string, fi os.FileInfo, err error) {
		assert.NoError(err, s)
		list[s] = true
	})
	for k, v := range list {
		assert.True(v, k)
	}
	assert.True(list[paths.TextsDir()], paths.TextsDir())
}

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("non-existent", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "non-existing")
		assert.NoFileExists(path)
		require.NoError(t, Create(path))
		assert.FileExists(path)
	})
	t.Run("existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "existing")
		fd, err := os.Create(path)
		require.NoError(t, err)
		fd.Close()
		assert.FileExists(path)
		require.NoError(t, Create(path))
		assert.FileExists(path)
	})
}

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "existing")
		fd, err := os.Create(path)
		require.NoError(t, err)
		fd.Close()
		assert.FileExists(path)
		require.NoError(t, Delete(path))
		assert.NoFileExists(path)
	})
	t.Run("non-existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "non-existing")
		assert.NoFileExists(path)
		require.NoError(t, Delete(path))
		assert.NoFileExists(path)
	})
}

func TestRead(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	path := filepath.Join(paths.TextsDir(), "file")
	require.NoError(t, os.WriteFile(path, []byte("some\ndata"), 0644))
	data, err := Read(path)
	require.NoError(t, err)
	assert.Equal([]string{"some", "data"}, data)
}

func TestWrite(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("non-existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "non-existing")
		assert.NoFileExists(path)
		require.NoError(t, Write(path, "some\ndata"))
		assert.FileExists(path)
		data, err := Read(path)
		require.NoError(t, err)
		assert.Equal([]string{"some", "data"}, data)
	})
	t.Run("existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "existing")
		require.NoError(t, os.WriteFile(path, []byte("weird\ndata\nstored\nhere"), 0644))
		assert.FileExists(path)
		data, err := Read(path)
		require.NoError(t, err)
		assert.Equal([]string{"weird", "data", "stored", "here"}, data)

		require.NoError(t, Write(path, "some\ndata"))
		assert.FileExists(path)
		data, err = Read(path)
		require.NoError(t, err)
		assert.Equal([]string{"some", "data"}, data)
	})
}

func TestAppend(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("non-existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "non-existing")
		assert.NoFileExists(path)
		require.NoError(t, Append(path, "some\ndata"))
		assert.FileExists(path)
		data, err := Read(path)
		require.NoError(t, err)
		assert.Equal([]string{"some", "data"}, data)
	})
	t.Run("existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "existing")
		require.NoError(t, Write(path, "weird\ndata\nstored\nhere"))
		assert.FileExists(path)
		data, err := Read(path)
		require.NoError(t, err)
		assert.Equal([]string{"weird", "data", "stored", "here"}, data)

		require.NoError(t, Append(path, "some\ndata"))
		assert.FileExists(path)
		data, err = Read(path)
		require.NoError(t, err)
		assert.Equal([]string{"weird", "data", "stored", "here", "some", "data"}, data)
	})
}

func TestRenamePath(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("happy", func(t *testing.T) {
		path, err := paths.NewPath("happy")
		require.NoError(t, err)
		require.NoError(t, Write(path.Path, "text"))
		assert.FileExists(path.Path)
		require.NoError(t, Write(path.Done(), "done"))
		assert.FileExists(path.Done())
		require.NoError(t, Write(path.Backup(), "texting"))
		assert.FileExists(path.Backup())
		require.NoError(t, Write(path.DoneBackup(), "texting"))
		assert.FileExists(path.Backup())

		new, err := paths.NewPath("happy-new")
		require.NoError(t, err)
		require.NoError(t, RenamePath(path, new))
		assert.NoFileExists(path.Path)
		assert.NoFileExists(path.Done())
		assert.NoFileExists(path.Backup())
		assert.NoFileExists(path.DoneBackup())
		assert.FileExists(new.Path)
		assert.FileExists(new.Done())
		assert.FileExists(new.Backup())
		assert.FileExists(new.DoneBackup())
	})
	t.Run("partial existence", func(t *testing.T) {
		path, err := paths.NewPath("partial")
		require.NoError(t, err)
		require.NoError(t, Write(path.Path, "text"))
		assert.FileExists(path.Path)
		require.NoError(t, Write(path.Done(), "done"))
		assert.FileExists(path.Done())
		assert.NoFileExists(path.Backup())

		new, err := paths.NewPath("partial-new")
		require.NoError(t, err)
		require.NoError(t, RenamePath(path, new))
		assert.NoFileExists(path.Path)
		assert.NoFileExists(path.Done())
		assert.NoFileExists(path.Backup())
		assert.FileExists(new.Path)
		assert.FileExists(new.Done())
		assert.NoFileExists(new.Backup())
	})
	t.Run("shadowing error", func(t *testing.T) {
		path, err := paths.NewPath("shadow")
		require.NoError(t, err)
		require.NoError(t, Write(path.Path, "text"))
		assert.FileExists(path.Path)
		require.NoError(t, Write(path.Done(), "done"))
		assert.FileExists(path.Done())
		require.NoError(t, Write(path.Backup(), "testing"))
		assert.FileExists(path.Backup())

		new, err := paths.NewPath("shadow-new")
		require.NoError(t, err)
		require.NoError(t, Write(new.Backup(), "old"))
		assert.FileExists(path.Backup())
		err = RenamePath(path, new)
		require.Error(t, err)
		assert.ErrorContains(err, "occupied")
		assert.FileExists(path.Path)
		assert.FileExists(path.Done())
		assert.FileExists(path.Backup())
		assert.NoFileExists(new.Path)
		assert.NoFileExists(new.Done())
		assert.FileExists(new.Backup())
		data, err := Read(new.Backup())
		require.NoError(t, err)
		assert.Equal([]string{"old"}, data)
	})
}

func TestCopy(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("non-existing", func(t *testing.T) {
		src := filepath.Join(paths.TextsDir(), "non-existing")
		dest := filepath.Join(paths.BackupDir(), "non-existing.bak")
		assert.NoFileExists(dest)
		require.NoError(t, Write(src, "some\ncontent"))
		assert.FileExists(src)
		require.NoError(t, Copy(src, dest))
		assert.FileExists(dest)
		data, err := Read(dest)
		require.NoError(t, err)
		assert.Equal([]string{"some", "content"}, data)
		sfi, err := os.Stat(src)
		require.NoError(t, err)
		dfi, err := os.Stat(dest)
		require.NoError(t, err)
		assert.Equal(sfi.Mode(), dfi.Mode())
	})
	t.Run("existing", func(t *testing.T) {
		src := filepath.Join(paths.TextsDir(), "non-existing")
		dest := filepath.Join(paths.BackupDir(), "non-existing.bak")
		require.NoError(t, Write(src, "weird"))
		assert.FileExists(dest)
		require.NoError(t, Write(src, "some\ncontent"))
		assert.FileExists(src)
		require.NoError(t, Copy(src, dest))
		assert.FileExists(dest)
		data, err := Read(dest)
		require.NoError(t, err)
		assert.Equal([]string{"some", "content"}, data)
		sfi, err := os.Stat(src)
		require.NoError(t, err)
		dfi, err := os.Stat(dest)
		require.NoError(t, err)
		assert.Equal(sfi.Mode(), dfi.Mode())
	})
}

func TestBackup(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("both", func(t *testing.T) {
		path, err := paths.NewPath("both")
		require.NoError(t, err)
		require.NoError(t, Write(path.Path, "path"))
		assert.FileExists(path.Path)
		require.NoError(t, Write(path.Done(), "done"))
		assert.FileExists(path.Done())

		require.NoError(t, Backup(path))
		assert.FileExists(path.Backup())
		data, err := Read(path.Backup())
		require.NoError(t, err)
		assert.Equal([]string{"path"}, data)
		assert.FileExists(path.DoneBackup())
		data, err = Read(path.DoneBackup())
		require.NoError(t, err)
		assert.Equal([]string{"done"}, data)
	})
	t.Run("neither", func(t *testing.T) {
		path, err := paths.NewPath("neither")
		require.NoError(t, err)
		assert.NoFileExists(path.Path)
		assert.NoFileExists(path.Done())

		require.NoError(t, Backup(path))
		assert.NoFileExists(path.Path)
		assert.NoFileExists(path.Done())
	})
	t.Run("only-one", func(t *testing.T) {
		path, err := paths.NewPath("first")
		require.NoError(t, err)
		require.NoError(t, Write(path.Path, "path"))
		assert.FileExists(path.Path)
		require.NoError(t, Backup(path))
		assert.FileExists(path.Backup())
		data, err := Read(path.Backup())
		require.NoError(t, err)
		assert.Equal([]string{"path"}, data)

		path, err = paths.NewPath("second")
		require.NoError(t, err)
		require.NoError(t, Write(path.Done(), "done"))
		assert.FileExists(path.Done())
		require.NoError(t, Backup(path))
		assert.FileExists(path.DoneBackup())
		data, err = Read(path.DoneBackup())
		require.NoError(t, err)
		assert.Equal([]string{"done"}, data)
	})
}

func TestArchive(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	path, err := paths.NewPath("file")
	require.NoError(t, err)
	require.NoError(t, Write(path.Path, "file"))
	assert.FileExists(path.Path)
	require.NoError(t, Write(path.Done(), "file.done"))
	assert.FileExists(path.Done())
	require.NoError(t, Write(path.Backup(), "file.bak"))
	assert.FileExists(path.Backup())
	require.NoError(t, Write(path.DoneBackup(), "file.done.bak"))
	assert.FileExists(path.DoneBackup())

	require.NoError(t, Archive(path))
	Walk(paths.ArchiveDir(), func(s string, fi os.FileInfo, err error) {
		require.NoError(t, err)
		if fi.IsDir() {
			return
		}
		ndx := strings.LastIndex(s, "file")
		suffix := s[ndx:]
		data, err := Read(s)
		require.NoError(t, err)
		assert.Equal(suffix, data[0])
	})
}

func TestList(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, MkStructure())

	t.Run("err", func(t *testing.T) {
		_, err := List(filepath.Dir(paths.TextsDir()))
		assert.ErrorIs(err, terrors.ErrArg)
		assert.ErrorIs(err, terrors.ErrPath)
		assert.ErrorContains(err, "must be beneath paths.TextsDir()")
	})
	t.Run("ok", func(t *testing.T) {
		pm := make(map[string]bool)
		for _, each := range []string{
			"file", "file2", "dir/", "dir/dir2/",
			"dir/file3", "dir/dir2/file4",
		} {
			pm[strings.TrimSuffix(each, "/")] = true
			if strings.HasSuffix(each, "/") {
				require.NoError(t, MkDir(filepath.Join(paths.TextsDir(), each), true))
				continue
			}
			require.NoError(t, Write(filepath.Join(paths.TextsDir(), each), "data"))
		}
		Walk(paths.TextsDir(), func(s string, fi os.FileInfo, err error) {
			if s == paths.TextsDir() {
				return
			}
			assert.Contains(pm, strings.Replace(s, paths.TextsDir()+"/", "", 1))
		})
	})
}
