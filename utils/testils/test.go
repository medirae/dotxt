package testils

import (
	"fmt"
	"os"
)

// TODO: develop a swap function where it'll swap variables of viper with another
//  value and then return a revert function that upon calling will swap them back.

func EnsureTestDir() {
	path := "/tmp/dotxt-testing"
	if err := os.Mkdir(path, 0755); err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
