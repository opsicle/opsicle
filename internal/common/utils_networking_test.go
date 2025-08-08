package common

import "testing"

func TestParseCidrs(t *testing.T) {
	cidrs := []string{"192.168.1.0/24", "bad", "10.0.0.1"}
	parsed, warnings, err := ParseCidrs(cidrs)
	if err != nil {
		t.Fatalf("ParseCidrs returned error: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 valid CIDRs, got %d", len(parsed))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}
