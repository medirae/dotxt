package task

import (
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	assert := assert.New(t)
	checkToken := func(task *Task, key string, value *time.Time) {
		tk, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, key))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.True(strings.HasPrefix(*tk.raw, "$"+key+"="))
			assert.Equal(*value, *tk.Value.(*TokenDateValue).Value)
		}
	}
	t.Run("non-temporal vars", func(t *testing.T) {
		task, _ := ParseTask(nil, "(A) +p #t @a $p=unit/10/100/cat $due=1w $dead=1w $c=2024-05-05T05-05")
		newTask, _ := ParseTask(nil, "(B) +p #t @a $p=unit/90/100/cat $due=1w $dead=1w")
		task.update(newTask)
		assert.Equal("(B)", *task.Priority)
		assert.Equal(90, task.Prog.Count)
		dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Exactly(*dt, *task.Time.CreationDate)
		task.Tokens.ForEach(func(tk *Token) {
			if tk.Type == TokenPriority && tk.Key == "priority" {
				assert.Equal("(B)", *tk.Value.(*string))
			} else if tk.Type == TokenProgress {
				assert.Equal(90, tk.Value.(*Progress).Count)
			} else if tk.Type == TokenDate && tk.Key == "c" {
				assert.True(strings.HasPrefix(*tk.raw, "$c="))
				assert.Equal(*dt, *tk.Value.(*TokenDateValue).Value)
			}
		})
	})
	t.Run("due + old c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05")
		ntask, _ := ParseTask(nil, "$due=1w")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
	})
	t.Run("new due + old c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05")
		ntask, _ := ParseTask(nil, "$due=2w")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-19T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
	})
	t.Run("irrelavent new c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05")
		ntask, _ := ParseTask(nil, "$due=1w $c=2025-03-03T03-03")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Equal(*dt, *task.Time.CreationDate)
		checkToken(task, "c", dt)
	})
	t.Run("general", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2024-05-05T05-05")
		ntask, _ := ParseTask(nil, "$c=2024-06-05T05-05 $due=1w $dead=1w $r=-2d")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Equal(*dt, *task.Time.CreationDate)
		checkToken(task, "c", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-19T05-05")
		assert.Equal(*dt, *task.Time.Deadline)
		checkToken(task, "dead", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-10T05-05")
		assert.Equal(*dt, *task.Time.Reminders[0])
		tk, ndx := task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenDate && tk.Key[0] == 'r'
		})
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.True(strings.HasPrefix(*tk.raw, "$r="))
			assert.Equal(*dt, *tk.Value.(*TokenDateValue).Value)
		}
	})
	t.Run("retain ID", func(t *testing.T) {
		id := 2
		task, _ := ParseTask(&id, "task")
		nid := 3
		ntask, _ := ParseTask(&nid, "newtask")
		task.update(ntask)
		if assert.NotNil(task.ID) {
			assert.Equal(2, *task.ID)
		}
		ntask, _ = ParseTask(nil, "even newer")
		task.update(ntask)
		if assert.NotNil(task.ID) {
			assert.Equal(2, *task.ID)
		}
	})
}

func TestUpdateDate(t *testing.T) {
	assert := assert.New(t)
	t.Run("token not found", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m")
		dt := rightNow.Add(7 * 24 * 60 * 60 * time.Second)
		err := task.updateDate("dead", &dt)
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, "token date for field 'dead' not found")
	})
	t.Run("relative", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m")
		dt := rightNow.Add(7 * 24 * 60 * 60 * time.Second)
		err := task.updateDate("due", &dt)
		require.NoError(t, err)
		assert.Equal("$due=1w", task.Norm())
		assert.Equal(dt, *task.Time.DueDate)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if assert.NotNil(tk) {
			assert.Equal("$due=1w", *tk.raw)
			assert.Equal(dt, *tk.Value.(*TokenDateValue).Value)
		}
	})
	t.Run("relative with var", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m $dead=c:2m")
		dt := rightNow.Add(3 * 30 * 24 * 60 * 60 * time.Second)
		err := task.updateDate("dead", &dt)
		require.NoError(t, err)
		assert.Equal("$due=1m $dead=c:3m", task.Norm())
		assert.Equal(dt, *task.Time.Deadline)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "dead"))
		if assert.NotNil(tk) {
			assert.Equal("$dead=c:3m", *tk.raw)
			assert.Equal(dt, *tk.Value.(*TokenDateValue).Value)
		}
	})
	t.Run("absolute", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2025-05-05T05-05 $due=2025-06-05T05-05")
		dt, _ := parseAbsoluteDatetime("2025-07-05T05-05")
		err := task.updateDate("due", dt)
		require.NoError(t, err)
		assert.Equal("$due=2025-07-05T05-05", task.Norm())
		assert.Equal(*dt, *task.Time.DueDate)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if assert.NotNil(tk) {
			assert.Equal("$due=2025-07-05T05-05", *tk.raw)
			assert.Equal(*dt, *tk.Value.(*TokenDateValue).Value)
		}
	})
}

func TestTokenValueTypes(t *testing.T) {
	assert := assert.New(t)
	task, _ := ParseTask(utils.MkPtr(2), "(prio) $c=2025-05-05T05-05 $due=1w $dead=1w +prj #tag @at !exclamation ?question *star &ampersand $id=1 $P=2 $every=1m $p=unit/10/100/cat")
	_, ok := task.Tokens[0].Value.(*string)
	assert.True(ok, "priority")
	_, ok = task.Tokens[1].Value.(*TokenDateValue)
	assert.True(ok, "c date")
	_, ok = task.Tokens[2].Value.(*TokenDateValue)
	assert.True(ok, "duedate")
	_, ok = task.Tokens[3].Value.(*TokenDateValue)
	assert.True(ok, "deadline")
	_, ok = task.Tokens[4].Value.(*string)
	assert.True(ok, "+hint")
	_, ok = task.Tokens[5].Value.(*string)
	assert.True(ok, "#hint")
	_, ok = task.Tokens[6].Value.(*string)
	assert.True(ok, "@hint")
	_, ok = task.Tokens[7].Value.(*string)
	assert.True(ok, "!hint")
	_, ok = task.Tokens[8].Value.(*string)
	assert.True(ok, "?hint")
	_, ok = task.Tokens[9].Value.(*string)
	assert.True(ok, "*hint")
	_, ok = task.Tokens[10].Value.(*string)
	assert.True(ok, "&hint")
	_, ok = task.Tokens[11].Value.(*string)
	assert.True(ok, "id")
	_, ok = task.Tokens[12].Value.(*string)
	assert.True(ok, "P")
	_, ok = task.Tokens[13].Value.(*time.Duration)
	assert.True(ok, "every")
	_, ok = task.Tokens[14].Value.(*Progress)
	assert.True(ok, "progress")
}

func TestLists(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("hello")
	var tasks []*Task
	for ndx := range 10 {
		task, _ := ParseTask(utils.MkPtr(ndx), fmt.Sprintf("%d", ndx))
		tasks = append(tasks, task)
	}
	t.Run("exists", func(t *testing.T) {
		Lists.Init(path)
		assert.True(Lists.Exists(path))
		Lists.Delete(path)
		assert.False(Lists.Exists(path))
	})
	t.Run("init", func(t *testing.T) {
		Lists.Delete(path)
		Lists.Init(path)
		assert.True(Lists.Exists(path))
		assert.NotNil(Lists[path].EIDs)
		assert.NotNil(Lists[path].PIDs)
		assert.Empty(Lists[path].Tasks)
		Lists.Init(path, tasks...)
		assert.NotEmpty(Lists[path].Tasks)
		Lists.Delete(path)
		Lists.Init(path, tasks...)
		assert.NotEmpty(path)
	})
	t.Run("empty", func(t *testing.T) {
		Lists.Delete(path)
		Lists.Init(path)
		Lists.Empty(path)
		assert.Empty(Lists[path].Tasks)
		Lists.Set(path, tasks)
		assert.NotEmpty(Lists[path].Tasks)
		Lists.Empty(path)
		assert.Empty(Lists[path].Tasks)
	})
	t.Run("set", func(t *testing.T) {
		Lists.Delete(path)
		Lists.Init(path)
		Lists.Set(path, tasks)
		assert.Equal(tasks, Lists[path].Tasks)
		Lists.Set(path, tasks[2:6])
		assert.Equal(tasks[2:6], Lists[path].Tasks)
	})
	t.Run("append", func(t *testing.T) {
		Lists.Append(path, tasks[0])
		assert.NotEqual(tasks[1], Lists[path].Tasks[len(Lists[path].Tasks)-1])
		Lists.Append(path, tasks[1])
		assert.Equal(tasks[1], Lists[path].Tasks[len(Lists[path].Tasks)-1])
	})
}

func TestString(t *testing.T) {
	assert := assert.New(t)
	task, _ := ParseTask(nil, "(A) +prj #tag @at !exclamation ?question *star &ampersand $due=1w $dead=1w $r=-2h $-id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
	helper := func(ndx int) string {
		return task.Tokens[ndx].String()
	}
	assert.Equal("(A)", helper(0))
	assert.Equal("+prj", helper(1))
	assert.Equal("#tag", helper(2))
	assert.Equal("@at", helper(3))
	assert.Equal("!exclamation", helper(4))
	assert.Equal("?question", helper(5))
	assert.Equal("*star", helper(6))
	assert.Equal("&ampersand", helper(7))
	assert.Equal("$due=1w", helper(8))
	assert.Equal("$dead=1w", helper(9))
	assert.Equal("$r=-2h", helper(10))
	assert.Equal("$-id=3", helper(11))
	assert.Equal("$P=2", helper(12))
	assert.Equal("$p=unit/2/15/cat", helper(13))
	assert.Equal("text", helper(14))
	assert.Equal("$r=-3d", helper(15))
	assert.Equal("$every=1m", helper(16))
}

func TestTokens(t *testing.T) {
	assert := assert.New(t)
	task, _ := ParseTask(nil, "(A) +prj #tag @at $due=1w $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
	keyValues := map[int]string{
		0: "(A)", 1: "+prj", 2: "#tag", 3: "@at", 4: "$due=1w",
		5: "$dead=1w", 6: "$r=-2h", 7: "$id=3", 8: "$P=2",
		9: "$p=unit/2/15/cat", 10: "text", 11: "$r=-3d",
		12: "$every=1m",
	}
	values := make([]string, len(keyValues))
	valuesKeys := make(map[string]int)
	for key, value := range keyValues {
		values[key] = value
		valuesKeys[value] = key
	}

	t.Run("simple ForEach", func(t *testing.T) {
		task.Tokens.ForEach(func(tk *Token) {
			if tk.Type == TokenDate && tk.Key == "c" {
				return
			}
			assert.Contains(values, tk.String())
		})
	})
	t.Run("find", func(t *testing.T) {
		tk, ndx := task.Tokens.Find(func(tk *Token) bool {
			if tk.Type == TokenID && tk.Key == "id" && *tk.Value.(*string) == "3" {
				return true
			}
			return false
		})
		assert.NotNil(tk)
		assert.Equal("$id=3", values[ndx])

		tk, ndx = task.Tokens.Find(func(tk *Token) bool {
			if tk.Type == TokenID && tk.Key == "id" && *tk.Value.(*string) == "something weird" {
				return true
			}
			return false
		})
		assert.Nil(tk)
		assert.Equal(-1, ndx)
	})
	t.Run("filter", func(t *testing.T) {
		task.Tokens.Filter(TkByType(TokenDate)).Filter(func(tk *Token) bool {
			return tk.Key != "c"
		}).ForEach(func(tk *Token) {
			key := tk.String()
			_, ok := valuesKeys[key]
			if assert.True(ok) {
				assert.Equal(TokenDate, tk.Type)
				assert.NotEqual("c", tk.Key)
			}
		})
		task.Tokens.Filter(TkByTypeKey(TokenID, "id")).ForEach(func(tk *Token) {
			assert.Equal(TokenID, tk.Type)
			assert.Equal("id", tk.Key)
			assert.Equal("3", *tk.Value.(*string))
		})
	})
	t.Run("\\; Join Norm", func(t *testing.T) {
		t.Run("semicolons", func(t *testing.T) {
			tc := "   \\;  \\;;asd\\;   a\\;     \\;a\\;b \\; \\;c d\\;    b "
			task, _ := ParseTask(nil, tc)
			for _, tk := range task.Tokens {
				if strings.Contains(*tk.raw, "\\;") || strings.Contains(*tk.raw, "  ") {
					assert.Equal(TokenText, tk.Type)
					assert.Equal(";", tk.Key)
				}
			}
			assert.Equal(tc, task.NormRegular())
		})
		t.Run("spaces", func(t *testing.T) {
			tc := ";asd  a   a    b  c d"
			task, _ = ParseTask(nil, tc)
			for _, tk := range task.Tokens {
				if strings.Contains(*tk.raw, "\\;") || strings.Contains(*tk.raw, "  ") {
					assert.Equal(TokenText, tk.Type)
					assert.Equal(";", tk.Key)
				}
			}
			assert.Equal(tc, task.NormRegular())
		})
		t.Run("space at beginning", func(t *testing.T) {
			for _, tc := range []string{" a", "   a"} {
				task, _ = ParseTask(nil, tc)
				for _, tk := range task.Tokens {
					if strings.Contains(*tk.raw, "\\;") || strings.Contains(*tk.raw, "  ") {
						assert.Equal(TokenText, tk.Type)
						assert.Equal(";", tk.Key)
					}
				}
				assert.Equal(tc, task.NormRegular())
			}
		})
		t.Run("space at end", func(t *testing.T) {
			for _, tc := range []string{"a ", "a    "} {
				task, _ = ParseTask(nil, tc)
				for _, tk := range task.Tokens {
					if strings.Contains(*tk.raw, "\\;") || strings.Contains(*tk.raw, "  ") {
						assert.Equal(TokenText, tk.Type)
						assert.Equal(";", tk.Key)
					}
				}
				assert.Equal(tc, task.NormRegular())
			}
		})
	})
	t.Run("priority and anti-priority", func(t *testing.T) {
		task, _ := ParseTask(nil, "(A)")
		assert.Equal("(A)", *task.Priority)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "priority"))
		assert.NotNil(tk)
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenPriority, "anti-priority"))
		assert.Nil(tk)

		task, _ = ParseTask(nil, "[A]")
		assert.Equal("[A]", *task.Priority)
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenPriority, "anti-priority"))
		assert.NotNil(tk)
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenPriority, "priority"))
		assert.Nil(tk)
	})

	t.Run("urgent", func(t *testing.T) {
		task, _ := ParseTask(nil, "$urgent")
		assert.True(task.Urgent)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "urgent"))
		if assert.NotNil(tk) {
			assert.Equal("$urgent", *tk.raw)
			if assert.NotNil(tk.Value) {
				assert.Equal("$urgent", *tk.Value.(*string))
			}
		}
	})
	t.Run("mit", func(t *testing.T) {
		task, _ := ParseTask(nil, "$mit=0")
		if assert.NotNil(task.MIT) {
			assert.Equal(0, *task.MIT)
		}
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "mit"))
		if assert.NotNil(tk) {
			assert.Equal("$mit=0", *tk.raw)
			if assert.NotNil(tk.Value) {
				assert.Equal(0, *tk.Value.(*int))
			}
		}
	})
}

func TestRoot(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("test")
	Lists.Empty(path)
	AddTaskFromStr("$-id=1", path)
	AddTaskFromStr("$id=2 $P=1", path)
	AddTaskFromStr("$id=3 $P=2", path)
	AddTaskFromStr("$P=3", path)
	AddTaskFromStr("$id=4 $P=2", path)
	AddTaskFromStr("$P=4", path)
	AddTaskFromStr("$id=5", path)
	AddTaskFromStr("$id=6 $P=5", path)
	AddTaskFromStr("$P=6", path)
	AddTaskFromStr("$id=7", path)
	for _, task := range Lists[path].Tasks {
		if task.PID != nil {
			continue
		}
		assert.Equal(task.Norm(), task.Root().Norm())
		stack := slices.Clone(task.Children)
		for len(stack) > 0 {
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			assert.Equal(task.Norm(), node.Root().Norm())
			stack = append(stack, slices.Clone(node.Children)...)
		}
	}
}

func TestRevertIDtoText(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("revertID")
	prep := func() {
		Lists.Empty(path)
		AddTaskFromStr("0 $id=0", path)
		AddTaskFromStr("0.1 $P=0", path)
		AddTaskFromStr("0.2 $P=0", path)
		AddTaskFromStr("1 $id=1 $P=tada", path)
		AddTaskFromStr("2 $P=1", path)
		AddTaskFromStr("empty", path)
	}
	get := func(ndx int) *Task {
		return Lists[path].Tasks[ndx]
	}
	t.Run("reverts id", func(t *testing.T) {
		prep()
		get(0).revertIDtoText("id")
		assert.Nil(get(0).EID)
		assert.Empty(get(0).Children)
		assert.NotNil(get(1).PID)
		assert.Nil(get(1).Parent)
		assert.NotNil(get(2).PID)
		assert.Nil(get(2).Parent)
	})
	t.Run("non-existing has no effect", func(t *testing.T) {
		prep()
		assert.Nil(get(5).EID)
		assert.Empty(get(5).Children)
		assert.Nil(get(5).PID)
		assert.Nil(get(5).Parent)
		get(5).revertIDtoText("id")
		get(5).revertIDtoText("P")
		assert.Nil(get(5).EID)
		assert.Empty(get(5).Children)
		assert.Nil(get(5).PID)
		assert.Nil(get(5).Parent)
	})
	t.Run("reverts P", func(t *testing.T) {
		prep()
		get(4).revertIDtoText("P")
		assert.Nil(get(4).PID)
		assert.Nil(get(4).Parent)
		assert.NotContains(get(3).Children, get(4))
	})
	t.Run("panics for unknown key", func(t *testing.T) {
		prep()
		defer func() {
			r := recover()
			assert.NotNil(r, "panicked")
		}()
		get(0).revertIDtoText("something")
	})
}

func TestUpdateByModifyingText(t *testing.T) {
	assert := assert.New(t)
	t.Run("normal", func(t *testing.T) {
		task, _ := ParseTask(nil, " $c=rn:0s ")
		task.updateByModifyingText("1", "1")
		assert.Equal("1 $c=rn:0s 1", task.Raw())
	})
	t.Run("spaces", func(t *testing.T) {
		task, _ := ParseTask(nil, "  $c=rn:0s  ")
		task.updateByModifyingText("1", "1")
		assert.Equal("1  $c=rn:0s  1", task.Raw())

		task, _ = ParseTask(nil, " $c=rn:0s ")
		task.updateByModifyingText(" ", " ")
		assert.Equal("  $c=rn:0s  ", task.Raw())
	})
	t.Run("semi-colons", func(t *testing.T) {
		task, _ := ParseTask(nil, "\\;$c=rn:0s\\;")
		task.updateByModifyingText("1", "1")
		assert.Equal("1\\;$c=rn:0s\\;1", task.Raw())

		task, _ = ParseTask(nil, "$c=rn:0s")
		task.updateByModifyingText("\\;", "\\;")
		assert.Equal("\\;$c=rn:0s\\;", task.Raw())
	})
	t.Run("priority", func(t *testing.T) {
		t.Run("spaces", func(t *testing.T) {
			task, _ := ParseTask(nil, "(0)  $c=rn:0s")
			task.updateByModifyingText("1", "")
			assert.Equal("(0) 1  $c=rn:0s", task.Raw())

			task, _ = ParseTask(nil, "(0) $c=rn:0s")
			task.updateByModifyingText(" ", "")
			assert.Equal("(0) $c=rn:0s", task.Raw())
		})
		t.Run("semi-colons", func(t *testing.T) {
			task, _ := ParseTask(nil, "(0)\\;$c=rn:0s")
			task.updateByModifyingText("1", "")
			assert.Equal("(0) 1\\;$c=rn:0s", task.Raw())

			task, _ = ParseTask(nil, "(0) $c=rn:0s")
			task.updateByModifyingText("\\;", "")
			assert.Equal("(0)\\;$c=rn:0s", task.Raw())
		})
	})
}
