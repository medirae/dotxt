package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/unicode/norm"
)

func TestRuneAt(t *testing.T) {
	assert := assert.New(t)
	assertPanic := func() {
		r := recover()
		assert.NotNil(r, "panicked")
	}
	t.Run("empty string panics", func(t *testing.T) {
		defer assertPanic()
		for ndx := -2; ndx <= 2; ndx++ {
			RuneAt("", ndx)
		}
	})
	t.Run("negative index panics", func(t *testing.T) {
		defer assertPanic()
		RuneAt("something", -1)
	})
	t.Run("index larger than or equals to length panics", func(t *testing.T) {
		defer assertPanic()
		RuneAt("abc", 3)
	})
	t.Run("normal ascii", func(t *testing.T) {
		tc := "abc"
		for ndx := range len(tc) {
			val := RuneAt(tc, ndx)
			assert.Equal(tc[ndx], byte(val))
		}
	})
	t.Run("normal utf-8", func(t *testing.T) {
		tc := "😄abcé世"
		tcRunes := []rune(tc)
		for ndx := range len(tcRunes) {
			val := RuneAt(tc, ndx)
			assert.Equal(tcRunes[ndx], val)
		}
	})
}

func TestRuneSlice(t *testing.T) {
	assert := assert.New(t)
	assertPanic := func() {
		r := recover()
		assert.NotNil(r, "panicked")
	}
	t.Run("empty string sliced with its min and max returns nothing", func(t *testing.T) {
		RuneSlice("", 0, 0)
	})
	t.Run("negative index panics", func(t *testing.T) {
		defer assertPanic()
		RuneSlice("hey", -1, 2)
	})
	t.Run("stop < start panics", func(t *testing.T) {
		defer assertPanic()
		RuneSlice("hey", 2, 1)
	})
	t.Run("start == stop returns nothing", func(t *testing.T) {
		val := RuneSlice("hey", 1, 1)
		assert.Empty(val)
	})
	t.Run("stop larger than length panics", func(t *testing.T) {
		defer assertPanic()
		RuneSlice("hey", 0, 4)
	})
	t.Run("stop equals length returns the whole string", func(t *testing.T) {
		val := RuneSlice("hey", 0, 3)
		assert.Equal("hey", val)
	})
	t.Run("normal ascii", func(t *testing.T) {
		tc := "ab"
		for ndx := range 2 {
			assert.Equal("", RuneSlice(tc, ndx, ndx))
		}
		assert.Equal("a", RuneSlice(tc, 0, 1))
		assert.Equal("b", RuneSlice(tc, 1, 2))
		assert.Equal("ab", RuneSlice(tc, 0, 2))
	})
	t.Run("normal utf-8", func(t *testing.T) {
		tc := norm.NFC.String("😄é世")
		tcRunes := []rune(tc)
		for ndx := range len(tcRunes) {
			assert.Equal("", RuneSlice(tc, ndx, ndx))
		}
		assert.Equal(norm.NFC.String("😄"), RuneSlice(tc, 0, 1))
		assert.Equal(norm.NFC.String("é"), RuneSlice(tc, 1, 2))
		assert.Equal(norm.NFC.String("世"), RuneSlice(tc, 2, 3))
		assert.Equal(norm.NFC.String("😄é"), RuneSlice(tc, 0, 2))
		assert.Equal(norm.NFC.String("é世"), RuneSlice(tc, 1, 3))
		assert.Equal(norm.NFC.String("😄é世"), RuneSlice(tc, 0, 3))
	})
	t.Run("default stop value", func(t *testing.T) {
		tc := norm.NFC.String("😄é世")
		assert.Equal(tc, RuneSlice(tc, 0))
		assert.Equal("ab", RuneSlice("ab", 0))
	})
}
