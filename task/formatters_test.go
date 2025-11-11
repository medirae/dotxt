package task

import (
	"bytes"
	"dotxt/utils"
	"fmt"
	"maps"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	assert := assert.New(t)
	helper := func(d int) string {
		v := time.Duration(d) * time.Second
		return formatDuration(&v)
	}
	const (
		minutes = 60
		hours   = minutes * 60
		days    = hours * 24
		weeks   = days * 7
		months  = days * 30
		years   = days * 365
	)
	t.Run("right now", func(t *testing.T) {
		s := helper(0)
		assert.Equal("rn", s)
	})
	t.Run("negative", func(t *testing.T) {
		s := helper(-10)
		assert.Equal("-10s", s)
	})
	t.Run("years >= 1.25", func(t *testing.T) {
		s := helper(1.265 * years)
		assert.Equal("1.2y", s)
		s = helper(12.882 * years)
		assert.Equal("12.8y", s)
		s = helper(5 * years)
		assert.Equal("5y", s)
	})
	t.Run("1 <= years < 1.25", func(t *testing.T) {
		s := helper(1.2375 * years)
		assert.Equal("1y2.8m", s)
		s = helper(1.164 * years)
		assert.Equal("1y1.9m", s)
		s = helper(1 * years)
		assert.Equal("1y", s)
		s = helper(1.004 * years)
		assert.Equal("1y", s)
	})
	t.Run("months >= 2", func(t *testing.T) {
		s := helper(12.166 * months)
		assert.Equal("12.1m", s)
		s = helper(12 * months)
		assert.Equal("12m", s)
		s = helper(2.5 * months)
		assert.Equal("2.5m", s)
		s = helper(2 * months)
		assert.Equal("2m", s)
	})
	t.Run("1 <= months < 2", func(t *testing.T) {
		s := helper(1.999 * months)
		assert.Equal("1m4.2w", s)
		s = helper(1.24 * months)
		assert.Equal("1m1w", s)
		s = helper(1.23 * months)
		assert.Equal("1m6d", s)
		s = helper(1.04 * months)
		assert.Equal("1m1d", s)
		s = helper(1.03 * months)
		assert.Equal("1m", s)
		s = helper(1 * months)
		assert.Equal("1m", s)
	})
	t.Run("weeks >= 1", func(t *testing.T) {
		s := helper(4.28 * weeks)
		assert.Equal("4w1d", s)
		s = helper(3.99 * weeks)
		assert.Equal("3w6d", s)
		s = helper(3.6 * weeks)
		assert.Equal("3w4d", s)
		s = helper(3 * weeks)
		assert.Equal("3w", s)
		s = helper(1.14 * weeks)
		assert.Equal("1w", s)
		s = helper(1 * weeks)
		assert.Equal("1w", s)
	})
	t.Run("days >= 2", func(t *testing.T) {
		s := helper(6.9 * days)
		assert.Equal("6d", s)
		s = helper(3 * days)
		assert.Equal("3d", s)
		s = helper(2 * days)
		assert.Equal("2d", s)
	})
	t.Run("1 <= days < 2", func(t *testing.T) {
		s := helper(1.99 * days)
		assert.Equal("1d23'", s)
		s = helper(1.5 * days)
		assert.Equal("1d12'", s)
		s = helper(1 * days)
		assert.Equal("1d", s)
	})
	t.Run("hours >= 2", func(t *testing.T) {
		s := helper(23.99 * hours)
		assert.Equal("23'59\"", s)
		s = helper(23 * hours)
		assert.Equal("23'", s)
		s = helper(2.01 * hours)
		assert.Equal("2'", s)
		s = helper(2 * hours)
		assert.Equal("2'", s)
	})
	t.Run("hours < 2", func(t *testing.T) {
		s := helper((1*hours + 59*minutes + 59))
		assert.Equal("1'59\"59s", s)
		s = helper((1*hours + 59*minutes))
		assert.Equal("1'59\"", s)
		s = helper((1*hours + 59))
		assert.Equal("1'59s", s)
		s = helper(1 * hours)
		assert.Equal("1'", s)
		s = helper((59*minutes + 59))
		assert.Equal("59\"59s", s)
		s = helper(59 * minutes)
		assert.Equal("59\"", s)
		s = helper(59)
		assert.Equal("59s", s)
		s = helper(1)
		assert.Equal("1s", s)
	})
}

func TestFormatAbsoluteDatetime(t *testing.T) {
	assert := assert.New(t)
	t.Run("nil dt", func(t *testing.T) {
		assert.Equal("", formatAbsoluteDatetime(nil, nil))
	})
	t.Run("nil relDt", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04-04")
		assert.Equal("2024-04-04T04-04", formatAbsoluteDatetime(dt, nil))
	})
	t.Run("duration eq", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04-04")
		assert.Equal("rn", formatAbsoluteDatetime(dt, dt))
	})
	t.Run("duration negative", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04")
		rel, _ := parseAbsoluteDatetime("2024-04-04T05-05")
		assert.Equal("-1'1\"", formatAbsoluteDatetime(dt, rel))
	})
	t.Run("duration positive", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04")
		rel, _ := parseAbsoluteDatetime("2024-04-04T03-03")
		assert.Equal("1'1\"", formatAbsoluteDatetime(dt, rel))
	})
}

func TestFormatPriorities(t *testing.T) {
	assert := assert.New(t)
	input := `
(A) $id=a                                  // 64
	(AA) $P=a                                  // 50
	(AAB) $P=a                                 // 60
	(AAC) $P=a                                 // 69
	(AB) $id=ab                                // 135
	(ABA) $P=ab                                // 114
	(ABB) $P=ab                                // 122
	(ABC) $P=ab                                // 131
	(ABCD) $P=ab                               // 139
	(ABCDE) $P=ab                              // 148
	(ABCDEA) $P=ab                             // 156
	(x) $P=a 1								   // 79
	(x) $P=a 2								   // 79
	(x) $P=a 3								   // 79
	(x) $P=a 4								   // 79
	(x) $P=a 5								   // 79
	(x) $P=a 6								   // 79
	(A)                                        // 64
	(AA)                                       // 87
	(AAB)                                      // 100
	(AAC)                                      // 106
	(AB)                                       // 135
	(ABA)                                      // 164
	(ABB)                                      // 170
	(ABC)                                      // 177
	(ABCD)                                     // 183
	(ABCDE)                                    // 190
	(ABCDEA)                                   // 196
	(t) 1									   // 264
	(t) 2									   // 264
	(t) 3 $id=t3							   // 264
	(t1) 3.1 $P=t3 							   // 240
	(t2) 3.2 $P=t3 							   // 256
	(t3) 3.3 $P=t3 $id=t33					   // 280
	(t31) 3.3.1 $P=t33  					   // 280
	(t) 4									   // 264
	(t) 5									   // 264
	(t) 6									   // 264
	(x) 1									   // 328
	(x) 2									   // 328
	(x) 3 $id=x3							   // 328
	(x1) 3.1 $P=x3 							   // 304
	(x2) 3.2 $P=x3 							   // 320
	(x3) 3.3 $P=x3 $id=x33					   // 344
	(x31) 3.3.1 $P=x33  					   // 344
	(x) 4									   // 328
	(x) 5									   // 328
	(x) 6									   // 328
	(XYZ)                                      // 215
	(XYZA)                                     // 222
	(ZYX)                                      // 228
	(LongPriorityStringThatExceedsDepthLimit)  // 202
	(Short)                                    // 209
	(1)                                        // 22
	(12)                                       // 29
	(123)                                      // 35
	(1A)                                       // 42
	(-)                                        // 3
	(--)                                       // 10
	(---)                                      // 16
	(AAAAAAAAAAAAAAAAAAAAAAAAAAAAA)            // 93
	$id=no-prio-parent						   // 0
	(prio-child) $P=no-prio-parent			   // 90
	(weird) $P=no-prio-parent			       // 270`
	lines := strings.Split(strings.TrimSpace(input), "\n")
	hues := make(map[*Task]int)
	lineMap := make(map[*Task]string)
	path, _ := parseFilepath("testPrio")
	Lists.Empty(path)
	for ndx, line := range lines {
		if utils.RuneAt(strings.TrimSpace(line), 0) == '/' {
			continue
		}
		parts := strings.SplitN(strings.TrimSpace(line), "//", 2)
		taskLine := strings.TrimSpace(parts[0])
		task, _ := ParseTask(utils.MkPtr(ndx), taskLine)
		Lists.Append(path, task)
		hueVal, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		hues[task] = hueVal
		lineMap[task] = line
	}
	CleanupRelations(path)
	rtasks := func() map[*Task]*rTask {
		out := make(map[*Task]*rTask)
		for _, t := range Lists[path].Tasks {
			out[t] = t.Render()
		}
		return out
	}()
	var rts []*rTask
	for rt := range maps.Values(rtasks) {
		rts = append(rts, rt)
	}
	formatPriorities(rts)
	for _, t := range Lists[path].Tasks {
		// line := lineMap[t] // uncomment these line to reprint the number for ease of modification
		// line = line[:strings.Index(line, "//")+2] + " "
		if t.Priority == nil {
			// fmt.Println(line + "0")
			continue
		}
		rt := rtasks[t]
		h, _, _ := utils.HexToHSL(rt.tokens[0].color)
		h = math.Round(h)
		// fmt.Println(line + strconv.Itoa(int(h)))
		assert.Equalf(float64(hues[t]), h, "=%s :color=%s", lineMap[t], rt.tokens[0].color)
	}
}

func TestFormatProgress(t *testing.T) {
	assert := assert.New(t)
	p := Progress{Unit: "unit", Category: "", Count: 14, DoneCount: 24}
	tokens := formatProgress(&p, 4, 5)
	var out strings.Builder
	for _, each := range tokens {
		out.WriteString(each.raw)
	}
	assert.Equal("  14/   24( 58%) ====>      (unit)", out.String())
}

func TestFormatListHeader(t *testing.T) {
	assert := assert.New(t)
	h := formatListHeader("/todosFile", 30)
	assert.Equal("> todosFile | ————————————————\n", h)
}

func TestResolvColor(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(viper.GetString("print.color-default"), resolvColor(""))
	assert.Equal(viper.GetString("print.color-default"), resolvColor("randommmm"))
	assert.Equal(viper.GetString("print.progress.count"), resolvColor("print.progress.count"))
}

func TestColorIds(t *testing.T) {
	assert := assert.New(t)
	out := colorizeIds(map[string]bool{"1": true, "2": true, "3": true})
	// 1:"#E09952", 2:"#99E052", 3:"#52E099"
	assert.Equal("#B48C64", out["1"])
	assert.Equal("#8CB464", out["2"])
	assert.Equal("#64B48C", out["3"])
}

func TestFormatCategoryHeader(t *testing.T) {
	assert := assert.New(t)
	l := rInfo{idLen: 2, countLen: 2, doneCountLen: 2, maxLen: 30}
	out := formatCategoryHeader("some", &l)
	assert.Equal("                     some ————\n", out)
	out = formatCategoryHeader("", &l)
	assert.Equal("                          ————\n", out)
}

func TestRender(t *testing.T) {
	assert := assert.New(t)
	id := 1

	t.Run("normal", func(t *testing.T) {
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1w $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
		rtask := task.Render()
		assert.Equal(1, rtask.rInfo.idLen)
		assert.Equal(103, rtask.rInfo.maxLen)
		assert.Equal(task, rtask.task, "task")
		assert.Equal(id, rtask.id, "id")
		assert.Equal("print.color-index", rtask.idColor, "idColor")
		assert.Equal("$p=unit/2/15/cat", *rtask.tokens[0].token.raw)
		assert.Equal("(A)", rtask.tokens[1].raw)
		assert.Equal("print.color-default", rtask.tokens[1].color)
		assert.Equal("+prj", rtask.tokens[2].raw)
		assert.Equal("print.hints.color-plus", rtask.tokens[2].color)
		assert.Equal("#tag", rtask.tokens[3].raw)
		assert.Equal("print.hints.color-tag", rtask.tokens[3].color)
		assert.Equal("@at", rtask.tokens[4].raw)
		assert.Equal("print.hints.color-at", rtask.tokens[4].color)
		assert.Equal("$due=1w", rtask.tokens[5].raw)
		assert.Equal("print.color-urgent", rtask.tokens[5].color)
		assert.Equal("$dead=1w", rtask.tokens[6].raw)
		assert.Equal("print.color-urgent", rtask.tokens[6].color)
		assert.Equal("$r=6d", rtask.tokens[7].raw)
		assert.Equal("print.color-date-r", rtask.tokens[7].color)
		assert.Equal("$id=3", rtask.tokens[8].raw)
		assert.Equal("print.color-default", rtask.tokens[8].color)
		assert.Equal("$P=2", rtask.tokens[9].raw)
		assert.Equal("print.color-default", rtask.tokens[9].color)
		assert.Equal("text", rtask.tokens[10].raw)
		assert.Equal("print.color-default", rtask.tokens[10].color)
		assert.Equal("$r=4d", rtask.tokens[11].raw)
		assert.Equal("print.color-date-r", rtask.tokens[11].color)
		assert.Equal("$every=1m", rtask.tokens[12].raw)
		assert.Equal("print.color-every", rtask.tokens[12].color)

		t.Run("quotes", func(t *testing.T) {
			line := `"t1 t1" "t2 \"t2\" t2" `
			line += `'t3 t3' 't4 \'t4\' t4' `
			line += "`t5 t5` `t6 \\`t6\\` t6` "
			line += `"t7 \"t7 \\\'t7 't7" `
			line += "\"t8 \\\"t8 \\\\\\'t8 't8 \\\\\\`t8 `t8\" "
			line += "\"t9 \\\"t9 \\\\\\'t9 't9 \\\\\\`t9 `t9 ```\""
			task, _ := ParseTask(utils.MkPtr(2), line)
			rtask := task.Render()
			assert.Equal(`"t1 t1"`, rtask.tokens[0].raw)
			assert.Equal("print.quotes.double", rtask.tokens[0].color)
			assert.Equal(`'t2 "t2" t2'`, rtask.tokens[1].raw)
			assert.Equal("print.quotes.double", rtask.tokens[1].color)
			assert.Equal(`'t3 t3'`, rtask.tokens[2].raw)
			assert.Equal("print.quotes.single", rtask.tokens[2].color)
			assert.Equal("`t4 't4' t4`", rtask.tokens[3].raw)
			assert.Equal("print.quotes.single", rtask.tokens[3].color)
			assert.Equal("`t5 t5`", rtask.tokens[4].raw)
			assert.Equal("print.quotes.backticks", rtask.tokens[4].color)
			assert.Equal("\"t6 `t6` t6\"", rtask.tokens[5].raw)
			assert.Equal("print.quotes.backticks", rtask.tokens[5].color)
			assert.Equal("`t7 \"t7 \\\\\\'t7 't7`", rtask.tokens[6].raw)
			assert.Equal("print.quotes.double", rtask.tokens[6].color)
			assert.Equal("```t8 \"t8 \\\\\\'t8 't8 \\\\\\`t8 `t8```", rtask.tokens[7].raw)
			assert.Equal("print.quotes.double", rtask.tokens[7].color)
			assert.Equal("```t9 \"t9 \\\\\\'t9 't9 \\\\\\`t9 `t9 \\`\\`\\````", rtask.tokens[8].raw)
			assert.Equal("print.quotes.double", rtask.tokens[8].color)
		})
		t.Run("escaped quote out of quote", func(t *testing.T) {
			task, _ := ParseTask(utils.MkPtr(3), "1\\\" 2\\' 3\\`")
			rtask := task.Render()
			assert.Equal("1\"", rtask.tokens[0].raw)
			assert.Equal("2'", rtask.tokens[1].raw)
			assert.Equal("3`", rtask.tokens[2].raw)
		})
		t.Run("extra hints", func(t *testing.T) {
			task, _ := ParseTask(&id, "+prj #tag @at !exclamation ?question *star &ampersand")
			rtask := task.Render()
			assert.Equal("+prj", rtask.tokens[0].raw)
			assert.Equal("print.hints.color-plus", rtask.tokens[0].color)
			assert.Equal("#tag", rtask.tokens[1].raw)
			assert.Equal("print.hints.color-tag", rtask.tokens[1].color)
			assert.Equal("@at", rtask.tokens[2].raw)
			assert.Equal("print.hints.color-at", rtask.tokens[2].color)
			assert.Equal("!exclamation", rtask.tokens[3].raw)
			assert.Equal("print.hints.color-exclamation", rtask.tokens[3].color)
			assert.Equal("?question", rtask.tokens[4].raw)
			assert.Equal("print.hints.color-question", rtask.tokens[4].color)
			assert.Equal("*star", rtask.tokens[5].raw)
			assert.Equal("print.hints.color-star", rtask.tokens[5].color)
			assert.Equal("&ampersand", rtask.tokens[6].raw)
			assert.Equal("print.hints.color-ampersand", rtask.tokens[6].color)
		})
		t.Run("anti-priority", func(t *testing.T) {
			task, _ := ParseTask(&id, "[z]")
			rtask := task.Render()
			assert.Equal("[z]", rtask.tokens[0].raw)
			assert.Equal("print.color-anti-priority", rtask.tokens[0].color)
		})
	})
	t.Run("after due", func(t *testing.T) {
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1d $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
		dt := rightNow.Add(-4 * 24 * 60 * 60 * time.Second)
		task.updateDate("due", &dt)
		rtask := task.Render()
		assert.Equal("$due=-4d", rtask.tokens[5].raw)
		assert.Equal("print.color-burnt", rtask.tokens[5].color)
		for _, tk := range rtask.tokens {
			assert.Equal("print.color-burnt", tk.dominantColor)
		}
	})
	t.Run("after due before end", func(t *testing.T) {
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1d $end=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
		dt := rightNow.Add(-4 * 24 * 60 * 60 * time.Second)
		task.updateDate("due", &dt)
		rtask := task.Render()
		assert.Equal("text", rtask.tokens[10].raw)
		assert.Equal("print.color-running-event-text", rtask.tokens[10].color)
		assert.Equal("$end=1w5d", rtask.tokens[6].raw)
		assert.Equal("print.color-running-event", rtask.tokens[6].color)
		assert.Equal("$due=-4d", rtask.tokens[5].raw)
		assert.Equal("print.color-burnt", rtask.tokens[5].color)
	})
	t.Run("after due before dead", func(t *testing.T) {
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
		dt := rightNow.Add(-4 * 24 * 60 * 60 * time.Second)
		task.updateDate("due", &dt)
		rtask := task.Render()
		assert.Equal("$dead=1w5d", rtask.tokens[6].raw)
		assert.Equal("print.color-imminent-deadline", rtask.tokens[6].color)
		assert.Equal("$due=-4d", rtask.tokens[5].raw)
		assert.Equal("print.color-burnt", rtask.tokens[5].color)

		task, _ = ParseTask(&id, "(A) +prj #tag @at $due=1d $dead=2m $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
		rtask = task.Render()
		assert.Equal("print.color-date-dead", rtask.tokens[6].color)
	})
	t.Run("urgent", func(t *testing.T) {
		prep := func(line string) *rTask {
			task, _ := ParseTask(&id, line)
			return task.Render()
		}
		for _, line := range []string{
			"$due=2m", "$end=2m $due=2m", "$dead=2m $due=2m",
		} {
			rtask := prep(line)
			tk := rtask.task.Tokens[0] // pay attention to ordering of dates in line
			assert.Equal("print.color-date-"+tk.Key, rtask.tokens[0].color)
		}
		for _, line := range []string{
			"$due=1w", "$end=1w $due=1w", "$dead=1w $due=1w",
		} {
			rtask := prep(line)
			assert.Equal("print.color-urgent", rtask.tokens[0].color)
		}
		rtask := prep("$urgent")
		assert.Equal("print.color-urgent", rtask.tokens[0].color)
	})
	t.Run("mit", func(t *testing.T) {
		task, _ := ParseTask(&id, "$mit=1")
		rtask := task.Render()
		assert.Equal("$mit=1", rtask.tokens[0].raw)
		assert.Equal("print.color-mit", rtask.tokens[0].color)
	})
}

func TestRenderList(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("test")
	Lists.Empty(path)

	id1 := 0
	task1, _ := ParseTask(&id1, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
	id2 := 1
	task2, _ := ParseTask(&id2, "normal task")
	id3 := 210
	task3, _ := ParseTask(&id3, "tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500/cat")

	Lists.Append(path, task1)
	Lists.Append(path, task2)
	Lists.Append(path, task3)
	rtasks, rinfo, err := RenderList(path)
	assert.NoError(err)
	t.Run("id color", func(t *testing.T) {
		assert.Equal("#64B464", rtasks[0].tokens[8].color)
		assert.Equal("#B48C64", rtasks[0].tokens[9].color)
	})
	t.Run("priority color", func(t *testing.T) {
		assert.Equal("#52E0E0", rtasks[0].tokens[1].color)
	})
	t.Run("lengths", func(t *testing.T) {
		assert.Equal(105, rinfo.maxLen)
		assert.Equal(3, rinfo.idLen)
		assert.Equal(3, rinfo.countLen)
		assert.Equal(4, rinfo.doneCountLen)
	})
	t.Run("id collapse filter", func(t *testing.T) {
		path, _ := parseFilepath("idC")
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
		CleanupRelations(path)
		rtasks, _, err := RenderList(path)
		assert.NoError(err)
		root := func(node *Task) *Task {
			for node.Parent != nil {
				node = node.Parent
			}
			return node
		}
		for _, task := range rtasks {
			if task.task.Norm() != "$-id=1" {
				assert.NotEqual("$-id=1", root(task.task).Norm())
			}
		}
	})
	t.Run("focus", func(t *testing.T) {
		path, _ := parseFilepath("focus")
		Lists.Empty(path)
		AddTaskFromStr("0 $id=1", path)
		AddTaskFromStr("01 $P=1", path)
		AddTaskFromStr("02 $P=1 $focus", path)
		AddTaskFromStr("03 $P=1", path)
		AddTaskFromStr("04 $P=1 $focus", path)
		AddTaskFromStr("05 $P=1", path)
		AddTaskFromStr("06 $id=2", path)
		AddTaskFromStr("07 $P=2 $focus", path)
		AddTaskFromStr("08 $id=3", path)
		AddTaskFromStr("09 $P=3", path)
		AddTaskFromStr("10 $P=3 $focus", path)
		AddTaskFromStr("11 $id=4", path)
		AddTaskFromStr("12 $P=4 $focus", path)
		AddTaskFromStr("13 $P=4", path)
		AddTaskFromStr("14 $id=5", path)
		AddTaskFromStr("15 $P=5", path)
		AddTaskFromStr("16 $P=5", path)
		AddTaskFromStr("17 $P=5", path)
		AddTaskFromStr("18", path)
		AddTaskFromStr("19 $id=19 $focus", path)
		AddTaskFromStr("20 $P=19 $id=20", path)
		AddTaskFromStr("21 $P=20 $id=21", path)
		AddTaskFromStr("22 $P=21 $id=22", path)
		AddTaskFromStr("23 $P=22 $id=23", path)
		AddTaskFromStr("24 $P=23", path)
		AddTaskFromStr("25 $P=23", path)
		AddTaskFromStr("26 $P=23", path)
		AddTaskFromStr("27 $P=23", path)
		AddTaskFromStr("28 $P=22", path)
		AddTaskFromStr("29 $P=20", path)
		AddTaskFromStr("30 $P=19", path)
		AddTaskFromStr("31", path)
		CleanupRelations(path)
		Lists.Sort(path)
		rtasks, listinfo, err := RenderList(path)
		assert.NoError(err)
		for _, rtask := range rtasks {
			rtask.rInfo.set(listinfo)
		}

		assert.Equal("00 0 $id=1", rtasks[0].stringify(false, 50))
		{
			assert.Equal("     ... -1 ...", rtasks[1].stringify(false, 50))
			assert.True(rtasks[1].decor)
			assert.Contains(rtasks[1].stringify(false, 50), "1")
		}
		assert.Equal("   02 02 $P=1 $focus", rtasks[2].stringify(false, 50))
		{
			assert.Equal("     ... -1 ...", rtasks[3].stringify(false, 50))
			assert.True(rtasks[3].decor)
			assert.Contains(rtasks[3].stringify(false, 50), "1")
		}
		assert.Equal("   04 04 $P=1 $focus", rtasks[4].stringify(false, 50))
		{
			assert.Equal("     ... -1 ...", rtasks[5].stringify(false, 50))
			assert.True(rtasks[5].decor)
			assert.Contains(rtasks[5].stringify(false, 50), "1")
		}
		assert.Equal("06 06 $id=2", rtasks[6].stringify(false, 50))
		assert.Equal("   07 07 $P=2 $focus", rtasks[7].stringify(false, 50))
		assert.Equal("08 08 $id=3", rtasks[8].stringify(false, 50))
		{
			assert.Equal("     ... -1 ...", rtasks[9].stringify(false, 50))
			assert.True(rtasks[9].decor)
			assert.Contains(rtasks[9].stringify(false, 50), "1")
		}
		assert.Equal("   10 10 $P=3 $focus", rtasks[10].stringify(false, 50))
		assert.Equal("11 11 $id=4", rtasks[11].stringify(false, 50))
		assert.Equal("   12 12 $P=4 $focus", rtasks[12].stringify(false, 50))
		{
			assert.Equal("     ... -1 ...", rtasks[13].stringify(false, 50))
			assert.True(rtasks[13].decor)
			assert.Contains(rtasks[13].stringify(false, 50), "1")
		}
		{
			assert.Equal("  ... -5 ...", rtasks[14].stringify(false, 50))
			assert.True(rtasks[14].decor)
			assert.Contains(rtasks[14].stringify(false, 50), "5")
		}
		assert.Equal("19 19 $id=19 $focus", rtasks[15].stringify(false, 50))
		assert.Equal("   20 20 $P=19 $id=20", rtasks[16].stringify(false, 50))
		assert.Equal("      21 21 $P=20 $id=21", rtasks[17].stringify(false, 50))
		assert.Equal("         22 22 $P=21 $id=22", rtasks[18].stringify(false, 50))
		assert.Equal("            23 23 $P=22 $id=23", rtasks[19].stringify(false, 50))
		assert.Equal("               24 24 $P=23", rtasks[20].stringify(false, 50))
		assert.Equal("               25 25 $P=23", rtasks[21].stringify(false, 50))
		assert.Equal("               26 26 $P=23", rtasks[22].stringify(false, 50))
		assert.Equal("               27 27 $P=23", rtasks[23].stringify(false, 50))
		assert.Equal("            28 28 $P=22", rtasks[24].stringify(false, 50))
		assert.Equal("      29 29 $P=20", rtasks[25].stringify(false, 50))
		assert.Equal("   30 30 $P=19", rtasks[26].stringify(false, 50))
		{
			assert.Equal("  ... -1 ...", rtasks[27].stringify(false, 50))
			assert.True(rtasks[27].decor)
			assert.Contains(rtasks[27].stringify(false, 50), "1")
		}
	})
}

func TestStringify(t *testing.T) {
	assert := assert.New(t)
	id1 := 12
	testLength := func(out string) bool {
		exceeds := false
		for _, each := range strings.Split(out, "\n") {
			if utils.RuneCount(each) > 50 {
				exceeds = true
			}
		}
		return exceeds
	}
	helper := func(line string) string {
		task, _ := ParseTask(&id1, line)
		rtask := task.Render()
		rtask.idLen = 2
		rtask.doneCountLen = 2
		rtask.countLen = 2
		return rtask.stringify(false, 50)
	}
	out := helper("(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/12/15/cat text $r=-3d $every=1m")
	assert.Equal("12 12/15( 80%) =======>   (unit) (A) +prj #tag @at\n                          $due=1d $dead=1w $r=22'\n                          $id=3 $P=2 text $r=-3d\n                          $every=1m", out)
	assert.Equal("12 ", out[:3], "id")
	assert.Equal("12/15( 80%) =======>   (unit) ", out[3:33], "progress")
	assert.False(testLength(out))
	t.Run("fold", func(t *testing.T) {
		// fits
		out = helper("===========")
		assert.Equal("12 ===========", out)
		assert.NotContains(out, "\n")
		assert.False(testLength(out))
		// string so long it has to be split
		out = helper("=============================================================================================================================")
		assert.Equal("12 ==============================================\\\n   ==============================================\\\n   =================================", out)
		assert.Contains(out, "\\")
		assert.Equal(2, strings.Count(out, "\\"))
		assert.Contains(out, "\n")
		assert.Equal(2, strings.Count(out, "\n"))
		assert.False(testLength(out))
		// line has no space whatsoever
		// string so long it has to be split
		// str is long enough
		out = helper("one two three four five six seven eight nine ten eleven ============================================================= twelve thirteen fourteen sixteen seventeen eighteen nineteen twenty twenty-one")
		assert.Equal("12 one two three four five six seven eight nine\n   ten eleven ===================================\\\n   ========================== twelve thirteen\n   fourteen sixteen seventeen eighteen nineteen\n   twenty twenty-one", out)
		assert.Contains(out, "nine\n   ten")
		assert.Contains(out, "thirteen\n   fourteen")
		assert.Contains(out, "nineteen\n   twenty")
		assert.Contains(out, "===================================\\\n   ==========================")
		assert.False(testLength(out))
	})
	t.Run("task depth", func(t *testing.T) {
		path, _ := parseFilepath("test")
		Lists.Empty(path)
		AddTaskFromStr("0 no id", path)
		AddTaskFromStr("1 $id=1", path)
		AddTaskFromStr("2 $id=2 $P=1", path)
		AddTaskFromStr("3 $id=3 $P=1", path)
		AddTaskFromStr("4 $id=4 $P=3", path)
		AddTaskFromStr("5 $id=5", path)
		AddTaskFromStr("6 $id=6 $P=5", path)
		AddTaskFromStr("7 $id=7 $P=6", path)
		AddTaskFromStr("8 no id", path)
		AddTaskFromStr("9 $P=7 =============================================================================================================================", path)
		AddTaskFromStr("10 $P=7 one two three four five six seven eight nine ten eleven ============================================================= twelve thirteen fourteen sixteen seventeen eighteen nineteen twenty twenty-one", path)
		AddTaskFromStr("11 $P=7 =============================================================", path)
		AddTaskFromStr("12 =============================================================================================================================", path)
		AddTaskFromStr("13 one two three four five six seven eight nine ten eleven ============================================================= twelve thirteen fourteen sixteen seventeen eighteen nineteen twenty twenty-one", path)
		AddTaskFromStr("14 =============================================================", path)
		CleanupRelations(path)
		Lists.Sort(path)
		helper := func(ndx int) string {
			task := Lists[path].Tasks[ndx]
			rtask := task.Render()
			rtask.idLen = 2
			str := rtask.stringify(false, 50)
			assert.False(testLength(str))
			return str
		}
		assert.Equal("00 0 no id", helper(0))
		assert.Equal("01 1 $id=1", helper(1))
		assert.Equal("   02 2 $id=2 $P=1", helper(2))
		assert.Equal("   03 3 $id=3 $P=1", helper(3))
		assert.Equal("      04 4 $id=4 $P=3", helper(4))
		assert.Equal("12 12 ===========================================\\\n   ==============================================\\\n   ====================================", helper(5))
		assert.Equal("13 13 one two three four five six seven eight nine\n   ten eleven ===================================\\\n   ========================== twelve thirteen\n   fourteen sixteen seventeen eighteen nineteen\n   twenty twenty-one", helper(6))
		assert.Equal("14 14 ===========================================\\\n   ==================", helper(7))
		assert.Equal("05 5 $id=5", helper(8))
		assert.Equal("   06 6 $id=6 $P=5", helper(9))
		assert.Equal("      07 7 $id=7 $P=6", helper(10))
		assert.Equal("         10 10 $P=7 one two three four five six\n            seven eight nine ten eleven =========\\\n            =====================================\\\n            =============== twelve thirteen\n            fourteen sixteen seventeen eighteen\n            nineteen twenty twenty-one", helper(11))
		assert.Equal("         11 11 $P=7 =============================\\\n            ================================", helper(12))
		assert.Equal("         09 9 $P=7 ==============================\\\n            =====================================\\\n            =====================================\\\n            =====================", helper(13))
		assert.Equal("08 8 no id", helper(14))
	})
	t.Run("id collapse", func(t *testing.T) {
		path, _ := parseFilepath("test")
		Lists.Empty(path)
		AddTaskFromStr("(testing) heyto $due=1w $-id=first $P=dead", path)
		AddTaskFromStr("$id=1", path)
		AddTaskFromStr("(testing) heyto $due=1w $P=1 $-id=second", path)
		AddTaskFromStr("$P=second 1", path)
		AddTaskFromStr("$P=second 2", path)
		AddTaskFromStr("$P=second 3", path)
		CleanupRelations(path)

		task := Lists[path].Tasks[0]
		rtask := task.Render()
		rtask.idLen = 2
		str := rtask.stringify(false, 50)
		assert.False(testLength(str))
		assert.Equal("00 + (testing) heyto $due=1w $-id=first $P=dead", str)

		task = Lists[path].Tasks[2]
		rtask = task.Render()
		rtask.idLen = 2
		str = rtask.stringify(false, 50)
		assert.False(testLength(str))
		assert.Equal("   02 +|3 (testing) heyto $due=1w $P=1 $-id=second", str)
	})
	t.Run("progress fold", func(t *testing.T) {
		path, _ := parseFilepath("test")
		Lists.Empty(path)
		AddTaskFromStr("(6) #Literature #classics +ugliness @y:1831 #rate:4.02/8k/211k @auth:Victor-Hugo The Hunchback of Notre-Dame $p=page/165/510/books $c=2025-05-17T17-06-25", path)
		CleanupRelations(path)
		task := Lists[path].Tasks[0]
		rtask := task.Render()
		rtask.idLen = 2
		rtask.countLen = 3
		rtask.doneCountLen = 3
		str := rtask.stringify(false, 50)
		assert.False(testLength(str))
	})
	t.Run("nested progress", func(t *testing.T) {
		path, _ := parseFilepath("test")
		Lists.Empty(path)
		AddTaskFromStr("test 0 $id=0", path)
		AddTaskFromStr("test 1 $P=0 $p=unit/1/3", path)
		AddTaskFromStr("(testing) heyto $due=1w $P=1 $-id=second", path)
		AddTaskFromStr("$P=second 1", path)
		AddTaskFromStr("$P=second 2", path)
		AddTaskFromStr("$P=second 3", path)
		CleanupRelations(path)

		task := Lists[path].Tasks[1]
		rtask := task.Render()
		rtask.idLen = 2
		rtask.countLen = 5
		rtask.doneCountLen = 6
		str := rtask.stringify(false, 50)
		assert.False(testLength(str))
		assert.Equal("   01 1/3( 33%) ==>        (unit) test 1 $P=0", str)
	})
	t.Run("skipping terminations", func(t *testing.T) {
		path, _ := parseFilepath("test")
		Lists.Empty(path)
		AddTaskFromStr(" this  #tag\\;.  a  b  c \\; d ", path)
		task := Lists[path].Tasks[0]
		rtask := task.Render()
		rtask.idLen = 1
		str := rtask.stringify(false, 50)
		assert.False(testLength(str))
		assert.Equal("0  this  #tag.  a  b  c  d", str)
	})
	t.Run("focus", func(t *testing.T) {
		path, _ := parseFilepath("focus")
		Lists.Empty(path)
		AddTaskFromStr("1 $focus", path)
		AddTaskFromStr("2", path)
		AddTaskFromStr("3 $focus", path)
		rtasks, _, err := RenderList(path)
		assert.NoError(err)
		assert.False(rtasks[0].decor)
		assert.False(rtasks[2].decor)
		assert.True(rtasks[1].decor)
		assert.Equal("... -1 ...", rtasks[1].stringify(false, 50))
		assert.Equal("print.color-hidden", rtasks[1].tokens[0].color)
	})
}

func TestPrintLists(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("printLists")
	Lists.Empty(path)

	id1 := 0
	task1, _ := ParseTask(&id1, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
	id2 := 1
	task2, _ := ParseTask(&id2, "normal task")
	id3 := 210
	task3, _ := ParseTask(&id3, "tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500")

	capture := func(maxlen, minlen int) []string {
		realStdout := os.Stdout
		defer func() { os.Stdout = realStdout }()
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w
		err = PrintLists([]string{path}, maxlen, minlen)
		require.Nil(t, err)
		w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		out := buf.String()
		return strings.Split(out, "\n")
	}

	t.Run("header len", func(t *testing.T) {
		Lists.Empty(path, task2)
		out := capture(80, 70)
		assert.Equal(70, utils.RuneCount(out[0]))
		out = capture(90, 70)
		assert.Equal(70, utils.RuneCount(out[0]))
		Lists.Empty(path, task1)
		out = capture(90, 70)
		assert.Equal(90, utils.RuneCount(out[0]))
		out = capture(160, 70)
		assert.Equal(105, utils.RuneCount(out[0]))
		out = capture(130, 70)
		assert.Equal(105, utils.RuneCount(out[0]))
	})
	t.Run("line len", func(t *testing.T) {
		Lists.Empty(path, task1, task2, task3)
		out := capture(50, 10)
		assert.Equal(50, utils.RuneCount(out[0])) // header
		assert.Equal(50, utils.RuneCount(out[1])) // category header

		assert.Equal(50, utils.RuneCount(out[2]))
		assert.Equal(46, utils.RuneCount(out[3]))
		assert.Equal(47, utils.RuneCount(out[4]))

		assert.Equal(39, utils.RuneCount(out[6]))
		assert.Equal(50, utils.RuneCount(out[7]))
		assert.Equal(50, utils.RuneCount(out[8]))

		assert.Equal(50, utils.RuneCount(out[9])) // category header
		assert.Equal(50, utils.RuneCount(out[10]))
	})
	t.Run("category headers", func(t *testing.T) {
		Lists.Empty(path, task1, task2, task3)
		out := capture(50, 10)
		assert.Equal(50, utils.RuneCount(out[1]))
		assert.Equal(41, utils.RuneCount(out[5]))
		assert.Equal(50, utils.RuneCount(out[9]))
	})
	t.Run("nested progress and category headers", func(t *testing.T) {
		Lists.Empty(path)
		AddTaskFromStr("0 $id=0 $p=unit/2/5", path)
		AddTaskFromStr("1 $P=0", path)
		AddTaskFromStr("2 $P=0 $p=unit/3/7", path)
		AddTaskFromStr("3 $P=0 $p=unit/4/8/b", path)
		AddTaskFromStr("4", path)
		AddTaskFromStr("5 $id=5 $p=unit/3/10/c", path)
		AddTaskFromStr("6 $P=5 $id=6 $p=unit/4/8/c", path)
		AddTaskFromStr("7 $P=5 $id=7 $p=unit/4/10/d", path)
		AddTaskFromStr("8 $P=7 $p=unit/5/15/z", path)
		AddTaskFromStr("9 $P=6", path)
		AddTaskFromStr("a $id=a $p=unit/5/20/a", path)
		AddTaskFromStr("b $P=b", path)
		AddTaskFromStr("c $P=a $p=unit/5/100/b", path)
		AddTaskFromStr("d $P=a $p=unit/70/100/a", path)
		AddTaskFromStr("e $P=a $p=unit/80/150", path)

		out := capture(60, 50)
		tc := `> printLists | ———————————————————————————————————
                         a ———————————————————————
10  5/ 20( 25%) =>         (unit) a $id=a
   13 70/100( 70%) ======>    (unit) d $P=a
   12 5/100(  5%)            (unit) c $P=a
   14 80/150( 53%) ====>      (unit) e $P=a
                         c ———————————————————————
05  3/ 10( 30%) ==>        (unit) 5 $id=5
   06 4/8( 50%) ====>      (unit) 6 $P=5 $id=6
      09 9 $P=6
   07 4/10( 40%) ===>       (unit) 7 $P=5 $id=7
      08 5/15( 33%) ==>        (unit) 8 $P=7
                         * ———————————————————————
00  2/  5( 40%) ===>       (unit) 0 $id=0
   03 4/8( 50%) ====>      (unit) 3 $P=0
   02 3/7( 42%) ===>       (unit) 2 $P=0
   01 1 $P=0
                           ———————————————————————
04 4
11 b $P=b`
		for ndx, line := range strings.Split(tc, "\n") {
			line = strings.TrimRightFunc(line, unicode.IsSpace)
			assert.Equal(line, out[ndx])
		}
	})
	t.Run("focus", func(t *testing.T) {
		Lists.Empty(path)
		AddTaskFromStr("0 $id=1 $focus", path)
		AddTaskFromStr("01 $P=1", path)
		AddTaskFromStr("02 $P=1 $focus", path)
		AddTaskFromStr("03 $P=1", path)
		AddTaskFromStr("04 $P=1 $focus", path)
		AddTaskFromStr("05 $P=1", path)
		AddTaskFromStr("06 $id=2 $focus", path)
		AddTaskFromStr("07 $P=2 $focus", path)
		AddTaskFromStr("08 $id=3 $focus", path)
		AddTaskFromStr("09 $P=3", path)
		AddTaskFromStr("10 $P=3 $focus", path)
		AddTaskFromStr("11 $id=4 $focus", path)
		AddTaskFromStr("12 $P=4 $focus", path)
		AddTaskFromStr("13 $P=4", path)
		AddTaskFromStr("14 $id=5", path)
		AddTaskFromStr("15 $P=5", path)
		AddTaskFromStr("16 $P=5", path)
		AddTaskFromStr("17 $P=5", path)
		AddTaskFromStr("18", path)
		AddTaskFromStr("19 $focus", path)

		out := capture(60, 50)
		tc := `> printLists | ———————————————————————————————————
00 0 $id=1 $focus
     ... -1 ...
   02 02 $P=1 $focus
     ... -1 ...
   04 04 $P=1 $focus
     ... -1 ...
06 06 $id=2 $focus
   07 07 $P=2 $focus
08 08 $id=3 $focus
     ... -1 ...
   10 10 $P=3 $focus
11 11 $id=4 $focus
   12 12 $P=4 $focus
     ... -1 ...
  ... -5 ...
19 19 $focus`
		for ndx, line := range strings.Split(tc, "\n") {
			line = strings.TrimRightFunc(line, unicode.IsSpace)
			assert.Equal(line, out[ndx])
		}
	})
}

func TestPrintTask(t *testing.T) {
	assert := assert.New(t)

	id1 := 0
	task1, _ := ParseTask(&id1, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
	id2 := 1
	task2, _ := ParseTask(&id2, "normal task")
	id3 := 210
	task3, _ := ParseTask(&id3, "tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500")
	path, _ := parseFilepath("printTask")
	Lists.Empty(path, task1, task2, task3)

	capture := func(id int) string {
		realStdout := os.Stdout
		defer func() { os.Stdout = realStdout }()
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w
		err = PrintTask(id, path, 80)
		require.Nil(t, err)
		w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		return buf.String()
	}

	out := capture(210)
	assert.Equal("210 223/3500(  6%)            (unit) tooooooooooooooooooooooooooooooooooooo\n                              looooooooooooooooooong\n", out)
	out = capture(0)
	assert.Equal("0 2/15( 13%) >          (unit) (A) +prj #tag @at $due=1d $dead=1w $r=22' $id=3\n                        $P=2 text $r=-3d $every=1m\n", out)
}

func TestOutputTask(t *testing.T) {
	assert := assert.New(t)

	id1 := 0
	task1, _ := ParseTask(&id1, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
	id2 := 1
	task2, _ := ParseTask(&id2, "normal task")
	id3 := 210
	task3, _ := ParseTask(&id3, "tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500")
	path, _ := parseFilepath("printTask")
	Lists.Empty(path, task1, task2, task3)
	rn := unparseAbsoluteDatetime(rightNow)

	capture := func(id int) string {
		realStdout := os.Stdout
		defer func() { os.Stdout = realStdout }()
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w
		err = OutputTask(id, path)
		require.Nil(t, err)
		w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		return buf.String()
	}

	out := capture(210)
	assert.Equal(fmt.Sprintf("tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500 $c=%s\n", rn), out)
	out = capture(0)
	assert.Equal(fmt.Sprintf("(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m $c=%s\n", rn), out)
}
