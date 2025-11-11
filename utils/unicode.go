package utils

import (
	"bufio"
	"strings"
	"unicode/utf8"
)

// errors for index out of range are not supported and the function will panic
func RuneAt(s string, index int) rune {
	r := bufio.NewReader(strings.NewReader(s))
	for i := 0; ; i++ {
		ch, _, err := r.ReadRune()
		if err != nil {
			panic(err)
		}
		if i == index {
			return ch
		}
	}
}

// only one extra variable can be provided as stop, otherwise the function panics
func RuneSlice(s string, start int, stops ...int) string {
	if len(stops) > 1 {
		panic("runtime error: extra unsupported values provided")
	}
	var stop int
	if len(stops) == 1 {
		stop = stops[0]
	} else {
		if start < 0 {
			panic("runtime error: slice bounds out of range")
		}
		return string([]rune(s)[start:])
	}

	if start < 0 || stop < start {
		panic("runtime error: slice bounds out of range")
	}
	length := stop - start
	out := make([]rune, length)
	runeNdx := 0
	writeNdx := 0
	for bytePos := 0; bytePos < len(s) && runeNdx < stop; {
		r, size := utf8.DecodeRuneInString(s[bytePos:])
		bytePos += size
		if runeNdx >= start {
			out[writeNdx] = r
			writeNdx++
		}
		runeNdx++
	}
	if runeNdx < stop {
		panic("runtime error: slice bounds out of range")
	}
	return string(out)
}

func RuneCount(s string) int {
	return utf8.RuneCountInString(s)
}
