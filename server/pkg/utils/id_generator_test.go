package utils

import (
	"regexp"
	"testing"
)

func TestGenerateOrderID(t *testing.T) {
	idSet := make(map[string]struct{})
	pattern := `^No-\d{8}-\d{6}-\d{3}$`
	re := regexp.MustCompile(pattern)
	for i := 0; i < 10; i++ {
		id := GenerateOrderID()
		if !re.MatchString(id) {
			t.Errorf("OrderID format error: %s", id)
		}
		if _, exists := idSet[id]; exists {
			t.Errorf("Duplicate OrderID generated: %s", id)
		}
		idSet[id] = struct{}{}
	}
}
