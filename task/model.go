package task

import (
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
)

var (
	rightNow        time.Time
	oneMonthFromNow time.Time
)

func AdjustTime() {
	rightNow = time.Now()
	oneMonthFromNow = rightNow.Add(30 * 24 * time.Hour)
}

func init() {
	AdjustTime()
}

func IsDateUrgent(dt *time.Time) bool {
	return dt != nil &&
		dt.After(rightNow) &&
		dt.Before(oneMonthFromNow)
}

type List struct {
	Tasks []*Task
	EIDs  map[string]*Task
	PIDs  map[*Task]string
}

type lists map[string]*List

var Lists lists = make(lists)

func (l *lists) Exists(path string) bool {
	_, ok := (*l)[path]
	return ok
}

// check existence and get
func (l *lists) Tasks(path string) ([]*Task, bool) {
	if !l.Exists(path) {
		return nil, false
	}
	return (*l)[path].Tasks, true
}

// create a list if it doesn't exist
func (l *lists) Init(path string, values ...*Task) {
	if !l.Exists(path) {
		(*l)[path] = new(List)
		(*l)[path].EIDs = make(map[string]*Task)
		(*l)[path].PIDs = make(map[*Task]string)
		if len(values) > 0 {
			(*l)[path].Tasks = append((*l)[path].Tasks, values...)
		}
	} else if len(values) > 0 {
		l.Set(path, values)
		CleanupRelations(path)
	}
}

// empties the tasks of this path if it exists
func (l *lists) Empty(path string, values ...*Task) {
	l.Init(path)
	if len(values) > 0 {
		l.Set(path, values)
	} else {
		(*l)[path].Tasks = make([]*Task, 0)
	}
}

func (l *lists) Set(path string, tasks []*Task) {
	l.Init(path)
	(*l)[path].Tasks = tasks
	CleanupRelations(path)
}

// append task to list if it exists
func (l *lists) Append(path string, task *Task) {
	l.Init(path)
	(*l)[path].Tasks = append((*l)[path].Tasks, task)
}

func (l *lists) Sort(path string) {
	if l.Exists(path) {
		(*l)[path].Tasks = sortTasks((*l)[path].Tasks)
	}
}

func (l *lists) SortFunc(path string, cmp func(*Task, *Task) int) {
	if l.Exists(path) {
		slices.SortFunc((*l)[path].Tasks, cmp)
	}
}

func (l *lists) Delete(path string) {
	if l.Exists(path) {
		delete(*l, path)
	}
}

// uses slices.Delete upon l.Tasks
func (l *lists) DeleteTasks(path string, start, end int) {
	if l.Exists(path) {
		(*l)[path].Tasks = slices.Delete((*l)[path].Tasks, start, end)
	}
}

func (l *lists) Len(path string) int {
	if l.Exists(path) {
		return len((*l)[path].Tasks)
	}
	return 0
}

func (l *lists) Clear() {
	for key := range *l {
		delete(*l, key)
	}
}

type TokenType int

const (
	TokenText TokenType = iota
	TokenID
	TokenHint
	TokenPriority
	TokenDate
	TokenDuration
	TokenProgress
	TokenFormat
)

type TokenDateValue struct {
	Value  *time.Time
	RelKey string
	RelVal *time.Time
	Offset *time.Duration
}

type Token struct {
	Type  TokenType
	raw   *string // an attempt to carry extra token metadata from the original text
	Key   string
	Value any // whatever the case this must be a pointer type
}

type Tokens []*Token

type TkFunc func(*Token)
type TkFuncIndex func(*Token, int)
type TkCond func(*Token) bool

func (lc TkCond) And(rc TkCond) TkCond {
	return func(tk *Token) bool {
		return lc(tk) && rc(tk)
	}
}

func (lc TkCond) Or(rc TkCond) TkCond {
	return func(tk *Token) bool {
		return lc(tk) || rc(tk)
	}
}

func (c TkCond) Not() TkCond {
	return func(tk *Token) bool {
		return !c(tk)
	}
}

func TkByType(tipe TokenType) TkCond {
	return func(tk *Token) bool {
		return tk.Type == tipe
	}
}

func TkByTypeKey(tipe TokenType, key string) TkCond {
	return func(tk *Token) bool {
		return tk.Type == tipe && tk.Key == key
	}
}

var TkPriorityPrefix TkCond = TkByTypeKey(TokenPriority, "priority").
	Or(TkByTypeKey(TokenPriority, "anti-priority"))

func (tks *Tokens) ForEach(fn TkFunc) {
	for _, tk := range *tks {
		fn(tk)
	}
}

func (tks *Tokens) ForEachIndex(fn TkFuncIndex) {
	for ndx, tk := range *tks {
		fn(tk, ndx)
	}
}

func (tks *Tokens) Find(cond TkCond) (*Token, int) {
	for ndx, tk := range *tks {
		if cond(tk) {
			return tk, ndx
		}
	}
	return nil, -1
}

func (tks *Tokens) FindFrom(cond TkCond, index int) (*Token, int) {
	index = min(max(index, 0), len(*tks))
	for ndx := index; ndx < len(*tks); ndx++ {
		tk := (*tks)[ndx]
		if cond(tk) {
			return tk, ndx
		}
	}
	return nil, -1
}

// if used to chain together commands,
// do note that this will make the whole thing run
// in at least 2 passes.
func (tks *Tokens) Filter(cond TkCond) *Tokens {
	var out Tokens
	for _, tk := range *tks {
		if cond(tk) {
			out = append(out, tk)
		}
	}
	return &out
}

func (tk *Token) String() string {
	switch tk.Type {
	case TokenText, TokenHint:
		return *tk.Value.(*string)
	case TokenPriority:
		if tk.Key == "mit" {
			return fmt.Sprintf("$mit=%d", *tk.Value.(*int))
		}
		return *tk.Value.(*string)
	case TokenID:
		val := *tk.Value.(*string)
		for _, prefix := range []string{"$id=", "$-id=", "$P="} {
			if strings.HasPrefix(*tk.raw, prefix) {
				return prefix + val
			}
		}
		return val
	case TokenDuration:
		return "$" + tk.Key + "=" + unparseDuration(*tk.Value.(*time.Duration))
	case TokenDate:
		val := tk.Value.(*TokenDateValue)
		if val.RelKey == "" {
			return fmt.Sprintf("$%s=%s", tk.Key, unparseAbsoluteDatetime(*val.Value))
		}
		return tk.unparseRelativeDatetime(nil)
	case TokenProgress:
		p, err := unparseProgress(*tk.Value.(*Progress))
		if err == nil {
			return "$p=" + p
		}
		return *tk.raw
	case TokenFormat:
		if tk.Key == "focus" {
			return "$focus"
		}
	}
	return ""
}

type Progress struct {
	Unit      string
	Category  string
	Count     int
	DoneCount int
}

type Temporal struct {
	CreationDate *time.Time
	DueDate      *time.Time
	Reminders    []*time.Time
	EndDate      *time.Time
	Deadline     *time.Time
	Every        *time.Duration
}

func (t *Temporal) getField(key string) (*time.Time, error) {
	switch key {
	case "rn":
		return &rightNow, nil
	case "c":
		return t.CreationDate, nil
	case "due":
		return t.DueDate, nil
	case "end":
		return t.EndDate, nil
	case "dead":
		return t.Deadline, nil
	}
	if key == "r" {
		return nil, fmt.Errorf("key 'r' not supported since it's a slice of *time.Time")
	}
	return nil, fmt.Errorf("%w: key '%s' not found", terrors.ErrNotFound, key)
}

func (t *Temporal) setField(key string, val *time.Time) error {
	switch key {
	case "c":
		t.CreationDate = val
	case "due":
		t.DueDate = val
	case "end":
		t.EndDate = val
	case "dead":
		t.Deadline = val
	}
	if key == "r" {
		return fmt.Errorf("key 'r' not supported since it's a slice of *time.Time")
	}
	return fmt.Errorf("%w: key '%s' not found", terrors.ErrNotFound, key)
}

// The default fields for each temporal field used for
// formatting datetime relatively
var temporalFormatFallback = map[string]string{
	"c": "rn", "due": "rn",
	"end": "due", "dead": "due",
	"r": "rn",
}

// The default fields for each temporal field used for
// parsing relative datetime
var temporalFallback = map[string]string{
	"rn":  "rn",
	"c":   "rn",
	"due": "c",
	"end": "due", "dead": "due", "r": "due",
}

// which RelKeys each Key is allowed to reference
var allowedTemporalRelations = map[string][]string{
	"rn":   {"rn"},
	"c":    {"rn"},
	"due":  {"c", "rn"},
	"end":  {"due", "c", "rn"},
	"dead": {"due", "c", "rn"},
	"r":    {"due", "c", "rn"},
}

type Format struct {
	Focus bool
}

type Task struct {
	Tokens   Tokens
	ID       *int
	Hints    []*string
	Priority *string
	EID      *string // explicit id ($id=)
	Children []*Task
	PID      *string // parent id ($P=)
	Parent   *Task
	Urgent   bool
	MIT      *int

	Time *Temporal
	Prog *Progress
	Fmt  *Format
}

func (t *Task) IsUrgent() bool {
	return t.Urgent || t.MIT != nil ||
		IsDateUrgent(t.Time.DueDate) ||
		IsDateUrgent(t.Time.EndDate) ||
		IsDateUrgent(t.Time.Deadline)
}

func (t *Task) String() string {
	if t.ID == nil {
		return t.Raw()
	}
	return fmt.Sprintf("%-2d %s", *t.ID, t.Raw())
}

func (t *Task) Depth() int {
	count := -1
	task := t
	for task != nil {
		task = task.Parent
		count++
	}
	return count
}

func (t *Task) Root() *Task {
	node := t
	for node.Parent != nil {
		node = node.Parent
	}
	return node
}

func (t *Task) IsCollapsed() bool {
	tk, _ := t.Tokens.Find(TkByTypeKey(TokenID, "id"))
	return tk != nil && strings.HasPrefix(*tk.raw, "$-id")
}

func (t *Task) IsParentCollapsed() bool {
	node := t.Parent
	for node != nil {
		if node.IsCollapsed() {
			return true
		}
		node = node.Parent
	}
	return false
}

func (t *Task) revertIDtoText(key string) {
	switch key {
	case "id":
		t.EID = nil
		for _, child := range t.Children {
			child.Parent = nil
		}
		t.Children = slices.Delete(t.Children, 0, len(t.Children))
	case "P":
		t.PID = nil
		if t.Parent != nil {
			for ndx := range t.Parent.Children {
				if t.Parent.Children[ndx] == t {
					t.Parent.Children = slices.Delete(t.Parent.Children, ndx, ndx+1)
					break
				}
			}
		}
		t.Parent = nil
	default:
		panic("key was supposed to be id or P but was '" + key + "'")
	}
	tk, _ := t.Tokens.Find(TkByTypeKey(TokenID, key))
	if tk != nil {
		tk.Type = TokenText
		tk.Key = ""
		tk.Value = tk.raw
	}
}

func (t *Task) update(new *Task) error {
	curCreationDtToken, _ := t.Tokens.Find(TkByTypeKey(TokenDate, "c"))
	if curCreationDtToken != nil {
		newCreationDtToken, _ := new.Tokens.Find(TkByTypeKey(TokenDate, "c"))
		if newCreationDtToken != nil {
			*newCreationDtToken.raw = *curCreationDtToken.raw
			newCreationDtToken.Value = curCreationDtToken.Value.(*TokenDateValue)
		} else {
			new.Tokens = append(new.Tokens, &Token{
				Type: TokenDate, Key: "c",
				raw:   utils.MkPtr(*curCreationDtToken.raw),
				Value: curCreationDtToken.Value.(*TokenDateValue),
			})
		}
	}
	new, err := ParseTask(t.ID, new.Raw())
	if err != nil {
		return err
	}
	id := t.ID
	*t = *new
	t.ID = id
	return nil
}

func (t *Task) updateFromText(new string) error {
	dummy, err := ParseTask(t.ID, new)
	if err != nil {
		return err
	}
	err = t.update(dummy)
	if err != nil {
		return err
	}
	return nil
}

func (t *Task) updateByModifyingText(prefix, postfix string) error {
	var newText string
	if n := utils.RuneCount(prefix); n > 0 {
		if t.Priority != nil {
			priority := *t.Priority
			t.Priority = nil
			_, ndx := t.Tokens.Find(TkPriorityPrefix)
			t.Tokens = slices.Delete(t.Tokens, ndx, ndx+1)

			leftOfPrefixIsSpace := unicode.IsSpace(utils.RuneAt(prefix, 0))
			leftOfPrefixIsSemicolon := utils.RuneAt(prefix, 0) == '\\' &&
				n >= 2 && utils.RuneAt(prefix, 1) == ';'
			if leftOfPrefixIsSpace || leftOfPrefixIsSemicolon {
				prefix = priority + prefix
			} else {
				prefix = priority + " " + prefix
			}
			n = utils.RuneCount(prefix)
		}

		newText = prefix
		firstTokenIsSpacy := len(t.Tokens) > 0 && t.Tokens[0].Type == TokenText &&
			t.Tokens[0].Key == ";"
		rightOfPrefixIsSpace := unicode.IsSpace(utils.RuneAt(prefix, n-1))
		rightOfPrefixIsSemicolon := utils.RuneAt(prefix, n-1) == ';' &&
			n >= 2 && utils.RuneAt(prefix, n-2) == '\\'
		if !firstTokenIsSpacy && !rightOfPrefixIsSpace && !rightOfPrefixIsSemicolon {
			newText += " "
		}

	}

	newText += t.Raw()

	if utils.RuneCount(postfix) > 0 {
		n := utils.RuneCount(postfix)
		tn := len(t.Tokens)
		lastTokenIsSpacy := tn > 0 && t.Tokens[tn-1].Type == TokenText && t.Tokens[tn-1].Key == ";"
		leftOfPostfixIsSpace := unicode.IsSpace(utils.RuneAt(postfix, 0))
		leftOfPostfixIsSemicolon := utils.RuneAt(postfix, 0) == '\\' &&
			n >= 2 && utils.RuneAt(postfix, 1) == ';'
		if !lastTokenIsSpacy && !leftOfPostfixIsSpace && !leftOfPostfixIsSemicolon {
			newText += " "
		}
		newText += postfix
	}
	return t.updateFromText(newText)
}

func (t *Task) updateDate(field string, newDt *time.Time) error {
	token, _ := t.Tokens.Find(TkByTypeKey(TokenDate, field))
	if token == nil {
		return fmt.Errorf("%w: token date for field '%s' not found", terrors.ErrNotFound, field)
	}
	var newDtTxt string
	tkDt := token.Value.(*TokenDateValue)
	if tkDt.RelKey == "" {
		newDtTxt = fmt.Sprintf("$%s=%s", field, unparseAbsoluteDatetime(*newDt))
	} else {
		newDtTxt = token.unparseRelativeDatetime(newDt)
	}
	*token.raw = newDtTxt
	tkDt.Value = newDt
	t.Time.setField(field, newDt)
	return nil
}

// this function is to be used in function that are turning the tokens of a task into a string
func preprocessTaskStrings(t *Task, index int, out *strings.Builder) {
	if index > 0 {
		prev := t.Tokens[index-1].Type == TokenText && t.Tokens[index-1].Key == ";"
		cur := t.Tokens[index].Type == TokenText && t.Tokens[index].Key == ";"
		if !prev && !cur {
			out.WriteRune(' ')
		}
	}
}

// A reduced form of the raw string that represents tasks
// more rigidly used for comparison
// :: everything besides $c
func (t *Task) Norm() string {
	var out strings.Builder
	// this must be filtered first so that preprocess doesn't have problem with indexing
	t.Tokens.Filter(func(tk *Token) bool {
		return !(tk.Type == TokenDate && tk.Key == "c")
	}).ForEachIndex(func(tk *Token, i int) {
		preprocessTaskStrings(t, i, &out)
		out.WriteString(tk.String())
	})
	return out.String()
}

// A reduced form of the raw string that represents tasks
// more rigidly via only regular texts used for comparison
// :: only non-special text
func (t *Task) NormRegular() string { // Text
	var out strings.Builder
	// this must be filtered first so that preprocess doesn't have problem with indexing
	t.Tokens.Filter(func(tk *Token) bool {
		return tk.Type == TokenText
	}).ForEachIndex(func(tk *Token, i int) {
		preprocessTaskStrings(t, i, &out)
		out.WriteString(tk.String())
	})
	return out.String()
}

// the text of the task joined in from the tokens
func (t *Task) Raw() string {
	var out strings.Builder
	t.Tokens.ForEachIndex(func(tk *Token, i int) {
		preprocessTaskStrings(t, i, &out)
		out.WriteString(tk.String())
	})
	return out.String()
}

func helper[T any](p T) string {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "nil"
		}
		return fmt.Sprintf("%v", v.Elem())
	}
	return fmt.Sprintf("%v", p)
}
func print(args ...any) {
	key, _ := args[0].(string)
	var values []any
	for ndx := 1; ndx < len(args); ndx++ {
		values = append(values, helper(args[ndx]))
	}
	fmt.Printf(key, values...)
}

func DebugTask(t *Task) {
	if t == nil {
		fmt.Println("task == nil")
		return
	}
	print("id: %v, explicitId: %v\ntext: %v\nhints: %v\npriority: %v\nparent: %v\n\ncreationDate: %v\n\ndueDate: %v\nreminders: %v\nendDate: %v\ndeadline: %v\nevery: %v\n\nprogress: %v\n",
		t.ID, t.EID, t.Raw(), t.Hints, t.Priority, t.PID, t.Time.CreationDate,
		t.Time.DueDate, t.Time.Reminders, t.Time.EndDate, t.Time.Deadline, t.Time.Every, t.Prog)
}
