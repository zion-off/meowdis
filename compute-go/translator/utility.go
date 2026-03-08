package translator

import "path"

func translatePing(args []string) (Translation, error) {
	if len(args) > 1 {
		return Translation{}, errWrongArgs("ping")
	}

	message := "PONG"
	if len(args) == 1 {
		message = args[0]
	}

	return Translation{
		Statements: nil,
		MapResult: func(results [][]map[string]any) (any, error) {
			return message, nil
		},
	}, nil
}

func translateDBSize(args []string) (Translation, error) {
	if len(args) != 0 {
		return Translation{}, errWrongArgs("dbsize")
	}

	stmts := []Statement{
		{
			SQL: "SELECT COUNT(*) as count FROM keys WHERE (expires_at IS NULL OR expires_at > unixepoch())",
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			if len(results[0]) == 0 {
				return int64(0), nil
			}
			count, ok := parseInt64(rowString(results[0][0], "count"))
			if !ok {
				return nil, ErrNotInteger
			}
			return count, nil
		},
	}, nil
}

func translateFlushDB(args []string) (Translation, error) {
	if len(args) != 0 {
		return Translation{}, errWrongArgs("flushdb")
	}

	stmts := []Statement{
		{
			SQL: "DELETE FROM keys",
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			return "OK", nil
		},
	}, nil
}

func translateKeys(args []string) (Translation, error) {
	if len(args) != 1 {
		return Translation{}, errWrongArgs("keys")
	}
	pattern := args[0]
	if pattern == "" {
		return Translation{}, errWrongArgs("keys")
	}

	stmts := []Statement{
		{
			SQL: "SELECT key FROM keys WHERE (expires_at IS NULL OR expires_at > unixepoch()) ORDER BY key",
		},
	}

	return Translation{
		Statements: stmts,
		MapResult: func(results [][]map[string]any) (any, error) {
			out := make([]any, 0, len(results[0]))
			for _, row := range results[0] {
				key := rowString(row, "key")
				match, err := path.Match(pattern, key)
				if err != nil {
					return nil, err
				}
				if match {
					out = append(out, key)
				}
			}
			return out, nil
		},
	}, nil
}
