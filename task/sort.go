package task

import (
	"dotxt/utils"
	"slices"
	"strings"
	"time"
)

// pay good attention to the values
func sortNil[T any](lv, rv *T) int {
	if lv != nil && rv == nil {
		return -1
	} else if lv == nil && rv != nil {
		return 1
	} else if lv == nil && rv == nil {
		return 2
	}
	return 3
}

func sortString(lv, rv string) int {
	lvn, rvn := utils.RuneCount(lv), utils.RuneCount(rv)
	if lvn != 0 && rvn == 0 {
		return -1
	} else if lvn == 0 && rvn != 0 {
		return 1
	} else if lv < rv {
		return -1
	} else if lv > rv {
		return 1
	}
	return 2
}

func sortId(lv, rv *int) int {
	if v := sortNil(lv, rv); v != 3 {
		return v
	}
	if *lv < *rv {
		return -1
	} else if *lv > *rv {
		return 1
	}
	return 2
}

func sortID(lv, rv *string) int {
	if v := sortNil(lv, rv); v != 3 {
		return v
	}
	if *lv < *rv {
		return -1
	} else if *lv > *rv {
		return 1
	}
	return 2
}

func sortTime(lv, rv *time.Time) int {
	if v := sortNil(lv, rv); v != 3 {
		return v
	}
	if lva, rva := lv.After(rightNow), rv.After(rightNow); lva && !rva {
		return -1
	} else if !lva && rva {
		return 1
	}
	if lv.Before(*rv) {
		return -1
	} else if lv.After(*rv) {
		return 1
	}
	return 2
}

func sortUrgentTime(lv, rv *time.Time) int {
	if lv == nil && rv == nil {
		return 2
	}
	lvu, rvu := IsDateUrgent(lv), IsDateUrgent(rv)
	if lvu && !rvu {
		return -1
	} else if !lvu && rvu {
		return 1
	} else if !lvu && !rvu {
		return 2
	} else if lv.Before(*rv) {
		return -1
	} else if lv.After(*rv) {
		return 1
	} else {
		return 2
	}
}

func sortUrgency(l, r *Task) int {
	if v := sortNil(l.MIT, r.MIT); v == -1 || v == 1 {
		return v
	} else if v == 3 {
		if *l.MIT < *r.MIT {
			return -1
		} else if *l.MIT > *r.MIT {
			return 1
		}
	}

	// TODO: (sort) in the case that DueDate is urgent and has not passed,
	//  DueDate takes precedence over End and Deadline
	//  only in the case that DueDate has been passed, End and Dead matter
	earliest := func(l, r *time.Time) *time.Time {
		if l == nil {
			return r
		} else if r == nil {
			return l
		} else if l.Before(*r) || l.Equal(*r) {
			return l
		} else {
			return r
		}
	}
	lED, rED :=
		earliest(l.Time.EndDate, l.Time.Deadline),
		earliest(r.Time.EndDate, r.Time.Deadline)

	if v := sortUrgentTime(lED, rED); v != 2 {
		return v
	} else if v := sortUrgentTime(l.Time.DueDate, r.Time.DueDate); v != 2 {
		return v
	}
	if l.Urgent == r.Urgent {
		return 2
	} else if l.Urgent {
		return -1
	} else {
		return 1
	}
}

func sortReminders(lr, rr []*time.Time) int {
	if len(lr) != 0 && len(rr) == 0 {
		return -1
	} else if len(lr) == 0 && len(rr) != 0 {
		return 1
	} else if len(lr) == 0 && len(rr) == 0 {
		return 2
	}
	lr = slices.Clone(lr)
	rr = slices.Clone(rr)
	slices.SortFunc(lr, func(l, r *time.Time) int {
		if v := sortTime(l, r); v != 2 {
			return v
		}
		return 0
	})
	slices.SortFunc(rr, func(l, r *time.Time) int {
		if v := sortTime(l, r); v != 2 {
			return v
		}
		return 0
	})
	minLen := min(len(lr), len(rr))
	for ndx := range minLen {
		if v := sortTime(lr[ndx], rr[ndx]); v != 2 {
			return v
		}
	}
	if len(lr) > len(rr) {
		return -1
	} else if len(lr) < len(rr) {
		return 1
	}
	return 2
}

func sortDatetime(l, r *Task) int {
	if v := sortTime(l.Time.EndDate, r.Time.EndDate); v != 2 {
		return v
	} else if v := sortTime(l.Time.Deadline, r.Time.Deadline); v != 2 {
		return v
	} else if v := sortTime(l.Time.DueDate, r.Time.DueDate); v != 2 {
		return v
	} else if v := sortTime(l.Time.CreationDate, r.Time.CreationDate); v != 2 { // this is pointless lol
		return v
	} else if v := sortReminders(l.Time.Reminders, r.Time.Reminders); v != 2 {
		return v
	}
	return 2
}

func sortDuration(lv, rv *time.Duration) int {
	if v := sortNil(lv, rv); v != 3 {
		return v
	}
	if *lv < *rv {
		return -1
	} else if *lv > *rv {
		return 1
	}
	return 2
}

func sortProgressValue(l, r *Task) int {
	if v := sortNil(l.Prog, r.Prog); v != 3 {
		return v
	}
	lP, rP := 100*l.Prog.Count/l.Prog.DoneCount, 100*r.Prog.Count/r.Prog.DoneCount
	if lP > rP {
		return -1
	} else if lP < rP {
		return 1
	}
	if l.Prog.DoneCount > r.Prog.DoneCount {
		return -1
	} else if l.Prog.DoneCount < r.Prog.DoneCount {
		return 1
	}
	return 2
}

func sortProgressCategory(l, r *Task) int {
	if v := sortNil(l.Prog, r.Prog); v != 3 {
		return v
	}
	return sortString(l.Prog.Category, r.Prog.Category)
}

func sortPriority(l, r *Task) int {
	if v := sortNil(l.Priority, r.Priority); v != 3 {
		return v
	}
	return sortString(*l.Priority, *r.Priority)
}

func sortAntiPriority(l, r *Task) int {
	if l.Priority == nil && r.Priority == nil {
		return 2
	}
	var ltk, rtk *Token
	if l.Priority != nil {
		ltk, _ = l.Tokens.Find(TkByTypeKey(TokenPriority, "anti-priority"))
	}
	if r.Priority != nil {
		rtk, _ = r.Tokens.Find(TkByTypeKey(TokenPriority, "anti-priority"))
	}

	if v := sortNil(rtk, ltk); v != 3 { // given in reverse so that having anti-priority means going down
		return v
	}
	return sortString(*l.Priority, *r.Priority)
}

func sortHints(l, r *Task) int {
	extractPlus := func(hints []*string) (int, string, int, string) {
		var out []string
		var rest []string
		for _, h := range hints {
			if utils.RuneCount(*h) > 0 && utils.RuneAt(*h, 0) == '+' {
				out = append(out, *h)
			} else {
				rest = append(rest, *h)
			}
		}
		slices.Sort(out)
		return len(out), strings.Join(out, ","), len(rest), strings.Join(rest, ",")
	}
	lPlusLen, lPlus, lHintsLen, lHints := extractPlus(l.Hints)
	rPlusLen, rPlus, rHintsLen, rHints := extractPlus(r.Hints)
	if lPlusLen > 0 && rPlusLen == 0 {
		return -1
	} else if lPlusLen == 0 && rPlusLen > 0 {
		return 1
	} else if lPlusLen > 0 && rPlusLen > 0 && lPlus < rPlus {
		return -1
	} else if lPlusLen > 0 && rPlusLen > 0 && lPlus > rPlus {
		return 1
	} else if lHintsLen > 0 && rHintsLen == 0 {
		return -1
	} else if lHintsLen == 0 && rHintsLen > 0 {
		return 1
	} else if lHintsLen > 0 && rHintsLen > 0 && lHints < rHints {
		return -1
	} else if lHintsLen > 0 && rHintsLen > 0 && lHints > rHints {
		return 1
	}
	return 2
}

func sortText(l, r *Task) int {
	lText, rText := l.NormRegular(), r.NormRegular()
	return sortString(lText, rText)
}

func sortChildren(l, r *Task) int {
	if len(l.Children) > 0 && len(r.Children) == 0 {
		return -1
	} else if len(l.Children) == 0 && len(r.Children) > 0 {
		return 1
	} else if len(l.Children) > 0 && len(r.Children) > 0 {
		lndx, rndx := 0, 0
		for lndx < len(l.Children) && rndx < len(r.Children) {
			if v := sortHelper(l.Children[lndx], r.Children[rndx]); v != 0 {
				return v
			}
			lndx++
			rndx++
		}
	}
	return 2
}

func sortHelper(l, r *Task) int {
	if v := sortNil(l, r); v != 3 {
		if v == 2 {
			return 0
		}
		return v
	} else if v = sortProgressCategory(l, r); v != 2 {
		return v
	} else if v = sortAntiPriority(l, r); v != 2 {
		return v
	} else if v = sortUrgency(l, r); v != 2 {
		return v
	} else if v = sortPriority(l, r); v != 2 {
		return v
	} else if v = sortProgressValue(l, r); v != 2 {
		return v
	} else if v = sortHints(l, r); v != 2 {
		return v
	} else if v = sortText(l, r); v != 2 {
		return v
	} else if v = sortDatetime(l, r); v != 2 {
		return v
	} else if v = sortDuration(l.Time.Every, r.Time.Every); v != 2 {
		return v
	} else if v = sortID(l.EID, r.EID); v != 2 {
		return v
	} else if v = sortID(l.PID, r.PID); v != 2 {
		return v
	} else if v = sortId(l.ID, r.ID); v != 2 {
		return v
	} else if v = sortChildren(l, r); v != 2 {
		return v
	}
	return 0
}

func sortTasks(tasks []*Task) []*Task {
	for ndx := len(tasks) - 1; ndx >= 0; ndx-- {
		if tasks[ndx].Parent != nil {
			tasks = slices.Delete(tasks, ndx, ndx+1)
		}
	}
	slices.SortFunc(tasks, sortHelper)
	var dfs func(int)
	dfs = func(ndx int) {
		children := slices.Clone(tasks[ndx].Children)
		slices.SortFunc(children, sortHelper)
		for sndx := len(children) - 1; sndx >= 0; sndx-- {
			tasks = slices.Insert(tasks, ndx+1, children[sndx])
			if len(children[sndx].Children) > 0 {
				dfs(ndx + 1)
			}
		}
	}
	for ndx := len(tasks) - 1; ndx >= 0; ndx-- {
		if len(tasks[ndx].Children) > 0 {
			dfs(ndx)
		}
	}
	return tasks
}
