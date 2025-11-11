package task

import (
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"slices"
	"sort"
	"strings"
)

func CleanupIDs(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	n := Lists.Len(path)
	ndxs := make(map[int]bool)
	for _, task := range Lists[path].Tasks {
		if task.ID != nil {
			if _, ok := ndxs[*task.ID]; *task.ID >= n || *task.ID < 0 || ok {
				task.ID = nil
			}
		}
		if task.ID != nil {
			ndxs[*task.ID] = true
		}
	}
	var stack []int
	for ndx := range n {
		_, ok := ndxs[ndx]
		if !ok {
			stack = append(stack, ndx)
		}
	}
	for _, task := range Lists[path].Tasks {
		if task.ID == nil {
			task.ID = utils.MkPtr(stack[0])
			stack = stack[1:]
		}
	}

	idSortFunc := func(l, r *Task) int {
		if *l.ID < *r.ID {
			return -1
		} else if *l.ID > *r.ID {
			return 1
		} else {
			return 0
		}
	}
	Lists.SortFunc(path, idSortFunc)
	return nil
}

func CleanupRelations(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	Lists[path].EIDs = make(map[string]*Task)
	Lists[path].PIDs = make(map[*Task]string)
	for _, task := range Lists[path].Tasks {
		if task.EID != nil {
			_, ok := Lists[path].EIDs[*task.EID]
			if ok {
				task.revertIDtoText("id")
			} else {
				Lists[path].EIDs[*task.EID] = task
			}
		}
		{ // find loop
			met := make(map[*Task]bool)
			node := task
			for node != nil && node.PID != nil {
				_, ok := met[node]
				if ok {
					node.revertIDtoText("P")
					break
				}
				met[node] = true
				node, ok = Lists[path].EIDs[*node.PID]
				if !ok {
					node = nil
				}
			}
		}
		if task.PID != nil {
			Lists[path].PIDs[task] = *task.PID
		}
		task.Children = make([]*Task, 0)
		task.Parent = nil
	}
	for task, pid := range Lists[path].PIDs {
		parent, ok := Lists[path].EIDs[pid]
		if !ok {
			continue
		}
		task.Parent = parent
		parent.Children = append(parent.Children, task)
	}
	for _, task := range Lists[path].Tasks {
		// note: this induced urgency is imperfect;
		//  since in sorting, among urgent tasks, the value
		//  of the $end, $dead, and $due are taken into account.
		//  whereas if a task has a sooner, say, $end value,
		//  if it is a child of an urgency-induced task,
		//  then it won't be prioritized correctly.
		//  same thing happens for values of '$mit='.
		// but this flaw is miniscule and therefore negligable.
		if task.IsUrgent() {
			node := task
			for node != nil {
				node.Urgent = true
				node = node.Parent
			}
		}
	}
	return nil
}

func AddTask(task *Task, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	Lists.Append(path, task)
	CleanupIDs(path)
	CleanupRelations(path)
	return nil
}

func AddTaskFromStr(task, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	t, err := ParseTask(nil, task)
	if err != nil {
		return err
	}
	return AddTask(t, path)
}

func getTaskIndexFromId(id int, path string) (int, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return -1, err
	}
	taskNdx := -1
	for ndx, t := range Lists[path].Tasks {
		if *t.ID == id {
			taskNdx = ndx
		}
	}
	if taskNdx == -1 {
		return taskNdx, fmt.Errorf("%w: task id '%d'", terrors.ErrNotFound, id)
	}
	return taskNdx, nil
}

func getTaskFromId(id int, path string) (*Task, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return nil, err
	}
	taskNdx, err := getTaskIndexFromId(id, path)
	if err != nil {
		return nil, err
	}
	return Lists[path].Tasks[taskNdx], nil
}

func AppendToTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	err = task.updateByModifyingText("", text)
	if err != nil {
		return err
	}
	task.ID = &id // TODO: why is this necessary?
	CleanupRelations(path)
	return nil
}

func PrependToTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	err = task.updateByModifyingText(text, "")
	if err != nil {
		return err
	}
	task.ID = &id // TODO: why is this necessary?
	CleanupRelations(path)
	return nil
}

func ReplaceTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	err = task.updateFromText(text)
	if err != nil {
		return err
	}
	task.ID = &id // TODO: why is this necessary?
	CleanupRelations(path)
	return nil
}

func DeduplicateList(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	var indexes []int
	var lines map[string][]int = make(map[string][]int)
	for ndx, task := range Lists[path].Tasks {
		taskNorm := task.Norm()
		_, ok := lines[taskNorm]
		if !ok {
			lines[taskNorm] = []int{ndx}
		} else {
			lines[taskNorm] = append(lines[taskNorm], ndx)
		}
	}
	for _, bucket := range lines {
		if len(bucket) > 1 {
			indexes = append(indexes, bucket[1:]...)
		}
	}
	return DeleteTasks(indexes, path)
}

func DeprioritizeTask(id int, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	if task.Priority == nil {
		return nil
	}

	_, ndx := task.Tokens.Find(TkPriorityPrefix)
	task.Tokens = slices.Delete(task.Tokens, ndx, ndx+1)
	task.Priority = nil
	return nil
}

func PrioritizeTask(id int, priority, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	priority = strings.TrimSpace(priority)
	n := utils.RuneCount(priority)
	if n > 0 && !strings.ContainsRune("[(", utils.RuneAt(priority, 0)) {
		priority = "(" + priority
		n += 1
	}
	if n > 0 && !strings.ContainsRune(")]", utils.RuneAt(priority, n-1)) {
		if utils.RuneAt(priority, 0) == '[' {
			priority = priority + "]"
		} else {
			priority = priority + ")"
		}
	}
	err = validatePriority(priority)
	if err != nil {
		return err
	}

	task.Priority = &priority
	tk, _ := task.Tokens.Find(TkByTypeKey(TokenPriority, "priority"))
	if tk != nil {
		tk.raw = &priority
		tk.Value = &priority
	} else {
		task.Tokens = append([]*Token{{
			Type: TokenPriority, Key: "priority",
			raw: &priority, Value: &priority,
		}}, task.Tokens...)
	}
	return nil
}

// The indexes are sorted in decreasing order
func getIndexesFromIds(ids []int, path string) ([]int, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return []int{}, err
	}
	if len(ids) == 0 {
		return []int{}, nil
	}

	var idsMap map[int]bool = make(map[int]bool)
	for _, val := range ids {
		idsMap[val] = true
	}
	var indexes []int
	for ndx, task := range Lists[path].Tasks {
		if task.ID == nil {
			continue
		}
		if _, ok := idsMap[*task.ID]; ok {
			indexes = append(indexes, ndx)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(indexes)))
	if len(indexes) != len(idsMap) {
		var ndxMap map[int]bool = make(map[int]bool)
		for _, val := range indexes {
			ndxMap[val] = true
		}
		var notFound []string
		for ndx := range idsMap {
			if _, ok := ndxMap[ndx]; !ok {
				notFound = append(notFound, fmt.Sprintf("%d", ndx))
			}
		}
		return []int{}, fmt.Errorf("%w: task ids '%s'", terrors.ErrNotFound, strings.Join(notFound, ", "))
	}
	return indexes, nil
}

func DeleteTasks(ids []int, path string) error {
	indexes, err := getIndexesFromIds(ids, path)
	if err != nil {
		return err
	}
	for _, ndx := range indexes {
		Lists.DeleteTasks(path, ndx, ndx+1)
	}
	err = CleanupIDs(path)
	if err != nil {
		return err
	}
	CleanupRelations(path)
	return nil
}

func DoneTask(ids []int, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	indexes, err := getIndexesFromIds(ids, path)
	if err != nil {
		return err
	}
	var tasks []*Task
	for _, ndx := range indexes {
		tasks = append(tasks, Lists[path].Tasks[ndx])
		Lists.DeleteTasks(path, ndx, ndx+1)
	}
	CleanupIDs(path)
	CleanupRelations(path)

	var out []string
	for _, task := range tasks {
		if task.Fmt != nil && task.Fmt.Focus {
			task.Fmt.Focus = false
			_, ndx := task.Tokens.Find(TkByTypeKey(TokenFormat, "focus"))
			task.Tokens = slices.Delete(task.Tokens, ndx, ndx+1)
		}
		out = append(out, task.Raw())
	}
	return appendToDoneFile(strings.Join(out, "\n"), path)
}

func MoveTask(from string, id int, to string) error {
	taskNdx, err := getTaskIndexFromId(id, from)
	if err != nil {
		return err
	}
	from, err = prepFileTaskFromPath(from)
	if err != nil {
		return err
	}
	to, err = prepFileTaskFromPath(to)
	if err != nil {
		return err
	}

	Lists.Append(to, Lists[from].Tasks[taskNdx])
	CleanupIDs(to)
	CleanupRelations(to)
	Lists.DeleteTasks(from, taskNdx, taskNdx+1)
	CleanupIDs(from)
	CleanupRelations(from)
	return nil
}

func RevertTask(ids []int, path string) error {
	texts, err := removeFromDoneFile(ids, path)
	if err != nil {
		return err
	}
	path, err = prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	for _, text := range texts {
		task, err := ParseTask(utils.MkPtr(Lists.Len(path)), text)
		if err != nil {
			return err
		}
		Lists.Append(path, task)
	}
	CleanupRelations(path)
	return nil
}

func MigrateTasks(from, to string) error {
	tasks, err := taskifyRandomFile(from)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if err := AddTask(&t, to); err != nil {
			return err
		}
	}
	return nil
}

func IncrementProgressCount(id int, path string, value int) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	if task.Prog == nil {
		return fmt.Errorf("%w: task with id '%d' has no progress", terrors.ErrValue, id)
	}
	rVal := task.Prog.Count + value
	task.Prog.Count = max(min(rVal, task.Prog.DoneCount), 0)
	progText, err := unparseProgress(*task.Prog)
	if err != nil {
		return err
	}
	progText = fmt.Sprintf("$p=%s", progText)
	pToken, _ := task.Tokens.Find(TkByType(TokenProgress))
	if pToken == nil {
		return fmt.Errorf("%w: task with id '%d' has no progress", terrors.ErrValue, id)
	}
	*pToken.raw = progText
	return nil
}

func SetProgressCount(id int, path string, value int) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	if task.Prog == nil {
		return fmt.Errorf("%w: task with id '%d' has no progress", terrors.ErrValue, id)
	}
	task.Prog.Count = max(min(value, task.Prog.DoneCount), 0)
	progText, err := unparseProgress(*task.Prog)
	if err != nil {
		return err
	}
	progText = fmt.Sprintf("$p=%s", progText)
	pToken, _ := task.Tokens.Find(TkByType(TokenProgress))
	if pToken == nil {
		return fmt.Errorf("%w: task with id '%d' has no progress", terrors.ErrValue, id)
	}
	*pToken.raw = progText
	return nil
}

func CheckAndRecurTasks(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	for _, task := range Lists[path].Tasks {
		if task.Time.Every != nil &&
			task.Time.DueDate != nil &&
			task.Time.DueDate.Before(rightNow) {

			newDt := *task.Time.DueDate
			for newDt.Before(rightNow) {
				newDt = newDt.Add(*task.Time.Every)
			}
			diff := newDt.Sub(*task.Time.DueDate) // must be before update!
			err := task.updateDate("due", &newDt)
			if err != nil {
				return err
			}

			if task.Time.EndDate != nil && task.Time.EndDate.Before(rightNow) {
				err = task.updateDate("end", utils.MkPtr(task.Time.EndDate.Add(diff)))
				if err != nil {
					return err
				}
			} else if task.Time.Deadline != nil && task.Time.Deadline.Before(rightNow) {
				err = task.updateDate("dead", utils.MkPtr(task.Time.Deadline.Add(diff)))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ToggleCollapsed(id int, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	tk, _ := task.Tokens.Find(TkByTypeKey(TokenID, "id"))
	if tk == nil {
		return fmt.Errorf("%w: task with id '%d' has no $id", terrors.ErrNotFound, id)
	}
	if strings.HasPrefix(*tk.raw, "$-id") {
		*tk.raw = strings.Replace(*tk.raw, "$-id", "$id", 1)
	} else {
		*tk.raw = strings.Replace(*tk.raw, "$id", "$-id", 1)
	}
	return nil
}

func SortList(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	Lists.Sort(path)
	return nil
}
