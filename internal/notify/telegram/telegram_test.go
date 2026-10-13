package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"tracker-server/internal/domain/entity"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func newTestTelegram(handler http.HandlerFunc) (*Telegram, *httptest.Server) {
	server := httptest.NewServer(handler)
	bot := &tgbotapi.BotAPI{
		Token:  "test-token",
		Client: server.Client(),
		Buffer: 100,
	}
	bot.SetAPIEndpoint(server.URL + "/bot%s/%s")

	return &Telegram{
		Bot:    bot,
		RoomID: 12345,
	}, server
}

func TestSendMessageStart(t *testing.T) {
	var capturedPath string
	var capturedBody string

	tg, server := newTestTelegram(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		bodyBytes, _ := io.ReadAll(r.Body)
		capturedBody, _ = url.QueryUnescape(string(bodyBytes))

		resp := tgbotapi.APIResponse{
			Ok:     true,
			Result: json.RawMessage(`{"message_id": 777, "date": 1234567, "chat": {"id": 12345}}`),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	msgID, err := tg.SendMessageStart("coding")
	if err != nil {
		t.Fatalf("SendMessageStart failed: %v", err)
	}

	if msgID != 777 {
		t.Errorf("expected msgID 777, got %d", msgID)
	}

	if capturedPath != "/bottest-token/sendMessage" {
		t.Errorf("expected path /bottest-token/sendMessage, got %s", capturedPath)
	}

	if !strings.Contains(capturedBody, "Work: coding") {
		t.Errorf("expected body to contain 'Work: coding', got %s", capturedBody)
	}

	// Verify inline keyboard buttons
	if !strings.Contains(capturedBody, "t_pause:coding") {
		t.Errorf("expected callback_data 't_pause:coding', got %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "t_stop:coding") {
		t.Errorf("expected callback_data 't_stop:coding', got %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "t_switch") {
		t.Errorf("expected callback_data 't_switch', got %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "⏸ Пауза") {
		t.Errorf("expected button '⏸ Пауза', got %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "⏹ Стоп") {
		t.Errorf("expected button '⏹ Стоп', got %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "🔄 Сменить") {
		t.Errorf("expected button '🔄 Сменить', got %s", capturedBody)
	}
}

func TestSendMessageCompletion_WithRemainingTasks(t *testing.T) {
	var deleteCalled bool
	var capturedSendBody string

	tg, server := newTestTelegram(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		bodyStr, _ := url.QueryUnescape(string(bodyBytes))

		if strings.HasSuffix(r.URL.Path, "/deleteMessage") {
			deleteCalled = true
			if !strings.Contains(bodyStr, "456") {
				t.Errorf("expected deleteMessage with message_id 456, got %s", bodyStr)
			}
			resp := tgbotapi.APIResponse{Ok: true, Result: json.RawMessage(`true`)}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			capturedSendBody = bodyStr
			resp := tgbotapi.APIResponse{
				Ok:     true,
				Result: json.RawMessage(`{"message_id": 888, "date": 1234567, "chat": {"id": 12345}}`),
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	})
	defer server.Close()

	err := tg.SendMessageCompletion("coding", 25, 25, 50, []string{"english", "math"}, "english", 456, nil)
	if err != nil {
		t.Fatalf("SendMessageCompletion failed: %v", err)
	}

	if !deleteCalled {
		t.Errorf("expected deleteMessage to be called for msgID 456")
	}

	// Verify formatted text
	expectedLine1 := "🎯 Задача <b>coding</b> (25 мин) завершена!"
	expectedLine2 := "📊 Сегодня: 25 / 50 мин (50% выполнено)"
	expectedLine3 := "📋 Остаток по плану: english, math"

	if !strings.Contains(capturedSendBody, expectedLine1) {
		t.Errorf("expected message to contain %q, got: %s", expectedLine1, capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, expectedLine2) {
		t.Errorf("expected message to contain %q, got: %s", expectedLine2, capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, expectedLine3) {
		t.Errorf("expected message to contain %q, got: %s", expectedLine3, capturedSendBody)
	}

	// Verify buttons
	if !strings.Contains(capturedSendBody, "▶️ Запустить english") || !strings.Contains(capturedSendBody, "t_start:english") {
		t.Errorf("expected start next task button, got: %s", capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, "☕️ Перерыв 10м") || !strings.Contains(capturedSendBody, "t_rest:10") {
		t.Errorf("expected rest button, got: %s", capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, "⏳ Продлить +15м") || !strings.Contains(capturedSendBody, "t_extend:coding:15") {
		t.Errorf("expected extend button, got: %s", capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, "🌙 Вечерний топ-3") || !strings.Contains(capturedSendBody, "t_evening_top3") {
		t.Errorf("expected evening top3 button, got: %s", capturedSendBody)
	}
}

func TestSendMessageCompletion_AllCompleted(t *testing.T) {
	var deleteCalled bool
	var capturedSendBody string

	tg, server := newTestTelegram(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		bodyStr, _ := url.QueryUnescape(string(bodyBytes))

		if strings.HasSuffix(r.URL.Path, "/deleteMessage") {
			deleteCalled = true
			resp := tgbotapi.APIResponse{Ok: true, Result: json.RawMessage(`true`)}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			capturedSendBody = bodyStr
			resp := tgbotapi.APIResponse{
				Ok:     true,
				Result: json.RawMessage(`{"message_id": 889, "date": 1234567, "chat": {"id": 12345}}`),
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	})
	defer server.Close()

	// msgID is 0 (should not call deleteMessage), percent 100%, no remaining tasks, nextTask is empty
	err := tg.SendMessageCompletion("reading", 30, 30, 30, nil, "", 0, nil)
	if err != nil {
		t.Fatalf("SendMessageCompletion failed: %v", err)
	}

	if deleteCalled {
		t.Errorf("deleteMessage should not be called when msgID == 0")
	}

	expectedLine1 := "🎯 Задача <b>reading</b> (30 мин) завершена!"
	expectedLine2 := "📊 Сегодня: 30 / 30 мин (100% выполнено ✅)"
	expectedLine3 := "📋 Остаток по плану: Все задачи на сегодня выполнены! 🎉"

	if !strings.Contains(capturedSendBody, expectedLine1) {
		t.Errorf("expected message to contain %q, got: %s", expectedLine1, capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, expectedLine2) {
		t.Errorf("expected message to contain %q, got: %s", expectedLine2, capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, expectedLine3) {
		t.Errorf("expected message to contain %q, got: %s", expectedLine3, capturedSendBody)
	}

	// When nextTask == "", no start button
	if strings.Contains(capturedSendBody, "t_start:") {
		t.Errorf("should not contain t_start button when nextTask is empty, got: %s", capturedSendBody)
	}

	// Should still contain rest, extend, evening buttons
	if !strings.Contains(capturedSendBody, "☕️ Перерыв 10м") {
		t.Errorf("expected rest button, got: %s", capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, "t_extend:reading:15") {
		t.Errorf("expected extend button, got: %s", capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, "🌙 Вечерний топ-3") {
		t.Errorf("expected evening top3 button, got: %s", capturedSendBody)
	}
}

func TestSendMessageCompletion_ZeroTargetDuration(t *testing.T) {
	var capturedSendBody string

	tg, server := newTestTelegram(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capturedSendBody, _ = url.QueryUnescape(string(bodyBytes))
		resp := tgbotapi.APIResponse{
			Ok:     true,
			Result: json.RawMessage(`{"message_id": 890, "date": 1234567, "chat": {"id": 12345}}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	// targetDuration == 0 should not panic (division by zero)
	err := tg.SendMessageCompletion("misc", 15, 15, 0, nil, "", 0, nil)
	if err != nil {
		t.Fatalf("SendMessageCompletion failed: %v", err)
	}

	expectedLine := "📊 Сегодня: 15 / 0 мин (0% выполнено)"
	if !strings.Contains(capturedSendBody, expectedLine) {
		t.Errorf("expected message to contain %q, got: %s", expectedLine, capturedSendBody)
	}
}

func TestTelegram_NilBot(t *testing.T) {
	tg := &Telegram{}

	if _, err := tg.SendMessageStart("test"); err == nil {
		t.Errorf("expected error for SendMessageStart with nil Bot")
	}
	if err := tg.SendMessageStop("test", 10, 1, "now"); err == nil {
		t.Errorf("expected error for SendMessageStop with nil Bot")
	}
	if err := tg.SendMessageCompletion("test", 10, 10, 10, nil, "", 1, nil); err == nil {
		t.Errorf("expected error for SendMessageCompletion with nil Bot")
	}
	if err := tg.SendCustomMessage("test"); err == nil {
		t.Errorf("expected error for SendCustomMessage with nil Bot")
	}
}

const longTaskName = "Изучение распределённых систем и консенсуса Raft" // > 64 bytes in UTF-8

func TestCallbackDataHashesLongNames(t *testing.T) {
	if got := callbackData("t_stop", "coding", ""); got != "t_stop:coding" {
		t.Errorf("expected plain callback for short name, got %q", got)
	}
	// Hash must match tracker_bot tests/test_remote_timer.py test_task_hash_matches_server_encoding.
	if got := callbackData("t_extend", longTaskName, ":15"); got != "t_extend:#68efd5438f7d:15" {
		t.Errorf("unexpected hashed callback %q", got)
	}
	for _, prefix := range []string{"t_pause", "t_stop", "t_start", "t_extend"} {
		if got := callbackData(prefix, longTaskName, ":15"); len(got) > maxCallbackBytes {
			t.Errorf("%s callback is %d bytes, exceeds %d", prefix, len(got), maxCallbackBytes)
		}
	}
}

func TestSendMessageCompletion_EscapesHTML(t *testing.T) {
	var capturedSendBody string
	tg, server := newTestTelegram(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			capturedSendBody, _ = url.QueryUnescape(string(bodyBytes))
		}
		resp := tgbotapi.APIResponse{Ok: true, Result: json.RawMessage(`{"message_id": 1, "date": 1, "chat": {"id": 12345}}`)}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	if err := tg.SendMessageCompletion("R&D", 10, 10, 20, []string{"a<b"}, "a<b", 0, nil); err != nil {
		t.Fatalf("SendMessageCompletion failed: %v", err)
	}
	if !strings.Contains(capturedSendBody, "<b>R&amp;D</b>") {
		t.Errorf("expected escaped task name, got %s", capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, "Остаток по плану: a&lt;b") {
		t.Errorf("expected escaped remaining task, got %s", capturedSendBody)
	}
}

func captureCompletionBody(t *testing.T, send func(tg *Telegram) error) string {
	t.Helper()
	var captured string
	tg, server := newTestTelegram(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			captured, _ = url.QueryUnescape(string(bodyBytes))
		}
		resp := tgbotapi.APIResponse{Ok: true, Result: json.RawMessage(`{"message_id": 1, "date": 1, "chat": {"id": 12345}}`)}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()
	if err := send(tg); err != nil {
		t.Fatalf("SendMessageCompletion failed: %v", err)
	}
	return captured
}

func TestSendMessageCompletion_BreakSelector(t *testing.T) {
	options := []entity.BreakOption{
		{TaskName: "movies", Role: "rest", WeeklyDone: 0, IsVariety: true},
		{TaskName: "games", Role: "rest", WeeklyDone: 10},
		{TaskName: "video", Role: "rest", WeeklyDone: 40},
	}
	body := captureCompletionBody(t, func(tg *Telegram) error {
		return tg.SendMessageCompletion("work", 25, 25, 50, nil, "", 0, options)
	})

	for _, want := range []string{
		"🎲 movies · за неделю 0м", "t_brk:movies:15",
		"t_brk:games:15", "t_brk:video:15",
		"✓ 15м", "t_brkdur:15", "t_brkdur:20", "t_brkdur:30",
		"t_rest:10", "t_extend:work:15", "t_evening_top3",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected keyboard to contain %q, got: %s", want, body)
		}
	}
	if strings.Count(body, "t_brk:movies:15") != 1 {
		t.Errorf("variety task must not be repeated in the mood row, got: %s", body)
	}
}

func TestSendMessageCompletion_NoBreakSelectorWithoutOptions(t *testing.T) {
	body := captureCompletionBody(t, func(tg *Telegram) error {
		return tg.SendMessageCompletion("video", 15, 15, 15, nil, "", 0, nil)
	})
	if strings.Contains(body, "t_brk") {
		t.Errorf("expected no break selector without options, got: %s", body)
	}
	if !strings.Contains(body, "t_rest:10") {
		t.Errorf("expected the regular rows to stay, got: %s", body)
	}
}

func TestSendMessageCompletion_BreakSelectorLimits(t *testing.T) {
	longName := strings.Repeat("очень_длинная_задача_", 4)
	options := []entity.BreakOption{{TaskName: longName, IsVariety: true}}
	for i := 0; i < 10; i++ {
		options = append(options, entity.BreakOption{TaskName: fmt.Sprintf("task%d", i)})
	}

	rows := breakSelectorRows(options)
	// variety + 6 mood buttons in rows of 3 + duration row
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	moodButtons := len(rows[1]) + len(rows[2])
	if moodButtons != breakMaxMoodOptions {
		t.Errorf("expected %d mood buttons, got %d", breakMaxMoodOptions, moodButtons)
	}
	for _, row := range rows {
		for _, btn := range row {
			if btn.CallbackData == nil || len(*btn.CallbackData) > maxCallbackBytes {
				t.Errorf("callback_data over %d bytes: %v", maxCallbackBytes, btn.CallbackData)
			}
		}
	}
	if data := *rows[0][0].CallbackData; !strings.HasPrefix(data, "t_brk:#") || !strings.HasSuffix(data, ":15") {
		t.Errorf("expected hashed callback for a long task name, got %q", data)
	}
}

func TestSendMessageCompletion_WeeklyDigestButtonOnSundayOnly(t *testing.T) {
	defer func() { timeNow = time.Now }()

	send := func(tg *Telegram) error {
		return tg.SendMessageCompletion("work", 25, 25, 50, nil, "", 0, nil)
	}

	timeNow = func() time.Time { return time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC) } // Sunday
	if body := captureCompletionBody(t, send); !strings.Contains(body, "week_digest") || !strings.Contains(body, "📊 Итоги недели") {
		t.Errorf("expected the weekly digest button on Sunday, got: %s", body)
	}

	timeNow = func() time.Time { return time.Date(2026, 10, 12, 20, 0, 0, 0, time.UTC) } // Monday
	if body := captureCompletionBody(t, send); strings.Contains(body, "week_digest") {
		t.Errorf("expected no weekly digest button on Monday, got: %s", body)
	}
}
