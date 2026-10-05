package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

type mockRampStorage struct {
	ramp         entity.RampInfo
	roles        map[string]string
	durationsMap map[string]int
	getErr       error
	saveErr      error
	resetErr     error
	updateErr    error

	resetStepCalled  bool
	updateStepCalled bool
	saveInfoCalled   bool
	lastUpdatedStep  int
}

func newMockRampStorage() *mockRampStorage {
	return &mockRampStorage{
		ramp:         entity.DefaultRampInfo(),
		roles:        make(map[string]string),
		durationsMap: make(map[string]int),
	}
}

func (m *mockRampStorage) GetRampInfo(ctx context.Context) (entity.RampInfo, error) {
	if m.getErr != nil {
		return entity.RampInfo{}, m.getErr
	}
	return m.ramp, nil
}

func (m *mockRampStorage) SaveRampInfo(ctx context.Context, ramp entity.RampInfo) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saveInfoCalled = true
	m.ramp = ramp
	return nil
}

func (m *mockRampStorage) ResetRampStep(ctx context.Context, date string) error {
	if m.resetErr != nil {
		return m.resetErr
	}
	m.resetStepCalled = true
	m.ramp.CurrentStep = 1
	m.ramp.Date = date
	return nil
}

func (m *mockRampStorage) UpdateRampStep(ctx context.Context, step int, date string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updateStepCalled = true
	m.lastUpdatedStep = step
	m.ramp.CurrentStep = step
	m.ramp.Date = date
	return nil
}

func (m *mockRampStorage) GetRole(taskName string) (string, error) {
	if role, ok := m.roles[strings.ToLower(taskName)]; ok {
		return role, nil
	}
	return "", fmt.Errorf("role not found")
}

func (m *mockRampStorage) GetTodayTaskDurationsMap(date string) (map[string]int, error) {
	return m.durationsMap, nil
}

// TestRampMath_Progression tests the arithmetic progression formula k = floor((1 + sqrt(1 + 8S)) / 2).
func TestRampMath_Progression(t *testing.T) {
	cases := []struct {
		focusMinutes int
		capMinutes   int
		expectedStep int
	}{
		{focusMinutes: 0, capMinutes: 25, expectedStep: 1},
		{focusMinutes: 1, capMinutes: 25, expectedStep: 2},
		{focusMinutes: 2, capMinutes: 25, expectedStep: 2},
		{focusMinutes: 3, capMinutes: 25, expectedStep: 3},
		{focusMinutes: 5, capMinutes: 25, expectedStep: 3},
		{focusMinutes: 6, capMinutes: 25, expectedStep: 4},
		{focusMinutes: 9, capMinutes: 25, expectedStep: 4},
		{focusMinutes: 10, capMinutes: 25, expectedStep: 5},
		{focusMinutes: 14, capMinutes: 25, expectedStep: 5},
		{focusMinutes: 15, capMinutes: 25, expectedStep: 6},
		{focusMinutes: 20, capMinutes: 25, expectedStep: 6},
		{focusMinutes: 21, capMinutes: 25, expectedStep: 7},
		{focusMinutes: 28, capMinutes: 25, expectedStep: 8},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("S=%d", tc.focusMinutes), func(t *testing.T) {
			step := CalculateStep(tc.focusMinutes, tc.capMinutes)
			if step != tc.expectedStep {
				t.Errorf("CalculateStep(%d, %d) = %d; want %d",
					tc.focusMinutes, tc.capMinutes, step, tc.expectedStep)
			}
		})
	}
}

// TestRampMath_CapClamping tests clamping at CapMinutes and edge cases.
func TestRampMath_CapClamping(t *testing.T) {
	t.Run("Clamp at default 25", func(t *testing.T) {
		step := CalculateStep(300, 25)
		if step != 25 {
			t.Errorf("expected step 25 for S=300, got %d", step)
		}
		stepLarge := CalculateStep(1000, 25)
		if stepLarge != 25 {
			t.Errorf("expected step 25 for S=1000, got %d", stepLarge)
		}
	})

	t.Run("Clamp at custom cap 10", func(t *testing.T) {
		step := CalculateStep(100, 10)
		if step != 10 {
			t.Errorf("expected step 10 for custom cap 10, got %d", step)
		}
	})

	t.Run("Negative focus minutes returns 1", func(t *testing.T) {
		step := CalculateStep(-10, 25)
		if step != 1 {
			t.Errorf("expected step 1 for negative minutes, got %d", step)
		}
	})
}

// TestRamp_MidnightReset tests that a stored date different from today triggers reset to step 1.
func TestRamp_MidnightReset(t *testing.T) {
	mock := newMockRampStorage()
	mock.ramp.Date = "22 September 2026"
	mock.ramp.CurrentStep = 12

	svc := NewRampService(mock)
	ctx := context.Background()

	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.CurrentStep != 1 {
		t.Errorf("expected CurrentStep=1 after midnight reset, got %d", status.CurrentStep)
	}

	today := time.Now().Format("2 January 2006")
	if status.Date != today {
		t.Errorf("expected Date=%s, got %s", today, status.Date)
	}

	if !mock.resetStepCalled {
		t.Errorf("expected ResetRampStep to be called on storage")
	}
}

// TestRamp_ManualReset tests explicit call to Reset().
func TestRamp_ManualReset(t *testing.T) {
	mock := newMockRampStorage()
	mock.ramp.CurrentStep = 18
	mock.durationsMap["work"] = 45

	svc := NewRampService(mock)
	ctx := context.Background()

	status, err := svc.Reset(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.CurrentStep != 1 {
		t.Errorf("expected CurrentStep=1 after manual reset, got %d", status.CurrentStep)
	}
	if !mock.resetStepCalled {
		t.Errorf("expected ResetRampStep to be called on storage")
	}
}

// TestRamp_Advance tests manual advancement +1 up to cap.
func TestRamp_Advance(t *testing.T) {
	mock := newMockRampStorage()
	mock.ramp.CurrentStep = 3
	mock.ramp.CapMinutes = 5

	svc := NewRampService(mock)
	ctx := context.Background()

	// 3 -> 4
	status, err := svc.Advance(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentStep != 4 {
		t.Errorf("expected step 4, got %d", status.CurrentStep)
	}

	// 4 -> 5 (cap)
	status, err = svc.Advance(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentStep != 5 {
		t.Errorf("expected step 5, got %d", status.CurrentStep)
	}
	if !status.IsCapped {
		t.Errorf("expected IsCapped=true when step == cap")
	}

	// 5 -> 5 (clamped)
	status, err = svc.Advance(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentStep != 5 {
		t.Errorf("expected step to remain 5 at cap, got %d", status.CurrentStep)
	}
}

// TestRamp_RoleFiltering tests GetDurationForTask with eligible vs excluded tasks/roles.
func TestRamp_RoleFiltering(t *testing.T) {
	mock := newMockRampStorage()
	mock.ramp.CurrentStep = 7
	mock.ramp.DefaultRestFallback = 15
	mock.roles["coding"] = "work"
	mock.roles["reading"] = "learn"
	mock.roles["gaming"] = "rest"

	svc := NewRampService(mock)
	ctx := context.Background()

	tests := []struct {
		taskName string
		expected int
	}{
		{taskName: "work", expected: 7},          // Direct enabled role
		{taskName: "learn", expected: 7},         // Direct enabled role
		{taskName: "coding", expected: 7},        // Role is "work"
		{taskName: "reading", expected: 7},       // Role is "learn"
		{taskName: "home_task", expected: 7},     // Direct enabled task
		{taskName: "video", expected: 15},        // Excluded task
		{taskName: "movies", expected: 15},       // Excluded task
		{taskName: "games", expected: 15},        // Excluded task
		{taskName: "telegram", expected: 15},     // Excluded task
		{taskName: "gaming", expected: 15},       // Role is "rest"
		{taskName: "unknown_task", expected: 15}, // Unknown, not eligible
		{taskName: "", expected: 15},             // Empty task
	}

	for _, tc := range tests {
		t.Run(tc.taskName, func(t *testing.T) {
			dur, err := svc.GetDurationForTask(ctx, tc.taskName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dur != tc.expected {
				t.Errorf("GetDurationForTask(%q) = %d; want %d", tc.taskName, dur, tc.expected)
			}
		})
	}
}

// TestRamp_UpdateConfig tests configuration validation and updates.
func TestRamp_UpdateConfig(t *testing.T) {
	mock := newMockRampStorage()
	svc := NewRampService(mock)
	ctx := context.Background()

	t.Run("Validation: CapMinutes < 5 fails", func(t *testing.T) {
		cfg := entity.RampConfig{CapMinutes: 4, DefaultRestFallback: 15}
		_, err := svc.UpdateConfig(ctx, cfg)
		if err == nil {
			t.Errorf("expected error for CapMinutes < 5, got nil")
		}
	})

	t.Run("Validation: DefaultRestFallback < 1 fails", func(t *testing.T) {
		cfg := entity.RampConfig{CapMinutes: 20, DefaultRestFallback: 0}
		_, err := svc.UpdateConfig(ctx, cfg)
		if err == nil {
			t.Errorf("expected error for DefaultRestFallback < 1, got nil")
		}
	})

	t.Run("Valid config updates and clamps current_step if needed", func(t *testing.T) {
		mock.ramp.CurrentStep = 15
		cfg := entity.RampConfig{
			CapMinutes:          10,
			EnabledRoles:        []string{"work", "learn", "side_project"},
			EnabledTasks:        []string{"planning"},
			ExcludedTasks:       []string{"social_media"},
			DefaultRestFallback: 20,
		}

		status, err := svc.UpdateConfig(ctx, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status.CapMinutes != 10 {
			t.Errorf("expected CapMinutes=10, got %d", status.CapMinutes)
		}
		if status.CurrentStep != 10 {
			t.Errorf("expected CurrentStep clamped to 10, got %d", status.CurrentStep)
		}
		if status.Config.DefaultRestFallback != 20 {
			t.Errorf("expected DefaultRestFallback=20, got %d", status.Config.DefaultRestFallback)
		}
		if !mock.saveInfoCalled {
			t.Errorf("expected SaveRampInfo to be called on storage")
		}
	})
}

// TestRamp_AutoRecalculateOnRecord tests automatic progression step recalculation.
func TestRamp_AutoRecalculateOnRecord(t *testing.T) {
	mock := newMockRampStorage()
	mock.ramp.CurrentStep = 1
	mock.roles["coding"] = "work"
	mock.roles["gaming"] = "rest"

	svc := NewRampService(mock)
	ctx := context.Background()

	t.Run("Eligible task advances step", func(t *testing.T) {
		// Today focus: 6 minutes (S=6 -> k=4)
		mock.durationsMap["coding"] = 6
		err := svc.AutoRecalculateOnRecord(ctx, "coding", 6)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !mock.updateStepCalled {
			t.Errorf("expected UpdateRampStep to be called")
		}
		if mock.lastUpdatedStep != 4 {
			t.Errorf("expected step 4 for S=6, got %d", mock.lastUpdatedStep)
		}
	})

	t.Run("Non-eligible task does not trigger recalculation", func(t *testing.T) {
		mock.updateStepCalled = false
		mock.durationsMap["gaming"] = 100

		err := svc.AutoRecalculateOnRecord(ctx, "gaming", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.updateStepCalled {
			t.Errorf("excluded/rest task should not trigger UpdateRampStep")
		}
	})

	t.Run("Does not downgrade step if today S yields lower k", func(t *testing.T) {
		mock.ramp.CurrentStep = 5
		mock.updateStepCalled = false
		mock.durationsMap["coding"] = 3 // S=3 -> k=3 < current_step 5

		err := svc.AutoRecalculateOnRecord(ctx, "coding", 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.updateStepCalled {
			t.Errorf("step should not be updated if k <= current_step")
		}
	})
}

// mockTaskRecordStorageForRamp implements TaskRecordStorage for integration test.
type mockTaskRecordStorageForRamp struct {
	records []entity.TaskRecord
	roles   map[string]string
}

func (m *mockTaskRecordStorageForRamp) GetRole(taskName string) (string, error) {
	if r, ok := m.roles[taskName]; ok {
		return r, nil
	}
	return "work", nil
}
func (m *mockTaskRecordStorageForRamp) AddTaskRecord(task entity.TaskRecord) error {
	m.records = append(m.records, task)
	return nil
}
func (m *mockTaskRecordStorageForRamp) AddRoleMinutes(task entity.TaskRecord) error { return nil }
func (m *mockTaskRecordStorageForRamp) AddRest(restTime int) error                  { return nil }
func (m *mockTaskRecordStorageForRamp) GetGroupPlanPercent() (int, error)           { return 0, nil }
func (m *mockTaskRecordStorageForRamp) GetGroupPercent(g int) (int, error)          { return 0, nil }
func (m *mockTaskRecordStorageForRamp) CheckIfPlanPercentEmpty() error              { return nil }
func (m *mockTaskRecordStorageForRamp) ChangeGroupPlanPercent(g int) error          { return nil }
func (m *mockTaskRecordStorageForRamp) GetGroupName(g int) (string, error)          { return "", nil }
func (m *mockTaskRecordStorageForRamp) GetTaskNamePlanPercent(g string, p int) (string, error) {
	return "", nil
}
func (m *mockTaskRecordStorageForRamp) DelGroupPercent(g string) error             { return nil }
func (m *mockTaskRecordStorageForRamp) GetTodayTaskDuration(t string) (int, error) { return 0, nil }
func (m *mockTaskRecordStorageForRamp) GetTaskParams(t string) (entity.TaskParams, error) {
	return entity.TaskParams{}, nil
}
func (m *mockTaskRecordStorageForRamp) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return entity.WeeklySchedule{}, nil
}
func (m *mockTaskRecordStorageForRamp) GetTaskDurationForDate(t string, d string, s string) (int, error) {
	return 0, nil
}
func (m *mockTaskRecordStorageForRamp) IsTaskStrict(t string) (bool, error) { return false, nil }
func (m *mockTaskRecordStorageForRamp) GetRecords() ([]entity.TaskRecord, error) {
	return m.records, nil
}
func (m *mockTaskRecordStorageForRamp) CleanRecords() {}

type mockRampRecalculator struct {
	called   bool
	taskName string
	duration int
}

func (m *mockRampRecalculator) AutoRecalculateOnRecord(ctx context.Context, taskName string, duration int) error {
	m.called = true
	m.taskName = taskName
	m.duration = duration
	return nil
}

// TestTaskRecordService_WithRampHook tests that TaskRecordService.AddRecord invokes RampRecalculator.
func TestTaskRecordService_WithRampHook(t *testing.T) {
	st := &mockTaskRecordStorageForRamp{roles: map[string]string{"work": "work"}}
	recalc := &mockRampRecalculator{}

	svc := NewTaskRecordService(st, recalc)

	req := entity.TaskRecordRequest{
		TaskName: "work",
		TimeDone: 25,
	}

	err := svc.AddRecord(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.called {
		t.Errorf("expected RampRecalculator.AutoRecalculateOnRecord to be called")
	}
	if recalc.taskName != "work" || recalc.duration != 25 {
		t.Errorf("expected taskName='work', duration=25; got taskName=%q, duration=%d",
			recalc.taskName, recalc.duration)
	}
}

// TestRamp_GetStatus_ReconcilesStep tests that GetStatus recalculates and updates the step
// when today's focus minutes exceed the stored step.
func TestRamp_GetStatus_ReconcilesStep(t *testing.T) {
	mock := newMockRampStorage()
	today := time.Now().Format("2 January 2006")
	mock.ramp.Date = today
	mock.ramp.CurrentStep = 1
	mock.ramp.CapMinutes = 25
	// 15 focus minutes on eligible task should calculate step 6 (CalculateStep(15, 25) == 6)
	mock.durationsMap["work"] = 15

	svc := NewRampService(mock)
	ctx := context.Background()

	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.CurrentStep != 6 {
		t.Errorf("expected reconciled CurrentStep=6, got %d", status.CurrentStep)
	}
	if !mock.updateStepCalled {
		t.Errorf("expected UpdateRampStep to be called to reconcile step in storage")
	}
	if mock.lastUpdatedStep != 6 {
		t.Errorf("expected storage updated with step 6, got %d", mock.lastUpdatedStep)
	}
	if mock.ramp.CurrentStep != 6 {
		t.Errorf("expected storage ramp.CurrentStep=6, got %d", mock.ramp.CurrentStep)
	}

	// Now check case where focus minutes do NOT exceed current step: should not downgrade step or update storage
	mock.updateStepCalled = false
	mock.ramp.CurrentStep = 10
	mock.durationsMap["work"] = 15 // expectedStep is 6, which is < 10

	status, err = svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.CurrentStep != 10 {
		t.Errorf("expected CurrentStep to stay at 10, got %d", status.CurrentStep)
	}
	if mock.updateStepCalled {
		t.Errorf("UpdateRampStep should not be called when expected step <= current step")
	}
}

