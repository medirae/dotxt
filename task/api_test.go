package task

import (
	"dotxt/config"
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupIDs(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("test")
	Lists.Empty(path)
	for ndx := range 10 {
		task, _ := ParseTask(utils.MkPtr(ndx), fmt.Sprintf("%d", ndx))
		Lists.Append(path, task)
	}
	t.Run("nil", func(t *testing.T) {
		for _, ndx := range []int{0, 2, 4, 6, 7, 9} {
			Lists[path].Tasks[ndx].ID = nil
		}
		CleanupIDs(path)
		for ndx := range 10 {
			task := Lists[path].Tasks[ndx]
			if assert.NotNil(task.ID) {
				assert.Equal(ndx, *task.ID)
			}
			assert.Equal(fmt.Sprintf("%d", ndx), task.Norm())
		}
	})
	t.Run("out of range", func(t *testing.T) {
		for ndx := range []int{0, 1, 4, 6, 7, 9} {
			sign := 1
			if ndx%2 == 0 {
				sign = -1
			}
			Lists[path].Tasks[ndx].ID = utils.MkPtr(sign * 10 * ndx)
		}
		CleanupIDs(path)
		for ndx := range 10 {
			task := Lists[path].Tasks[ndx]
			if assert.NotNil(task.ID) {
				assert.Equal(ndx, *task.ID)
			}
			assert.Equal(fmt.Sprintf("%d", ndx), task.Norm())
		}
	})
	t.Run("duplicates", func(t *testing.T) {
		for ndx := range []int{1, 4, 6, 7, 9} {
			Lists[path].Tasks[ndx].ID = utils.MkPtr(0)
		}
		CleanupIDs(path)
		for ndx := range 10 {
			task := Lists[path].Tasks[ndx]
			if assert.NotNil(task.ID) {
				assert.Equal(ndx, *task.ID)
			}
			assert.Equal(fmt.Sprintf("%d", ndx), task.Norm())
		}
	})
	Lists.Delete(path)
}

func TestAddTask(t *testing.T) {
	assert := assert.New(t)

	mkId := func(a int) *int {
		return &a
	}
	path, _ := parseFilepath("file")
	task, _ := ParseTask(mkId(1), "1")
	Lists.Empty(path, task)
	task, _ = ParseTask(nil, "nil")
	err := AddTask(task, path)
	require.Nil(t, err)
	assert.Equal(2, Lists.Len(path))
	task1 := Lists[path].Tasks[0]
	assert.Equal(0, *task1.ID)
	assert.Equal("nil", task1.Norm())
	task2 := Lists[path].Tasks[1]
	assert.Equal(1, *task2.ID)
	assert.Equal("1", task2.Norm())
}

func TestAddTaskFromStr(t *testing.T) {
	assert := assert.New(t)
	mkId := func(a int) *int {
		return &a
	}
	path, _ := parseFilepath("file")
	task, _ := ParseTask(mkId(1), "1")
	Lists.Empty(path, task)
	err := AddTaskFromStr("nil", path)
	require.Nil(t, err)
	assert.Equal("nil", Lists[path].Tasks[0].NormRegular())
}

func TestGetTaskIndexFromId(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2", path)
	id, err := getTaskIndexFromId(1, path)
	require.Nil(t, err)
	assert.Equal(1, id)
	_, err = getTaskIndexFromId(5, path)
	require.NotNil(t, err)
	assert.ErrorIs(err, terrors.ErrNotFound)
	assert.ErrorContains(err, "task id '5'")
}

func TestGetTaskFromId(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2", path)
	task, err := getTaskFromId(1, path)
	require.Nil(t, err)
	assert.Equal("1", task.NormRegular())
}

func TestAppendToTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1 $c=2024-05-05T05-05", path)
	AddTaskFromStr("2", path)
	err := AppendToTask(1, "new data +prj @at #tag $due=1w", path)
	require.Nil(t, err)
	task := Lists[path].Tasks[1]
	dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
	assert.Equal(*dt, *task.Time.DueDate)
	assert.Len(task.Hints, 3)
	assert.Equal([]string{"+prj", "@at", "#tag"}, func() []string {
		var out []string
		for _, h := range task.Hints {
			out = append(out, *h)
		}
		return out
	}())
	assert.Equal("1 new data", task.NormRegular())
	if assert.NotNil(task.ID) {
		assert.Equal(1, *task.ID)
	}
}

func TestPrependToTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1 $c=2024-05-05T05-05", path)
	AddTaskFromStr("(A) 2", path)
	AddTaskFromStr("(B)  3", path)
	t.Run("1", func(t *testing.T) {
		err := PrependToTask(1, "new data +prj @at #tag $due=1w", path)
		require.Nil(t, err)
		task := Lists[path].Tasks[1]
		dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		assert.Len(task.Hints, 3)
		assert.Equal([]string{"+prj", "@at", "#tag"}, func() []string {
			var out []string
			for _, h := range task.Hints {
				out = append(out, *h)
			}
			return out
		}())
		assert.Equal("new data 1", task.NormRegular())
		if assert.NotNil(task.ID) {
			assert.Equal(1, *task.ID)
		}
	})
	t.Run("2", func(t *testing.T) {
		err := PrependToTask(2, "something new", path)
		require.Nil(t, err)
		task := Lists[path].Tasks[2]
		assert.Equal("(A)", *task.Priority)
		assert.Equal("(A) something new 2", task.Norm())
	})
	t.Run("3", func(t *testing.T) {
		err := PrependToTask(3, "something new", path)
		require.Nil(t, err)
		task := Lists[path].Tasks[3]
		assert.Equal("(B)", *task.Priority)
		assert.Equal("(B) something new  3", task.Norm())
	})
}

func TestReplaceTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1 $c=2024-05-05T05-05", path)
	AddTaskFromStr("2", path)
	err := ReplaceTask(1, "new data +prj @at #tag $due=1w", path)
	require.Nil(t, err)
	task := Lists[path].Tasks[1]
	dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
	assert.Equal(*dt, *task.Time.DueDate)
	assert.Len(task.Hints, 3)
	assert.Equal([]string{"+prj", "@at", "#tag"}, func() []string {
		var out []string
		for _, h := range task.Hints {
			out = append(out, *h)
		}
		return out
	}())
	assert.Equal("new data", task.NormRegular())
	if assert.NotNil(task.ID) {
		assert.Equal(1, *task.ID)
	}
	assert.NotContains(task.Norm(), "1 ")
	assert.Contains(task.Raw(), "$c=2024-05-05T05-05")
}

func TestDeduplicateList(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0", path)
	AddTaskFromStr("(1) +prj @at #tag $due=1w $c=2024-05-05T05-05 $p=page/0/1287/books $every=1m", path)
	AddTaskFromStr("(1) +prj @at #tag $due=1w $c=1960-05-05T05-05 $p=page/0/1287/books $every=1m", path)
	AddTaskFromStr("3", path)
	err := DeduplicateList(path)
	require.NoError(t, err)
	assert.Equal(3, Lists.Len(path))
	dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
	assert.Equal(*dt, *Lists[path].Tasks[1].Time.CreationDate)
	assert.Equal("0", Lists[path].Tasks[0].Norm())
	assert.Equal("3", Lists[path].Tasks[2].Norm())
}

func TestDeprioritizeTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("no priority", path)
	AddTaskFromStr("(with) priority", path)
	t.Run("no priority", func(t *testing.T) {
		assert.Nil(Lists[path].Tasks[0].Priority)
		err := DeprioritizeTask(0, path)
		assert.NoError(err)
		assert.Nil(Lists[path].Tasks[0].Priority)
	})
	t.Run("priority", func(t *testing.T) {
		assert.Equal("(with)", *Lists[path].Tasks[1].Priority)
		err := DeprioritizeTask(1, path)
		assert.NoError(err)
		assert.Nil(Lists[path].Tasks[1].Priority)
		_, ndx := Lists[path].Tasks[1].Tokens.Find(TkByTypeKey(TokenPriority, "priority"))
		assert.Negative(ndx)
	})
}

func TestPrioritizeTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("no priority", path)
	AddTaskFromStr("(with) priority", path)
	t.Run("no prior priority", func(t *testing.T) {
		assert.Nil(Lists[path].Tasks[0].Priority)
		err := PrioritizeTask(0, "prio", path)
		assert.NoError(err)
		assert.Equal("(prio)", *Lists[path].Tasks[0].Priority)
		assert.Equal("(prio)", *Lists[path].Tasks[0].Tokens[0].raw)
	})
	t.Run("prior priority", func(t *testing.T) {
		assert.Equal("(with)", *Lists[path].Tasks[1].Priority)
		err := PrioritizeTask(1, "prio", path)
		assert.NoError(err)
		assert.Equal("(prio)", *Lists[path].Tasks[1].Priority)
		assert.Equal("(prio)", *Lists[path].Tasks[1].Tokens[0].raw)
	})
	t.Run("ineffective parentheses", func(t *testing.T) {
		err := PrioritizeTask(1, "(NewPrio)", path)
		assert.NoError(err)
		assert.Equal("(NewPrio)", *Lists[path].Tasks[1].Priority)
		assert.Equal("(NewPrio)", *Lists[path].Tasks[1].Tokens[0].raw)
	})
}

func TestGetIndexesFromIds(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	for ndx := range 5 {
		AddTaskFromStr(fmt.Sprintf("%d", ndx*10), path)
	}
	t.Run("happy path", func(t *testing.T) {
		ndxs, err := getIndexesFromIds([]int{2, 3, 4, 1, 0}, path)
		require.NoError(t, err)
		assert.ElementsMatch([]int{0, 1, 2, 3, 4}, ndxs)
		assert.Equal([]int{4, 3, 2, 1, 0}, ndxs)
	})
	t.Run("missing id", func(t *testing.T) {
		_, err := getIndexesFromIds([]int{9}, path)
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, "task ids '9'")
		assert.ErrorContains(err, "9")
	})
	t.Run("support duplicates", func(t *testing.T) {
		ndxs, err := getIndexesFromIds([]int{1, 1, 2, 1, 1}, path)
		require.NoError(t, err)
		assert.ElementsMatch([]int{1, 2}, ndxs)
		assert.Equal([]int{2, 1}, ndxs)
	})
	t.Run("empty", func(t *testing.T) {
		ndxs, err := getIndexesFromIds([]int{}, path)
		require.NoError(t, err)
		assert.Empty(ndxs)
	})
}

func TestDeleteTasks(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0 $P=2", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $id=2", path)
	AddTaskFromStr("3", path)
	AddTaskFromStr("4 $P=2", path)
	AddTaskFromStr("5", path)
	err := DeleteTasks([]int{5, 2}, path)
	require.NoError(t, err)
	assert.Equal(4, Lists.Len(path))
	assert.Equal(0, *Lists[path].Tasks[0].ID)
	assert.Equal("0 $P=2", Lists[path].Tasks[0].Norm())
	assert.Equal(1, *Lists[path].Tasks[1].ID)
	assert.Equal("1", Lists[path].Tasks[1].Norm())
	assert.Equal(2, *Lists[path].Tasks[2].ID)
	assert.Equal("4 $P=2", Lists[path].Tasks[2].Norm())
	assert.Equal(3, *Lists[path].Tasks[3].ID)
	assert.Equal("3", Lists[path].Tasks[3].Norm())
}

func TestDoneTask(t *testing.T) {
	assert := assert.New(t)

	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0 $P=2", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $id=2", path)
	AddTaskFromStr("3", path)
	AddTaskFromStr("4 $P=2", path)
	AddTaskFromStr("5 $focus", path)

	err = DoneTask([]int{5, 2}, path)
	require.NoError(t, err)
	assert.Equal(4, Lists.Len(path))
	assert.Equal(0, *Lists[path].Tasks[0].ID)
	assert.Equal("0 $P=2", Lists[path].Tasks[0].Norm())
	assert.Equal(1, *Lists[path].Tasks[1].ID)
	assert.Equal("1", Lists[path].Tasks[1].Norm())
	assert.Equal(2, *Lists[path].Tasks[2].ID)
	assert.Equal("4 $P=2", Lists[path].Tasks[2].Norm())
	assert.Equal(3, *Lists[path].Tasks[3].ID)
	assert.Equal("3", Lists[path].Tasks[3].Norm())
	raw, err := os.ReadFile(filepath.Join(todosDir(), "_etc", "file.done"))
	require.NoError(t, err)
	tasks := strings.Split(string(raw), "\n")
	assert.True(strings.HasPrefix(tasks[0], "5"))
	assert.NotContains(tasks[0], "$focus")
	assert.True(strings.HasPrefix(tasks[1], "2 $id=2"))
}

func TestMoveTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("src")
	Lists.Empty(path)
	pathDst, _ := parseFilepath("dst")
	Lists.Empty(pathDst)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2", path)
	AddTaskFromStr("3", pathDst)
	AddTaskFromStr("4", pathDst)
	AddTaskFromStr("5", pathDst)
	err := MoveTask(path, 1, pathDst)
	require.NoError(t, err)
	assert.Equal(2, Lists.Len(path))
	assert.Equal(1, *Lists[path].Tasks[1].ID)
	assert.Equal(4, Lists.Len(pathDst))
	assert.Equal(3, *Lists[pathDst].Tasks[3].ID)
	assert.Equal("1", Lists[pathDst].Tasks[3].Norm())
}

func TestRevertTask(t *testing.T) {
	assert := assert.New(t)

	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")

	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0 $P=2", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $id=2", path)
	AddTaskFromStr("3", path)
	AddTaskFromStr("4 $P=2", path)
	AddTaskFromStr("5", path)

	donePath := filepath.Join(todosDir(), "_etc", "file.done")
	err = os.WriteFile(donePath, []byte("1\n2\n3"), 0o655)
	require.NoError(t, err)

	err = RevertTask([]int{1}, path)
	require.NoError(t, err)
	assert.Equal(7, Lists.Len(path))
	assert.Equal(6, *Lists[path].Tasks[6].ID)
	assert.Equal("2", Lists[path].Tasks[6].Norm())
}

func TestIncrementProgressCount(t *testing.T) {
	assert := assert.New(t)

	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0 $p=unit/10/100/cat", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $p=unit/10/100/cat", path)
	AddTaskFromStr("3 $p=unit/10/100/cat", path)
	AddTaskFromStr("4 $p=unit/10/100/cat", path)
	t.Run("no progress", func(t *testing.T) {
		err := IncrementProgressCount(1, path, 2)
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrValue)
		assert.ErrorContains(err, "task with id '1' has no progress")
	})
	t.Run("positive", func(t *testing.T) {
		task := Lists[path].Tasks[0]
		err := IncrementProgressCount(0, path, 2)
		require.NoError(t, err)
		assert.Equal(12, task.Prog.Count)
		tk, ndx := task.Tokens.Find(TkByType(TokenProgress))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Equal(12, tk.Value.(*Progress).Count)
			assert.Contains(*tk.raw, "12/100")
		}
	})
	t.Run("negative", func(t *testing.T) {
		task := Lists[path].Tasks[2]
		err := IncrementProgressCount(2, path, -2)
		require.NoError(t, err)
		assert.Equal(8, task.Prog.Count)
		tk, ndx := task.Tokens.Find(TkByType(TokenProgress))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Equal(8, tk.Value.(*Progress).Count)
			assert.Contains(*tk.raw, "8/100")
		}
	})
	t.Run("exceed positive", func(t *testing.T) {
		task := Lists[path].Tasks[3]
		err := IncrementProgressCount(3, path, 200)
		require.NoError(t, err)
		assert.Equal(100, task.Prog.Count)
		tk, ndx := task.Tokens.Find(TkByType(TokenProgress))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Equal(100, tk.Value.(*Progress).Count)
			assert.Contains(*tk.raw, "100/100")
		}
	})
	t.Run("exceed negative", func(t *testing.T) {
		task := Lists[path].Tasks[4]
		err := IncrementProgressCount(4, path, -200)
		require.NoError(t, err)
		assert.Equal(0, task.Prog.Count)
		tk, ndx := task.Tokens.Find(TkByType(TokenProgress))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Equal(0, tk.Value.(*Progress).Count)
			assert.Contains(*tk.raw, "unit/0/100/cat")
		}
	})
}

func TestCheckAndRecurTasks(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	Lists.Empty(path)
	AddTaskFromStr("0 $c=2024-05-05T05-05 $due=1m $every=1y", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $c=2024-05-05T05-05 $every=1y", path)
	AddTaskFromStr("3 $due=1w $every=1y", path)
	AddTaskFromStr("4 $due=1w $dead=1m $every=1y", path)
	AddTaskFromStr("5 $due=1w $end=1m $every=1y", path)
	AddTaskFromStr("6 $c=2024-05-05T05-05 $due=1m $dead=1m $every=1y", path)
	err := CheckAndRecurTasks(path)
	require.NoError(t, err)
	t.Run("valid", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-06-04T05-05")
		for dt.Before(rightNow) {
			dt = utils.MkPtr(dt.Add(365 * 24 * 60 * 60 * time.Second))
		}
		assert.Equal(*dt, *Lists[path].Tasks[0].Time.DueDate)
		dueStr := unparseRelativeDatetime(*Lists[path].Tasks[0].Time.DueDate, *Lists[path].Tasks[0].Time.CreationDate)
		tk, ndx := Lists[path].Tasks[0].Tokens.Find(TkByTypeKey(TokenDate, "due"))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Equal(fmt.Sprintf("$due=%s", dueStr), *tk.raw)
		}
		assert.Equal("2 $every=1y $due=2y", Lists[path].Tasks[2].Norm())
	})
	t.Run("invalid", func(t *testing.T) {
		assert.Equal("1", Lists[path].Tasks[1].Norm())
		assert.Equal("3 $due=1w $every=1y", Lists[path].Tasks[3].Norm())
		assert.Equal("4 $due=1w $dead=1m $every=1y", Lists[path].Tasks[4].Norm())
		assert.Equal("5 $due=1w $end=1m $every=1y", Lists[path].Tasks[5].Norm())
	})
	t.Run("valid with dead", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-06-04T05-05")
		for dt.Before(rightNow) {
			dt = utils.MkPtr(dt.Add(365 * 24 * 60 * 60 * time.Second))
		}
		assert.Equal(*dt, *Lists[path].Tasks[6].Time.DueDate)
		dueStr := unparseRelativeDatetime(*Lists[path].Tasks[6].Time.DueDate, *Lists[path].Tasks[6].Time.CreationDate)
		tk, ndx := Lists[path].Tasks[6].Tokens.Find(TkByTypeKey(TokenDate, "due"))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Equal(fmt.Sprintf("$due=%s", dueStr), *tk.raw)
		}
		assert.Equal(dt.Add(30*24*60*60*time.Second), *Lists[path].Tasks[6].Time.Deadline)
	})
}

func TestCleanupRelations(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("test")
	Lists.Empty(path)
	for ndx, line := range []string{
		"0 id $id=first",
		"1 nested $id=fourth $P=first",
		"2 no id",
		"3 repetetive $-id=first",
		"4 id $id=second",
		"5 parent $P=first",
		"6 parent $P=second",
		"7 id+parent $id=third $P=second",
		"8 nested first $P=third",
		"9 nested second repetetive $id=fourth $P=third",
		"10 nested nested $P=fourth",
	} {
		task, _ := ParseTask(utils.MkPtr(ndx), line)
		Lists.Append(path, task)
	}
	CleanupRelations(path)
	get := func(ndx int) *Task {
		return Lists[path].Tasks[ndx]
	}
	t.Run("0 id $id=first", func(t *testing.T) {
		assert.Nil(get(0).Parent)
		if assert.NotEmpty(get(0).Children) {
			for _, child := range get(0).Children {
				assert.Equal(get(0), child.Parent)
				assert.Contains([]int{1, 5}, *child.ID)
			}
		}
	})
	t.Run("1 nested $id=fourth $P=first", func(t *testing.T) {
		if assert.NotNil(get(1).Parent) {
			assert.Equal(0, *get(1).Parent.ID)
		}
		if assert.NotEmpty(get(1).Children) {
			for _, child := range get(1).Children {
				assert.Equal(get(1), child.Parent)
				assert.Contains([]int{10}, *child.ID)
			}
		}
	})
	t.Run("2 no id", func(t *testing.T) {
		assert.Nil(get(2).EID)
		assert.Empty(get(2).Children)
		assert.Nil(get(2).PID)
		assert.Nil(get(2).Parent)
	})
	t.Run("3 repetetive $-id=first", func(t *testing.T) {
		assert.Nil(get(3).EID)
		assert.Empty(get(3).Children)
	})
	t.Run("4 id $id=second", func(t *testing.T) {
		assert.Nil(get(4).Parent)
		if assert.NotEmpty(get(4).Children) {
			for _, child := range get(4).Children {
				assert.Equal(get(4), child.Parent)
				assert.Contains([]int{6, 7}, *child.ID)
			}
		}
	})
	t.Run("5 parent $P=first", func(t *testing.T) {
		if assert.NotNil(get(5).Parent) {
			assert.Equal(0, *get(5).Parent.ID)
		}
		assert.Equal(1, get(5).Depth())
	})
	t.Run("6 parent $P=second", func(t *testing.T) {
		if assert.NotNil(get(6).Parent) {
			assert.Equal(4, *get(6).Parent.ID)
		}
	})
	t.Run("7 id+parent $id=third $P=second", func(t *testing.T) {
		if assert.NotNil(get(7).Parent) {
			assert.Equal(4, *get(7).Parent.ID)
		}
		assert.NotNil(get(7).EID)
		if assert.NotEmpty(get(7).Children) {
			for _, child := range get(7).Children {
				assert.Equal(get(7), child.Parent)
				assert.Contains([]int{8, 9}, *child.ID)
			}
		}
	})
	t.Run("8 nested first $P=third", func(t *testing.T) {
		if assert.NotNil(get(8).Parent) {
			assert.Equal(7, *get(8).Parent.ID, "pid=third")
		}
		assert.Equal(2, get(8).Depth())
	})
	t.Run("9 nested second repetetive $id=fourth $P=third", func(t *testing.T) {
		assert.Nil(get(9).EID)
		assert.Empty(get(9).Children)
		assert.NotNil(get(9).PID)
		if assert.NotNil(get(9).Parent) {
			assert.Equal(7, *get(9).Parent.ID)
		}
	})
	t.Run("10 nested nested $P=fourth", func(t *testing.T) {
		if assert.NotNil(get(10).Parent) {
			assert.Equal(1, *get(10).Parent.ID)
		}
	})
	t.Run("looping", func(t *testing.T) {
		Lists.Empty(path)
		for ndx, line := range []string{
			"0 $id=0 $P=0",

			"1 $id=1 $P=2",
			"2 $id=2 $P=1",

			"3 $id=3 $P=5",
			"4 $id=4 $P=3",
			"5 $id=5 $P=4",
		} {
			task, _ := ParseTask(utils.MkPtr(ndx), line)
			Lists.Append(path, task)
		}
		CleanupRelations(path)
		t.Run("self loop", func(t *testing.T) {
			assert.Nil(get(0).PID)
			assert.Nil(get(0).Parent)
			assert.NotNil(get(0).EID)
		})
		t.Run("two-node loop", func(t *testing.T) {
			{
				assert.NotNil(get(1).PID)
				assert.NotNil(get(1).Parent)
				assert.Equal(get(1).Parent, get(2))
			}
			{
				assert.Nil(get(2).PID)
				assert.Nil(get(2).Parent)
			}
		})
		t.Run("three-node loop", func(t *testing.T) {
			{
				assert.NotNil(get(3).PID)
				assert.NotNil(get(3).Parent)
				assert.Equal(get(3).Parent, get(5))
			}
			{
				assert.NotNil(get(4).PID)
				assert.NotNil(get(4).Parent)
				assert.Equal(get(4).Parent, get(3))
			}
			{
				assert.Nil(get(5).PID)
				assert.Nil(get(5).Parent)
			}
		})
	})
}

func TestToggleCollapsed(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("toggleC")
	Lists.Empty(path)
	AddTaskFromStr("was collapsed $-id=first", path)
	AddTaskFromStr("was expanded $id=second", path)

	err := ToggleCollapsed(0, path)
	assert.NoError(err)
	assert.False(Lists[path].Tasks[0].IsCollapsed())

	err = ToggleCollapsed(1, path)
	assert.NoError(err)
	assert.True(Lists[path].Tasks[1].IsCollapsed())
}

func TestSortList(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("sortList")
	Lists.Empty(path)
	for _, line := range []string{ // from sort_test
		"$p=unit/0/100/cat", "$p=unit/0/100",
		"$p=unit/0/100/catA", "$p=unit/0/100/catB",
		"$p=unit/0/300", "$p=unit/10/100",
		"$p=unit/20/100", "$p=unit/200/1000",
		"(A) s", "s",
		"(B) s", "(C) s",
		"+b +a", "s",
		"+b +a +z", "+z +b +0",
		"#a", "#ab", "#a #b #c",
		"@a", "@ab", "@a @b @c",
		"#a @a", "#a @b", "#b @a",
		"@a #b", "@a #b", "@b #a",
		"a +b", "+b",
		"c", "d",
		"a",
		"b $id=1",
		"z.1 $P=1",
		"a.2 $P=1",
		"c",
		"$id=0", "$id=1", "$id=2",
		"$P=0", "$P=1", "$P=2",
		"$id=0", "$id=1", "$id=2",
		"$P=0", "$P=1", "$P=2",
		"$id=-1", "$id=4", "$id=5",
		"$P=-2", "$P=-3", "$P=-4",
		"$due=1w $dead=1w", "$due=1w $dead=2w",
		"$due=1w $end=1w", "$due=1w $end=2w",
		"$due=2d", "$due=4d",
		"$c=2024-feb", "$c=2024-mar",
		"$r=20d $r=1w $r=1h", "$r=1w $r=1h", "$r=10d $r=1h", "$r=1h $r=2d",
		"$every=6m $due=3w", "$every=1y $due=3w",
	} {
		AddTaskFromStr(line, path)
	}
	err := StoreFile(path)
	require.NoError(t, err)
	tasks := slices.Clone(Lists[path].Tasks)
	Lists.Empty(path)
	err = LoadFile(path)
	require.NoError(t, err)
	err = SortList(path)
	require.NoError(t, err)
	err = StoreFile(path)
	require.NoError(t, err)
	Lists.Empty(path)
	err = LoadFile(path)
	require.NoError(t, err)
	tasks = sortTasks(tasks)
	assert.Equal(len(tasks), Lists.Len(path))
	for ndx := range tasks {
		assert.Equal(tasks[ndx].Raw(), Lists[path].Tasks[ndx].Raw())
	}
}
