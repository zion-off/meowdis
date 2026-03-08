package translator

func translateHGet(args []string) (Translation, error) {
	if len(args) != 2 {
		return Translation{}, errWrongArgs("hget")
	}

	key := args[0]
	field := args[1]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT value FROM hashes WHERE key = ? AND field = ?",
			Params: []any{key, field},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "hash") {
				return nil, ErrWrongType
			}
			if len(results[2]) == 0 {
				return nil, nil
			}
			return rowString(results[2][0], "value"), nil
		},
	}, nil
}

func translateHSet(args []string) (Translation, error) {
	if len(args) < 3 || (len(args)-1)%2 != 0 {
		return Translation{}, errWrongArgs("hset")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'hash')",
			Params: []any{key},
		},
	}

	insertIndexes := make([]int, 0, (len(args)-1)/2)
	for i := 1; i < len(args); i += 2 {
		field := args[i]
		value := args[i+1]
		insertIndexes = append(insertIndexes, len(stmts))
		stmts = append(stmts,
			Statement{
				SQL:    "INSERT OR IGNORE INTO hashes (key, field, value) VALUES (?, ?, ?) RETURNING field",
				Params: []any{key, field, value},
			},
			Statement{
				SQL:    "UPDATE hashes SET value = ? WHERE key = ? AND field = ? AND NOT EXISTS (SELECT 1 WHERE changes() > 0)",
				Params: []any{value, key, field},
			},
		)
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "hash") {
				return nil, ErrWrongType
			}
			var count int64
			for _, index := range insertIndexes {
				count += int64(len(results[index]))
			}
			return count, nil
		},
	}, nil
}

func translateHDel(args []string) (Translation, error) {
	if len(args) < 2 {
		return Translation{}, errWrongArgs("hdel")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
	}

	deleteIndexes := make([]int, 0, len(args)-1)
	for i := 1; i < len(args); i++ {
		field := args[i]
		deleteIndexes = append(deleteIndexes, len(stmts))
		stmts = append(stmts, Statement{
			SQL:    "DELETE FROM hashes WHERE key = ? AND field = ? RETURNING field",
			Params: []any{key, field},
		})
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "hash") {
				return nil, ErrWrongType
			}
			var count int64
			for _, index := range deleteIndexes {
				count += int64(len(results[index]))
			}
			return count, nil
		},
	}, nil
}

func translateHGetAll(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("hgetall")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT field, value FROM hashes WHERE key = ? ORDER BY field",
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "hash") {
				return nil, ErrWrongType
			}
			out := make([]any, 0, len(results[2])*2)
			for _, row := range results[2] {
				out = append(out, rowString(row, "field"), rowString(row, "value"))
			}
			return out, nil
		},
	}, nil
}

func translateHExists(args []string) (Translation, error) {
	if len(args) != 2 {
		return Translation{}, errWrongArgs("hexists")
	}

	key := args[0]
	field := args[1]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT 1 FROM hashes WHERE key = ? AND field = ?",
			Params: []any{key, field},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "hash") {
				return nil, ErrWrongType
			}
			if len(results[2]) > 0 {
				return int64(1), nil
			}
			return int64(0), nil
		},
	}, nil
}

func translateHKeys(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("hkeys")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT field FROM hashes WHERE key = ? ORDER BY field",
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "hash") {
				return nil, ErrWrongType
			}
			out := make([]any, 0, len(results[2]))
			for _, row := range results[2] {
				out = append(out, rowString(row, "field"))
			}
			return out, nil
		},
	}, nil
}

func translateHVals(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("hvals")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT value FROM hashes WHERE key = ? ORDER BY field",
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "hash") {
				return nil, ErrWrongType
			}
			out := make([]any, 0, len(results[2]))
			for _, row := range results[2] {
				out = append(out, rowString(row, "value"))
			}
			return out, nil
		},
	}, nil
}
