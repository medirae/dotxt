package server

import "fmt"

// TODO: ponder on moving these to rpc/shared

type ArgTask struct {
	Task string
	Path string
}

type ArgNil struct{}

var Nil = &ArgNil{}

type ArgErr struct {
	Err string
}

func (e *ArgErr) IsNil() bool {
	return e.Err == ""
}

func (e *ArgErr) Error() error {
	if e.IsNil() {
		return nil
	}
	return fmt.Errorf("%s", e.Err)
}
