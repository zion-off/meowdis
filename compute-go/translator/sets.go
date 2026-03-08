package translator

func translateSAdd(args []string) (Translation, error) {
	if len(args) < 2 {
		return Translation{}, errWrongArgs("sadd")
	}

	key := args[0]
	members := args[1:]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'set')",
			Params: []any{key},
		},
	}

	insertIndexes := make([]int, 0, len(members))
	for _, member := range members {
		insertIndexes = append(insertIndexes, len(stmts))
		stmts = append(stmts, Statement{
			SQL:    "INSERT OR IGNORE INTO sets (key, member) VALUES (?, ?) RETURNING member",
			Params: []any{key, member},
		})
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "set") {
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

func translateSRem(args []string) (Translation, error) {
	if len(args) < 2 {
		return Translation{}, errWrongArgs("srem")
	}

	key := args[0]
	members := args[1:]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
	}

	deleteIndexes := make([]int, 0, len(members))
	for _, member := range members {
		deleteIndexes = append(deleteIndexes, len(stmts))
		stmts = append(stmts, Statement{
			SQL:    "DELETE FROM sets WHERE key = ? AND member = ? RETURNING member",
			Params: []any{key, member},
		})
	}

	stmts = append(stmts, Statement{
		SQL:    "DELETE FROM keys WHERE key = ? AND NOT EXISTS (SELECT 1 FROM sets WHERE key = ?)",
		Params: []any{key, key},
	})

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "set") {
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

func translateSMembers(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("smembers")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT member FROM sets WHERE key = ? ORDER BY member",
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "set") {
				return nil, ErrWrongType
			}
			out := make([]any, 0, len(results[2]))
			for _, row := range results[2] {
				out = append(out, rowString(row, "member"))
			}
			return out, nil
		},
	}, nil
}

func translateSIsMember(args []string) (Translation, error) {
	if len(args) != 2 {
		return Translation{}, errWrongArgs("sismember")
	}

	key := args[0]
	member := args[1]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT 1 FROM sets WHERE key = ? AND member = ?",
			Params: []any{key, member},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "set") {
				return nil, ErrWrongType
			}
			if len(results[2]) > 0 {
				return int64(1), nil
			}
			return int64(0), nil
		},
	}, nil
}

func translateSCard(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("scard")
	}

	key := args[0]
	stmts := []Statement{
		deleteIfExpired(key),
		{
			SQL:    "SELECT type FROM keys WHERE key = ?",
			Params: []any{key},
		},
		{
			SQL:    "SELECT COUNT(*) as count FROM sets WHERE key = ?",
			Params: []any{key},
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if wrongTypeFor(results, 1, "set") {
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
