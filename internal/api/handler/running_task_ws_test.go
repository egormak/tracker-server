package handler

import (
	"net"
	"sync"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/realtime"
	"tracker-server/internal/services"

	fws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp/fasthttputil"
)

// wsTestStorage is an in-memory services.RunningTaskStorage safe for concurrent use.
type wsTestStorage struct {
	mu    sync.Mutex
	tasks map[string]entity.RunningTask
}

func newWSTestStorage() *wsTestStorage {
	return &wsTestStorage{tasks: make(map[string]entity.RunningTask)}
}

func (s *wsTestStorage) GetRunningTask(taskName string) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tasks[taskName], nil
}

func (s *wsTestStorage) GetActiveRunningTask() (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tasks {
		if t.IsRunning {
			return t, nil
		}
	}
	return entity.RunningTask{}, nil
}

func (s *wsTestStorage) GetAllRunningTasks() ([]entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []entity.RunningTask
	for _, t := range s.tasks {
		list = append(list, t)
	}
	return list, nil
}

func (s *wsTestStorage) UpsertRunningTask(task entity.RunningTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.TaskName] = task
	return nil
}

func (s *wsTestStorage) DeleteRunningTask(taskName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, taskName)
	return nil
}

func (s *wsTestStorage) GetRole(taskName string) (string, error)     { return "", nil }
func (s *wsTestStorage) AddTaskRecord(task entity.TaskRecord) error  { return nil }
func (s *wsTestStorage) AddRoleMinutes(task entity.TaskRecord) error { return nil }
func (s *wsTestStorage) TimeListDelDB(timeDuretion int) error        { return nil }
func (s *wsTestStorage) GetTodayTaskDuration(string) (int, error)    { return 0, nil }
func (s *wsTestStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return entity.WeeklySchedule{}, nil
}
func (s *wsTestStorage) GetTaskParams(taskName string) (entity.TaskParams, error) {
	return entity.TaskParams{}, nil
}
func (s *wsTestStorage) IsTaskStrict(taskName string) (bool, error) { return false, nil }
func (s *wsTestStorage) GetTaskDurationForDate(taskName string, date string, sourceDay string) (int, error) {
	return 0, nil
}

// TestWebSocketHandler_ConcurrentBroadcastAndHeartbeat drives hub broadcasts and
// heartbeat replies to one connection at the same time. Without a single writer per
// connection this panics with a concurrent write, or fails under -race.
func TestWebSocketHandler_ConcurrentBroadcastAndHeartbeat(t *testing.T) {
	const n = 100

	st := newWSTestStorage()
	_ = st.UpsertRunningTask(entity.RunningTask{TaskName: "focus", Role: "work", IsRunning: true, StartTime: time.Now()})

	hub := realtime.NewHub()
	go hub.Run()

	service := services.NewRunningTaskService(st, nil)
	service.SetHub(hub)
	h := NewRunningTaskHandler(service, hub)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/ws", h.WebSocketUpgradeCheck, h.WebSocketHandler())

	ln := fasthttputil.NewInmemoryListener()
	go func() { _ = app.Listener(ln) }()

	dialer := fws.Dialer{
		NetDial:          func(network, addr string) (net.Conn, error) { return ln.Dial() },
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial("ws://tracker.test/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		_ = app.ShutdownWithTimeout(2 * time.Second)
	})

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	var first realtime.Event
	if err := conn.ReadJSON(&first); err != nil {
		t.Fatalf("read first message: %v", err)
	}
	if first.Type != realtime.EventStateSync {
		t.Fatalf("first message type = %q, want %q", first.Type, realtime.EventStateSync)
	}

	// Each heartbeat yields two HEARTBEAT_ACK (the direct reply and the hub broadcast),
	// each broadcast one TASK_ADJUSTED.
	const want = 3 * n

	type readResult struct {
		counts map[realtime.EventType]int
		err    error
	}
	var countsMu sync.Mutex
	counts := make(map[realtime.EventType]int)
	readDone := make(chan readResult, 1)
	go func() {
		for i := 0; i < want; i++ {
			var ev realtime.Event
			if err := conn.ReadJSON(&ev); err != nil {
				countsMu.Lock()
				snapshot := copyCounts(counts)
				countsMu.Unlock()
				readDone <- readResult{counts: snapshot, err: err}
				return
			}
			countsMu.Lock()
			counts[ev.Type]++
			countsMu.Unlock()
		}
		countsMu.Lock()
		snapshot := copyCounts(counts)
		countsMu.Unlock()
		readDone <- readResult{counts: snapshot}
	}()

	start := make(chan struct{})
	writeErr := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine A: the only writer on the client side of the connection.
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < n; i++ {
			if err := conn.WriteMessage(fws.TextMessage, []byte(`{"type":"heartbeat","task_name":"focus"}`)); err != nil {
				writeErr <- err
				return
			}
		}
	}()

	// Goroutine B: hub broadcasts to the same connection.
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < n; i++ {
			hub.Broadcast(realtime.Event{Type: realtime.EventTaskAdjusted, TaskName: "focus"})
		}
	}()

	close(start)
	wg.Wait()

	select {
	case err := <-writeErr:
		t.Fatalf("write heartbeat: %v", err)
	default:
	}

	var res readResult
	select {
	case res = <-readDone:
	case <-time.After(15 * time.Second):
		countsMu.Lock()
		snapshot := copyCounts(counts)
		countsMu.Unlock()
		t.Fatalf("timed out waiting for %d messages, got %v", want, snapshot)
	}
	if res.err != nil {
		t.Fatalf("read after %v: %v", res.counts, res.err)
	}

	if got := res.counts[realtime.EventHeartbeatAck]; got != 2*n {
		t.Errorf("HEARTBEAT_ACK = %d, want %d (all counts %v)", got, 2*n, res.counts)
	}
	if got := res.counts[realtime.EventTaskAdjusted]; got != n {
		t.Errorf("TASK_ADJUSTED = %d, want %d (all counts %v)", got, n, res.counts)
	}
	if len(res.counts) != 2 {
		t.Errorf("unexpected event types: %v", res.counts)
	}
}

func copyCounts(m map[realtime.EventType]int) map[realtime.EventType]int {
	out := make(map[realtime.EventType]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
