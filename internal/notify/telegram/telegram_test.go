package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

	err := tg.SendMessageCompletion("coding", 25, 25, 50, []string{"english", "math"}, "english", 456)
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
	err := tg.SendMessageCompletion("reading", 30, 30, 30, nil, "", 0)
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
	err := tg.SendMessageCompletion("misc", 15, 15, 0, nil, "", 0)
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
	if err := tg.SendMessageCompletion("test", 10, 10, 10, nil, "", 1); err == nil {
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

	if err := tg.SendMessageCompletion("R&D", 10, 10, 20, []string{"a<b"}, "a<b", 0); err != nil {
		t.Fatalf("SendMessageCompletion failed: %v", err)
	}
	if !strings.Contains(capturedSendBody, "<b>R&amp;D</b>") {
		t.Errorf("expected escaped task name, got %s", capturedSendBody)
	}
	if !strings.Contains(capturedSendBody, "Остаток по плану: a&lt;b") {
		t.Errorf("expected escaped remaining task, got %s", capturedSendBody)
	}
}
