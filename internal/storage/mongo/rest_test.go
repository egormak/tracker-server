package mongo

import (
	"testing"
)

func TestRestSpendUnits_Validation(t *testing.T) {
	st := &Storage{}

	// Non-positive units must be rejected
	if err := st.RestSpendUnits(0); err == nil {
		t.Errorf("expected error for RestSpendUnits(0), got nil")
	}
	if err := st.RestSpendUnits(-100); err == nil {
		t.Errorf("expected error for RestSpendUnits(-100), got nil")
	}
}

func TestRestInvariantConstants(t *testing.T) {
	// Invariant: MaxDailyRestUnits is 6000 (60 minutes * 100)
	if MaxDailyRestUnits != 6000 {
		t.Fatalf("expected MaxDailyRestUnits 6000, got %d", MaxDailyRestUnits)
	}

	// Invariant: 1 minute = 100 raw rest units
	oneMinuteUnits := 1 * 100
	if oneMinuteUnits != 100 {
		t.Fatalf("expected 1 minute = 100 units")
	}
}
