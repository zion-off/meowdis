package translator

import (
	"strings"
	"testing"
)

func TestTranslateListCounts(t *testing.T) {
	lpush, err := Translate([]string{"LPUSH", "k", "a", "b"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(lpush.Statements) != 6 {
		t.Fatalf("expected 6 statements, got %d", len(lpush.Statements))
	}
	results := make([][]map[string]any, 6)
	results[5] = []map[string]any{{"count": "2"}}
	value, err := lpush.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(2) {
		t.Fatalf("expected 2, got %v", value)
	}

	rpush, err := Translate([]string{"RPUSH", "k", "a"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(rpush.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(rpush.Statements))
	}
	results = make([][]map[string]any, 5)
	results[4] = []map[string]any{{"count": "1"}}
	value, err = rpush.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != int64(1) {
		t.Fatalf("expected 1, got %v", value)
	}
}

func TestTranslatePopUsesQuotedIndex(t *testing.T) {
	lpop, err := Translate([]string{"LPOP", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if !hasQuotedIndex(lpop.Statements[2].SQL) {
		t.Fatalf("expected quoted index in LPOP SQL")
	}
	results := make([][]map[string]any, 4)
	results[2] = []map[string]any{{"value": "v"}}
	value, err := lpop.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != "v" {
		t.Fatalf("expected v, got %v", value)
	}

	rpop, err := Translate([]string{"RPOP", "k"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if !hasQuotedIndex(rpop.Statements[2].SQL) {
		t.Fatalf("expected quoted index in RPOP SQL")
	}
}

func TestTranslateLRange(t *testing.T) {
	translation, err := Translate([]string{"LRANGE", "k", "1", "2"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	results := make([][]map[string]any, 3)
	results[2] = []map[string]any{{"value": "a"}, {"value": "b"}, {"value": "c"}, {"value": "d"}}
	value, err := translation.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	list := value.([]any)
	if len(list) != 2 || list[0] != "b" || list[1] != "c" {
		t.Fatalf("unexpected LRANGE result: %v", list)
	}

	negative, err := Translate([]string{"LRANGE", "k", "-2", "-1"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	value, err = negative.MapResult(results)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	list = value.([]any)
	if len(list) != 2 || list[0] != "c" || list[1] != "d" {
		t.Fatalf("unexpected LRANGE result: %v", list)
	}
}

func hasQuotedIndex(sql string) bool {
	if sql == "" {
		return false
	}
	stripped := replaceAll(sql, "\"index\"", "")
	return !strings.Contains(stripped, "index")
}

func replaceAll(value, old, new string) string {
	for {
		idx := strings.Index(value, old)
		if idx < 0 {
			return value
		}
		value = value[:idx] + new + value[idx+len(old):]
	}
}
