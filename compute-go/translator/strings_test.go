package translator

import (
	"strings"
	"testing"
)

func TestTranslateGetMissingKey(t *testing.T) {
	translation, err := Translate([]string{"GET", "missing"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(translation.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(translation.Statements))
	}
	results := make([][]map[string]any, 3)
	value, err := translation.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}
}

func TestTranslateSetVariants(t *testing.T) {
	basic, err := Translate([]string{"SET", "k", "v"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(basic.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(basic.Statements))
	}
	results := make([][]map[string]any, 4)
	value, err := basic.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != "OK" {
		t.Fatalf("expected OK, got %v", value)
	}

	nx, err := Translate([]string{"SET", "k", "v", "NX"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(nx.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(nx.Statements))
	}
	results = make([][]map[string]any, 3)
	value, err = nx.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}

	xx, err := Translate([]string{"SET", "k", "v", "XX"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(xx.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(xx.Statements))
	}
	results = make([][]map[string]any, 3)
	value, err = xx.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}

	exp, err := Translate([]string{"SET", "k", "v", "EX", "10"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	var hasExpiry bool
	for _, stmt := range exp.Statements {
		if strings.Contains(stmt.SQL, "expires_at") {
			for _, param := range stmt.Params {
				if param != nil {
					hasExpiry = true
					break
				}
			}
		}
	}
	if !hasExpiry {
		t.Fatalf("expected expires_at param to be set")
	}
}

func TestTranslateIncrVariants(t *testing.T) {
	translation, err := Translate([]string{"INCR", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(translation.Statements) != 6 {
		t.Fatalf("expected 6 statements, got %d", len(translation.Statements))
	}
	results := make([][]map[string]any, 6)
	results[4] = []map[string]any{{"value": "0"}}
	results[5] = []map[string]any{{"value": "1"}}
	value, err := translation.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(1) {
		t.Fatalf("expected 1, got %v", value)
	}

	incrBy, err := Translate([]string{"INCRBY", "k", "5"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	results = make([][]map[string]any, 6)
	results[4] = []map[string]any{{"value": "0"}}
	results[5] = []map[string]any{{"value": "5"}}
	value, err = incrBy.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(5) {
		t.Fatalf("expected 5, got %v", value)
	}

	badResults := make([][]map[string]any, 6)
	_, err = translation.MapResult(badResults)
	if err != ErrNotInteger {
		t.Fatalf("expected ErrNotInteger, got %v", err)
	}
}

func TestTranslateStringErrors(t *testing.T) {
	if _, err := Translate([]string{"GET"}); err == nil {
		t.Fatalf("expected error for wrong args")
	}
	if _, err := Translate([]string{"SET", "k", "v", "NX", "XX"}); err == nil {
		t.Fatalf("expected error for NX+XX")
	}
	if _, err := Translate([]string{"NOPE"}); err == nil {
		t.Fatalf("expected error for unknown command")
	}
}
