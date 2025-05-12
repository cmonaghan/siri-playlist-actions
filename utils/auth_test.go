package utils

import (
	"testing"
)

func TestGenerateAPIKey_Length(t *testing.T) {
	key := GenerateAPIKey()
	if len(key) != 32 {
		t.Errorf("expected API key length 32, got %d", len(key))
	}
}
