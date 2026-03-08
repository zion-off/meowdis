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

type Translation struct {
	Statements []Statement
	MapResult  func(results [][]map[string]any) (any, error)
}

func Translate(cmd []string) (Translation, error) {
	if len(cmd) == 0 {
		return Translation{}, errors.New("ERR empty command")
	}
	switch strings.ToUpper(cmd[0]) {
	case "GET":
		return translateGet(cmd[1:])
	case "SET":
		return translateSet(cmd[1:])
	case "DEL":
		return translateDel(cmd[1:])
	case "EXISTS":
		return translateExists(cmd[1:])
	case "INCR":
		return translateIncr(cmd[1:])
	case "INCRBY":
		return translateIncrBy(cmd[1:])
	case "DECR":
		return translateDecr(cmd[1:])
	case "DECRBY":
		return translateDecrBy(cmd[1:])
	case "MGET":
		return translateMGet(cmd[1:])
	case "MSET":
		return translateMSet(cmd[1:])
	default:
		return Translation{}, errUnknownCommand(cmd[0])
	}
}

func deleteIfExpired(key string) Statement {
	return Statement{
		SQL:    "DELETE FROM keys WHERE key = ? AND expires_at IS NOT NULL AND expires_at <= unixepoch()",
		Params: []any{key},
	}
}
