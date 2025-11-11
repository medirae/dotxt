package task

import (
	"dotxt/utils"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/viper"
)

// Used to carry intermediary info for a token,
// or a part of a token,
// or a formatted string from a token
type rToken struct {
	token         *Token
	raw           string
	color         string
	dominantColor string
}

func (rt *rToken) getColor() string {
	if rt.dominantColor != "" {
		return rt.dominantColor
	}
	if rt.color != "" {
		return rt.color
	}
	return "print.color-default"
}

type rInfo struct {
	maxLen       int
	idLen        int
	countLen     int
	doneCountLen int
}

func (ri *rInfo) set(alt *rInfo) {
	ri.maxLen = max(ri.maxLen, alt.maxLen)
	ri.idLen = max(ri.idLen, alt.idLen)
	ri.countLen = max(ri.countLen, alt.countLen)
	ri.doneCountLen = max(ri.doneCountLen, alt.doneCountLen)
}

// Used to carry intermediary info for a task
type rTask struct {
	task    *Task
	tokens  []*rToken
	id      int
	idColor string
	decor   bool
	depth   int
	focused bool
	rInfo
}

func (r *rTask) stringify(toColor bool, maxWidth int) string {
	var idPrefix string
	// metadata
	md := struct {
		length        int
		out           strings.Builder
		newLinePrefix string
		newLineLen    int
	}{
		newLinePrefix: strings.Repeat(" ", r.idLen+1),
		newLineLen:    r.idLen + 1,
	}
	{
		if depth := r.depth * (r.idLen + 1); depth > 0 {
			depthSpace := strings.Repeat(" ", depth)
			md.newLinePrefix += depthSpace
			md.newLineLen += depth
			idPrefix += depthSpace
		}
		if r.task != nil && r.task.Prog != nil {
			progLen := r.countLen + 1 + r.doneCountLen + 6 + 1 +
				viper.GetInt("print.progress.bartext-len") + 1
			md.newLinePrefix += strings.Repeat(" ", progLen)
			md.newLineLen += progLen
		}
	}
	var fold func(string) string
	fold = func(text string) string {
		if maxWidth == -1 { // for total length purposes
			return text
		}
		n := utils.RuneCount(text)
		if n+md.length <= maxWidth { // fits
			md.length += n
			return text
		}
		if md.length >= maxWidth-1 { // current line has no space whatsoever
			md.length = md.newLineLen
			if text == " " {
				return "\n" + md.newLinePrefix
			}
			return "\n" + md.newLinePrefix + fold(text)
		}
		if n > maxWidth || r.idLen+1+n > maxWidth ||
			md.newLineLen+r.idLen+1+n > maxWidth { // string is so long it has to be split
			oldLen := md.length
			md.length = md.newLineLen
			return utils.RuneSlice(text, 0, maxWidth-oldLen-1) + "\\\n" +
				md.newLinePrefix +
				fold(utils.RuneSlice(text, maxWidth-oldLen-1))
		}
		// str is long enough to not fit current line and not long enough to be splitted
		md.length = md.newLineLen + n
		return "\n" + md.newLinePrefix + text
	}
	write := func(color, text string) {
		out := fold(text)
		if toColor {
			out = colorize(color, out)
		}
		md.out.WriteString(out)
	}
	writeSpace := func() {
		md.out.WriteString(fold(" "))
	}

	if !r.decor {
		write(r.idColor, idPrefix)
		write(r.idColor, fmt.Sprintf("%0*d", r.idLen, r.id))
		writeSpace()
	} else {
		write("print.color-default", fmt.Sprintf("%s%s", idPrefix, strings.Repeat(" ", r.idLen)))
	}

	if r.task != nil && r.task.IsCollapsed() {
		count, stack := 0, slices.Clone(r.task.Children)
		for len(stack) > 0 {
			stack = append(stack[1:], stack[0].Children...)
			count++
		}
		val := "+"
		if count > 0 {
			val += fmt.Sprintf("|%d", count)
		}
		write("print.color-collapsed", val)
		writeSpace()
	}

	for ndx, tk := range r.tokens {
		if ndx > 0 {
			isPrevSemicolon := r.tokens[ndx-1].token != nil && r.tokens[ndx-1].token.Type == TokenText &&
				r.tokens[ndx-1].token.Key == ";"
			isCurSemicolon := tk.token != nil && tk.token.Type == TokenText &&
				tk.token.Key == ";"
			if !isPrevSemicolon && !isCurSemicolon {
				writeSpace()
			}
		}
		if tk.token != nil && tk.token.Type == TokenProgress {
			cl, dcl := r.countLen, r.doneCountLen
			prog := tk.token.Value.(*Progress)
			if r.task.PID != nil {
				cl, dcl = utils.RuneCount(strconv.Itoa(prog.Count)), utils.RuneCount(strconv.Itoa(prog.DoneCount))
			}
			parts := formatProgress(prog, cl, dcl)
			for _, pt := range parts {
				write(pt.getColor(), pt.raw)
			}
			continue
		}
		write(tk.getColor(), tk.raw)
	}
	return strings.ReplaceAll(strings.TrimRightFunc(md.out.String(), unicode.IsSpace), " \n", "\n")
}
