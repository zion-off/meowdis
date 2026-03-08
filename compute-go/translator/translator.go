// Package translator takes in Redis commands in a string array and translates them into SQLite queries.
package translator

import (
	"errors"
	"strings"
)


type Statement struct {
	SQL    string
	Params []any
}

func Translate(cmd []string) ([]Statement, error) {
	if len(cmd) == 0 {
		return nil, errors.New("ERR empty command")
	}
	switch strings.ToUpper(cmd[0]) {
	case "GET":
		return translateGet(cmd[1:])
	default:
		return nil, errUnknownCommand(cmd[0])
	}
}

func deleteIfExpired(key string) Statement {
	return Statement{
		SQL:    "DELETE FROM keys WHERE key = ? AND expires_at IS NOT NULL AND expires_at <= unixepoch()",
		Params: []any{key},
	}
}
