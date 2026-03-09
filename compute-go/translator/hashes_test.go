package translator

import "testing"

func TestTranslateHashes(t *testing.T) {
	hset, err := Translate([]string{"HSET", "k", "f", "v"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(hset.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(hset.Statements))
	}
	results := make([][]map[string]any, 5)
	results[3] = []map[string]any{{"field": "f"}}
	value, err := hset.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(1) {
		t.Fatalf("expected 1, got %v", value)
	}

	hget, err := Translate([]string{"HGET", "k", "f"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	results = make([][]map[string]any, 3)
	value, err = hget.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}

	hgetAll, err := Translate([]string{"HGETALL", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	results = make([][]map[string]any, 3)
	results[2] = []map[string]any{{"field": "a", "value": "1"}, {"field": "b", "value": "2"}}
	value, err = hgetAll.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	list := value.([]any)
	if len(list) != 4 || list[0] != "a" || list[1] != "1" || list[2] != "b" || list[3] != "2" {
		t.Fatalf("unexpected HGETALL result: %v", list)
	}
}
