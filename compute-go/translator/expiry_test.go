package translator

import "testing"

func TestTranslateExpiry(t *testing.T) {
	expire, err := Translate([]string{"EXPIRE", "k", "10"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(expire.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(expire.Statements))
	}
	results := make([][]map[string]any, 3)
	results[1] = []map[string]any{{"expires_at": 1}}
	results[2] = []map[string]any{{"key": "k"}}
	value, err := expire.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(1) {
		t.Fatalf("expected 1, got %v", value)
	}

	persist, err := Translate([]string{"PERSIST", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	results = make([][]map[string]any, 2)
	value, err = persist.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(0) {
		t.Fatalf("expected 0, got %v", value)
	}

	ttl, err := Translate([]string{"TTL", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	results = make([][]map[string]any, 2)
	results[1] = []map[string]any{{"expires_at": "110", "now": "100"}}
	value, err = ttl.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(10) {
		t.Fatalf("expected 10, got %v", value)
	}

	pttl, err := Translate([]string{"PTTL", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	value, err = pttl.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(10000) {
		t.Fatalf("expected 10000, got %v", value)
	}

	missingResults := make([][]map[string]any, 2)
	value, err = ttl.MapResult(missingResults)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(-2) {
		t.Fatalf("expected -2, got %v", value)
	}

	persistentResults := make([][]map[string]any, 2)
	persistentResults[1] = []map[string]any{{"expires_at": nil, "now": "100"}}
	value, err = ttl.MapResult(persistentResults)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(-1) {
		t.Fatalf("expected -1, got %v", value)
	}
}
