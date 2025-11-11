package task

import (
	"dotxt/config"
	"dotxt/terrors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkDirs(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	assert.DirExists(tmpDir)
	assert.NoDirExists(filepath.Join(tmpDir, "todos"))
	assert.NoDirExists(filepath.Join(tmpDir, "todos", "_etc"))
	assert.NoDirExists(filepath.Join(tmpDir, "todos", "_archive"))
	err = mkDirs("")
	require.NoError(t, err)
	assert.DirExists(filepath.Join(tmpDir, "todos"))
	assert.DirExists(filepath.Join(tmpDir, "todos", "_etc"))
	assert.DirExists(filepath.Join(tmpDir, "todos", "_archive"))
	err = mkDirs("nestedDir/nested-er-Dir/")
	require.NoError(t, err)
	assert.DirExists(filepath.Join(tmpDir, "todos", "nestedDir", "nested-er-Dir"))
}

func TestParseFilepath(t *testing.T) {
	assert := assert.New(t)
	helper := func(path string) string {
		out, err := parseFilepath(path)
		assert.NoError(err)
		return out
	}
	t.Run("empty", func(t *testing.T) {
		assert.Equal(filepath.Join(todosDir(), "todo"), helper(""))
	})
	t.Run("absolute", func(t *testing.T) {
		assert.Equal(filepath.Join(todosDir(), "file"), helper(filepath.Join(todosDir(), "file")))
	})
	t.Run("error not under confpath/todos", func(t *testing.T) {
		_, err := parseFilepath("/tmp/file")
		assert.Error(err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "filepath not under /todos")
	})
	t.Run("basename", func(t *testing.T) {
		assert.Equal(filepath.Join(todosDir(), "file"), helper("file"))
	})
	t.Run("not ending in /", func(t *testing.T) {
		_, err := parseFilepath("/tom/file/")
		assert.Error(err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "path cannot end in a /")
	})
}

func TestParseDirpath(t *testing.T) {
	assert := assert.New(t)
	helper := func(path string) string {
		out, err := parseDirpath(path)
		assert.NoError(err)
		return out
	}
	t.Run("empty", func(t *testing.T) {
		assert.Equal(filepath.Join(todosDir(), "todo"), helper(""))
	})
	t.Run("absolute", func(t *testing.T) {
		assert.Equal(filepath.Join(todosDir(), "file"), helper(filepath.Join(todosDir(), "file")))
	})
	t.Run("can end in /", func(t *testing.T) {
		assert.Equal(filepath.Join(todosDir(), "file")+"/", helper(filepath.Join(todosDir(), "file")+"/"))
	})
	t.Run("error not under confpath/todos", func(t *testing.T) {
		_, err := parseDirpath("/tmp/file")
		assert.Error(err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "dirpath not under /todos")
	})
	t.Run("basename", func(t *testing.T) {
		assert.Equal(filepath.Join(todosDir(), "file"), helper("file"))
	})
}

func TestPrepFileTaskFromPath(t *testing.T) {
	assert := assert.New(t)
	Lists = make(lists)
	path := "file"
	_, err := prepFileTaskFromPath(path)
	assert.ErrorIs(err, terrors.ErrListNotInMemory)
	assert.ErrorContains(err, path)
	path, _ = parseFilepath(path)
	Lists.Empty(path)
	_, err = prepFileTaskFromPath(path)
	assert.NoError(err)
}

func TestLocateFiles(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "todos", "storage"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "to-be-pointed-to"), 0755)

	CreateFile(filepath.Join(tmpDir, "todos", "file1"))
	CreateFile(filepath.Join(tmpDir, "todos", "nestedDir", "nestedFile"))

	os.Symlink(filepath.Join(tmpDir, "to-be-pointed-to"), filepath.Join(tmpDir, "todos", "nestedSymDir"))
	CreateFile(filepath.Join(tmpDir, "todos", "nestedSymDir", "regular"))

	CreateFile(filepath.Join(tmpDir, "todos", "storage", "regular"))
	os.Symlink(filepath.Join(tmpDir, "todos", "storage", "regular"), filepath.Join(tmpDir, "todos", "nestedSymDir", "sym"))

	CreateFile(filepath.Join(tmpDir, "todos", "storage", "regular2"))
	os.Symlink(filepath.Join(tmpDir, "todos", "storage", "regular2"), filepath.Join(tmpDir, "todos", "sym"))

	Lists = make(lists)
	err = locateFiles()
	assert.NoError(err)

	assert.True(Lists.Exists(filepath.Join(tmpDir, "todos", "file1")))
	assert.True(Lists.Exists(filepath.Join(tmpDir, "todos", "nestedDir", "nestedFile")))
	assert.True(Lists.Exists(filepath.Join(tmpDir, "todos", "nestedSymDir", "regular")))
	assert.True(Lists.Exists(filepath.Join(tmpDir, "todos", "nestedSymDir", "sym")))
	assert.True(Lists.Exists(filepath.Join(tmpDir, "todos", "sym")))
}

func TestReolveSymlinkPath(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")

	t.Run("sym -> regular target", func(t *testing.T) {
		target := filepath.Join(tmpDir, "target")
		err = os.WriteFile(target, []byte(""), 0644)
		require.NoError(t, err)
		sym := filepath.Join(tmpDir, "sym")
		err = os.Symlink(target, sym)
		require.NoError(t, err)
		path, err := resolveSymlinkPath(sym)
		require.NoError(t, err)
		assert.Equal(path, target)
	})
	t.Run("regular file", func(t *testing.T) {
		target := filepath.Join(tmpDir, "target")
		err = os.WriteFile(target, []byte(""), 0644)
		require.NoError(t, err)
		path, err := resolveSymlinkPath(target)
		require.NoError(t, err)
		assert.Equal(path, target)
	})
	t.Run("non existing", func(t *testing.T) {
		target := filepath.Join(tmpDir, "random")
		path, err := resolveSymlinkPath(target)
		require.NoError(t, err)
		assert.Equal(path, target)
	})
	t.Run("sym -> non existing", func(t *testing.T) {
		target := filepath.Join(tmpDir, "non-existing")
		sym := filepath.Join(tmpDir, "fail-sym")
		err = os.Symlink(target, sym)
		require.NoError(t, err)
		_, err := resolveSymlinkPath(sym)
		require.Error(t, err)
		assert.ErrorContains(err, "could not resolve")
	})
	t.Run("sym -> dir", func(t *testing.T) {
		target := filepath.Join(tmpDir, "dir")
		err = os.MkdirAll(target, 0755)
		require.NoError(t, err)
		sym := filepath.Join(tmpDir, "sym-dir")
		err = os.Symlink(target, sym)
		require.NoError(t, err)
		_, err := resolveSymlinkPath(sym)
		require.Error(t, err)
		assert.ErrorContains(err, "non-regular")
	})
}

func TestAppendToDoneFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")

	t.Run("append to non-existing", func(t *testing.T) {
		randName := filepath.Base(tmpDir)
		path := filepath.Join(todosDir(), "_etc", randName+".done")
		assert.NoFileExists(path)
		err := appendToDoneFile("text", randName)
		require.NoError(t, err)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("text", string(rawData))
	})
	t.Run("append to empty", func(t *testing.T) {
		name := "file"
		path := filepath.Join(todosDir(), "_etc", name+".done")
		os.WriteFile(path, []byte(""), 0o655)
		assert.FileExists(path)
		appendToDoneFile("text", name)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("text", string(rawData))
	})
	t.Run("append to file ending with \\n", func(t *testing.T) {
		name := "file2"
		path := filepath.Join(todosDir(), "_etc", name+".done")
		os.WriteFile(path, []byte("1\n2\n"), 0o655)
		assert.FileExists(path)
		appendToDoneFile("text", name)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("1\n2\ntext", string(rawData))
	})
	t.Run("append to file not ending with \\n", func(t *testing.T) {
		name := "file3"
		path := filepath.Join(todosDir(), "_etc", name+".done")
		os.WriteFile(path, []byte("1\n2"), 0o655)
		assert.FileExists(path)
		appendToDoneFile("text", name)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("1\n2\ntext", string(rawData))
	})
	t.Run("append to file which is nested", func(t *testing.T) {
		name := "nested/file4"
		path := filepath.Join(todosDir(), "_etc", name+".done")
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("1"), 0o655)
		assert.FileExists(path)
		appendToDoneFile("text", name)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("1\ntext", string(rawData))
	})
}

func TestRemoveFromDoneFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")
	name := "file"
	path := filepath.Join(todosDir(), "_etc", name+".done")
	os.WriteFile(path, []byte(""), 0o655)

	t.Run("validate ids", func(t *testing.T) {
		_, err := removeFromDoneFile([]int{}, "file")
		require.NotNil(t, err)
		assert.ErrorIs(err, terrors.ErrValue)
		assert.ErrorContains(err, "empty array ids")
	})
	t.Run("non-existing id", func(t *testing.T) {
		_, err := removeFromDoneFile([]int{5}, name)
		require.NotNil(t, err)
		assert.ErrorIs(err, terrors.ErrValue)
		assert.ErrorContains(err, "id '5' exceeds number of lines in done file")
	})
	t.Run("normal", func(t *testing.T) {
		os.WriteFile(path, []byte("a\nb\nc\n \nd\n \ne\n \nf"), 0o655)
		tasks, _ := removeFromDoneFile([]int{0}, name)
		assert.Equal("a", tasks[0])
		tasks, _ = removeFromDoneFile([]int{1, 3, 5}, name)
		assert.Equal([]string{"e", "d", "c"}, tasks)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("b\n \n \n \nf", string(rawData))
	})
	t.Run("nested", func(t *testing.T) {
		name := "nested/file"
		mkDirs(filepath.Dir(name))
		path := filepath.Join(todosDir(), "_etc", name+".done")
		os.WriteFile(path, []byte("a\nb\nc\n \nd\n \ne\n \nf"), 0o655)
		tasks, err := removeFromDoneFile([]int{0}, name)
		require.NoError(t, err)
		assert.Equal("a", tasks[0])
		tasks, _ = removeFromDoneFile([]int{1, 3, 5}, name)
		assert.Equal([]string{"e", "d", "c"}, tasks)
		rawData, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("b\n \n \n \nf", string(rawData))
	})
}

func TestCreateFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")
	name := "file"
	path := filepath.Join(todosDir(), name)
	assert.NoFileExists(path)
	err = CreateFile(name)
	require.Nil(t, err)
	assert.FileExists(path)
	raw, err := os.ReadFile(path)
	require.Nil(t, err)
	assert.Equal("", string(raw))
	assert.True(Lists.Exists(path))

	t.Run("nested", func(t *testing.T) {
		name := "nested/file"
		path := filepath.Join(todosDir(), name)
		assert.NoFileExists(path)
		err = CreateFile(name)
		require.Nil(t, err)
		assert.FileExists(path)
		raw, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("", string(raw))
		assert.True(Lists.Exists(path))
	})
}

func TestLoadFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")
	name := "file"
	path := filepath.Join(todosDir(), name)

	t.Run("non-existing", func(t *testing.T) {
		err := LoadFile("random")
		assert.Error(err)
		assert.ErrorIs(err, os.ErrNotExist)
	})
	t.Run("empty file", func(t *testing.T) {
		os.WriteFile(path, []byte(""), 0o655)
		err := LoadFile(name)
		require.Nil(t, err)
		assert.True(Lists.Exists(path))
		tasks, ok := Lists.Tasks(path)
		assert.True(ok)
		assert.Empty(tasks)
	})
	t.Run("full file", func(t *testing.T) {
		os.WriteFile(path, []byte("1\n2\n3"), 0o655)
		err := LoadFile(name)
		require.Nil(t, err)
		assert.True(Lists.Exists(path))
		tasks, ok := Lists.Tasks(path)
		assert.True(ok)
		assert.Len(tasks, 3)
	})
	t.Run("nested", func(t *testing.T) {
		mkDirs("nested/")
		name := "nested/file"
		path = filepath.Join(todosDir(), name)
		os.WriteFile(path, []byte("1\n2\n3"), 0o655)
		err := LoadFile(name)
		require.Nil(t, err)
		assert.True(Lists.Exists(path))
		tasks, ok := Lists.Tasks(path)
		assert.True(ok)
		assert.Len(tasks, 3)
	})
}

func TestStoreFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")
	name := "file"
	path := filepath.Join(todosDir(), name)

	t.Run("empty tasks", func(t *testing.T) {
		os.WriteFile(path, []byte("1\n2\n"), 0o655)
		LoadFile(name)
		tasks := Lists[path].Tasks
		assert.Len(tasks, 2)
		Lists.Empty(path)
		err = StoreFile(path)
		require.Nil(t, err)
		raw, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal("", string(raw))
	})
	t.Run("full file", func(t *testing.T) {
		os.WriteFile(path+"1", []byte("1\n2\n3\n"), 0o655)
		os.WriteFile(path, []byte(""), 0o655)
		LoadFile(name + "1")
		LoadFile(name)
		assert.True(Lists.Exists(path))
		Lists[path].Tasks = Lists[path+"1"].Tasks
		err := StoreFile(path)
		require.Nil(t, err)
		raw, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal(3, len(strings.Split(string(raw), "\n")))
	})
	t.Run("nested", func(t *testing.T) {
		mkDirs("nested/")
		name := "nested/file"
		path := filepath.Join(todosDir(), name)
		os.WriteFile(path+"1", []byte("1\n2\n3\n"), 0o655)
		os.WriteFile(path, []byte(""), 0o655)
		LoadFile(name + "1")
		LoadFile(name)
		assert.True(Lists.Exists(path))
		Lists[path].Tasks = Lists[path+"1"].Tasks
		err := StoreFile(path)
		require.Nil(t, err)
		raw, err := os.ReadFile(path)
		require.Nil(t, err)
		assert.Equal(3, len(strings.Split(string(raw), "\n")))
	})
}

func TestBackupFile(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")
	name := "bak"
	path := filepath.Join(todosDir(), name)

	t.Run("non-existing file", func(t *testing.T) {
		err := BackupFile(path)
		require.NoError(t, err)
		bakPath := filepath.Join(etcDir(), strings.TrimPrefix(path, todosDir()+"/")+".bak")
		require.NoFileExists(t, bakPath)
	})
	t.Run("file backed up", func(t *testing.T) {
		name := "bakFile"
		path := filepath.Join(todosDir(), name)
		err := os.WriteFile(path, []byte("1\n2\n3"), 0644)
		require.NoError(t, err)
		err = LoadFile(path)
		require.NoError(t, err)
		assert.Len(Lists[path].Tasks, 3)
		err = BackupFile(path)
		require.NoError(t, err)
		bakPath := filepath.Join(etcDir(), strings.TrimPrefix(path, todosDir()+"/")+".bak")
		require.FileExists(t, bakPath)
		data, err := os.ReadFile(bakPath)
		require.NoError(t, err)
		assert.Equal("1\n2\n3", string(data))
	})
}
