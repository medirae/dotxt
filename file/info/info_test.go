package info_test

import (
	"dotxt/config"
	"dotxt/file"
	"dotxt/file/info"
	"dotxt/file/paths"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	path := "/tmp/dotxt-testing/paths"
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

func TestEvalSymlink(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, file.MkStructure())

	t.Run("path is target", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "regular")
		_, err := os.Create(path)
		require.NoError(t, err)

		destAddr, info, err := info.EvalSymlink(path)
		require.NoError(t, err)
		assert.Equal(destAddr, path)
		assert.Equal(info.Name(), filepath.Base(path))
	})
	t.Run("symlink to existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "sym-to-regular")
		targetPath := filepath.Join(paths.TextsDir(), "regular")
		// assuming /regular file exists
		require.NoError(t, os.Symlink(targetPath, path))

		destAddr, info, err := info.EvalSymlink(path)
		require.NoError(t, err)
		assert.Equal(destAddr, targetPath)
		assert.Equal(info.Name(), filepath.Base(targetPath))
	})
	t.Run("symlink to non-existing", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "sym-to-nope")
		targetPath := filepath.Join(paths.TextsDir(), "nope")
		// assuming /nope file does not exists
		require.NoError(t, os.Symlink(targetPath, path))

		_, _, err := info.EvalSymlink(path)
		assert.Error(err)
		assert.ErrorContains(err, "could not Lstat")
	})
	t.Run("symlink chain", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "sym1")
		path2 := filepath.Join(paths.TextsDir(), "sym2")
		path3 := filepath.Join(paths.TextsDir(), "sym3")
		targetPath := filepath.Join(paths.TextsDir(), "regular")
		// assuming /regular file exists
		require.NoError(t, os.Symlink(targetPath, path3))
		require.NoError(t, os.Symlink(path3, path2))
		require.NoError(t, os.Symlink(path2, path))

		destAddr, info, err := info.EvalSymlink(path)
		require.NoError(t, err)
		assert.Equal(destAddr, targetPath)
		assert.Equal(info.Name(), filepath.Base(targetPath))
	})
	t.Run("symlink chain to non-existent", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "sym-non-1")
		path2 := filepath.Join(paths.TextsDir(), "sym-non-2")
		path3 := filepath.Join(paths.TextsDir(), "sym-non-3")
		targetPath := filepath.Join(paths.TextsDir(), "nope")
		// assuming /regular file exists
		require.NoError(t, os.Symlink(targetPath, path3))
		require.NoError(t, os.Symlink(path3, path2))
		require.NoError(t, os.Symlink(path2, path))

		_, _, err := info.EvalSymlink(path)
		assert.Error(err)
		assert.ErrorContains(err, "could not Lstat")
	})
	t.Run("loop", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "sym-loop-1")
		path2 := filepath.Join(paths.TextsDir(), "sym-loop-2")
		path3 := filepath.Join(paths.TextsDir(), "sym-loop-3")
		targetPath := filepath.Join(paths.TextsDir(), "sym-loop-1")
		// assuming /regular file exists
		require.NoError(t, os.Symlink(targetPath, path3))
		require.NoError(t, os.Symlink(path3, path2))
		require.NoError(t, os.Symlink(path2, path))

		_, _, err := info.EvalSymlink(path)
		assert.Error(err)
		assert.ErrorContains(err, "loop")
	})
}

func TestHasPerms(t *testing.T) {
	// tbh I don't know how to test this...
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, file.MkStructure())

	tests := []struct {
		name  string
		mode  os.FileMode
		read  bool
		write bool
	}{
		{"owner_rw", 0600, true, true},
		{"owner_r_only", 0400, true, false},
		{"owner_w_only", 0200, false, true},
		{"group_r", 0040, true, false}, // because we are owner, group perms irrelevant unless owner denies
		{"other_r", 0004, true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(paths.TextsDir(), tc.name)
			err := os.WriteFile(path, []byte(""), tc.mode)
			require.NoError(t, err, tc.name)

			r, w, _, err := info.HasPerms(path, nil)
			require.NoError(t, err, tc.name)
			assert.Equal(tc.read, r, tc.name)
			assert.Equal(tc.write, w, tc.name)
		})
	}
}

func TestEnsureInfo(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, file.MkStructure())

	t.Run("no-addr with-path", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "1")
		require.NoError(t, os.WriteFile(path, []byte(""), 0644))
		i, err := info.EnsureInfo(nil, path, false)
		require.NoError(t, err)
		assert.Equal(path, i.Addr)
		assert.True(i.Exists)
		assert.True(i.HasRead)
		assert.True(i.HasWrite)
		assert.EqualValues(0, i.Size)
	})
	t.Run("addr no-path", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "2")
		require.NoError(t, os.WriteFile(path, []byte(""), 0644))
		i := info.NewInfo(path)
		_, err := info.EnsureInfo(i, "", false)
		require.NoError(t, err)
		assert.Equal(path, i.Addr)
		assert.True(i.Exists)
		assert.True(i.HasRead)
		assert.True(i.HasWrite)
		assert.EqualValues(0, i.Size)
	})
}

func TestNeedsRefresh(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	require.NoError(t, file.MkStructure())

	t.Run("shallow", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "file")
		i := info.NewInfo(path)
		assert.True(i.NeedsRefresh())
	})
	t.Run("recent", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "file")
		i := info.NewInfo(path)
		require.NoError(t, i.Refresh(true))
		assert.False(i.NeedsRefresh())
	})
	t.Run("unchanged", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "file")
		i := info.NewInfo(path)
		require.NoError(t, os.WriteFile(path, []byte(""), 0644))
		require.NoError(t, i.Refresh(true))
		time.Sleep(1100 * time.Millisecond)
		assert.False(i.NeedsRefresh())
	})
	t.Run("time changed", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "file")
		i := info.NewInfo(path)
		require.NoError(t, os.WriteFile(path, []byte(""), 0644))
		require.NoError(t, i.Refresh(true))
		time.Sleep(1100 * time.Millisecond)
		require.NoError(t, os.WriteFile(path, []byte(""), 0644))
		assert.True(i.NeedsRefresh())
	})
	t.Run("size changed", func(t *testing.T) {
		path := filepath.Join(paths.TextsDir(), "file")
		i := info.NewInfo(path)
		require.NoError(t, os.WriteFile(path, []byte(""), 0644))
		require.NoError(t, i.Refresh(true))
		time.Sleep(1100 * time.Millisecond)

		sinfo, err := os.Stat(path)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, []byte("a"), 0644))
		require.NoError(t, os.Chtimes(path, sinfo.ModTime(), sinfo.ModTime().Add(-1100*time.Millisecond)))

		assert.True(i.NeedsRefresh())
	})
}
