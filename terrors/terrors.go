package terrors

import (
	"errors"
	"fmt"
)

var (
	ErrArg             = errors.New("arg error")
	ErrNoArgsProvided  = fmt.Errorf("%w: no args provided error", ErrArg)
	ErrEmptyText       = errors.New("empty text error")
	ErrParse           = errors.New("failed to parse error")
	ErrValue           = errors.New("value error")
	ErrType            = errors.New("type error")
	ErrConf            = errors.New("config error")
	ErrFlag            = errors.New("flag error")
	ErrListNotInMemory = errors.New("list not in memory error")
	ErrNotFound        = errors.New("not found error")
	ErrRPC             = errors.New("rpc error")
	ErrConcurrency     = errors.New("concurrency error")
	ErrPath            = errors.New("path error")
	ErrFile            = errors.New("file error")
)

func ErrorArgNotProvided(field string) error {
	return fmt.Errorf("%w: arg '%s' not provided error", ErrArg, field)
}

func ErrorArgParse(arg string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %w: arg %s", ErrArg, ErrParse, arg)
	}
	return fmt.Errorf("%w: %w: arg %s: %w", ErrArg, ErrParse, arg, err)
}
