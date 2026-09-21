package services

import (
	"errors"
	"testing"
)

type mockRestStorage struct {
	restUnits int
	addErr    error
	spendErr  error
	getErr    error
	resetErr  error
}

func (m *mockRestStorage) AddRest(restTime int) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.restUnits += restTime * 30
	if m.restUnits > MaxDailyRestUnits {
		m.restUnits = MaxDailyRestUnits
	}
	return nil
}

func (m *mockRestStorage) AddRestMinutes(minutes int) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.restUnits += minutes * 100
	if m.restUnits > MaxDailyRestUnits {
		m.restUnits = MaxDailyRestUnits
	}
	return nil
}

func (m *mockRestStorage) RestSpend(restTime int) error {
	if m.spendErr != nil {
		return m.spendErr
	}
	m.restUnits -= restTime * 100
	if m.restUnits < 0 {
		m.restUnits = 0
	}
	return nil
}

func (m *mockRestStorage) GetRest() (int, error) {
	if m.getErr != nil {
		return 0, m.getErr
	}
	return m.restUnits, nil
}

func (m *mockRestStorage) ResetRest() error {
	if m.resetErr != nil {
		return m.resetErr
	}
	m.restUnits = 0
	return nil
}

func TestRestService_ResetRest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		st := &mockRestStorage{restUnits: 3000}
		svc := NewRestService(st)

		err := svc.ResetRest()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if st.restUnits != 0 {
			t.Errorf("expected rest units to be 0, got %d", st.restUnits)
		}
	})

	t.Run("storage error", func(t *testing.T) {
		st := &mockRestStorage{
			restUnits: 3000,
			resetErr:  errors.New("db error"),
		}
		svc := NewRestService(st)

		err := svc.ResetRest()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRestService_AddRest(t *testing.T) {
	t.Run("invalid rest time", func(t *testing.T) {
		st := &mockRestStorage{}
		svc := NewRestService(st)

		if err := svc.AddRest(0); err == nil {
			t.Error("expected error for restTime=0, got nil")
		}
		if err := svc.AddRest(-5); err == nil {
			t.Error("expected error for restTime=-5, got nil")
		}
	})

	t.Run("success (minutes * 100 units scaling)", func(t *testing.T) {
		st := &mockRestStorage{restUnits: 100}
		svc := NewRestService(st)

		if err := svc.AddRest(10); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// 100 + 10 * 100 = 1100
		if st.restUnits != 1100 {
			t.Errorf("expected 1100, got %d", st.restUnits)
		}
	})

	t.Run("storage error", func(t *testing.T) {
		st := &mockRestStorage{addErr: errors.New("db failure")}
		svc := NewRestService(st)

		if err := svc.AddRest(10); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRestService_RestSpend(t *testing.T) {
	t.Run("invalid rest time", func(t *testing.T) {
		st := &mockRestStorage{}
		svc := NewRestService(st)

		if err := svc.RestSpend(0); err == nil {
			t.Error("expected error for restTime=0, got nil")
		}
		if err := svc.RestSpend(-1); err == nil {
			t.Error("expected error for restTime=-1, got nil")
		}
	})

	t.Run("success", func(t *testing.T) {
		st := &mockRestStorage{restUnits: 500}
		svc := NewRestService(st)

		if err := svc.RestSpend(2); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// 500 - 2*100 = 300
		if st.restUnits != 300 {
			t.Errorf("expected 300, got %d", st.restUnits)
		}
	})

	t.Run("storage error", func(t *testing.T) {
		st := &mockRestStorage{spendErr: errors.New("db failure")}
		svc := NewRestService(st)

		if err := svc.RestSpend(5); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRestService_GetRest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		st := &mockRestStorage{restUnits: 1500}
		svc := NewRestService(st)

		val, err := svc.RestGet()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if val != 1500 {
			t.Errorf("expected 1500, got %d", val)
		}

		val2, err := svc.GetRest()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if val2 != 1500 {
			t.Errorf("expected 1500, got %d", val2)
		}
	})

	t.Run("storage error", func(t *testing.T) {
		st := &mockRestStorage{getErr: errors.New("db failure")}
		svc := NewRestService(st)

		_, err := svc.RestGet()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRestCapAndClamping(t *testing.T) {
	// Verify the invariant constant
	if MaxDailyRestUnits != 6000 {
		t.Fatalf("expected MaxDailyRestUnits to be 6000, got %d", MaxDailyRestUnits)
	}

	st := &mockRestStorage{restUnits: 5000}
	svc := NewRestService(st)

	// Adding rest that exceeds 6000 should clamp to 6000
	if err := svc.AddRest(100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.restUnits != 6000 {
		t.Errorf("expected rest to clamp at 6000, got %d", st.restUnits)
	}

	// Spending more rest than available should clamp to 0
	if err := svc.RestSpend(100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.restUnits != 0 {
		t.Errorf("expected rest to clamp at 0 on overspend, got %d", st.restUnits)
	}

	// ResetRest should reset to 0
	st.restUnits = 4000
	if err := svc.ResetRest(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.restUnits != 0 {
		t.Errorf("expected reset to set rest to 0, got %d", st.restUnits)
	}
}

func TestRestService_ConcurrentAccess(t *testing.T) {
	st := &mockRestStorage{restUnits: 1000}
	svc := NewRestService(st)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_ = svc.AddRest(1)
			_ = svc.RestSpend(1)
			_, _ = svc.RestGet()
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		_ = svc.AddRest(1)
		_ = svc.RestSpend(1)
		_, _ = svc.RestGet()
	}
	<-done
}
