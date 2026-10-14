package services

import (
	"errors"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

// mockDigestStore keeps counters and the snapshot in memory, like the Mongo collections.
type mockDigestStore struct {
	counters map[entity.UsageCounter]int // key: counter with Count == 0
	snapshot *entity.DigestSnapshot
	incErr   error
}

func newMockDigestStore() *mockDigestStore {
	return &mockDigestStore{counters: make(map[entity.UsageCounter]int)}
}

func (m *mockDigestStore) IncUsageCounter(week, event, client string) error {
	if m.incErr != nil {
		return m.incErr
	}
	m.counters[entity.UsageCounter{Week: week, Event: event, Client: client}]++
	return nil
}

func (m *mockDigestStore) GetUsageCounters(week string) ([]entity.UsageCounter, error) {
	var out []entity.UsageCounter
	for key, count := range m.counters {
		if key.Week == week {
			key.Count = count
			out = append(out, key)
		}
	}
	return out, nil
}

func (m *mockDigestStore) SaveDigestSnapshot(snapshot entity.DigestSnapshot) error {
	m.snapshot = &snapshot
	return nil
}

func (m *mockDigestStore) GetDigestSnapshot() (entity.DigestSnapshot, bool, error) {
	if m.snapshot == nil {
		return entity.DigestSnapshot{}, false, nil
	}
	return *m.snapshot, true, nil
}

func (m *mockDigestStore) count(event, client string) int {
	total := 0
	for key, count := range m.counters {
		if key.Event == event && key.Client == client {
			total += count
		}
	}
	return total
}

func TestUsageService_TrackUsesMondayOfCurrentWeek(t *testing.T) {
	store := newMockDigestStore()
	usage := NewUsageService(store)
	// Sunday 11 October 2026 belongs to the week that starts on Monday 5 October.
	usage.nowFunc = func() time.Time { return time.Date(2026, 10, 11, 23, 0, 0, 0, time.UTC) }

	usage.Track(UsageRampReset, "cli")
	usage.Track(UsageRampReset, "")

	week := "5 October 2026"
	if got := store.counters[entity.UsageCounter{Week: week, Event: UsageRampReset, Client: "cli"}]; got != 1 {
		t.Errorf("cli counter for %s = %d, want 1", week, got)
	}
	if got := store.counters[entity.UsageCounter{Week: week, Event: UsageRampReset, Client: ClientUnknown}]; got != 1 {
		t.Errorf("unknown-client counter for %s = %d, want 1", week, got)
	}
}

func TestUsageService_TrackNeverPanics(t *testing.T) {
	var nilService *UsageService
	nilService.Track(UsageRampReset, "cli") // a service without counters must stay usable

	store := newMockDigestStore()
	store.incErr = errors.New("mongo down")
	NewUsageService(store).Track(UsageRampReset, "cli")
	if len(store.counters) != 0 {
		t.Errorf("expected no counters after a storage error, got %v", store.counters)
	}
}

func TestRunningTaskService_StartWith_RecordsClientAndOrigin(t *testing.T) {
	cases := []struct {
		name      string
		task      string
		origin    string
		wantEvent string
	}{
		{name: "manual", task: "work", origin: "", wantEvent: UsageStartManual},
		{name: "unknown origin is manual", task: "work", origin: "whatever", wantEvent: UsageStartManual},
		{name: "evening", task: "video", origin: OriginEvening, wantEvent: UsageStartEvening},
		{name: "combo", task: "video", origin: OriginCombo, wantEvent: UsageStartCombo},
		// mockBreakOptionsProvider marks "movies" as the variety pick.
		{name: "break variety", task: "Movies", origin: OriginBreak, wantEvent: UsageStartBreakVariety},
		{name: "break mood", task: "games", origin: OriginBreak, wantEvent: UsageStartBreakMood},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storage := NewMockRunningTaskStorage()
			store := newMockDigestStore()
			service := NewRunningTaskService(storage, nil)
			service.SetBreakOptionsProvider(&mockBreakOptionsProvider{})
			service.SetUsageService(NewUsageService(store))

			task, err := service.StartWith(StartOptions{TaskName: tc.task, Role: "rest", TargetDuration: 15, Client: "web", Origin: tc.origin})
			if err != nil {
				t.Fatalf("StartWith: %v", err)
			}
			if task.SourceDevice != "web" {
				t.Errorf("running task source_device = %q, want web", task.SourceDevice)
			}
			if got := store.count(tc.wantEvent, "web"); got != 1 {
				t.Errorf("counter %s/web = %d, want 1 (all: %v)", tc.wantEvent, got, store.counters)
			}
			if len(store.counters) != 1 {
				t.Errorf("expected exactly one counter, got %v", store.counters)
			}

			record, err := service.Stop(tc.task, "manual")
			if err != nil {
				t.Fatalf("Stop: %v", err)
			}
			if record.SourceDevice != "web" {
				t.Errorf("record source_device = %q, want web", record.SourceDevice)
			}
		})
	}
}

func TestRunningTaskService_StartWith_CountsOnlyNewStarts(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	store := newMockDigestStore()
	service := NewRunningTaskService(storage, nil)
	service.SetUsageService(NewUsageService(store))

	if _, err := service.StartWith(StartOptions{TaskName: "work", Role: "work", Client: "cli"}); err != nil {
		t.Fatalf("first start: %v", err)
	}
	// Already running: a repeated start from another client is a no-op.
	if _, err := service.StartWith(StartOptions{TaskName: "work", Role: "work", Client: "web"}); err != nil {
		t.Fatalf("repeated start: %v", err)
	}
	if _, err := service.Pause("work"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	// Resuming a paused task through start keeps the original client.
	resumed, err := service.StartWith(StartOptions{TaskName: "work", Role: "work", Client: "web"})
	if err != nil {
		t.Fatalf("resume through start: %v", err)
	}

	if resumed.SourceDevice != "cli" {
		t.Errorf("source_device after resume = %q, want the starting client cli", resumed.SourceDevice)
	}
	if got := store.count(UsageStartManual, "cli"); got != 1 {
		t.Errorf("start_manual/cli = %d, want 1", got)
	}
	if got := store.count(UsageStartManual, "web"); got != 0 {
		t.Errorf("start_manual/web = %d, want 0", got)
	}
}

func TestWeeklyDigest_ClientsAndUsage(t *testing.T) {
	srv := newDigestService(time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC))
	storage := srv.st.(*mockDigestStorage)
	storage.records[0].SourceDevice = "cli" // work 150
	storage.records[1].SourceDevice = "web" // work 50
	storage.records[2].SourceDevice = "cli" // video 90
	// records[3] (telegram 30) has no client

	store := newMockDigestStore()
	week := "5 October 2026"
	store.counters[entity.UsageCounter{Week: week, Event: UsageRampReset, Client: "web"}] = 2
	store.counters[entity.UsageCounter{Week: week, Event: UsageStartManual, Client: "cli"}] = 5
	store.counters[entity.UsageCounter{Week: week, Event: UsageStartManual, Client: "web"}] = 7
	store.counters[entity.UsageCounter{Week: week, Event: "future_event", Client: "cli"}] = 1
	store.counters[entity.UsageCounter{Week: "12 October 2026", Event: UsageEveningSkip, Client: "cli"}] = 9
	srv.SetDigestStore(store)

	digest, err := srv.GetWeeklyDigest(WeekCurrent)
	if err != nil {
		t.Fatalf("GetWeeklyDigest: %v", err)
	}
	if digest.Source != entity.DigestSourceLive {
		t.Errorf("source = %q, want live", digest.Source)
	}

	wantClients := []entity.WeeklyDigestClient{
		{Client: "cli", Records: 2, Minutes: 240, SharePercent: 75},
		{Client: "web", Records: 1, Minutes: 50, SharePercent: 15},
		{Client: ClientUnknown, Records: 1, Minutes: 30, SharePercent: 9},
	}
	if len(digest.Clients) != len(wantClients) {
		t.Fatalf("clients = %+v, want %+v", digest.Clients, wantClients)
	}
	for i, want := range wantClients {
		if digest.Clients[i] != want {
			t.Errorf("clients[%d] = %+v, want %+v", i, digest.Clients[i], want)
		}
	}

	// Fixed event order, unknown events last, other weeks' counters left out.
	wantEvents := []string{UsageStartManual, UsageRampReset, "future_event"}
	if len(digest.Usage) != len(wantEvents) {
		t.Fatalf("usage = %+v, want events %v", digest.Usage, wantEvents)
	}
	for i, want := range wantEvents {
		if digest.Usage[i].Event != want {
			t.Errorf("usage[%d].event = %q, want %q", i, digest.Usage[i].Event, want)
		}
	}
	manual := digest.Usage[0]
	if manual.Count != 12 || len(manual.Clients) != 2 || manual.Clients[0] != (entity.WeeklyDigestUsageClient{Client: "web", Count: 7}) {
		t.Errorf("start_manual = %+v, want 12 with web(7) first", manual)
	}
}

func TestWeeklyDigest_SnapshotServesPreviousWeekAfterClean(t *testing.T) {
	// Monday 12 October, before the wipe: the week of 5..11 October is "previous".
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	srv := newDigestService(now)
	storage := srv.st.(*mockDigestStorage)
	store := newMockDigestStore()
	store.counters[entity.UsageCounter{Week: "5 October 2026", Event: UsageRampReset, Client: "cli"}] = 3
	srv.SetDigestStore(store)

	before, err := srv.GetWeeklyDigest(WeekPrevious)
	if err != nil {
		t.Fatalf("digest before clean: %v", err)
	}
	if before.Source != entity.DigestSourceLive || before.TotalDone != 320 {
		t.Fatalf("before clean: source=%q total=%d, want live/320", before.Source, before.TotalDone)
	}

	srv.SnapshotBeforeClean()
	if store.snapshot == nil || store.snapshot.WeekStartDate != "5 October 2026" {
		t.Fatalf("snapshot = %+v, want the week of 5 October 2026", store.snapshot)
	}

	// The wipe: records and counters are gone.
	storage.records = nil
	store.counters = make(map[entity.UsageCounter]int)

	after, err := srv.GetWeeklyDigest(WeekPrevious)
	if err != nil {
		t.Fatalf("digest after clean: %v", err)
	}
	if after.Source != entity.DigestSourceSnapshot {
		t.Fatalf("after clean: source = %q, want snapshot", after.Source)
	}
	if after.Week != WeekPrevious || after.TotalDone != 320 || after.SnapshotTakenAt != "2026-10-12T09:00:00Z" {
		t.Errorf("after clean: week=%q total=%d taken_at=%q", after.Week, after.TotalDone, after.SnapshotTakenAt)
	}
	if len(after.Usage) != 1 || after.Usage[0].Count != 3 {
		t.Errorf("after clean: usage = %+v, want the pre-wipe ramp_reset counter", after.Usage)
	}

	// A second wipe with nothing logged must not replace the snapshot with an empty one.
	srv.SnapshotBeforeClean()
	if store.snapshot.WeekStartDate != "5 October 2026" {
		t.Errorf("empty wipe replaced the snapshot: %+v", store.snapshot)
	}

	// The current week never answers from the snapshot.
	current, err := srv.GetWeeklyDigest(WeekCurrent)
	if err != nil {
		t.Fatalf("current digest: %v", err)
	}
	if current.Source != entity.DigestSourceLive || current.TotalDone != 0 {
		t.Errorf("current: source=%q total=%d, want live/0", current.Source, current.TotalDone)
	}

	// A week later the snapshot is of another week and must not be served as "previous".
	srv.nowFunc = func() time.Time { return now.AddDate(0, 0, 7) }
	stale, err := srv.GetWeeklyDigest(WeekPrevious)
	if err != nil {
		t.Fatalf("digest a week later: %v", err)
	}
	if stale.Source != entity.DigestSourceLive || stale.TotalDone != 0 {
		t.Errorf("a week later: source=%q total=%d, want live/0", stale.Source, stale.TotalDone)
	}
}

func TestWeeklyDigest_SnapshotTakesCurrentWeekOnMidWeekClean(t *testing.T) {
	// Sunday 11 October: the week of 5..11 October is still current, the previous one is empty.
	srv := newDigestService(time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC))
	store := newMockDigestStore()
	srv.SetDigestStore(store)

	srv.SnapshotBeforeClean()

	if store.snapshot == nil || store.snapshot.WeekStartDate != "5 October 2026" {
		t.Fatalf("snapshot = %+v, want the current week of 5 October 2026", store.snapshot)
	}
}

func TestTaskRecordService_CleanRunsHookBeforeWipe(t *testing.T) {
	var order []string
	storage := &cleanOrderStorage{order: &order}
	srv := &TaskRecordService{st: storage}
	srv.SetBeforeClean(func() { order = append(order, "snapshot") })

	srv.CleanRecords()

	if len(order) != 2 || order[0] != "snapshot" || order[1] != "clean" {
		t.Errorf("order = %v, want [snapshot clean]", order)
	}
}

// cleanOrderStorage records when the wipe happens; every other method is unused.
type cleanOrderStorage struct {
	TaskRecordStorage
	order *[]string
}

func (s *cleanOrderStorage) CleanRecords() { *s.order = append(*s.order, "clean") }
