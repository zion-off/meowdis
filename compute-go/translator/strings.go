package translator

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func translateGet(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("get")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT value FROM strings WHERE key = ?",
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if hasWrongType(results, 1) {
				return nil, ErrWrongType
			}
			if len(results[2]) == 0 {
				return nil, nil
			}
			return rowString(results[2][0], "value"), nil
		},
	}, nil
}

func translateSet(args []string) (Translation, error) {
	if len(args) < 2 {
		return Translation{}, errWrongArgs("set")
	}

	key := args[0]
	value := args[1]
	options, err := parseSetOptions(args[2:])
	if err != nil {
		return Translation{}, err
	}

	var expiresAt any
	if !options.keepTTL {
		if options.expiresAt != nil {
			expiresAt = *options.expiresAt
		} else {
			expiresAt = nil
		}
	}

	stmts := []Statement{deleteIfExpired(key)}
	getIndex := -1
	if options.get {
		getIndex = len(stmts)
		stmts = append(stmts, Statement{
			SQL:    "SELECT value FROM strings WHERE key = ?",
			Params: []any{key},
		})
	}

	okIndex := -1
	switch {
	case options.nx:
		okIndex = len(stmts)
		stmts = append(stmts,
			Statement{
				SQL:    "INSERT OR IGNORE INTO keys (key, type, expires_at) VALUES (?, 'string', ?) RETURNING key",
				Params: []any{key, expiresAt},
			},
			Statement{
				SQL:    "INSERT INTO strings (key, value) SELECT ?, ? WHERE changes() > 0",
				Params: []any{key, value},
			},
		)
	case options.xx:
		okIndex = len(stmts)
		if options.keepTTL {
			stmts = append(stmts, Statement{
				SQL:    "UPDATE keys SET type = 'string' WHERE key = ? RETURNING key",
				Params: []any{key},
			})
		} else {
			stmts = append(stmts, Statement{
				SQL:    "UPDATE keys SET type = 'string', expires_at = ? WHERE key = ? RETURNING key",
				Params: []any{expiresAt, key},
			})
		}
		stmts = append(stmts, Statement{
			SQL:    "INSERT OR REPLACE INTO strings (key, value) SELECT ?, ? WHERE changes() > 0",
			Params: []any{key, value},
		})
	default:
		if options.keepTTL {
			stmts = append(stmts,
				Statement{
					SQL:    "UPDATE keys SET type = 'string' WHERE key = ? RETURNING key",
					Params: []any{key},
				},
				Statement{
					SQL:    "INSERT INTO keys (key, type) SELECT ?, 'string' WHERE changes() = 0",
					Params: []any{key},
				},
				Statement{
					SQL:    "INSERT OR REPLACE INTO strings (key, value) VALUES (?, ?)",
					Params: []any{key, value},
				},
			)
		} else {
			stmts = append(stmts,
				Statement{
					SQL:    "DELETE FROM keys WHERE key = ?",
					Params: []any{key},
				},
				Statement{
					SQL:    "INSERT INTO keys (key, type, expires_at) VALUES (?, 'string', ?)",
					Params: []any{key, expiresAt},
				},
				Statement{
					SQL:    "INSERT INTO strings (key, value) VALUES (?, ?)",
					Params: []any{key, value},
				},
			)
		}
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			var oldValue any
			if getIndex >= 0 {
				if len(results[getIndex]) == 0 {
					oldValue = nil
				} else {
					oldValue = rowString(results[getIndex][0], "value")
				}
			}

			if okIndex >= 0 && len(results[okIndex]) == 0 {
				return nil, nil
			}
			if options.get {
				return oldValue, nil
			}
			return "OK", nil
		},
	}, nil
}

func translateDel(args []string) (Translation, error) {
	if len(args) == 0 {
		return Translation{}, errWrongArgs("del")
	}

	stmts := make([]Statement, 0, len(args))
	for _, key := range args {
		stmts = append(stmts, Statement{
			SQL:    "DELETE FROM keys WHERE key = ? RETURNING key",
			Params: []any{key},
		})
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			var count int64
			for _, res := range results {
				count += int64(len(res))
			}
			return count, nil
		},
	}, nil
}

func translateExists(args []string) (Translation, error) {
	if len(args) == 0 {
		return Translation{}, errWrongArgs("exists")
	}

	stmts := make([]Statement, 0, len(args))
	for _, key := range args {
		stmts = append(stmts, Statement{
			SQL:    "SELECT 1 FROM keys WHERE key = ? AND (expires_at IS NULL OR expires_at > unixepoch())",
			Params: []any{key},
		})
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			var count int64
			for _, res := range results {
				if len(res) > 0 {
					count++
				}
			}
			return count, nil
		},
	}, nil
}

func translateIncr(args []string) (Translation, error) {
	return translateIncrByDelta(args, 1, "incr")
}

func translateIncrBy(args []string) (Translation, error) {
	if len(args) != 2 {
		return Translation{}, errWrongArgs("incrby")
	}

	delta, ok := parseInt64(args[1])
	if !ok {
		return Translation{}, ErrNotInteger
	}
	return translateIncrByDelta(args[:1], delta, "incrby")
}

func translateDecr(args []string) (Translation, error) {
	return translateIncrByDelta(args, -1, "decr")
}

func translateDecrBy(args []string) (Translation, error) {
	if len(args) != 2 {
		return Translation{}, errWrongArgs("decrby")
	}

	delta, ok := parseInt64(args[1])
	if !ok {
		return Translation{}, ErrNotInteger
	}
	return translateIncrByDelta(args[:1], -delta, "decrby")
}

func translateIncrByDelta(args []string, delta int64, cmd string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs(cmd)
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'string')",
			Params: []any{key},
		},
		{
			SQL:    "INSERT OR IGNORE INTO strings (key, value) VALUES (?, '0')",
			Params: []any{key},
		},
		{
			SQL:    "SELECT value FROM strings WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "UPDATE strings SET value = CAST(CAST(value AS INTEGER) + ? AS INTEGER) WHERE key = ? RETURNING value",
			Params: []any{delta, key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if hasWrongType(results, 1) {
				return nil, ErrWrongType
			}
			if len(results[4]) == 0 {
				return nil, ErrNotInteger
			}
			before := rowString(results[4][0], "value")
			if _, ok := parseInt64(before); !ok {
				return nil, ErrNotInteger
			}
			if len(results[5]) == 0 {
				return nil, ErrNotInteger
			}
			after := rowString(results[5][0], "value")
			value, ok := parseInt64(after)
			if !ok {
				return nil, ErrNotInteger
			}
			return value, nil
		},
	}, nil
}

func translateMGet(args []string) (Translation, error) {
	if len(args) == 0 {
		return Translation{}, errWrongArgs("mget")
	}

	stmts := make([]Statement, 0, len(args)*2)
	for _, key := range args {
		stmts = append(stmts,
			deleteIfExpired(key),
			Statement{
				SQL:    "SELECT value FROM strings WHERE key = ?",
				Params: []any{key},
			},
		)
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			out := make([]any, 0, len(args))
			for i := range args {
				res := results[i*2+1]
				if len(res) == 0 {
					out = append(out, nil)
					continue
				}
				out = append(out, rowString(res[0], "value"))
			}
			return out, nil
		},
	}, nil
}

func translateMSet(args []string) (Translation, error) {
	if len(args) == 0 || len(args)%2 != 0 {
		return Translation{}, errWrongArgs("mset")
	}

	stmts := make([]Statement, 0, (len(args)/2)*3)
	for i := 0; i < len(args); i += 2 {
		key := args[i]
		value := args[i+1]
		stmts = append(stmts,
			Statement{
				SQL:    "DELETE FROM keys WHERE key = ?",
				Params: []any{key},
			},
			Statement{
				SQL:    "INSERT INTO keys (key, type) VALUES (?, 'string')",
				Params: []any{key},
			},
			Statement{
				SQL:    "INSERT INTO strings (key, value) VALUES (?, ?)",
				Params: []any{key, value},
			},
		)
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			return "OK", nil
		},
	}, nil
}

type setOptions struct {
	nx        bool
	xx        bool
	get       bool
	keepTTL   bool
	expiresAt *int64
}

func parseSetOptions(args []string) (setOptions, error) {
	var opts setOptions
	var hasExpiry bool
	for i := 0; i < len(args); i++ {
		switch strings.ToUpper(args[i]) {
		case "NX":
			opts.nx = true
		case "XX":
			opts.xx = true
		case "GET":
			opts.get = true
		case "KEEPTTL":
			opts.keepTTL = true
		case "EX", "PX", "EXAT", "PXAT":
			if hasExpiry {
				return setOptions{}, errWrongArgs("set")
			}
			if i+1 >= len(args) {
				return setOptions{}, errWrongArgs("set")
			}
			value, ok := parseInt64(args[i+1])
			if !ok {
				return setOptions{}, ErrNotInteger
			}
			var expires int64
			switch strings.ToUpper(args[i]) {
			case "EX":
				expires = time.Now().Unix() + value
			case "PX":
				expires = time.Now().Unix() + value/1000
			case "EXAT":
				expires = value
			case "PXAT":
				expires = value / 1000
			}
			opts.expiresAt = &expires
			hasExpiry = true
			i++
		default:
			return setOptions{}, errWrongArgs("set")
		}
	}

	if opts.nx && opts.xx {
		return setOptions{}, errWrongArgs("set")
	}
	if opts.keepTTL && hasExpiry {
		return setOptions{}, errWrongArgs("set")
	}

	return opts, nil
}

func hasWrongType(results [][]map[string]any, index int) bool {
	return wrongTypeFor(results, index, "string")
}

func rowString(row map[string]any, key string) string {
	value, ok := row[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func parseInt64(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
