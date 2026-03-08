package translator

func translateGet(args []string) ([]Statement, error) {
	if len(args) != 1 {
		return nil, errWrongArgs("get")
	}

	return []Statement{
		deleteIfExpired(args[0]),
		{
			SQL:    "SELECT value FROM strings WHERE key = ?",
			Params: []any{args[0]},
		},
	}, nil
}

