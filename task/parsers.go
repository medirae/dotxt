package task

import (
	"dotxt/logging"
	"dotxt/terrors"
	"dotxt/utils"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

/*
ommitted fields will be replaced from RightNow, I think
%Y:2006, %y:06, %m:01, %d:02, %H:15, %M:04, %S:05, %b:Jan

datetime: [date][[time]]
time:

	3-parter: T%H-%M-%S
	2-parter:
		T<a>-<b>:
			if a <= 23: T%H-%M
			else:		T%M-%S
	1-parter:
		T<a>:
			if a <= 23: T%H
			else: 		T%M

date:

	3-parter:
		%Y-%m-%d
		%y-%m-%d
		%Y-%b-%d
		%y-%b-%d
	2-parter:
		<a>-<b>:
			if a >= 13 || <b> == %b:
				- %Y-%m
				- %Y-%b
				- %y-%m
				- %y-%b
			else:
				- %m-%d
				- %b-%d
	1-parter:
		<a>:
			if a >= 13:
				- %Y
				- %y
			else:
				- %m
				- %b
*/
func parseAbsoluteDatetime(absDt string) (*time.Time, error) {
	if absDt == "" {
		return nil, fmt.Errorf("%w: empty", terrors.ErrParse)
	}
	var timeStr string
	if ndx := strings.IndexRune(absDt, 'T'); ndx != -1 {
		timeStr = absDt[ndx+1:]
		absDt = absDt[:ndx]
		if len(timeStr) == 0 {
			return nil, fmt.Errorf("%w: invalid use of 'T'", terrors.ErrParse)
		}
		parts := strings.Split(timeStr, "-")
		n := len(parts)
		if n == 0 || n > 3 {
			return nil, fmt.Errorf("%w: invalid use 'T': either no string afterwards or too many dashes", terrors.ErrParse)
		}
		vals := make([]int, n)
		for ndx := range parts {
			val, err := strconv.Atoi(parts[ndx])
			if err != nil {
				return nil, fmt.Errorf("%w: %w: invalid hour, minute or second: %w", terrors.ErrParse, terrors.ErrValue, err)
			}
			if val >= 60 {
				return nil, fmt.Errorf("%w: %w: invalid hour, minute or second value: '%d'", terrors.ErrParse, terrors.ErrValue, val)
			}
			vals[ndx] = val
			parts[ndx] = fmt.Sprintf("%02s", parts[ndx])
		}
		switch n {
		case 3:
			if vals[0] >= 24 {
				return nil, fmt.Errorf("%w: %w: invalid hour value: '%d'", terrors.ErrParse, terrors.ErrValue, vals[0])
			}
			timeStr = fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
		case 2:
			if vals[0] <= 23 {
				timeStr = fmt.Sprintf("%s-%s-00", parts[0], parts[1])
			} else {
				timeStr = fmt.Sprintf("00-%s-%s", parts[0], parts[1])
			}
		case 1:
			if vals[0] <= 23 {
				timeStr = fmt.Sprintf("%s-00-00", parts[0])
			} else {
				timeStr = fmt.Sprintf("00-%s-00", parts[0])
			}
		}
	} else {
		timeStr = "00-00-00"
	}

	var dateStr string
	if len(absDt) > 0 {
		parts := strings.Split(absDt, "-")
		n := len(parts)
		if n == 0 || n > 3 {
			return nil, fmt.Errorf("%w: invalid use of date: too many dashes", terrors.ErrParse)
		}
		vals := make([]int, n)
		for ndx := range parts {
			val, err := strconv.Atoi(parts[ndx])
			if err == nil {
				if val >= 3000 {
					return nil, fmt.Errorf("%w: %w: invalid year, month or day value: '%d'",
						terrors.ErrParse, terrors.ErrValue, val)
				}
				vals[ndx] = val
			} else {
				t, err := time.ParseInLocation("Jan", parts[ndx], time.Local)
				if err != nil {
					return nil, fmt.Errorf("%w: %w: invalid date value which is neither 'Jan' nor a number: '%s'",
						terrors.ErrParse, terrors.ErrValue, parts[ndx])
				}
				vals[ndx] = math.MaxInt
				parts[ndx] = t.Format("01")
			}
			parts[ndx] = fmt.Sprintf("%02s", parts[ndx])
		}
		handleSmallYear := func(year string) string {
			t, _ := time.ParseInLocation("06", year, time.Local)
			return t.Format("2006")
		}
		switch n {
		case 3:
			if vals[0] < 100 {
				parts[0] = handleSmallYear(parts[0])
			}
			dateStr = fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
		case 2:
			if (vals[0] >= 13 && vals[0] < 3000) || vals[1] == math.MaxInt {
				if vals[0] < 100 {
					parts[0] = handleSmallYear(parts[0])
				}
				dateStr = fmt.Sprintf("%s-%s-01", parts[0], parts[1])
			} else {
				dateStr = fmt.Sprintf("%s-%s-%s", rightNow.Format("2006"), parts[0], parts[1])
			}
		case 1:
			if vals[0] >= 13 && vals[0] < 3000 {
				if vals[0] < 100 {
					parts[0] = handleSmallYear(parts[0])
				}
				dateStr = fmt.Sprintf("%s-01-01", parts[0])
			} else {
				dateStr = fmt.Sprintf("%s-%s-01", rightNow.Format("2006"), parts[0])
			}
		}
	} else {
		dateStr = fmt.Sprintf("%s-01-01", rightNow.Format("2006"))
	}

	t, err := time.ParseInLocation("2006-01-02T15-04-05", fmt.Sprintf("%sT%s", dateStr, timeStr), time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

/*
datetime: [date]T[[time]]
date:

	if month == 1 and day == 1: %Y
	else if day == 1:			%Y-%m
	else: 						%Y-%m-%d

time:

	if hour == 0 and minute == 0 and second == 0:
	else if minute == 0 and second == 0:       	  T%H
	else if second == 0:						  T%H-%M
	else:										  T%H-%M-%S
*/
func unparseAbsoluteDatetime(absDt time.Time) string {
	var dateStr string
	if absDt.Month() == 1 && absDt.Day() == 1 {
		dateStr = absDt.Format("2006")
	} else if absDt.Day() == 1 {
		dateStr = absDt.Format("2006-01")
	} else {
		dateStr = absDt.Format("2006-01-02")
	}
	var timeStr string
	if absDt.Hour() == 0 && absDt.Minute() == 0 && absDt.Second() == 0 {
	} else if absDt.Minute() == 0 && absDt.Second() == 0 {
		timeStr = absDt.Format("15")
	} else if absDt.Second() == 0 {
		timeStr = absDt.Format("15-04")
	} else {
		timeStr = absDt.Format("15-04-05")
	}
	if len(timeStr) > 0 {
		timeStr = "T" + timeStr
	}
	return fmt.Sprintf("%s%s", dateStr, timeStr)
}

func parseDuration(dur string) (*time.Duration, error) {
	if err := validateEmptyText(dur); err != nil {
		return nil, err
	}
	sign := 1
	if dur[0] == '-' {
		sign = -1
		dur = dur[1:]
	} else if dur[0] == '+' {
		dur = dur[1:]
	}
	if dur == "0" {
		return utils.MkPtr(time.Duration(0)), nil
	}

	const day = 24 * time.Hour
	var duration float64
	var numStr string
	for _, char := range dur {
		if unicode.IsDigit(char) || char == '.' {
			numStr += string(char)
			continue
		}
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: number conversion of '%s' failed: %w", terrors.ErrParse, numStr, err)
		}
		var multiplier time.Duration
		switch char {
		case 'y':
			multiplier = 365 * day
		case 'm':
			multiplier = 30 * day
		case 'w':
			multiplier = 7 * day
		case 'd':
			multiplier = day
		case 'h':
			multiplier = time.Hour
		case 'M':
			multiplier = time.Minute
		case 's':
			multiplier = time.Second
		default:
			return nil, fmt.Errorf("%w: unexpected time unit '%q'", terrors.ErrParse, char)
		}
		duration += float64(multiplier) * num
		numStr = ""
	}
	if numStr != "" {
		return nil, fmt.Errorf("%w: trailing numbers without a time unit '%q'", terrors.ErrParse, numStr)
	}
	return utils.MkPtr(time.Duration(sign) * time.Duration(duration)), nil
}

func unparseDuration(dur time.Duration) string {
	totalSec := int(dur.Seconds())
	if totalSec == 0 {
		return "0s"
	}
	var sign string
	if totalSec < 0 {
		sign = "-"
		totalSec *= -1
	}

	const (
		secPerMin = 60
		secPerHr  = secPerMin * 60
		secPerDay = secPerHr * 24
		secPerWk  = secPerDay * 7
		secPerMo  = secPerDay * 30
		secPerYr  = secPerDay * 365
	)

	years := totalSec / secPerYr
	totalSec %= secPerYr
	months := totalSec / secPerMo
	totalSec %= secPerMo
	weeks := totalSec / secPerWk
	totalSec %= secPerWk
	days := totalSec / secPerDay
	totalSec %= secPerDay
	hours := totalSec / secPerHr
	totalSec %= secPerHr
	mins := totalSec / secPerMin
	secs := totalSec % secPerMin

	parts := []string{sign}
	if years > 0 {
		parts = append(parts, fmt.Sprintf("%dy", years))
	}
	if months > 0 {
		parts = append(parts, fmt.Sprintf("%dm", months))
	}
	if weeks > 0 {
		parts = append(parts, fmt.Sprintf("%dw", weeks))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if mins > 0 {
		parts = append(parts, fmt.Sprintf("%dM", mins))
	}
	if secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}

	return strings.Join(parts, "")
}

func unparseRelativeDatetime(dt, rel time.Time) string {
	return unparseDuration(dt.Sub(rel))
}

// TODO: rewrite supporting errors. unparseRelativeDatetime should't be called on TokenDates that aren't relative
func (tk *Token) unparseRelativeDatetime(val *time.Time) string {
	tkDt := tk.Value.(*TokenDateValue)
	if val == nil {
		val = tkDt.Value
	}
	relVal := tkDt.RelVal
	if relVal == nil {
		relVal = &rightNow
	}
	newDtTxt := unparseRelativeDatetime(*val, *relVal)
	if strings.ContainsRune(*tk.raw, ':') {
		newDtTxt = fmt.Sprintf("%s:%s", tkDt.RelKey, newDtTxt)
	}
	newDtTxt = fmt.Sprintf("$%s=%s", tk.Key, newDtTxt)
	return newDtTxt
}

func getTemporalFallback(field, dt string) (string, string, error) {
	fallback, ok := temporalFallback[field]
	if !ok {
		return "", "", fmt.Errorf("%w: %w: field '%s' not in temporalFallback map", terrors.ErrParse, terrors.ErrNotFound, field)
	}
	if ndx := strings.IndexRune(dt, ':'); ndx != -1 {
		if ndx == 0 || ndx == utils.RuneCount(dt)-1 {
			return "", "", fmt.Errorf("%w: invalid use of 'var:dt': '%s'", terrors.ErrParse, dt)
		}
		fallback = utils.RuneSlice(dt, 0, ndx)
		_, ok := temporalFallback[fallback]
		if !ok {
			return "", "", fmt.Errorf("%w: %w: field '%s' not in temporalFallback map", terrors.ErrParse, terrors.ErrNotFound, fallback)
		}
		dt = utils.RuneSlice(dt, ndx+1)
	}
	return fallback, dt, nil
}

func parseTmpRelativeDatetime(field, dt string) (string, *time.Duration, error) {
	fallback, dt, err := getTemporalFallback(field, dt)
	if err != nil {
		return "", nil, err
	}
	duration, err := parseDuration(dt)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed parsing tmp relative datetime: %w", terrors.ErrParse, err)
	}
	return fallback, duration, nil
}

func parseProgress(token string) (*Progress, error) {
	parts := strings.Split(token, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("%w: $progress: number of '/' is less than 2: required fields not provided in '%s'", terrors.ErrParse, token)
	}
	unit, count, doneCount := parts[0], parts[1], parts[2]
	var category string
	switch len(parts) {
	case 4:
		category = parts[3]
	}
	if unit == "" {
		return nil, fmt.Errorf("%w: %w: $progress: unit is empty", terrors.ErrParse, terrors.ErrValue)
	}
	doneCountInt, err := strconv.Atoi(doneCount)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: $progress: doneCount to int: %w", terrors.ErrParse, terrors.ErrValue, err)
	}
	if doneCountInt < 1 {
		return nil, fmt.Errorf("%w: %w: $progress: doneCount minimum is 1: '%d'", terrors.ErrParse, terrors.ErrValue, doneCountInt)
	}
	countInt, err := strconv.Atoi(count)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: $progress: count to int: '%w'", terrors.ErrParse, terrors.ErrValue, err)
	}
	if countInt < 0 {
		return nil, fmt.Errorf("%w: %w: $progress: count minimum is 0: '%d'", terrors.ErrParse, terrors.ErrValue, countInt)
	}
	countInt = min(countInt, doneCountInt)
	return &Progress{
		Unit: unit, Category: category,
		Count: countInt, DoneCount: doneCountInt,
	}, nil
}

func unparseProgress(progress Progress) (string, error) {
	if progress.Unit == "" {
		return "", fmt.Errorf("%w: progress unit cannot be empty", terrors.ErrValue)
	}
	if progress.DoneCount <= 0 {
		return "", fmt.Errorf("%w: progress doneCount '%d' cannot be less than 1", terrors.ErrValue, progress.DoneCount)
	}
	if progress.Count < 0 {
		return "", fmt.Errorf("%w: progress count '%d' cannot be less than 0", terrors.ErrValue, progress.Count)
	}
	if progress.Count > progress.DoneCount {
		return "", fmt.Errorf("%w: progress count '%d' cannot be greater than doneCount '%d'", terrors.ErrValue, progress.Count, progress.DoneCount)
	}
	base := fmt.Sprintf("%s/%d/%d", progress.Unit, progress.Count, progress.DoneCount)
	if progress.Category != "" {
		base += "/" + progress.Category
	}
	return base, nil
}

/*
ai says to look out for these:
  - The for len(resolved)-1 < dtCount loop can hang
  - If changed is false you never break or update len(resolved), so the loop never exits.
  - You also decrement dtCount inside the !changed block, but only for the ones you explicitly revert—what about other unresolved nodes?
  - Magic “3” iteration depth
  - That will only chase at most 3 hops down the fallback chain. If you ever extend temporalFallback (e.g. add more levels), you’d have to bump that constant.
  - Dual key-remapping for reminders
  - You temporarily rename "r" to "r<idx>" in the first pass, then map everything using nodes[key] loops over the hard-coded slice append([]string{"c", "due", …}, rKeys…). If someone writes two reminders back-to-back but with the same raw text (e.g. duplicate tokens), the index hack could mismatch.
    TODO: write an exhaustive testcase generator against these
*/
func resolveDates(tokens []*Token) []error {
	var errs []error
	var dtCount int
	var reminders []*Token
	var rKeys []string
	nodes := make(map[string]*Token)
	resolved := make(map[string]*TokenDateValue)
	resolved["rn"] = &TokenDateValue{Value: &rightNow}
	for ndx, tk := range tokens {
		if tk.Type != TokenDate {
			continue
		}
		if tk.Key == "r" {
			tk.Key = fmt.Sprintf("r%d", ndx)
			reminders = append(reminders, tk)
			rKeys = append(rKeys, tk.Key)
		}

		tdv := tk.Value.(*TokenDateValue)
		if tdv.Value != nil { // validate absolute
			dtCount++
			resolved[tk.Key] = tdv
			continue
		}

		if tdv.Offset == nil || tdv.RelKey == "" {
			dateToTextToken(tk)
			continue
		}
		k := tk.Key
		if strings.HasPrefix(k, "r") {
			k = "r"
		}
		if allowedRels, exists := allowedTemporalRelations[k]; !exists {
			dateToTextToken(tk)
			continue
		} else if !slices.Contains(allowedRels, tdv.RelKey) {
			dateToTextToken(tk)
			continue
		}

		// validate relative
		dtCount++
		nodes[tk.Key] = tk
	}
	for len(resolved)-1 < dtCount { // ?
		changed := false
		// this order is based on temporalFallback and please review this if you change that
		for _, key := range append([]string{"c", "due", "end", "dead"}, rKeys...) {
			tk, ok := nodes[key]
			if !ok { // validate relative
				continue
			}
			if _, ok := resolved[key]; ok { // validate unresolved
				continue
			}
			tdv := tk.Value.(*TokenDateValue)
			ref := tdv.RelKey
			for range 3 { // 3 is the max depth from temporalFallback's current state
				if base, ok := resolved[ref]; ok {
					tdv.RelVal = base.Value
					tdv.Value = utils.MkPtr(tdv.RelVal.Add(*tdv.Offset))
					resolved[key] = tdv
					changed = true
				} else {
					tmp, ok := temporalFallback[ref]
					if !ok {
						continue
					}
					ref = tmp
				}
			}
		}
		if !changed {
			errs = append(errs, fmt.Errorf("%w: dependency of date fields are not resolvable", terrors.ErrValue))
			for key, tk := range nodes {
				if tdv, ok := resolved[key]; ok && tdv.Value != nil && (tdv.RelKey == "" || tdv.RelVal != nil) {
					continue
				} else {
					errs = append(errs, fmt.Errorf("%w: somehow the date '%s' was not resolved", terrors.ErrNotFound, key))
					dateToTextToken(tk)
					dtCount--
				}
			}
		}
	}
	for _, tk := range reminders {
		tk.Key = "r"
	}
	return errs
}

func dateToTextToken(dt *Token) {
	dt.Key = ""
	dt.Type = TokenText
	dt.Value = dt.raw
}

/*
tokenizeLine splits a line into tokens with the following rules:
  - A single unescaped space is just a separator.  Extra spaces beyond the first
    become explicit " " tokens (e.g. "a  b" → ["a","b"," "]).
  - "\ " produces a literal space in the token.
  - Double, single, or back-tick quotes group everything (including spaces) until
    the matching closing quote; inside them only \" (or \' or \`) is special.
    The opening and closing characters of \", \', and \` are also kept intact in the token.
  - If a quote never closes, we abandon the quote and treat the rest as plain text.
  - If a \; is provided the token is forced to be split
  - The function replaces any \n with '\ '
*/
func tokenizeLine(line string) []string {
	line = strings.NewReplacer("\t", "    ", "\r", " ", "\n", " ", "\v", " ", "\f", " ").Replace(line)
	var tokens []string
	var cur strings.Builder
	rs := []rune(line)
	n := len(rs)

	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	quotes := []rune{'"', '\'', '`'}
	isQuote := func(q rune) bool {
		return slices.Contains(quotes, q)
	}
	isSep := func(i int) bool {
		return i+1 < n && rs[i] == '\\' && rs[i+1] == ';'
	}

	for i := 0; i < n; {
		r := rs[i]

		if r == '\\' && i+1 < n && (rs[i+1] == ' ' || isQuote(rune(rs[i+1]))) {
			if isQuote(rune(rs[i+1])) {
				cur.WriteRune('\\')
			}
			cur.WriteRune(rs[i+1])
			i += 2
			continue
		}

		if isQuote(r) {
			snapTokens := slices.Clone(tokens)
			snapStr := cur.String()
			snapI := i

			open := r
			cur.WriteRune(open)
			i++ // consume the opening quote
			for i < n {
				if rs[i] == '\\' && i+1 < n && rs[i+1] == open {
					cur.WriteRune('\\')
					cur.WriteRune(open)
					i += 2
					continue
				}
				if rs[i] == open {
					cur.WriteRune(open)
					i++ // consume closing
					break
				}
				cur.WriteRune(rs[i])
				i++
			}
			// check unterminated
			if i > n || (i <= n && rs[i-1] != open) {
				tokens = snapTokens
				cur.Reset()
				cur.WriteString(snapStr)
				i = snapI
			} else {
				flush()
				continue
			}
		}

		if isSep(i) || r == ' ' {
			flush()
			j := i
			for j < n {
				if isSep(j) {
					j += 2
					continue
				}
				if rs[j] == ' ' {
					j++
					continue
				}
				break
			}
			count := j - i
			if !((i > 0 && i < n-1) && count == 1 && r == ' ') { // skip one space between two tokens
				tokens = append(tokens, string(rs[i:j]))
			}
			i = j
			continue
		}

		if r == '\\' && i+1 < n && rs[i+1] == ';' {
			flush()
			cur.WriteString("\\;")
			flush()
			i += 2
			continue
		}

		cur.WriteRune(r)
		i++
	}

	flush()
	return tokens
}

func parseTokens(line string) ([]*Token, []error) {
	specialFields := make(map[string]bool)
	var tokens []*Token
	var errs []error
	handleTokenText := func(tokenStr string, err error) {
		if err != nil {
			errs = append(errs, err)
		}
		tokens = append(tokens, &Token{Type: TokenText, raw: &tokenStr, Value: &tokenStr})
	}
	tokenStrings := tokenizeLine(line)
	if len(tokenStrings) > 0 {
		if err := validatePriority(tokenStrings[0]); err == nil {
			prioKey := "priority"
			if tokenStrings[0][0] == '[' {
				prioKey = "anti-priority"
			}
			tokens = append(tokens, &Token{
				Type: TokenPriority, Key: prioKey,
				raw: &tokenStrings[0], Value: &tokenStrings[0],
			})
			tokenStrings = tokenStrings[1:]
		}
	}
	for ndx, tokenStr := range tokenStrings {
		if n := utils.RuneCount(tokenStr); (utils.RuneAt(tokenStr, 0) == ' ' ||
			(utils.RuneAt(tokenStr, 0) == '\\' && n >= 2 && utils.RuneAt(tokenStr, 1) == ';')) &&
			func() bool { // ";" text token
				spaces := 0
				for i := range n {
					switch utils.RuneAt(tokenStr, i) {
					case ' ':
						spaces++
					case '\\':
						if i+1 < n && utils.RuneAt(tokenStr, i+1) == ';' {
							return true
						}
						return false
					default:
						return false
					}
				}
				return spaces >= 2 || (spaces >= 1 && (ndx == len(tokenStrings)-1 || ndx == 0))
			}() {
			tokens = append(tokens, &Token{
				Type: TokenText, raw: &tokenStr,
				Key: ";", Value: &tokenStr,
			})
			continue
		}
		switch utils.RuneAt(tokenStr, 0) {
		case '+', '@', '#', '!', '?', '*', '&':
			if err := validateHint(tokenStr); err != nil {
				handleTokenText(tokenStr, nil)
				continue
			}
			tokens = append(tokens, &Token{
				Type: TokenHint,
				Key:  utils.RuneSlice(tokenStr, 0, 1),
				raw:  &tokenStr, Value: &tokenStr,
			})
		case '$':
			// $key
			if utils.RuneCount(tokenStr) >= 2 && !strings.ContainsRune(tokenStr, '=') {
				key := utils.RuneSlice(tokenStr, 1, utils.RuneCount(tokenStr))
				switch key {
				case "focus":
					tokens = append(tokens, &Token{
						Type: TokenFormat, Key: "focus",
						raw: &tokenStr,
					})
				case "urgent":
					tokens = append(tokens, &Token{
						Type: TokenPriority, Key: "urgent",
						raw: &tokenStr, Value: &tokenStr,
					})
				default:
					handleTokenText(tokenStr, nil)
				}
				continue
			}

			// $key=value
			keyValue := strings.SplitN(utils.RuneSlice(tokenStr, 1), "=", 2)
			if len(keyValue) != 2 {
				errs = append(errs, fmt.Errorf("%w: zero or multiple `=` were found: %s", terrors.ErrParse, tokenStr))
				handleTokenText(tokenStr, nil)
				continue
			}
			key, value := keyValue[0], keyValue[1]
			if validateEmptyText(value) != nil {
				handleTokenText(tokenStr, nil)
				continue
			}
			_, seenKey := specialFields[key]
			if key != "r" && seenKey {
				continue
			} else if key != "r" {
				specialFields[key] = true
			}
			switch key {
			case "-id", "id", "P":
				k := strings.Replace(key, "-", "", 1)
				tokens = append(tokens, &Token{
					Type: TokenID, raw: &tokenStr,
					Key: k, Value: &value,
				})
			case "c", "due", "end", "dead", "r":
				var err error
				var tkValue TokenDateValue
				tkValue.Value, err = parseAbsoluteDatetime(value)
				if err != nil {
					tkValue.RelKey, tkValue.Offset, err = parseTmpRelativeDatetime(key, value)
					if err != nil {
						handleTokenText(tokenStr, fmt.Errorf("%w: $%s", err, key))
						continue
					}
				}
				tokens = append(tokens, &Token{
					Type: TokenDate, raw: &tokenStr,
					Key: key, Value: &tkValue,
				})
			case "every":
				duration, err := parseDuration(value)
				oneDay := 24 * 60 * 60 * time.Second
				tenYears := 10 * 365 * 24 * 60 * 60 * time.Second
				if err != nil || *duration < oneDay || tenYears <= *duration {
					handleTokenText(tokenStr, fmt.Errorf("%w: $every: %w", terrors.ErrParse, err))
					continue
				}
				tokens = append(tokens, &Token{
					Type: TokenDuration, raw: &tokenStr,
					Key: key, Value: duration,
				})
			case "p":
				progress, err := parseProgress(value)
				if err != nil {
					handleTokenText(tokenStr, err)
					continue
				}
				tokens = append(tokens, &Token{
					Type: TokenProgress, raw: &tokenStr,
					Key: "p", Value: progress,
				})
			case "mit":
				mitVal, err := strconv.Atoi(value)
				if err != nil {
					handleTokenText(tokenStr, fmt.Errorf("%w: $mit=%s: %w", terrors.ErrParse, value, err))
					continue
				} else if mitVal < 0 {
					handleTokenText(tokenStr, fmt.Errorf("%w: %w: $mit value '%d' must be greater than or equal to '0'", terrors.ErrParse, terrors.ErrValue, mitVal))
					continue
				}
				tokens = append(tokens, &Token{
					Type: TokenPriority, raw: &tokenStr,
					Key: "mit", Value: &mitVal,
				})
			default:
				handleTokenText(tokenStr, nil)
			}
		case '\'', '"', '`':
			if utils.RuneCount(tokenStr) == 1 ||
				!slices.Contains([]rune{'"', '\'', '`'}, utils.RuneAt(tokenStr, utils.RuneCount(tokenStr)-1)) {
				handleTokenText(tokenStr, nil)
				continue
			}
			tokens = append(tokens, &Token{Type: TokenText, Key: "quote", raw: &tokenStr, Value: &tokenStr})
		default:
			handleTokenText(tokenStr, nil)
		}
	}
	tmpErrs := resolveDates(tokens)
	if len(tmpErrs) > 0 {
		errs = append(errs, tmpErrs...)
	}
	return tokens, errs
}

func ParseTask(id *int, line string) (*Task, error) {
	if err := validateEmptyText(line); err != nil {
		return nil, err
	}

	// this tries to append as many extra character accents back into
	// the previous rune as it is feasible to do so; which depends
	// on whether the combined rune exists or not.
	//
	// if there are extra accents upon a rune, it's highly
	// likely that there won't be a dedicated rune point,
	// so there will be probably be multiple runes left.
	// also, NFC doesn't guarantee the order of the accents,
	// to be as was given. they will be ordered using `ccc`.
	line = norm.NFC.String(line)

	task := &Task{ID: id, Time: new(Temporal)}
	tokens, warns := parseTokens(line)
	for ndx := range tokens {
		token := tokens[ndx]
		switch token.Type {
		case TokenID:
			val := token.Value.(*string)
			switch token.Key {
			case "id":
				task.EID = val
			case "P":
				task.PID = val
			}
		case TokenHint:
			task.Hints = append(task.Hints, token.Value.(*string))
		case TokenPriority:
			switch token.Key {
			case "urgent":
				task.Urgent = true
			case "mit":
				task.MIT = token.Value.(*int)
			default:
				task.Priority = token.Value.(*string)
			}
		case TokenDate:
			switch token.Key {
			case "c":
				val := token.Value.(*TokenDateValue)
				if val.Value.After(rightNow) {
					dateToTextToken(tokens[ndx])
					continue
				}
				task.Time.CreationDate = val.Value
			case "due":
				task.Time.DueDate = token.Value.(*TokenDateValue).Value
			case "r":
				task.Time.Reminders = append(task.Time.Reminders, token.Value.(*TokenDateValue).Value)
			case "end":
				task.Time.EndDate = token.Value.(*TokenDateValue).Value
			case "dead":
				task.Time.Deadline = token.Value.(*TokenDateValue).Value
			}
		case TokenDuration:
			task.Time.Every = token.Value.(*time.Duration)
		case TokenProgress:
			task.Prog = token.Value.(*Progress)
		case TokenFormat:
			if task.Fmt == nil {
				task.Fmt = new(Format)
			}
			switch token.Key {
			case "focus":
				task.Fmt.Focus = true
			}
		}
	}
	task.Tokens = tokens
	if task.Time.CreationDate == nil {
		tmp := rightNow
		task.Time.CreationDate = &tmp
		task.Tokens = append(task.Tokens, &Token{
			Type: TokenDate,
			Key:  "c",
			raw:  utils.MkPtr(fmt.Sprintf("$c=%s", unparseAbsoluteDatetime(rightNow))),
			Value: &TokenDateValue{
				Value: task.Time.CreationDate,
			},
		})
	}
	if task.Time.DueDate == nil && task.Time.Every != nil {
		task.Time.DueDate = utils.MkPtr(task.Time.CreationDate.Add(*task.Time.Every))
		task.Tokens = append(task.Tokens, &Token{
			Type: TokenDate,
			Key:  "due",
			raw:  utils.MkPtr(fmt.Sprintf("$due=%s", unparseRelativeDatetime(*task.Time.DueDate, *task.Time.CreationDate))),
			Value: &TokenDateValue{
				Value: task.Time.DueDate, RelKey: "due", RelVal: task.Time.CreationDate,
				Offset: task.Time.Every,
			},
		})
	}
	if task.Time.DueDate != nil && !task.Time.DueDate.After(*task.Time.CreationDate) {
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if tk != nil {
			dateToTextToken(tk)
			task.Time.DueDate = nil
		} else {
			warns = append(warns, fmt.Errorf("%w: due date token", terrors.ErrNotFound))
		}
	}
	if task.Time.Deadline != nil && (task.Time.DueDate == nil || !task.Time.Deadline.After(*task.Time.DueDate)) {
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "dead"))
		if tk != nil {
			dateToTextToken(tk)
			task.Time.Deadline = nil
		} else {
			warns = append(warns, fmt.Errorf("%w: dead date token", terrors.ErrNotFound))
		}
	}
	if task.Time.EndDate != nil && (task.Time.DueDate == nil || !task.Time.EndDate.After(*task.Time.DueDate)) {
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "end"))
		if tk != nil {
			dateToTextToken(tk)
			task.Time.EndDate = nil
		} else {
			warns = append(warns, fmt.Errorf("%w: end date token", terrors.ErrNotFound))
		}
	}
	if task.Time.EndDate != nil && task.Time.Deadline != nil {
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "dead"))
		if tk != nil {
			dateToTextToken(tk)
			task.Time.Deadline = nil
		} else {
			warns = append(warns, fmt.Errorf("%w: dead date token", terrors.ErrNotFound))
		}
		tk, _ = task.Tokens.Find(TkByTypeKey(TokenDate, "end"))
		if tk != nil {
			dateToTextToken(tk)
			task.Time.EndDate = nil
		} else {
			warns = append(warns, fmt.Errorf("%w: end date token", terrors.ErrNotFound))
		}
	}
	for ndx := len(task.Time.Reminders) - 1; ndx >= 0; ndx-- {
		if !task.Time.Reminders[ndx].After(*task.Time.CreationDate) {
			tk, _ := task.Tokens.Find(func(tk *Token) bool {
				return tk.Type == TokenDate &&
					strings.HasPrefix(tk.Key, "r") &&
					*tk.Value.(*TokenDateValue).Value == *task.Time.Reminders[ndx]
			})
			if tk != nil {
				task.Time.Reminders = slices.Delete(task.Time.Reminders, ndx, ndx+1)
				dateToTextToken(tk)
			}
		}
	}
	if task.EID != nil && task.PID != nil && *task.EID == *task.PID {
		task.revertIDtoText("P")
	}
	if task.Urgent && IsDateUrgent(task.Time.DueDate) {
		task.Urgent = false
		_, ndx := task.Tokens.Find(TkByTypeKey(TokenPriority, "urgent"))
		task.Tokens = slices.Delete(task.Tokens, ndx, ndx+1)
	}
	for _, err := range warns {
		logging.Logger.Debugf("task=\"%s\" warn=\"%s\"", task.String(), err)
	}
	return task, nil
}

// TODO: refactor this and file.go
func ParseTasks(filepath string) ([]*Task, error) {
	if !utils.FileExists(filepath) {
		return []*Task{}, os.ErrNotExist
	}
	data, err := os.ReadFile(filepath)
	if err != nil {
		return []*Task{}, err
	}
	lines := strings.Split(string(data), "\n")
	var tasks []*Task
	var errs error
	for id, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		task, err := ParseTask(&id, line)
		if err != nil {
			if errors.Is(err, terrors.ErrEmptyText) {
				continue
			}
			if errs == nil {
				errs = fmt.Errorf("line %d: %w", id, err)
			} else {
				errs = fmt.Errorf("%w\nline %d: %w", errs, id, err)
			}
		}
		tasks = append(tasks, task)
	}
	return tasks, errs
}
