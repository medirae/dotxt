package task

import (
	"dotxt/config"
	"dotxt/utils"
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
)

/*
	duration format rule of thumb

if 1.25 < year: 1.5y
if 1 < year < 1.25: 1y2m
if year < 1:

	if 2 < month: 3m
	if 1 < month < 2:
		if 1 < week: 1m2w
		if week < 1: 1m12d
	if month < 1:
		if 1 < week: 1w2d
		if week < 1:
			if 2 < day: 3d
			if 1 < day < 2: 1d2h
			if day < 1:
				if 2 < hour: 3h30M
				if hour < 2: 1h30M25s
*/
func formatDuration(d *time.Duration) string {
	totalSec := d.Seconds()
	var sign string
	if totalSec < 0 {
		sign = "-"
		totalSec = -1 * totalSec
	}

	const (
		secPerMin  = float64(60)
		secPerHr   = secPerMin * 60
		secPerDay  = secPerHr * 24
		secPerWeek = secPerDay * 7
		secPerMo   = secPerDay * 30
		secPerYr   = secPerDay * 365
	)

	// this function assumes that all formats are .1f or .0f
	fmtFloat := func(fpoints int, format string, val float64) string {
		valD := decimal.NewFromFloat(val)
		if fpoints != -1 {
			valD = valD.Truncate(int32(fpoints))
		}
		if valD.IsZero() {
			return ""
		}
		return valD.String() + format
	}

	if totalSec == 0 {
		return "rn"
	}
	var dtStr string = func() string {
		if years := totalSec / secPerYr; 1.25 <= years {
			return fmtFloat(1, "y", years)
		}
		if years := totalSec / secPerYr; 1 <= years && years < 1.25 {
			return fmtFloat(0, "y", years) + fmtFloat(1, "m", math.Mod(totalSec, secPerYr)/secPerMo)
		}
		if months := totalSec / secPerMo; 2 <= months {
			return fmtFloat(1, "m", months)
		}
		if months := totalSec / secPerMo; 1 <= months && months < 2 {
			if weeks := math.Mod(totalSec, secPerMo) / secPerWeek; 1 <= weeks {
				return fmtFloat(0, "m", months) + fmtFloat(1, "w", weeks)
			}
			return fmtFloat(0, "m", months) + fmtFloat(0, "d", math.Mod(totalSec, secPerMo)/secPerDay)
		}
		if weeks := totalSec / secPerWeek; 1 <= weeks {
			return fmtFloat(0, "w", weeks) + fmtFloat(0, "d", math.Mod(totalSec, secPerWeek)/secPerDay)
		}
		if days := totalSec / secPerDay; 2 <= days {
			return fmtFloat(0, "d", days)
		}
		if days := totalSec / secPerDay; 1 <= days && days < 2 {
			return fmtFloat(0, "d", days) + fmtFloat(0, "'", math.Mod(totalSec, secPerDay)/secPerHr)
		}
		if hours := totalSec / secPerHr; 2 <= hours {
			return fmtFloat(0, "'", hours) + fmtFloat(0, `"`, math.Mod(totalSec, secPerHr)/secPerMin)
		}
		hours := totalSec / secPerHr
		mins := math.Mod(totalSec, secPerHr) / secPerMin
		secs := math.Mod(math.Mod(totalSec, secPerHr), secPerMin)
		return fmtFloat(0, "'", hours) + fmtFloat(0, `"`, mins) + fmtFloat(0, "s", secs)
	}()

	return sign + dtStr
}

func formatAbsoluteDatetime(dt *time.Time, relDt *time.Time) string {
	if dt == nil {
		return ""
	}
	if relDt == nil {
		return dt.Format("2006-01-02T15-04")
	}
	return formatDuration(utils.MkPtr(dt.Sub(*relDt)))
}

func formatPriorities(tasks []*rTask) {
	if len(tasks) == 0 {
		return
	}

	// in case this is to be modified:
	//  the tasks in `tasks` are the ones that are to be shown.
	//  the tasks in a given task's set of children are not necessarily to be shown.
	//  therefore it is mandatory to skip tasks not in `tasks` whilst iterating a given set of children.

	colorSaturation := viper.GetFloat64("print.priority.saturation")
	colorLightness := viper.GetFloat64("print.priority.lightness")
	colorStartHue := viper.GetFloat64("print.priority.start-hue")
	colorEndHue := viper.GetFloat64("print.priority.end-hue")

	taskToRTask := func() map[*Task]*rTask {
		out := make(map[*Task]*rTask)
		for _, t := range tasks {
			if t.task != nil {
				out[t.task] = t
			}
		}
		return out
	}()

	weightMemo := make(map[*Task]int)
	var countWeight func([]*Task) int
	countWeight = func(tasks []*Task) int {
		count := 0
		prios := make(map[string]bool)
		for _, t := range tasks {
			if _, ok := taskToRTask[t]; !ok {
				continue
			}
			if w, ok := weightMemo[t]; ok {
				count += w
				continue
			}
			tcnt := 0
			if t.Priority != nil {
				if _, ok := prios[*t.Priority]; !ok {
					prios[*t.Priority] = true
					tcnt = 1
				}
			}
			tcnt += countWeight(t.Children)
			weightMemo[t] = tcnt
			count += tcnt
		}
		return count
	}

	subtreeDepth := make(map[*Task]int)
	var computeDepth func(*Task) int
	computeDepth = func(t *Task) int {
		if d, ok := subtreeDepth[t]; ok {
			return d
		}
		maximum := 0
		for _, c := range t.Children {
			if _, ok := taskToRTask[c]; !ok {
				continue
			}
			maximum = max(maximum, computeDepth(c))
		}
		subtreeDepth[t] = maximum + 1
		return subtreeDepth[t]
	}

	assignColor := func(task *rTask, color string) {
		for _, tk := range task.tokens {
			if tk.token != nil && tk.token.Type == TokenPriority && tk.token.Key == "priority" {
				tk.color = color
				break
			}
		}
	}

	sortByPriority := func(tasks []*rTask) []*rTask {
		slices.SortFunc(tasks, func(l, r *rTask) int {
			if r := sortPriority(l.task, r.task); r != 2 {
				return r
			}
			return sortHelper(l.task, r.task)
		})
		return tasks
	}

	var assignHue func(tasks []*rTask, startHue, endHue float64, depth int)
	assignHue = func(tasks []*rTask, startHue, endHue float64, depth int) {
		if len(tasks) == 0 {
			return
		}

		var prioed []*rTask
		// horizontal call part 1
		for _, rt := range tasks {
			if rt.task == nil {
				continue
			}
			ptk, _ := rt.task.Tokens.Find(TkByTypeKey(TokenPriority, "priority"))
			if ptk == nil {
				children := []*rTask{}
				for _, child := range rt.task.Children {
					if crt, ok := taskToRTask[child]; ok {
						children = append(children, crt)
					}
				}
				assignHue(sortByPriority(children), startHue, endHue, 0)
			} else {
				prioed = append(prioed, rt)
			}
		}
		tasks = prioed

		if len(tasks) == 0 {
			return
		}
		maxDepth := 0
		for _, t := range tasks {
			maxDepth = max(maxDepth, utils.RuneCount(*t.task.Priority))
		}
		if depth > maxDepth {
			return
		}

		prefixes := []string{}
		groups := make(map[string][]*rTask)
		for _, rt := range tasks {
			prio := *rt.task.Priority // TODO: document what prio and prefix are doing
			prefix := prio
			if utils.RuneCount(prio) > depth {
				prefix = utils.RuneSlice(prio, 0, depth+1)
			}
			if _, exists := groups[prefix]; !exists {
				prefixes = append(prefixes, prefix)
			}
			groups[prefix] = append(groups[prefix], rt)
		}
		sort.Strings(prefixes)

		weights := make([]int, len(prefixes))
		totalWeight := 0
		for i, prefix := range prefixes {
			weights[i] += countWeight(func() []*Task {
				out := make([]*Task, len(groups[prefix]))
				for ndx, rt := range groups[prefix] {
					out[ndx] = rt.task
				}
				return out
			}())
			totalWeight += weights[i]
		}

		span := endHue - startHue
		currentPrefixHue := startHue
		for i, prefix := range prefixes {
			group := groups[prefix]

			ratio := float64(weights[i]) / float64(totalWeight)
			nextPrefixHue := currentPrefixHue + ratio*span
			currentSpan := nextPrefixHue - currentPrefixHue
			currentHue := currentPrefixHue

			leavesWeight := make(map[string]int)
			leavesByPrio := make(map[string][]*rTask)
			var branch []*rTask
			branchWeight := 0
			for _, rt := range group {
				if utils.RuneCount(*rt.task.Priority) <= depth+1 || len(group) == 1 { // if there are leaves, all of them are always behind branches!
					leavesByPrio[*rt.task.Priority] = append(leavesByPrio[*rt.task.Priority], rt)
					leavesWeight[*rt.task.Priority] += weightMemo[rt.task]
				} else {
					branch = append(branch, rt)
					branchWeight += weightMemo[rt.task]
				}
			}

			for prio, samePrioLeafGroup := range leavesByPrio {
				nextHue := currentHue + float64(leavesWeight[prio])/float64(weights[i])*currentSpan
				midHue := (currentHue + nextHue) / 2
				for _, rt := range samePrioLeafGroup {
					assignColor(rt, utils.HslToHex(midHue, colorSaturation, colorLightness))
					// horizontal call part 2
					var children []*rTask
					for _, child := range rt.task.Children {
						if crt, ok := taskToRTask[child]; ok {
							children = append(children, crt)
						}
					}
					assignHue(sortByPriority(children), currentHue, nextHue, 0)
				}
				currentHue = nextHue
			}

			// vertical call
			assignHue(branch, currentHue, nextPrefixHue, depth+1)
			currentPrefixHue = nextPrefixHue
		}
	}

	roots := []*rTask{}
	for _, rt := range tasks {
		if rt.task != nil && rt.task.Parent == nil {
			roots = append(roots, rt)
		}
	}

	for _, rt := range roots {
		_ = countWeight([]*Task{rt.task})
	}

	assignHue(sortByPriority(roots), colorStartHue, colorEndHue, 0)
}

func formatProgress(p *Progress, countLen, doneCountLen int) []rToken {
	colorizePercentage := func(percent int) string {
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		hue := 120.0 * float64(percent) / 100.0

		startSaturation := viper.GetFloat64("print.progress.percentage.start-saturation")
		endSaturation := viper.GetFloat64("print.progress.percentage.end-saturation")
		saturation := startSaturation + (endSaturation-startSaturation)*((math.Pow(math.E, float64(percent)/100)-1)/(math.E-1))

		startLightness := viper.GetFloat64("print.progress.percentage.start-lightness")
		endLightness := viper.GetFloat64("print.progress.percentage.end-lightness")
		lightness := startLightness + (endLightness-startLightness)*((math.Pow(math.E, 1-float64(percent)/100)-1)/(math.E-1))

		return utils.HslToHex(hue, saturation, lightness)
	}

	percentage := 100 * p.Count / p.DoneCount
	percentageColor := colorizePercentage(percentage)
	barText := func(width int) string {
		pLen := width * p.Count / p.DoneCount
		switch pLen {
		case 0:
			return strings.Repeat(" ", width)
		case 1:
			return ">" + strings.Repeat(" ", width-1)
		case width:
			return strings.Repeat("=", width)
		}
		return strings.Repeat("=", pLen-1) + ">" + strings.Repeat(" ", width-pLen)
	}(viper.GetInt("print.progress.bartext-len"))
	return []rToken{
		{raw: fmt.Sprintf("%*d", countLen, p.Count), color: "print.progress.count"},
		{raw: fmt.Sprintf("/%*d", doneCountLen, p.DoneCount), color: "print.progress.done-count"},
		{raw: fmt.Sprintf("(%3d%%)", percentage), color: percentageColor},
		{raw: fmt.Sprintf(" %s ", barText), color: percentageColor},
		{raw: fmt.Sprintf("(%s)", p.Unit), color: "print.progress.unit"},
	}
}

func formatListHeader(path string, maxLen int) string {
	var out strings.Builder
	out.WriteString("> ")
	out.WriteString(filepath.Base(path))
	out.WriteString(" | ")
	out.WriteString(strings.Repeat("—", maxLen-out.Len()))
	out.WriteRune('\n')
	return colorize("print.color-header", out.String())
}

func resolvColor(color string) string {
	if strings.TrimSpace(color) == "" {
		color = "print.color-default"
	}
	if validateHexColor(color) != nil {
		color = viper.GetString(color)
		if validateHexColor(color) != nil {
			color = viper.GetString("print.color-default")
		}
	}
	return color
}

func colorize(color, text string) string {
	color = resolvColor(color)
	if config.Color && color != "" {
		return fmt.Sprintf("${color %s}%s", color, text)
	}
	return text
}

func colorizeIds(ids map[string]bool) map[string]string {
	out := make(map[string]string)
	if len(ids) == 0 {
		return out
	}
	startHue := viper.GetFloat64("print.ids.start-hue")
	endHue := viper.GetFloat64("print.ids.end-hue")
	saturation := viper.GetFloat64("print.ids.saturation")
	lightness := viper.GetFloat64("print.ids.lightness")
	ndx := 0
	n := float64(len(ids))
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		h := startHue + float64(ndx)*(endHue-startHue)/n
		out[id] = utils.HslToHex(h, saturation, lightness)
		ndx++
	}
	return out
}

func formatID(tk Token) string {
	var idCollapse string
	if strings.HasPrefix(*tk.raw, "$-id=") {
		idCollapse = "-"
	}
	return fmt.Sprintf("$%s%s=%s", idCollapse, tk.Key, *tk.Value.(*string))
}

func formatCategoryHeader(category string, info *rInfo) string {
	var out strings.Builder
	out.WriteString(strings.Repeat(" ", info.idLen+1+
		info.countLen+1+info.doneCountLen+utils.RuneCount("(100%) ")+
		viper.GetInt("print.progress.bartext-len")+1+
		-utils.RuneCount(category)-1,
	))
	out.WriteString(fmt.Sprintf("%s ", category))
	out.WriteString(strings.Repeat("—", info.maxLen-out.Len()))
	out.WriteRune('\n')
	return colorize("print.progress.header", out.String())
}

func (t *Task) Render() *rTask {
	out := rTask{task: t, id: *t.ID, idColor: "print.color-index", decor: false, depth: t.Depth()}
	addAsRegular := func(token *Token) {
		out.tokens = append(out.tokens, &rToken{token: token, raw: token.String(), color: "print.color-default"})
	}

	if t.Prog != nil {
		tk, _ := t.Tokens.Find(TkByType(TokenProgress))
		out.tokens = append(out.tokens, &rToken{token: tk})
		out.countLen = utils.RuneCount(strconv.Itoa(t.Prog.Count))
		out.doneCountLen = utils.RuneCount(strconv.Itoa(t.Prog.DoneCount))
	}
	if t.Priority != nil {
		tk, _ := t.Tokens.Find(TkPriorityPrefix)
		color := "print.color-default"
		if tk.Key == "anti-priority" {
			color = "print.color-anti-priority"
		}
		out.tokens = append(out.tokens, &rToken{
			token: tk, raw: *t.Priority, color: color,
		})
	}

	var dominantColor, defaultColor string = func() (string, string) {
		if t.Time.DueDate != nil && t.Time.DueDate.Sub(rightNow) <= 0 {
			if t.Time.EndDate == nil && t.Time.Deadline == nil {
				return "print.color-burnt", ""
			}
			if t.Time.EndDate != nil {
				if t.Time.EndDate.Sub(rightNow) <= 0 {
					return "print.color-burnt", ""
				}
				return "", "print.color-running-event-text"
			}
			if t.Time.Deadline != nil {
				if t.Time.Deadline.Sub(rightNow) <= 0 {
					return "print.color-burnt", ""
				}
				return "", ""
			}
		}
		return "", ""
	}()

	var reminderCount int
	t.Tokens.ForEach(func(tk *Token) {
		switch tk.Type {
		case TokenProgress:
			return
		case TokenPriority:
			switch tk.Key {
			case "urgent":
				out.tokens = append(out.tokens, &rToken{
					token: tk, raw: tk.String(), color: "print.color-urgent",
				})
			case "mit":
				out.tokens = append(out.tokens, &rToken{
					token: tk, raw: tk.String(), color: "print.color-mit",
				})
			default:
				return
			}
		case TokenText:
			switch tk.Key {
			case "quote":
				val := tk.String()
				quote := val[0]
				val = val[1 : len(val)-1]
				var color string
				switch quote {
				case '"':
					color = "print.quotes.double"
				case '\'':
					color = "print.quotes.single"
				case '`':
					color = "print.quotes.backticks"
				}
				quotes := []byte{'"', '\'', '`'}
				quotes = func() []byte { // rotate such that: index -> 0
					k := slices.Index(quotes, quote)
					n := len(quotes)
					k = (n - (k % n)) % n
					return append(quotes[n-k:], quotes[:n-k]...)
				}()
				var found bool
				for ndx, q := range quotes {
					if strings.ContainsRune(val, rune(q)) {
						if ndx == 0 {
							val = strings.ReplaceAll(val, "\\"+string(q), string(q))
						}
						continue
					}
					val = fmt.Sprintf("%c%s%c", q, val, q)
					found = true
					break
				}
				if !found {
					val = strings.ReplaceAll(val, "```", "\\`\\`\\`")
					val = fmt.Sprintf("```%s```", val)
				}
				out.tokens = append(out.tokens, &rToken{
					token: tk, raw: val,
					color: color,
				})
			case ";":
				replacer := strings.NewReplacer("\\'", "'", "\\\"", "\"", "\\`", "`", "\\;", "")
				out.tokens = append(out.tokens, &rToken{
					token: tk, raw: replacer.Replace(tk.String()), color: "print.color-default",
				})
			default:
				replacer := strings.NewReplacer("\\'", "'", "\\\"", "\"", "\\`", "`")
				out.tokens = append(out.tokens, &rToken{
					token: tk, raw: replacer.Replace(tk.String()), color: "print.color-default",
				})
			}
		case TokenID:
			out.tokens = append(out.tokens, &rToken{
				token: tk,
				raw:   formatID(*tk),
				color: "print.color-default",
			})
		case TokenHint:
			var color string
			switch tk.Key {
			case "@":
				color = "print.hints.color-at"
			case "#":
				color = "print.hints.color-tag"
			case "+":
				color = "print.hints.color-plus"
			case "!":
				color = "print.hints.color-exclamation"
			case "?":
				color = "print.hints.color-question"
			case "*":
				color = "print.hints.color-star"
			case "&":
				color = "print.hints.color-ampersand"
			}
			out.tokens = append(out.tokens, &rToken{
				token: tk,
				raw:   tk.String(), color: color,
			})
		case TokenDate:
			if slices.Contains([]string{"due", "end", "dead"}, tk.Key) {
				val, err := t.Time.getField(tk.Key)
				if err != nil {
					addAsRegular(tk)
					return
				}
				relStr, ok := temporalFormatFallback[tk.Key]
				if !ok {
					addAsRegular(tk)
					return
				}
				rel, err := t.Time.getField(relStr)
				if err != nil {
					addAsRegular(tk)
					return
				}
				color := "print.color-date-" + tk.Key
				if (tk.Key == "due" && IsDateUrgent(t.Time.DueDate)) ||
					(tk.Key == "end" && IsDateUrgent(t.Time.EndDate)) ||
					(tk.Key == "dead" && IsDateUrgent(t.Time.Deadline)) {
					color = "print.color-urgent"
				}
				if t.Time.DueDate != nil &&
					t.Time.DueDate.Sub(rightNow) <= 0 {
					if tk.Key == "due" {
						color = "print.color-burnt"
					} else if tk.Key == "dead" &&
						IsDateUrgent(t.Time.Deadline) {
						color = "print.color-imminent-deadline"
					} else if tk.Key == "end" &&
						t.Time.EndDate != nil &&
						t.Time.EndDate.Sub(rightNow) > 0 {
						color = "print.color-running-event"
					}
				}
				out.tokens = append(out.tokens, &rToken{
					token: tk,
					raw:   fmt.Sprintf("$%s=%s", tk.Key, formatAbsoluteDatetime(val, rel)),
					color: color,
				})
			}
			if strings.HasPrefix(tk.Key, "r") {
				if reminderCount >= len(t.Time.Reminders) {
					addAsRegular(tk)
					return
				}
				val := t.Time.Reminders[reminderCount]
				reminderCount++
				relStr, ok := temporalFormatFallback["r"]
				if !ok {
					addAsRegular(tk)
					return
				}
				rel, err := t.Time.getField(relStr)
				if err != nil {
					addAsRegular(tk)
					return
				}
				if val.Sub(rightNow) < 0 {
					return
				}
				out.tokens = append(out.tokens, &rToken{
					token: tk,
					raw:   fmt.Sprintf("$r=%s", formatAbsoluteDatetime(val, rel)),
					color: "print.color-date-r",
				})
			}
		case TokenDuration:
			out.tokens = append(out.tokens, &rToken{
				token: tk,
				raw:   fmt.Sprintf("$every=%s", formatDuration(t.Time.Every)),
				color: "print.color-every",
			})
		case TokenFormat:
			if tk.Key == "focus" {
				out.focused = true
				out.tokens = append(out.tokens, &rToken{
					token: tk,
					raw:   "$focus",
					color: "print.color-focus",
				})
			}
		default:
			addAsRegular(tk)
		}
	})
	out.maxLen = utils.RuneCount(out.stringify(false, -1))
	out.idLen = utils.RuneCount(strconv.Itoa(*t.ID))
	if dominantColor != "" || defaultColor != "" {
		for _, rtk := range out.tokens {
			rtk.dominantColor = dominantColor
			if rtk.token.Type == TokenText && defaultColor != "" {
				rtk.color = defaultColor
			}
		}
	}
	return &out
}

func RenderList(path string) ([]*rTask, *rInfo, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return nil, nil, err
	}

	// first pass: preprocessing
	taskHasFocus := func(t *Task) bool {
		return t.Fmt != nil && t.Fmt.Focus
	}
	taskHasDerivedFocus := func(rt *rTask) bool {
		return rt.focused && !taskHasFocus(rt.task)
	}

	parentToChildren := make(map[*Task][]*Task)
	parentToChildrenDepth := map[*Task]int{nil: 0}
	parentToChildrenFocus := make(map[*Task]bool)
	taskToRTask := make(map[*Task]*rTask)
	Lists.Sort(path)

	var parentStack []*Task
	isParentInStack := func(t *Task) bool {
		for ndx := len(parentStack) - 1; ndx >= 0; ndx-- {
			if parentStack[ndx] == t {
				return true
			}
		}
		return false
	}
	var parent *Task
	for _, task := range Lists[path].Tasks {
		if task.Parent != nil && !isParentInStack(task.Parent) {
			// going into a new depth
			parentStack = append(parentStack, task.Parent)
			parent = task.Parent
		} else if (task.Parent == nil && parent != nil) ||
			(task.Parent != nil && task.Parent != parent) {
			// coming out of a depth
			for ndx := len(parentStack) - 1; ndx >= 0; ndx-- {
				if parentStack[ndx] == task.Parent {
					break
				}
				parentStack = parentStack[:ndx]
			}
			parent = task.Parent
		}

		rtask := task.Render()
		taskToRTask[task] = rtask
		parentToChildren[task.Parent] = append(parentToChildren[task.Parent], task)
		if rtask.focused {
			node := task
			for node != nil {
				taskToRTask[node].focused = true
				node = node.Parent
			}
		}
		parentToChildrenDepth[task.Parent] = len(parentStack)
		parentToChildrenFocus[task.Parent] = parentToChildrenFocus[task.Parent] || taskHasFocus(task)
	}

	// second pass: filtering and processing
	var out []*rTask
	idList := make(map[string]bool)
	listInfo := rInfo{}

	flushEllipsis := func(count, depth int) {
		out = append(out, &rTask{
			tokens: []*rToken{
				{raw: "...", color: "print.color-hidden"},
				{raw: "-" + strconv.Itoa(count), color: "print.color-hidden"},
				{raw: "...", color: "print.color-hidden"},
			},
			depth: depth,
			decor: true,
		})
	}

	var render func(*Task)
	render = func(node *Task) {
		siblings, ok := parentToChildren[node]
		if !ok {
			return
		}
		siblingDepth := parentToChildrenDepth[node]
		var shf bool = parentToChildrenFocus[node] // siblings have focus
		var hiddenCount int
		for _, task := range siblings {
			rtask := taskToRTask[task]
			if shf && !taskHasFocus(task) && !taskHasDerivedFocus(rtask) {
				hiddenCount += 1 + len(task.Children)
				continue
			}
			if shf && (taskHasFocus(task) || taskHasDerivedFocus(rtask)) && hiddenCount > 0 {
				flushEllipsis(hiddenCount, siblingDepth)
			}
			if shf && (taskHasFocus(task) || taskHasDerivedFocus(rtask)) {
				hiddenCount = 0
			}
			if task.IsParentCollapsed() {
				continue
			}
			out = append(out, rtask)
			if rtask.task != nil {
				if rtask.task.EID != nil {
					idList[*rtask.task.EID] = true
				}
				if rtask.task.PID != nil {
					idList[*rtask.task.PID] = true
				}
			}
			listInfo.set(&rtask.rInfo)

			if task.EID != nil {
				render(task)
			}
		}
		if hiddenCount > 0 {
			flushEllipsis(hiddenCount, siblingDepth)
		}
	}
	render(nil)
	idColors := colorizeIds(idList)
	for _, rtask := range out {
		for _, tk := range rtask.tokens {
			if tk.token != nil && tk.token.Type == TokenID {
				tk.color = idColors[*tk.token.Value.(*string)]
				if tk.dominantColor == "" &&
					((tk.token.Key == "id" && len(rtask.task.Children) == 0) ||
						(tk.token.Key == "P" && rtask.task.Parent == nil)) {
					tk.dominantColor = "print.color-dead-relations"
				}
			}
		}
	}
	formatPriorities(out)
	return out, &listInfo, nil
}

func PrintLists(paths []string, maxLen, minLen int) error {
	var err error
	for ndx := range paths {
		paths[ndx], err = prepFileTaskFromPath(paths[ndx])
		if err != nil {
			return err
		}
	}
	rtasks := make(map[string][]*rTask)
	sessionInfo := rInfo{}
	for _, path := range paths {
		var listInfo *rInfo
		rtasks[path], listInfo, err = RenderList(path)
		if err != nil {
			return err
		}
		sessionInfo.set(listInfo)
	}

	sessionInfo.maxLen = max(min(sessionInfo.maxLen, maxLen), minLen)
	for _, path := range paths { // propogate downwards
		for _, rtask := range rtasks[path] {
			rtask.rInfo.set(&sessionInfo)
		}
	}
	var out strings.Builder
	for _, path := range paths {
		emptyCatThere := false
		categories := make(map[string]bool)
		for _, rtask := range rtasks[path] {
			if rtask.task != nil && rtask.task.Prog != nil {
				if root := rtask.task.Root(); root == rtask.task { // only take root tasks into account
					if rtask.task.Prog.Category == "" {
						emptyCatThere = true
					}
					categories[rtask.task.Prog.Category] = true
				}
			}
		}
		useCatHeader := !((len(categories) == 1 && emptyCatThere) || len(categories) == 0)
		var lastCat string
		firstNonCat := true

		out.WriteString(formatListHeader(path, sessionInfo.maxLen))
		for _, rtask := range rtasks[path] {
			if useCatHeader && rtask.task != nil && rtask.task.Prog != nil && rtask.task.Prog.Category != lastCat {
				if root := rtask.task.Root(); root == rtask.task { // not a nested progress
					cat := rtask.task.Prog.Category
					if cat == "" {
						cat = "*"
					}
					out.WriteString(formatCategoryHeader(cat, &sessionInfo))
					lastCat = rtask.task.Prog.Category
				}
			}
			if useCatHeader && rtask.task != nil && rtask.task.Prog == nil && firstNonCat {
				if root := rtask.task.Root(); root == rtask.task { // not a nested progress
					firstNonCat = false
					out.WriteString(formatCategoryHeader("", &sessionInfo))
				}
			}

			out.WriteString(rtask.stringify(true, sessionInfo.maxLen))
			out.WriteRune('\n')
		}
		out.WriteRune('\n')
	}
	fmt.Print(out.String())
	return nil
}

// single task
func PrintTask(id int, path string, maxWidth int) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	fmt.Println(task.Render().stringify(true, maxWidth))
	return nil
}

func OutputTask(id int, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	fmt.Println(task.Raw())
	return nil
}
