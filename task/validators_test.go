package task

import (
	"dotxt/terrors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePriority(t *testing.T) {
	assert := assert.New(t)
	var err error
	t.Run("invalid: opening symbol", func(t *testing.T) {
		err = validatePriority("(some")
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, ")")

		err = validatePriority("[some")
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, "]")
	})
	t.Run("invalid: closing symbol", func(t *testing.T) {
		err = validatePriority("some)")
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, "(")

		err = validatePriority("some]")
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, "[")
	})
	t.Run("valid: spaces", func(t *testing.T) {
		err = validatePriority("(some thuing)")
		assert.NoError(err)

		err = validatePriority("[some thuing]")
		assert.NoError(err)
	})
	t.Run("valid: normal", func(t *testing.T) {
		err = validatePriority("(eyo!!)")
		assert.NoError(err)

		err = validatePriority("[eyo!!]")
		assert.NoError(err)
	})
}

func TestValidateHint(t *testing.T) {
	assert := assert.New(t)
	t.Run("valid: symbol", func(t *testing.T) {
		for _, char := range "#@+!?*&" {
			err := validateHint(fmt.Sprintf("%c1", char))
			assert.NoError(err, char)
		}
	})
	t.Run("invalid: first", func(t *testing.T) {
		for _, char := range "`1234567890-" +
			"~$%^()_" +
			"qwertyuiop[]\\" +
			"QWERTYUIOP{}|" +
			"asdfghjkl;'" +
			"ASDFGHJKL:\"" +
			"zxcvbnm,./ZXCVBNM<>" {
			err := validateHint(fmt.Sprintf("%c1", char))
			if assert.Error(err, string(char)) {
				assert.ErrorIs(err, terrors.ErrValue)
				assert.ErrorContains(err, "unsupported opening symbol")
			}
		}
	})
	t.Run("invalid: length", func(t *testing.T) {
		for _, str := range []string{"!", "+  "} {
			err := validateHint(str)
			if assert.Error(err) {
				assert.ErrorIs(err, terrors.ErrValue)
				assert.ErrorContains(err, "not long enough")
			}
		}
	})
}

func TestValidateEmptyString(t *testing.T) {
	assert := assert.New(t)
	for _, str := range []string{
		"", " ", "  ",
		"\t", "\n", "\v", "\f", "\r", string(rune(0x85)), string(rune(0xA0)),
	} {
		err := validateEmptyText(str)
		if assert.Error(err, str) {
			assert.ErrorIs(err, terrors.ErrEmptyText)
		}
	}
	assert.NoError(validateEmptyText("2"))
}

func TestValidateHexColor(t *testing.T) {
	assert := assert.New(t)
	t.Run("length", func(t *testing.T) {
		for count := range 10 {
			if count != 7 {
				err := validateHexColor(strings.Repeat("1", count))
				if assert.Error(err, count) {
					assert.ErrorIs(err, terrors.ErrValue)
					assert.ErrorContains(err, "length of hex color must be '7'")
				}
			}
		}
	})
	t.Run("symbol", func(t *testing.T) {
		err := validateHexColor("!123456")
		if assert.Error(err) {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "hex color must start with '#'")
		}
	})
	t.Run("valid hex", func(t *testing.T) {
		for _, str := range []string{
			"#!@#$%^", "#&*()_+", "#=ZXCVB", "#M<>?GH",
		} {
			err := validateHexColor(str)
			if assert.Error(err, str) {
				assert.ErrorIs(err, terrors.ErrValue)
				assert.ErrorContains(err, "hex color must only consist of letters and digits")
			}
		}
	})
	assert.NoError(validateHexColor("#Abcd12"))
}
