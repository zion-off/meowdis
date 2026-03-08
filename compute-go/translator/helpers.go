package translator

import (
	"fmt"
	"strings"
)

func wrongTypeFor(results [][]map[string]any, index int, expected string) bool {
	if index < 0 || index >= len(results) {
		return false
	}
	if len(results[index]) == 0 {
		return false
	}
	value, ok := results[index][0]["type"]
	if !ok || value == nil {
		return false
	}
	return strings.ToLower(fmt.Sprint(value)) != expected
}
