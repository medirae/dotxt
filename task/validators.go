package task

import (
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"strings"
	"unicode"
)

func validateHint(token string) error {
	if ln := utils.RuneCount(strings.TrimSpace(token)); ln <= 1 {
		return fmt.Errorf("%w: '%s' is not long enough with len '%d'", terrors.ErrValue, token, ln)
	}
	if !strings.ContainsRune("#@+!?*&", utils.RuneAt(token, 0)) {
		return fmt.Errorf("%w: '%s' has unsupported opening symbol '%c'", terrors.ErrValue, token, utils.RuneAt(token, 0))
	}
	return nil
}

func validateEmptyText(text string) error {
	if strings.TrimSpace(text) == "" {
		return terrors.ErrEmptyText
	}
	return nil
}

func validateHexColor(color string) error {
	if utils.RuneCount(color) != 7 {
		return fmt.Errorf("%w: length of hex color must be '7'", terrors.ErrValue)
	}
	if color[0] != '#' {
		return fmt.Errorf("%w: hex color must start with '#'", terrors.ErrValue)
	}
	for _, char := range color[1:] {
		if !(unicode.IsDigit(char) || (unicode.IsLetter(char) && strings.ContainsRune("AaBbCcDdEeFf", char))) {
			return fmt.Errorf("%w: hex color must only consist of letters and digits", terrors.ErrValue)
		}
	}
	return nil
}

func validatePriority(token string) error {
	if utils.RuneCount(token) <= 2 {
		return terrors.ErrEmptyText
	}
	prioMatch := ')'
	if prioChar := utils.RuneAt(token, 0); prioChar != '(' && prioChar != '[' {
		return fmt.Errorf("%w: %w: '(' nor '['", terrors.ErrParse, terrors.ErrNotFound)
	} else if prioChar == '[' {
		prioMatch = ']'
	}
	n := utils.RuneCount(token)
	if utils.RuneAt(token, n-1) != prioMatch {
		return fmt.Errorf("%w: %w: '%c'", terrors.ErrParse, terrors.ErrNotFound, prioMatch)
	}
	return nil
}
