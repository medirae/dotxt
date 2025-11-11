package task

import (
	"dotxt/config"
	"dotxt/terrors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dtFormat = "2006-01-02T15-04"

func TestMain(m *testing.M) {
	if err := os.RemoveAll("/tmp/dotxt-testing"); err != nil {
		panic(err)
	}
	config.InitViper("/tmp/dotxt-testing")
	if err := os.MkdirAll("/tmp/dotxt-testing", 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// TODO: remove
func clearTasks(path string) {
	Lists.Empty(path)
}

// TODO: incinerate
func mockLoad(path string) error {
	path, err := parseFilepath(path)
	clearTasks(path)
	return err
}

func TestAdd(t *testing.T) {
	assert := assert.New(t)
	path := "todo"
	mockLoad(path)
	path, err := prepFileTaskFromPath(path)
	if assert.NoError(err) {
		assert.True(strings.HasSuffix(path, "/todos/todo"))
	}

	err = AddTaskFromStr("testing", path)
	if !assert.NoError(err) {
		assert.FailNow(err.Error())
	}
	task := Lists[path].Tasks[0]
	assert.Equal(*task.ID, 0)
	assert.Equal(task.Norm(), "testing")
}

func TestParseTask(t *testing.T) {
	assert := assert.New(t)
	var found bool
	var count int
	var err error
	var task *Task
	var dt time.Time

	t.Run("deformed", func(t *testing.T) {
		ParseTask(nil, "hey $yoyo")
	})

	t.Run("weird char support", func(t *testing.T) {
		weirdChars := "`!@#$%^&**()-_=+\\/'\"[]{};:.,"
		for _, char := range weirdChars {
			task, err = ParseTask(nil, string(char))
			if assert.NoError(err, "ParseTask") {
				assert.Equalf(task.Norm(), string(char), "char '%s'", string(char))
			}
		}
	})

	t.Run("validate auto creation of creationDate", func(t *testing.T) {
		task, _ := ParseTask(nil, "testing")
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "c"))
		if assert.NotNil(tk) {
			assert.Exactly(&rightNow, tk.Value.(*TokenDateValue).Value, "'c'")
		}
	})

	t.Run("validate empty", func(t *testing.T) {
		_, err = ParseTask(nil, "")
		if !assert.Error(err, "ParseTask") {
			assert.FailNow(err.Error())
		}
		assert.ErrorIs(err, terrors.ErrEmptyText)
	})

	t.Run("validate priority #4", func(t *testing.T) {
		task, _ = ParseTask(nil, "(!@`#$$%^&*([]{}./';\",)")
		assert.Equal("(!@`#$$%^&*([]{}./';\",)", *task.Priority)
		task, _ = ParseTask(nil, "[!@`#$$%^&*([]{}./';\",]")
		assert.Equal("[!@`#$$%^&*([]{}./';\",]", *task.Priority)
	})

	t.Run("validate hints #1", func(t *testing.T) {
		task, _ = ParseTask(nil, "# + @ ! ? * &")
		_, ndx := task.Tokens.Find(TkByType(TokenHint))
		assert.Equal(-1, ndx)
		assert.Len(task.Hints, 0)
	})
	t.Run("validate hints #2", func(t *testing.T) {
		task, _ = ParseTask(nil, "#hint +hint @hint !hint ?hint *hint &hint")
		count = 0
		ndxMap := map[int]rune{
			0: '#', 1: '+', 2: '@', 3: '!',
			4: '?', 5: '*', 6: '&',
		}
		task.Tokens.Filter(TkByType(TokenHint)).ForEachIndex(func(tk *Token, index int) {
			count++
			assert.Equal(string(ndxMap[index])+"hint", *tk.Value.(*string))
		})
		assert.Equal(7, count)
		assert.Equal("#hint +hint @hint !hint ?hint *hint &hint", task.Norm())
	})

	t.Run("validate invalid key value: no key", func(t *testing.T) {
		task, _ = ParseTask(nil, "$")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$", *tk.Value.(*string))
		}
	})
	t.Run("validate invalid key value: no equal sign", func(t *testing.T) {
		task, _ = ParseTask(nil, "$key")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$key", *tk.Value.(*string))
		}
	})
	t.Run("validate invalid key value: no value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$key=")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$key=", *tk.Value.(*string))
		}
	})
	t.Run("validate invalid key value: unknown key", func(t *testing.T) {
		task, _ = ParseTask(nil, "$key=value")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$key=value", *tk.Value.(*string))
		}
	})
	t.Run("validate invalid key value: known key but no value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$id=")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$id=", *tk.Value.(*string))
		}
	})

	t.Run("validate $id $P: strings", func(t *testing.T) {
		task, _ = ParseTask(nil, "$id=noway $P=nada")
		tk, _ := task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenID && strings.HasPrefix(*tk.raw, "$id=")
		})
		if assert.NotNil(tk) {
			assert.Equal("noway", *tk.Value.(*string))
		}
		if assert.NotNil(task.EID) {
			assert.Equal("noway", *task.EID, "EID")
		}
		tk, _ = task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenID && strings.HasPrefix(*tk.raw, "$P=")
		})
		if assert.NotNil(tk) {
			assert.Equal("nada", *tk.Value.(*string))
		}
		if assert.NotNil(task.PID) {
			assert.Equal("nada", *task.PID, "PID")
		}
	})
	t.Run("validate $id $P: valid", func(t *testing.T) {
		task, _ = ParseTask(nil, "$id=20002 $P=534")
		tk, _ := task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenID && strings.HasPrefix(*tk.raw, "$id=")
		})
		if assert.NotNil(tk) {
			assert.Equal("20002", *tk.Value.(*string))
		}
		if assert.NotNil(task.EID, "EID") {
			assert.Equal("20002", *task.EID, "EID")
		}
		tk, _ = task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenID && strings.HasPrefix(*tk.raw, "$P=")
		})
		if assert.NotNil(tk) {
			assert.Equal("534", *tk.Value.(*string))
		}
		if assert.NotNil(task.PID, "PID") {
			assert.Equal("534", *task.PID, "PID")
		}
	})
	t.Run("validate equal $id $P: revert id", func(t *testing.T) {
		task, _ := ParseTask(nil, "$id=200 $P=200")
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenID, "id"))
		if assert.NotNil(tk) && assert.NotNil(task.EID) {
			assert.Equal(tk.Value.(*string), task.EID)
		}
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenID, "P"))
		assert.Nil(tk)
		assert.Nil(task.Parent)
	})

	t.Run("validate invalid absolute dates: %Y", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025")
		assert.Nil(task.Time.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04")
		assert.Nil(task.Time.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m-%d", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04-02")
		assert.Nil(task.Time.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m-%d", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04-02T")
		assert.Nil(task.Time.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m-%dT%H", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04-02T20")
		assert.Nil(task.Time.DueDate)
	})

	t.Run("validate absolute dates", func(t *testing.T) {
		dtVal := "2025-05-20T00-00"
		dt, _ := time.ParseInLocation(dtFormat, dtVal, time.Local)
		got, err := parseAbsoluteDatetime(dtVal)
		if assert.NoError(err, "parseAbsoluteDatetime.err") {
			assert.Exactly(dt, *got, "parseAbsoluteDatetime")
		}
	})

	t.Run("validate invalid relative dates: no value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$due=", *tk.raw)
		}
		assert.Nil(task.Time.DueDate)
	})
	t.Run("validate invalid relative dates: unknown unit", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=+2a")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$due=+2a", *tk.raw)
		}
		assert.Nil(task.Time.DueDate)
	})
	t.Run("validate invalid relative dates: unknown relation", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=abc:1y")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$due=abc:1y", *tk.raw)
		}
		assert.Nil(task.Time.DueDate)
	})
	t.Run("validate invalid relative dates: wrong syntax", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=c;123")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$due=c;123", *tk.raw)
		}
		assert.Nil(task.Time.DueDate)
	})

	t.Run("validate valid relative dates: relative dates", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=+1y2m3w4d5h6M7s")
		dt = rightNow.Add(38898367 * time.Second)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if assert.NotNil(tk) {
			assert.Exactly(dt, *tk.Value.(*TokenDateValue).Value)
		}
		assert.Exactly(dt, *task.Time.DueDate)
	})
	t.Run("validate valid relative dates: valid", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=c:+1y2m3w4d5h6M7s")
		day := 24 * 60 * 60 * time.Second
		dt = rightNow.Add(365*day + 2*30*day + 3*7*day + 4*day + 5*60*60*time.Second + 6*60*time.Second + 7*time.Second)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if assert.NotNil(tk) {
			assert.Exactly(dt, *tk.Value.(*TokenDateValue).Value)
		}
		assert.Exactly(dt, *task.Time.DueDate)
	})
	t.Run("validate valid relative dates: resolve function #1", func(t *testing.T) {
		task, _ = ParseTask(nil, "$dead=6m $due=1w $c=2020-01-01T01-01")
		dt, _ := parseAbsoluteDatetime("2020-01-01T01-01")
		assert.Exactly(*dt, *task.Time.CreationDate, "CreationDate")
		assert.Exactly(dt.Add(7*24*60*60*time.Second), *task.Time.DueDate, "DueDate")
		assert.Exactly(dt.Add((7*24*60*60+6*30*24*60*60)*time.Second), *task.Time.Deadline, "Deadline")
	})
	t.Run("validate valid relative dates: resolve function #2", func(t *testing.T) {
		task, _ = ParseTask(nil, "$r=-2d $due=1w $end=4m")
		assert.Exactly(rightNow.Add(7*24*60*60*time.Second), *task.Time.DueDate, "DueDate")
		assert.Exactly(rightNow.Add((7*24*60*60+4*30*24*60*60)*time.Second), *task.Time.EndDate, "EndDate")
		found = false
		for _, r := range task.Time.Reminders {
			if r.Equal(rightNow.Add((7 - 2) * 24 * 60 * 60 * time.Second)) {
				found = true
			}
		}
		assert.True(found)
	})

	t.Run("validate date semantics: maximum count", func(t *testing.T) {
		for _, key := range []string{"c", "due", "end", "dead", "r"} {
			// if not `r`, only the first value is accepted
			// 	and any other ones are disposed of
			task, _ = ParseTask(nil, fmt.Sprintf("$%s=2026-06-06T00-00 $%s=2027-06-06T00-00", key, key))
			dt, _ := parseAbsoluteDatetime("2026-06-06T00-00")
			if key == "r" {
				assert.Equal(2, len(task.Time.Reminders), "r count")
				rCount := 0
				for _, r := range task.Time.Reminders {
					if r.Equal(*dt) || r.Equal(dt.Add(365*24*60*60*time.Second)) {
						rCount++
					}
				}
				assert.Equal(2, rCount)
				continue
			}
			tdt, err := task.Time.getField(key)
			if !assert.NoError(err, key) {
				assert.Exactly(*dt, *tdt, key)
			}
		}
	})
	t.Run("validate date semantics: c maximum value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$c=1y")
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "c"))
		if assert.NotNil(tk) {
			assert.Exactly(rightNow, *tk.Value.(*TokenDateValue).Value)
		}
		tk, _ = task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$c=1y", *tk.raw)
		}
		assert.Exactly(rightNow, *task.Time.CreationDate, "CreationDate")
	})
	t.Run("validate date semantics: dead-due existence dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$dead=2026-06-06T00-00")
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, "dead"))
		assert.Equal(-1, ndx)
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$dead=2026-06-06T00-00", *tk.raw)
		}
		assert.Nil(task.Time.Deadline, "Deadline") // when there's deadline but no due, deadline loses depth
	})
	t.Run("validate date semantics: end-due existence dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$end=1w")
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, "end"))
		assert.Equal(-1, ndx)
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$end=1w", *tk.raw)
		}
		assert.Nil(task.Time.EndDate, "EndDate") // when there's end but no due, end loses depth
	})
	t.Run("validate date semantics: dead-due value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=10d $dead=c:2d")
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, "dead"))
		assert.Equal(-1, ndx)
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$dead=c:2d", *tk.raw)
		}
		assert.Nil(task.Time.Deadline, "Deadline") // when deadline <= due, deadline loses depth
	})
	t.Run("validate date semantics: end-due value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=10d $end=c:2d")
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, "end"))
		assert.Equal(-1, ndx)
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$end=c:2d", *tk.raw)
		}
		assert.Nil(task.Time.EndDate, "EndDate") // when end <= due, end loses depth
	})
	t.Run("validate date semantics: dead-end existence dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$dead=10d $end=1w $due=2w")
		tk, _ := task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenText && strings.Contains(*tk.raw, "dead")
		})
		if assert.NotNil(tk) {
			assert.Equal("$dead=10d", *tk.raw)
		}
		tk, _ = task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenText && strings.Contains(*tk.raw, "end")
		})
		if assert.NotNil(tk) {
			assert.Equal("$end=1w", *tk.raw)
		}
		assert.Nil(task.Time.Deadline, "Deadline")
		assert.Nil(task.Time.EndDate, "EndDate") // when end & dead, then both lose depth
	})
	t.Run("validate date semantics: due-c value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$c=2025-05-05T05-05 $due=2023-05-05T05-05")
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$due=2023-05-05T05-05", *tk.raw)
		}
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		assert.Equal(-1, ndx)
		assert.Nil(task.Time.DueDate, "DueDate") // when due <= c, due loses depth
	})
	t.Run("validate date semantics: r-c value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$r=c:-1w $r=c:1w $c=2025-05-05T05-05")
		dt, _ := parseAbsoluteDatetime("2025-05-12T05-05")
		if assert.Len(task.Time.Reminders, 1, "r count") {
			assert.Exactly(*dt, *task.Time.Reminders[0], "Reminders")
		}
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		assert.Equal("$r=c:-1w", *tk.raw)
	})

	t.Run("validate date semantics: every-due dependency", func(t *testing.T) {
		task, _ := ParseTask(nil, "$every=1y") // every with no due
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if assert.NotNil(task.Time.DueDate) {
			due := task.Time.CreationDate.Add(*task.Time.Every)
			assert.Equal(due, *task.Time.DueDate)
		}
		if assert.NotNil(tk) {
			assert.Equal("$due=1y", tk.String())
		}
	})

	t.Run("validate every", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=9y364d23h59M59s")
		val := (9*365*24*60*60 + 364*24*60*60 + 23*60*60 + 59*60 + 59) * time.Second
		assert.Exactly(val, *task.Time.Every, "Every")
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDuration, "every"))
		if assert.NotNil(tk) {
			assert.Equal(val, *tk.Value.(*time.Duration))
		}
	})
	t.Run("validate every: minimum", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=23h59M59s")
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDuration, "every"))
		assert.Equal(-1, ndx)
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$every=23h59M59s", *tk.raw)
		}
		assert.Nil(task.Time.Every, "Every")
	})
	t.Run("validate every: maximum", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=10y")
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDuration, "every"))
		assert.Equal(-1, ndx)
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$every=10y", *tk.raw)
		}
		assert.Nil(task.Time.Every, "Every")
	})
	t.Run("validate every: negative", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=-1w")
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenDuration, "every"))
		assert.Equal(-1, ndx)
		tk, _ := task.Tokens.Find(TkByType(TokenText))
		if assert.NotNil(tk) {
			assert.Equal("$every=-1w", *tk.raw)
		}
		assert.Nil(task.Time.Every, "Every")
	})

	t.Run("validate quoted texts", func(t *testing.T) {
		task, _ := ParseTask(nil, "t1 't2 \\' t2' \"t3 \\\" t3\" `t4 \\` t4` \\'\\'\\'")
		tk, ndx := task.Tokens.Find(TkByTypeKey(TokenText, ""))
		if assert.NotNil(tk) {
			assert.Equal("t1", *tk.Value.(*string))
			assert.Equal("t1", *tk.raw)
		}
		tk, ndx = task.Tokens.FindFrom(TkByTypeKey(TokenText, "quote"), ndx+1)
		if assert.NotNil(tk) {
			assert.Equal("'t2 \\' t2'", *tk.Value.(*string))
			assert.Equal("'t2 \\' t2'", *tk.raw)
		}
		tk, ndx = task.Tokens.FindFrom(TkByTypeKey(TokenText, "quote"), ndx+1)
		if assert.NotNil(tk) {
			assert.Equal("\"t3 \\\" t3\"", *tk.Value.(*string))
			assert.Equal("\"t3 \\\" t3\"", *tk.raw)
		}
		tk, _ = task.Tokens.FindFrom(TkByTypeKey(TokenText, "quote"), ndx+1)
		if assert.NotNil(tk) {
			assert.Equal("`t4 \\` t4`", *tk.Value.(*string))
			assert.Equal("`t4 \\` t4`", *tk.raw)
		}
		tk = task.Tokens[len(task.Tokens)-2]
		assert.Empty(tk.Key)
		assert.Equal(`\'\'\'`, *tk.Value.(*string))
	})

	t.Run("validate token focus", func(t *testing.T) {
		task, _ := ParseTask(nil, "hellow $focus")
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenFormat, "focus"))
		require.NotNil(t, tk)
		assert.Equal("$focus", *tk.raw)
		assert.Nil(tk.Value)
		require.NotNil(t, task.Fmt)
		assert.True(task.Fmt.Focus)
	})

	t.Run("validate token urgent", func(t *testing.T) {
		task, _ := ParseTask(nil, "hellow $urgent")
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "urgent"))
		require.NotNil(t, tk)
		assert.Equal("$urgent", *tk.raw)
		if assert.NotNil(tk.Value) {
			assert.Equal("$urgent", *tk.Value.(*string))
		}
		assert.True(task.Urgent)
	})
	t.Run("validate urgencty: due urgent dependency", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $urgent")
		assert.False(task.Urgent)
		assert.NotNil(task.Time.DueDate)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "urgent"))
		assert.Nil(tk)
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		assert.NotNil(tk)

		task, _ = ParseTask(nil, "$due=5w $urgent")
		assert.True(task.Urgent)
		assert.NotNil(task.Time.DueDate)
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenPriority, "urgent"))
		assert.NotNil(tk)
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		assert.NotNil(tk)
	})

	t.Run("validate token mit", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			task, _ := ParseTask(nil, "hello $mit=0")
			tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "mit"))
			require.NotNil(t, tk)
			assert.Equal("$mit=0", *tk.raw)
			if assert.NotNil(tk.Value) {
				assert.Equal(0, *tk.Value.(*int))
			}
			if assert.NotNil(task.MIT) {
				assert.Equal(0, *task.MIT)
			}
		})
		t.Run("invalid number", func(t *testing.T) {
			for _, line := range []string{
				"$mit=something", "$mit=-2",
			} {
				task, _ := ParseTask(nil, line)
				tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "mit"))
				require.Nil(t, tk)
				require.Nil(t, task.MIT)
				tk, _ = task.Tokens.Find(TkByType(TokenText))
				require.NotNil(t, tk)
				assert.Equal(line, *tk.raw)
				assert.Equal(line, *tk.Value.(*string))
			}
		})
	})
}

func TestParseDuration(t *testing.T) {
	assert := assert.New(t)
	var err error
	t.Run("zero value", func(t *testing.T) {
		for _, each := range "ymwdhMs" {
			d, err := parseDuration("0" + string(each))
			require.NoError(t, err)
			assert.Equal(time.Duration(0), *d)
		}
		for _, each := range []string{"0", "+0", "-0"} {
			d, err := parseDuration(each)
			require.NoError(t, err)
			assert.Equal(time.Duration(0), *d)
		}
	})
	t.Run("invalid: empty", func(t *testing.T) {
		_, err = parseDuration("")
		if assert.Error(err) {
			assert.ErrorIs(err, terrors.ErrEmptyText)
		}
	})
	t.Run("invalid: number conversion", func(t *testing.T) {
		_, err = parseDuration("+1y*")
		if assert.Error(err) {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "number conversion of ")
		}
	})
	t.Run("invalid: unexpected time unit", func(t *testing.T) {
		_, err = parseDuration("+1y2*")
		if assert.Error(err) {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "unexpected time unit")
		}
	})
	t.Run("invalid: trailing numbers", func(t *testing.T) {
		_, err = parseDuration("+1y23")
		if assert.Error(err) {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "trailing numbers without a ")
		}
	})
	t.Run("valid: positive sign", func(t *testing.T) {
		val, err := parseDuration("+1y2m3w4d5h6M7s")
		if assert.NoError(err, "error") {
			assert.Equal((7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
	t.Run("valid: no sign", func(t *testing.T) {
		val, err := parseDuration("1y2m3w4d5h6M7s")
		if assert.NoError(err, "error") {
			assert.Equal((7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
	t.Run("valid: negative sign", func(t *testing.T) {
		val, err := parseDuration("-1y2m3w4d5h6M7s")
		if assert.NoError(err, "error") {
			assert.Equal(-(7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
	t.Run("valid: scrambled", func(t *testing.T) {
		val, err := parseDuration("2m4d3w7s6M1y5h")
		if assert.NoError(err, "error") {
			assert.Equal((7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
	t.Run("valid: float values", func(t *testing.T) {
		val, err := parseDuration("1.5y1.5m1.555w1.002d12.27h3.14M0.0s")
		if assert.NoError(err) {
			sec := float64(time.Second)
			day := 24 * 60 * 60 * sec
			var v float64 = 1.5*365*day + 1.5*30*day + 1.555*7*day + 1.002*day + 12.27*60*60*sec + 3.14*60*sec
			assert.Equal(time.Duration(v), *val)
		}
	})
}

func TestUnparseDuration(t *testing.T) {
	assert := assert.New(t)
	const (
		sec   = 1 * time.Second
		min   = 60 * sec
		hour  = min * 60
		day   = hour * 24
		month = day * 30
		year  = month * 365
	)
	t.Run("huge positive", func(t *testing.T) {
		dur := unparseDuration(6*year + 1000*month + 1000*day + 1000*hour + 1000*min + 1000*sec)
		assert.Equal("265y2w3d8h56M40s", dur)
	})
	t.Run("negative", func(t *testing.T) {
		dur := unparseDuration(-1 * day)
		assert.Equal("-1d", dur)
	})
	t.Run("zero", func(t *testing.T) {
		dur := unparseDuration(time.Duration(0))
		assert.Equal("0s", dur)
	})
}

func TestParseProgress(t *testing.T) {
	assert := assert.New(t)
	t.Run("valid: 4-parter", func(t *testing.T) {
		p, err := parseProgress("unit/10/100/cat")
		if assert.NoError(err, "err") {
			assert.Equal("unit", p.Unit, "Unit")
			assert.Equal("cat", p.Category, "Category")
			assert.Equal(10, p.Count, "Count")
			assert.Equal(100, p.DoneCount, "DoneCount")
		}
	})
	t.Run("valid: 3-parter", func(t *testing.T) {
		p, err := parseProgress("unit/10/100")
		if assert.NoError(err, "err") {
			assert.Equal("unit", p.Unit, "Unit")
			assert.Equal("", p.Category, "Category")
			assert.Equal(10, p.Count, "Count")
			assert.Equal(100, p.DoneCount, "DoneCount")
		}
	})
	t.Run("invalid: less than 3-parter", func(t *testing.T) {
		_, err := parseProgress("unit/100")
		assert.Error(err, "err")
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "$progress: number of '/' is less than 2: required fields not provided")
		_, err = parseProgress("unit")
		assert.Error(err, "err")
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "$progress: number of '/' is less than 2: required fields not provided")
	})
	t.Run("invalid: doneCount", func(t *testing.T) {
		for _, val := range []string{"unit/10/!!/cat", "unit/10/!!"} {
			_, err := parseProgress(val)
			if assert.Error(err, "err") {
				assert.ErrorIs(err, terrors.ErrParse)
				assert.ErrorIs(err, terrors.ErrValue)
				assert.ErrorContains(err, "$progress: doneCount to int")
				assert.ErrorContains(err, "!!")
			}
		}
	})
	t.Run("invalid: count", func(t *testing.T) {
		for _, val := range []string{"unit/!!/100/cat", "unit/!!/100"} {
			_, err := parseProgress(val)
			if assert.Error(err, "err") {
				assert.ErrorIs(err, terrors.ErrParse)
				assert.ErrorIs(err, terrors.ErrValue)
				assert.ErrorContains(err, "$progress: count to int")
				assert.ErrorContains(err, "!!")
			}
		}
	})
	t.Run("invalid: minimum doneCount", func(t *testing.T) {
		_, err := parseProgress("unit/10/-1000/cat")
		if assert.Error(err, "err") {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "$progress: doneCount minimum is 1")
		}
	})
	t.Run("invalid: minimum count", func(t *testing.T) {
		_, err := parseProgress("unit/-10/100/cat")
		if assert.Error(err, "err") {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "$progress: count minimum is 0")
		}
	})
	t.Run("invalid: maximum count", func(t *testing.T) {
		p, err := parseProgress("unit/200/100/cat")
		assert.NoError(err, "err")
		assert.Equal(p.Count, 100) // if count is greater than or equals to doneCount, it becomes doneCount
	})
}

func TestUnparseProgress(t *testing.T) {
	assert := assert.New(t)
	t.Run("valid: 3-parter", func(t *testing.T) {
		val, err := unparseProgress(Progress{Unit: "unit", DoneCount: 100})
		assert.NoError(err, "err")
		assert.Equal("unit/0/100", val)
		val, err = unparseProgress(Progress{Unit: "unit", Count: 10, DoneCount: 100})
		assert.NoError(err, "err")
		assert.Equal("unit/10/100", val)
	})
	t.Run("valid: 4-parter", func(t *testing.T) {
		val, err := unparseProgress(Progress{Unit: "unit", Category: "cat", Count: 10, DoneCount: 100})
		assert.NoError(err, "err")
		assert.Equal("unit/10/100/cat", val)
	})
	t.Run("invalid: no unit", func(t *testing.T) {
		_, err := unparseProgress(Progress{DoneCount: 100})
		if assert.Error(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "unit cannot be empty")
		}
	})
	t.Run("invalid: minimum doneCount", func(t *testing.T) {
		_, err := unparseProgress(Progress{Unit: "unit"})
		if assert.Error(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "doneCount '0' cannot be less than 1")
		}
	})
	t.Run("invalid: minimum count", func(t *testing.T) {
		_, err := unparseProgress(Progress{Unit: "unit", Count: -1, DoneCount: 100})
		if assert.Error(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "count '-1' cannot be less than 0")
		}
	})
	t.Run("invalid: maximum count", func(t *testing.T) {
		_, err := unparseProgress(Progress{Unit: "unit", Count: 200, DoneCount: 100})
		if assert.Error(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "count '200' cannot be greater than doneCount '100'")
		}
	})
}

func TestParseAbsoluteDatetime(t *testing.T) {
	assert := assert.New(t)
	helper := func(dt string) *time.Time {
		val, err := parseAbsoluteDatetime(dt)
		require.NoError(t, err)
		return val
	}
	timeCheck := func(val *time.Time, hour, minute, second int) {
		assert.Equal(hour, val.Hour())
		assert.Equal(minute, val.Minute())
		assert.Equal(second, val.Second())
	}
	t.Run("valid time: 3-parter", func(t *testing.T) {
		v := helper("T01-02-03")
		timeCheck(v, 1, 2, 3)
	})
	t.Run("valid time: 2-parter: %H-%M", func(t *testing.T) {
		v := helper("T1-2")
		timeCheck(v, 1, 2, 0)
	})
	t.Run("valid time: 2-parter: %M-%S", func(t *testing.T) {
		v := helper("T24-2")
		timeCheck(v, 0, 24, 2)
	})
	t.Run("valid time: 1-parter: %H", func(t *testing.T) {
		v := helper("T2")
		timeCheck(v, 2, 0, 0)
	})
	t.Run("valid time: 1-parter: %M", func(t *testing.T) {
		v := helper("T24")
		timeCheck(v, 0, 24, 0)
	})
	dateCheck := func(val *time.Time, year, month, day int) {
		assert.Equal(year, val.Year())
		assert.EqualValues(month, val.Month())
		assert.Equal(day, val.Day())
	}
	t.Run("valid date: 3-parter", func(t *testing.T) {
		t.Run("%Y-%m-%d", func(t *testing.T) {
			v := helper("2025-01-02")
			dateCheck(v, 2025, 1, 2)
		})
		t.Run("%y-%m-%d", func(t *testing.T) {
			v := helper("25-01-02")
			dateCheck(v, 2025, 1, 2)
		})
		t.Run("%Y-%b-%d", func(t *testing.T) {
			v := helper("2025-jan-02")
			dateCheck(v, 2025, 1, 2)
		})
		t.Run("%y-%b-%d", func(t *testing.T) {
			v := helper("25-jan-02")
			dateCheck(v, 2025, 1, 2)
		})
	})
	t.Run("valid date: 2-parter", func(t *testing.T) {
		t.Run("%Y-%m", func(t *testing.T) {
			v := helper("2025-2")
			dateCheck(v, 2025, 2, 1)
		})
		t.Run("%Y-%b", func(t *testing.T) {
			v := helper("2025-feb")
			dateCheck(v, 2025, 2, 1)
		})
		t.Run("%y-%m", func(t *testing.T) {
			v := helper("25-2")
			dateCheck(v, 2025, 2, 1)
		})
		t.Run("%y-%b", func(t *testing.T) {
			v := helper("25-feb")
			dateCheck(v, 2025, 2, 1)
		})
		t.Run("%m-%d", func(t *testing.T) {
			v := helper("12-2")
			dateCheck(v, 2025, 12, 2)
		})
		t.Run("%b-%d", func(t *testing.T) {
			v := helper("dec-2")
			dateCheck(v, 2025, 12, 2)
			v = helper("jan-01")
			dateCheck(v, 2025, 1, 1)
			v = helper("dec-29")
			dateCheck(v, 2025, 12, 29)
		})
	})
	t.Run("valid date: 1-parter", func(t *testing.T) {
		t.Run("%Y", func(t *testing.T) {
			v := helper("2025")
			dateCheck(v, 2025, 1, 1)
		})
		t.Run("%y", func(t *testing.T) {
			v := helper("94")
			dateCheck(v, 1994, 1, 1)
		})
		t.Run("%m", func(t *testing.T) {
			v := helper("5")
			dateCheck(v, rightNow.Year(), 5, 1)
		})
		t.Run("%b", func(t *testing.T) {
			v := helper("dec")
			dateCheck(v, rightNow.Year(), 12, 1)
		})
	})
	t.Run("valid: zero fills", func(t *testing.T) {
		v := helper("25-2-3T5-6-7")
		dateCheck(v, 2025, 2, 3)
		timeCheck(v, 5, 6, 7)
	})
	t.Run("invalid time: empty T", func(t *testing.T) {
		_, err := parseAbsoluteDatetime("2025-05-05T")
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "invalid use of 'T'")
	})
	t.Run("invalid time: too many dashes after T", func(t *testing.T) {
		_, err := parseAbsoluteDatetime("2025-05-05T05-05-05-40")
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorContains(err, "invalid use 'T': either no string afterwards or too many dashes")
	})
	t.Run("invalid time: non-integer value for time", func(t *testing.T) {
		for _, dt := range []string{"Ta-1-2", "T1-a-2", "T1-2-a"} {
			_, err := parseAbsoluteDatetime(dt)
			require.Error(t, err)
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "invalid hour, minute or second")
		}
	})
	t.Run("invalid time: over 60 time value", func(t *testing.T) {
		for _, dt := range []string{"T60-1-2", "T1-60-2", "T1-2-60"} {
			_, err := parseAbsoluteDatetime(dt)
			require.Error(t, err)
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "invalid hour, minute or second value")
		}
	})
	t.Run("invalid time 3-parter: over 24 hours", func(t *testing.T) {
		_, err := parseAbsoluteDatetime("T24-1-2")
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrParse)
		assert.ErrorIs(err, terrors.ErrValue)
		assert.ErrorContains(err, "invalid hour value")
	})
	t.Run("invalid date: too many dashes", func(t *testing.T) {
		_, err := parseAbsoluteDatetime("2025-05-05-05-05")
		if assert.Error(err, "err") {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "invalid use of date: too many dashes")
		}
	})
	t.Run("invalid date: over 3000 value", func(t *testing.T) {
		for _, dt := range []string{"3000-2-3", "25-3000-3", "25-3-3000"} {
			_, err := parseAbsoluteDatetime(dt)
			require.Error(t, err)
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "invalid year, month or day value")
		}
	})
	t.Run("invalid date: weird value", func(t *testing.T) {
		for _, dt := range []string{"a-2-3", "25-a-3", "25-3-a"} {
			_, err := parseAbsoluteDatetime(dt)
			require.Error(t, err)
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "invalid date value which is neither 'Jan' nor a number")
		}
	})
}

func TestUnparseAbsoluteDatetime(t *testing.T) {
	assert := assert.New(t)
	helper := func(dt string) string {
		val, err := parseAbsoluteDatetime(dt)
		require.NoError(t, err)
		return unparseAbsoluteDatetime(*val)
	}
	t.Run("%Y", func(t *testing.T) {
		v := helper("2025")
		assert.Equal("2025", v)
	})
	t.Run("%Y-%m", func(t *testing.T) {
		v := helper("2025-dec")
		assert.Equal("2025-12", v)
	})
	t.Run("%Y-%m-%d", func(t *testing.T) {
		v := helper("2025-dec-2")
		assert.Equal("2025-12-02", v)
	})
	t.Run("%Y-%m-%dT%H", func(t *testing.T) {
		v := helper("2025-05-05T2")
		assert.Equal("2025-05-05T02", v)
	})
	t.Run("%Y-%m-%dT%H-%M", func(t *testing.T) {
		v := helper("2025-05-05T2-3")
		assert.Equal("2025-05-05T02-03", v)
	})
	t.Run("%Y-%m-%dT%H-%M-%S", func(t *testing.T) {
		v := helper("2025-05-05T2-3-4")
		assert.Equal("2025-05-05T02-03-04", v)
	})
}

func TestResolveDates(t *testing.T) {
	assert := assert.New(t)
	helper := func(line string) ([]*Token, []error) {
		var tokens []*Token
		for token := range strings.SplitSeq(line, " ") {
			token := strings.TrimSpace(token)
			if token == "" {
				continue
			}
			switch token[0] {
			case '$':
				keyValue := strings.SplitN(token[1:], "=", 2)
				if len(keyValue) != 2 {
					continue
				}
				key, value := keyValue[0], keyValue[1]
				if validateEmptyText(value) != nil {
					continue
				}
				switch key {
				case "c", "due", "end", "dead", "r":
					var tkValue TokenDateValue
					var err error
					tkValue.Value, err = parseAbsoluteDatetime(value)
					if err != nil {
						tkValue.RelKey, tkValue.Offset, err = parseTmpRelativeDatetime(key, value)
						if err != nil {
							continue
						}
					}
					tokens = append(tokens, &Token{
						Type: TokenDate, raw: &token,
						Key: key, Value: &tkValue,
					})
				}
			}
		}
		return tokens, resolveDates(tokens)
	}
	t.Run("normal", func(t *testing.T) {
		tokens, errs := helper("$c=2025-05-05T05-05 $due=1w $dead=2w $end=3w $r=-5h $r=+5M")
		require.Empty(t, errs)
		dt, _ := parseAbsoluteDatetime("2025-05-05T05-05")
		tkM := make(map[string]*Token)
		for _, tk := range tokens {
			tkM[tk.Key] = tk
		}
		for _, tk := range tokens {
			assert.Equal(TokenDate, tk.Type)
			tdv, ok := tk.Value.(*TokenDateValue)
			assert.True(ok)
			switch tk.Key {
			case "c":
				assert.Equal(*dt, *tdv.Value)
			case "due":
				if assert.NotNil(tdv.Value) {
					assert.Equal(dt.Add(1*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "c") {
					assert.Equal(*tdv.RelVal, *tkM["c"].Value.(*TokenDateValue).Value)
				}
			case "dead":
				if assert.NotNil(tdv.Value) {
					assert.Equal(tkM["due"].Value.(*TokenDateValue).Value.Add(2*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "due") {
					assert.Equal(*tdv.RelVal, *tkM["due"].Value.(*TokenDateValue).Value)
				}
			case "end":
				if assert.NotNil(tdv.Value) {
					assert.Equal(tkM["due"].Value.(*TokenDateValue).Value.Add(3*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "due") {
					assert.Equal(*tdv.RelVal, *tkM["due"].Value.(*TokenDateValue).Value)
				}
			case "r":
				if *tk.raw == "$r=-5h" {
					if assert.NotNil(tdv.Value) {
						assert.Equal(tkM["due"].Value.(*TokenDateValue).Value.Add(-5*60*60*time.Second), *tdv.Value)
					}
					if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "due") {
						assert.Equal(*tdv.RelVal, *tkM["due"].Value.(*TokenDateValue).Value)
					}
				} else if *tk.raw == "$r=+5M" {
					if assert.NotNil(tdv.Value) {
						assert.Equal(tkM["due"].Value.(*TokenDateValue).Value.Add(5*60*time.Second), *tdv.Value)
					}
					if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "due") {
						assert.Equal(*tdv.RelVal, *tkM["due"].Value.(*TokenDateValue).Value)
					}
				} else {
					t.Error("reminder(s) is/are missing")
				}
			}
		}
	})
	t.Run("normal with custom relations", func(t *testing.T) {
		tokens, errs := helper("$c=2025-05-05T05-05 $due=rn:1w $dead=c:2w $end=rn:3w $r=c:-5h $r=due:+5M")
		require.Empty(t, errs)
		tkM := make(map[string]*Token)
		for _, tk := range tokens {
			tkM[tk.Key] = tk
		}
		for _, tk := range tokens {
			assert.Equal(TokenDate, tk.Type)
			tdv, ok := tk.Value.(*TokenDateValue)
			assert.True(ok)
			switch tk.Key {
			case "c":
				assert.Equal(*tkM["c"].Value.(*TokenDateValue).Value, *tdv.Value)
			case "due":
				if assert.NotNil(tdv.Value) {
					assert.Equal(rightNow.Add(1*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "rn") {
					assert.Equal(*tdv.RelVal, rightNow)
				}
			case "dead":
				if assert.NotNil(tdv.Value) {
					assert.Equal(tkM["c"].Value.(*TokenDateValue).Value.Add(2*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "c") {
					assert.Equal(*tdv.RelVal, *tkM["c"].Value.(*TokenDateValue).Value)
				}
			case "end":
				if assert.NotNil(tdv.Value) {
					assert.Equal(rightNow.Add(3*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "rn") {
					assert.Equal(*tdv.RelVal, rightNow)
				}
			case "r":
				if *tk.raw == "$r=c:-5h" {
					if assert.NotNil(tdv.Value) {
						assert.Equal(tkM["c"].Value.(*TokenDateValue).Value.Add(-5*60*60*time.Second), *tdv.Value)
					}
					if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "c") {
						assert.Equal(*tdv.RelVal, *tkM["c"].Value.(*TokenDateValue).Value)
					}
				} else if *tk.raw == "$r=due:+5M" {
					if assert.NotNil(tdv.Value) {
						assert.Equal(tkM["due"].Value.(*TokenDateValue).Value.Add(5*60*time.Second), *tdv.Value)
					}
					if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "due") {
						assert.Equal(*tdv.RelVal, *tkM["due"].Value.(*TokenDateValue).Value)
					}
				} else {
					t.Error("reminder(s) is/are missing raw=", tk.raw)
				}
			}
		}
	})
	t.Run("normal with custom relations and shuffled tokens", func(t *testing.T) {
		tokens, errs := helper("$due=rn:1w $r=due:+5M $end=rn:3w $r=c:-5h $c=2025-05-05T05-05 $dead=c:2w")
		require.Empty(t, errs)
		tkM := make(map[string]*Token)
		for _, tk := range tokens {
			tkM[tk.Key] = tk
		}
		for _, tk := range tokens {
			assert.Equal(TokenDate, tk.Type)
			tdv, ok := tk.Value.(*TokenDateValue)
			assert.True(ok)
			switch tk.Key {
			case "c":
				assert.Equal(*tkM["c"].Value.(*TokenDateValue).Value, *tdv.Value)
			case "due":
				if assert.NotNil(tdv.Value) {
					assert.Equal(rightNow.Add(1*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "rn") {
					assert.Equal(*tdv.RelVal, rightNow)
				}
			case "dead":
				if assert.NotNil(tdv.Value) {
					assert.Equal(tkM["c"].Value.(*TokenDateValue).Value.Add(2*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "c") {
					assert.Equal(*tdv.RelVal, *tkM["c"].Value.(*TokenDateValue).Value)
				}
			case "end":
				if assert.NotNil(tdv.Value) {
					assert.Equal(rightNow.Add(3*7*24*60*60*time.Second), *tdv.Value)
				}
				if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "rn") {
					assert.Equal(*tdv.RelVal, rightNow)
				}
			case "r":
				if *tk.raw == "$r=c:-5h" {
					if assert.NotNil(tdv.Value) {
						assert.Equal(tkM["c"].Value.(*TokenDateValue).Value.Add(-5*60*60*time.Second), *tdv.Value)
					}
					if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "c") {
						assert.Equal(*tdv.RelVal, *tkM["c"].Value.(*TokenDateValue).Value)
					}
				} else if *tk.raw == "$r=due:+5M" {
					if assert.NotNil(tdv.Value) {
						assert.Equal(tkM["due"].Value.(*TokenDateValue).Value.Add(5*60*time.Second), *tdv.Value)
					}
					if assert.NotNil(tdv.RelVal) && assert.Equal(tdv.RelKey, "due") {
						assert.Equal(*tdv.RelVal, *tkM["due"].Value.(*TokenDateValue).Value)
					}
				} else {
					t.Error("reminder(s) is/are missing raw=", *tk.raw)
				}
			}
		}
	})
}

func TestParseTasks(t *testing.T) {
	assert := assert.New(t)
	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs("")
	path, _ := parseFilepath("test")
	os.WriteFile(path, []byte("1\n2\n3"), 0o644)

	data, err := ParseTasks(path)
	require.NoError(t, err)
	assert.Equal("1", data[0].Norm())
	assert.Equal(0, *data[0].ID)
	assert.Equal("2", data[1].Norm())
	assert.Equal(1, *data[1].ID)
	assert.Equal("3", data[2].Norm())
	assert.Equal(2, *data[2].ID)
}

func TestTokenizeLine(t *testing.T) {
	assert := assert.New(t)
	t.Run("single spaces", func(t *testing.T) {
		v := tokenizeLine("token1 token2 token3")
		assert.Equal("token1", v[0])
		assert.Equal("token2", v[1])
		assert.Equal("token3", v[2])
	})
	t.Run("extra spaces preserved", func(t *testing.T) {
		v := tokenizeLine("t1 t2    t3")
		assert.Equal("t1", v[0])
		assert.Equal("t2", v[1])
		assert.Equal("    ", v[2])
		assert.Equal("t3", v[3])
	})
	t.Run("initial and final spaces preserved", func(t *testing.T) {
		v := tokenizeLine(" t1  t2 ")
		assert.Equal(" ", v[0])
		assert.Equal("t1", v[1])
		assert.Equal("  ", v[2])
		assert.Equal("t2", v[3])
		assert.Equal(" ", v[4])
	})
	t.Run("escaped space", func(t *testing.T) {
		v := tokenizeLine("t1 t2\\ t3")
		assert.Equal("t1", v[0])
		assert.Equal("t2 t3", v[1])
	})
	t.Run("double quotes", func(t *testing.T) {
		v := tokenizeLine("t1 \"t2 still t2\"")
		assert.Equal("t1", v[0])
		assert.Equal("\"t2 still t2\"", v[1])
	})
	t.Run("nested quotes", func(t *testing.T) {
		v := tokenizeLine("t1 't2 \"t3\\ t4` t5`'")
		assert.Equal("t1", v[0])
		assert.Equal("'t2 \"t3\\ t4` t5`'", v[1])
	})
	t.Run("escaping quotes results with backslash", func(t *testing.T) {
		v := tokenizeLine("t1 't2 \\' t3'")
		assert.Equal("t1", v[0])
		assert.Equal("'t2 \\' t3'", v[1])
		v = tokenizeLine("t1 \"t2 \\\" t3\"")
		assert.Equal("t1", v[0])
		assert.Equal("\"t2 \\\" t3\"", v[1])
		v = tokenizeLine("t1 \\' t2 \\\" t3 \\`")
		assert.Equal("t1", v[0])
		assert.Equal("\\'", v[1])
		assert.Equal("t2", v[2])
		assert.Equal("\\\"", v[3])
		assert.Equal("t3", v[4])
		assert.Equal("\\`", v[5])
	})
	t.Run("unterminated quote rolls back", func(t *testing.T) {
		v := tokenizeLine("t1 \" t2 t3")
		assert.Equal("t1", v[0])
		assert.Equal("\"", v[1])
		assert.Equal("t2", v[2])
		assert.Equal("t3", v[3])
	})
	t.Run("force terminate token", func(t *testing.T) {
		v := tokenizeLine("\\; t1 t\\;2 #hint\\;.")
		assert.Equal("\\; ", v[0])
		assert.Equal("t1", v[1])
		assert.Equal("t", v[2])
		assert.Equal("\\;", v[3])
		assert.Equal("2", v[4])
		assert.Equal("#hint", v[5])
		assert.Equal("\\;", v[6])
		assert.Equal(".", v[7])
	})
	t.Run("terminations and semicolons", func(t *testing.T) {
		v := tokenizeLine("   \\;  \\;;asd\\;   a\\;     \\;a\\;b \\; \\;c d\\;    b ")
		assert.Equal("   \\;  \\;", v[0])
		assert.Equal(";asd", v[1])
		assert.Equal("\\;   ", v[2])
		assert.Equal("a", v[3])
		assert.Equal("\\;     \\;", v[4])
		assert.Equal("a", v[5])
		assert.Equal("\\;", v[6])
		assert.Equal("b", v[7])
		assert.Equal(" \\; \\;", v[8])
		assert.Equal("c", v[9])
		assert.Equal("d", v[10])
		assert.Equal("\\;    ", v[11])
		assert.Equal("b", v[12])
		assert.Equal(" ", v[13])
	})
	t.Run("quotes are kept in the token", func(t *testing.T) {
		v := tokenizeLine("t1 \"t2 t2\" 't3 t3' `t4 t4`")
		assert.Equal("t1", v[0])
		assert.Equal("\"t2 t2\"", v[1])
		assert.Equal("'t3 t3'", v[2])
		assert.Equal("`t4 t4`", v[3])
	})
}

func TestParseUnparseCoherency(t *testing.T) {
	assert := assert.New(t)
	t.Run("duration", func(t *testing.T) {
		helper := func(d string) string {
			val, err := parseDuration(d)
			if !assert.NoError(err, d) {
				return ""
			}
			return unparseDuration(*val)
		}
		t.Run("zero value", func(t *testing.T) {
			for _, d := range []string{
				"0y", "-0y", "+0y",
				"0m", "-0m", "+0m",
				"0w", "-0w", "+0w",
				"0d", "-0d", "+0d",
				"0h", "-0h", "+0h",
				"0M", "-0M", "+0M",
				"0s", "-0s", "+0s",
			} {
				assert.Equal("0s", helper(d))
			}
		})
		t.Run("positive sign isn't kept", func(t *testing.T) {
			assert.Equal("2y", helper("+2y"))
			assert.Equal("2y", helper("2y"))
			assert.Equal("1y2m4d5h6M7s", helper("+1y2m4d5h6M7s"))
		})
		t.Run("negative sign is kept", func(t *testing.T) {
			assert.Equal("-1d", helper("-1d"))
		})
		t.Run("weeks are turned to days", func(t *testing.T) {
			assert.Equal("1y2m3w4d5h6M7s", helper("1y2m3w4d5h6M7s"))
		})
		t.Run("overflows are corrected", func(t *testing.T) {
			assert.Equal("1d6h", helper("30h"))
			assert.Equal("2y7m1w2d13h31M20s", helper("1y15m9w70d36h90M80s"))
		})
		t.Run("float values are poured to other units", func(t *testing.T) {
			assert.Equal("1y6m2d12h", helper("1.5y"))
		})
		t.Run("units are ordered descending", func(t *testing.T) {
			assert.Equal("1y2m3w4d5h6M7s", helper("2m4d3w7s6M1y5h"))
		})
	})
	t.Run("progress", func(t *testing.T) {
		helper := func(p string) string {
			val, err := parseProgress(p)
			if !assert.NoError(err, p) {
				return ""
			}
			v, err := unparseProgress(*val)
			if !assert.NoError(err, p) {
				return ""
			}
			return v
		}
		t.Run("no difference", func(t *testing.T) {
			for _, p := range []string{
				"unit/10/100", "unit/0/100",
				"unit/10/100", "unit/10/100/cat",
			} {
				assert.Equal(p, helper(p), p)
			}
		})
		t.Run("overvalued counts are corrected", func(t *testing.T) {
			assert.Equal("unit/100/100/cat", helper("unit/200/100/cat"))
		})
	})
	t.Run("absolute date", func(t *testing.T) {
		helper := func(dt string) string {
			val, err := parseAbsoluteDatetime(dt)
			if !assert.NoError(err, dt) {
				return ""
			}
			return unparseAbsoluteDatetime(*val)
		}
		year := rightNow.Format("2006")
		t.Run("dates are always provided", func(t *testing.T) {
			for _, dt := range []string{
				"T01-02", "T23-02", "T02", "T23",
			} {
				assert.Equal(year+dt, helper(dt))
			}
		})
		t.Run("padding is always done", func(t *testing.T) {
			assert.Equal(year+"-02-03T04-05-06", helper("25-2-3T4-5-6"))
		})
		t.Run("hour is always there", func(t *testing.T) {
			assert.Equal(year+"T23", helper("T23"))
			assert.Equal(year+"T00-24", helper("T24"))
			assert.Equal(year+"T00-24-30", helper("T24-30"))
		})
		t.Run("month strings are turned integer", func(t *testing.T) {
			assert.Equal("2025-01-02", helper("2025-jan-02"))
			assert.Equal("2025-02", helper("25-feb"))
		})
		t.Run("year is always there", func(t *testing.T) {
			assert.Equal(year+"-02", helper("feb"))
			assert.Equal(year+"-02-02", helper("feb-02"))
			assert.Equal(year+"-10", helper("10"))
			assert.Equal("2024", helper("24"))
		})
	})
	t.Run("relative date", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			for _, each := range []string{
				"$due=1m $dead=1m",
				"$due=1m $dead=c:1y",
			} {
				task, _ := ParseTask(nil, each)
				tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "dead"))
				if assert.NotNil(tk) {
					out := tk.unparseRelativeDatetime(nil)
					assert.Equal(strings.Split(each, " ")[1], out)
				}
			}
			task, _ := ParseTask(nil, "$due=c:1y")
			tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
			if assert.NotNil(tk) {
				out := tk.unparseRelativeDatetime(nil)
				assert.Equal("$due=c:1y", out)
			}
		})
		t.Run("absolute tokens unparsed as relative will be relative", func(t *testing.T) {
			task, _ := ParseTask(nil, "$due="+strconv.Itoa(rightNow.Year()+1)+"-dec-25")
			tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
			if assert.NotNil(tk) {
				out := tk.unparseRelativeDatetime(nil)
				tVal, _ := time.ParseInLocation("2006-Jan-02", strconv.Itoa(rightNow.Year()+1)+"-dec-25", time.Local)
				assert.Equal("$due="+unparseRelativeDatetime(tVal, rightNow), out)
			}
		})
	})
}

func TestTokenUnparseRelativeDatetime(t *testing.T) {
	assert := assert.New(t)
	findToken := func(task *Task, key string) *Token {
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, key))
		assert.NotNil(tk, key, task.Norm())
		return tk
	}
	t.Run("normal relative", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m")
		tk := findToken(task, "due")
		out := tk.unparseRelativeDatetime(nil)
		assert.Equal("$due=1m", out)
		t.Run("update", func(t *testing.T) {
			tVal := rightNow.Add(6 * 30 * 24 * 60 * 60 * time.Second)
			out = tk.unparseRelativeDatetime(&tVal)
			assert.Equal("$due=6m", out)
		})
	})
	t.Run("custom field relative", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m $dead=c:1y")
		tk := findToken(task, "dead")
		out := tk.unparseRelativeDatetime(nil)
		assert.Equal("$dead=c:1y", out)
		t.Run("update", func(t *testing.T) {
			tVal := rightNow.Add(6 * 30 * 24 * 60 * 60 * time.Second)
			out = tk.unparseRelativeDatetime(&tVal)
			assert.Equal("$dead=c:6m", out)
		})
	})
	t.Run("absolute date", func(t *testing.T) {
		year := rightNow.Format("2006")
		task, _ := ParseTask(nil, fmt.Sprintf("$due=%s-12-26", year))
		tk := findToken(task, "due")
		out := tk.unparseRelativeDatetime(nil)
		assert.Equal("$due="+unparseRelativeDatetime(*tk.Value.(*TokenDateValue).Value, rightNow), out)
		t.Run("update", func(t *testing.T) {
			tVal := rightNow.Add(6 * 30 * 24 * 60 * 60 * time.Second)
			out = tk.unparseRelativeDatetime(&tVal)
			assert.Equal("$due=6m", out)
		})
	})
}
