package translator

import (
	"errors"
	"fmt"
	"strings"
)

var ErrWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
var ErrNotInteger = errors.New("ERR value is not an integer or out of range")

func errUnknownCommand(cmd string) error {
	return fmt.Errorf("ERR unknown command '%s'", strings.ToLower(cmd))
}

func errWrongArgs(cmd string) error {
	return fmt.Errorf("ERR wrong number of arguments for '%s' command", strings.ToLower(cmd))
}
