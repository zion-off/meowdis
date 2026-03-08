package translator

import (
	"sort"
	"strconv"
)

func translateLPush(args []string) (Translation, error) {
	if len(args) < 2 {
		return Translation{}, errWrongArgs("lpush")
	}

	key := args[0]
	values := args[1:]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'list')",
			Params: []any{key},
		},
	}

	for _, value := range values {
		stmts = append(stmts, Statement{
			SQL:    `INSERT INTO lists (key, "index", value) SELECT ?, COALESCE(MIN("index"), 1.0) - 1.0, ? FROM lists WHERE key = ?`,
			Params: []any{key, value, key},
		})
	}

	countIndex := len(stmts)
	stmts = append(stmts, Statement{
		SQL:    "SELECT COUNT(*) as count FROM lists WHERE key = ?",
		Params: []any{key},
	})

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "list") {
				return nil, ErrWrongType
			}
			if len(results[countIndex]) == 0 {
				return int64(0), nil
			}
			count, ok := parseInt64(rowString(results[countIndex][0], "count"))
			if !ok {
				return nil, ErrNotInteger
			}
			return count, nil
		},
	}, nil
}

func translateRPush(args []string) (Translation, error) {
	if len(args) < 2 {
		return Translation{}, errWrongArgs("rpush")
	}

	key := args[0]
	values := args[1:]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'list')",
			Params: []any{key},
		},
	}

	for _, value := range values {
		stmts = append(stmts, Statement{
			SQL:    `INSERT INTO lists (key, "index", value) SELECT ?, COALESCE(MAX("index"), 0.0) + 1.0, ? FROM lists WHERE key = ?`,
			Params: []any{key, value, key},
		})
	}

	countIndex := len(stmts)
	stmts = append(stmts, Statement{
		SQL:    "SELECT COUNT(*) as count FROM lists WHERE key = ?",
		Params: []any{key},
	})

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "list") {
				return nil, ErrWrongType
			}
			if len(results[countIndex]) == 0 {
				return int64(0), nil
			}
			count, ok := parseInt64(rowString(results[countIndex][0], "count"))
			if !ok {
				return nil, ErrNotInteger
			}
			return count, nil
		},
	}, nil
}

func translateLPop(args []string) (Translation, error) {
	return translatePop(args, "lpop", true)
}

func translateRPop(args []string) (Translation, error) {
	return translatePop(args, "rpop", false)
}

func translatePop(args []string, cmd string, left bool) (Translation, error) {
	if len(args) < 1 || len(args) > 2 {
		return Translation{}, errWrongArgs(cmd)
	}

	key := args[0]
	var count int64
	var hasCount bool
	if len(args) == 2 {
		parsed, ok := parseInt64(args[1])
		if !ok {
			return Translation{}, ErrNotInteger
		}
		if parsed < 0 {
			return Translation{}, ErrNotInteger
		}
		hasCount = true
		count = parsed
	}

	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
	}

	var popSQL string
	var popParams []any
	if hasCount {
		order := "ASC"
		if !left {
			order = "DESC"
		}
		popSQL = `DELETE FROM lists WHERE key = ? AND "index" IN (SELECT "index" FROM lists WHERE key = ? ORDER BY "index" ` + order + ` LIMIT ?) RETURNING "index", value`
		popParams = []any{key, key, count}
	} else {
		if left {
			popSQL = `DELETE FROM lists WHERE key = ? AND "index" = (SELECT MIN("index") FROM lists WHERE key = ?) RETURNING "index", value`
			popParams = []any{key, key}
		} else {
			popSQL = `DELETE FROM lists WHERE key = ? AND "index" = (SELECT MAX("index") FROM lists WHERE key = ?) RETURNING "index", value`
			popParams = []any{key, key}
		}
	}

	popIndex := len(stmts)
	stmts = append(stmts, Statement{
		SQL:    popSQL,
		Params: popParams,
	})
	stmts = append(stmts, Statement{
		SQL:    "DELETE FROM keys WHERE key = ? AND NOT EXISTS (SELECT 1 FROM lists WHERE key = ?)",
		Params: []any{key, key},
	})

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "list") {
				return nil, ErrWrongType
			}
			rows := results[popIndex]
			if !hasCount {
				if len(rows) == 0 {
					return nil, nil
				}
				return rowString(rows[0], "value"), nil
			}
			if count == 0 {
				return []any{}, nil
			}
			values := make([]struct {
				index float64
				value string
			}, 0, len(rows))
			for _, row := range rows {
				indexValue, err := strconv.ParseFloat(rowString(row, "index"), 64)
				if err != nil {
					return nil, ErrNotInteger
				}
				values = append(values, struct {
					index float64
					value string
				}{
					index: indexValue,
					value: rowString(row, "value"),
				})
			}

			sort.Slice(values, func(i, j int) bool {
				if left {
					return values[i].index < values[j].index
				}
				return values[i].index > values[j].index
			})

			out := make([]any, 0, len(values))
			for _, item := range values {
				out = append(out, item.value)
			}
			return out, nil
		},
	}, nil
}

func translateLRange(args []string) (Translation, error) {
	if len(args) != 3 {
		return Translation{}, errWrongArgs("lrange")
	}

	key := args[0]
	start, ok := parseInt64(args[1])
	if !ok {
		return Translation{}, ErrNotInteger
	}
	stop, ok := parseInt64(args[2])
	if !ok {
		return Translation{}, ErrNotInteger
	}

	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    `SELECT value FROM lists WHERE key = ? ORDER BY "index" ASC`,
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "list") {
				return nil, ErrWrongType
			}
			if len(results[2]) == 0 {
				return []any{}, nil
			}
			values := make([]string, 0, len(results[2]))
			for _, row := range results[2] {
				values = append(values, rowString(row, "value"))
			}

			length := len(values)
			startIndex := int(start)
			stopIndex := int(stop)
			if startIndex < 0 {
				startIndex = length + startIndex
			}
			if stopIndex < 0 {
				stopIndex = length + stopIndex
			}
			if startIndex < 0 {
				startIndex = 0
			}
			if stopIndex > length-1 {
				stopIndex = length - 1
			}
			if startIndex > stopIndex || startIndex >= length || stopIndex < 0 {
				return []any{}, nil
			}

			out := make([]any, 0, stopIndex-startIndex+1)
			for _, value := range values[startIndex : stopIndex+1] {
				out = append(out, value)
			}
			return out, nil
		},
	}, nil
}

func translateLLen(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("llen")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT COUNT(*) as count FROM lists WHERE key = ?",
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "list") {
				return nil, ErrWrongType
			}
			if len(results[2]) == 0 {
				return int64(0), nil
			}
			count, ok := parseInt64(rowString(results[2][0], "count"))
			if !ok {
				return nil, ErrNotInteger
			}
			return count, nil
		},
	}, nil
}
