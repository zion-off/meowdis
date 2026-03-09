package translator

import "testing"

func TestTranslateSets(t *testing.T) {
	sadd, err := Translate([]string{"SADD", "k", "a", "b"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(sadd.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(sadd.Statements))
	}
	results := make([][]map[string]any, 5)
	results[3] = []map[string]any{{"member": "a"}}
	results[4] = []map[string]any{{"member": "b"}}
	value, err := sadd.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(2) {
		t.Fatalf("expected 2, got %v", value)
	}

	smembers, err := Translate([]string{"SMEMBERS", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	results = make([][]map[string]any, 3)
	results[2] = []map[string]any{{"member": "a"}, {"member": "b"}}
	value, err = smembers.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	list := value.([]any)
	if len(list) != 2 || list[0] != "a" || list[1] != "b" {
		t.Fatalf("unexpected SMEMBERS result: %v", list)
	}

	srem, err := Translate([]string{"SREM", "k", "a"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(srem.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(srem.Statements))
	}
	results = make([][]map[string]any, 4)
	results[2] = []map[string]any{{"member": "a"}}
	value, err = srem.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(1) {
		t.Fatalf("expected 1, got %v", value)
	}
}
