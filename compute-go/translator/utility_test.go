package translator

import "testing"

func TestTranslatePing(t *testing.T) {
	translation, err := Translate([]string{"PING"})
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if len(translation.Statements) != 0 {
		t.Fatalf("expected 0 statements, got %d", len(translation.Statements))
	}
	value, err := translation.MapResult(nil)
	if err != nil {
		t.Fatalf("MapResult error: %v", err)
	}
	if value != "PONG" {
		t.Fatalf("expected PONG, got %v", value)
	}
}
