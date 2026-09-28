package telegram

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html"
	"log/slog"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Telegram struct {
	Bot       *tgbotapi.BotAPI
	MessageID int
	RoomID    int64
}

// Telegram rejects the whole message if any button's callback_data exceeds 64 bytes.
const maxCallbackBytes = 64

// callbackData builds "prefix:taskName[suffix]", falling back to "prefix:#<sha1[:12]>[suffix]"
// when the task name would push it past Telegram's limit. tracker_bot's
// keyboards/callback.py resolves the hash back to a task name; keep the two in sync.
func callbackData(prefix, taskName, suffix string) string {
	data := prefix + ":" + taskName + suffix
	if len(data) <= maxCallbackBytes {
		return data
	}
	sum := sha1.Sum([]byte(taskName))
	return prefix + ":#" + hex.EncodeToString(sum[:])[:12] + suffix
}

func TelegramNew(apiKey string, roomID int64) *Telegram {
	bot, err := tgbotapi.NewBotAPI(apiKey)
	if err != nil {
		slog.Error("Can't connect to telegram", "error", err)
		os.Exit(1)
	}

	return &Telegram{
		Bot:    bot,
		RoomID: roomID,
	}
}

func (t *Telegram) SendMessageStart(taskName string) (int, error) {
	if t.Bot == nil {
		return 0, fmt.Errorf("telegram bot is not initialized")
	}

	msgTheme := "\xE2\x9A\xA0 Start Timer \xE2\x9A\xA0"
	msgBody := fmt.Sprintf("\xF0\x9F\x94\x83 Work: %s", html.EscapeString(taskName))
	msgTime := fmt.Sprintf("\xF0\x9F\x95\x9C TimeBegin: %s", time.Now().Format("2 January 2006 15:04"))
	msg := tgbotapi.NewMessage(t.RoomID, fmt.Sprintf("%s\n%s\n%s", msgTheme, msgBody, msgTime))

	msg.ParseMode = "HTML"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏸ Пауза", callbackData("t_pause", taskName, "")),
			tgbotapi.NewInlineKeyboardButtonData("⏹ Стоп", callbackData("t_stop", taskName, "")),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сменить", "t_switch"),
		),
	)
	msg.ReplyMarkup = keyboard

	msg_ack, err := t.Bot.Send(msg)
	if err != nil {
		slog.Error("SendMessageStart: failed to send message", "error", err)
		return 0, err
	}
	t.MessageID = msg_ack.MessageID

	return t.MessageID, nil
}

func (t *Telegram) SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error {
	if t.Bot == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}

	msgTheme := "\xE2\x9C\x85 End Timer \xE2\x9C\x85"
	msgBody := fmt.Sprintf("\xF0\x9F\x94\x83 Work: %s", taskName)
	msgTime := fmt.Sprintf("\xF0\x9F\x95\x9C TimeEnd: %s", timeEnd)
	msgDone := fmt.Sprintf("Done: %d", timeDone)

	msgD := tgbotapi.NewDeleteMessage(t.RoomID, msgID)
	msg := tgbotapi.NewMessage(t.RoomID, fmt.Sprintf("%s\n%s\n%s\n%s", msgTheme, msgBody, msgTime, msgDone))

	_, _ = t.Bot.Request(msgD)
	_, _ = t.Bot.Send(msg)

	return nil
}

func (t *Telegram) SendMessageCompletion(taskName string, timeDone int, todayDone int, targetDuration int, remainingTasks []string, nextTask string, msgID int) error {
	if t.Bot == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}

	if msgID > 0 {
		msgD := tgbotapi.NewDeleteMessage(t.RoomID, msgID)
		if _, err := t.Bot.Request(msgD); err != nil {
			slog.Warn("SendMessageCompletion: failed to delete previous message", "msgID", msgID, "error", err)
		}
	}

	percent := 0
	if targetDuration > 0 {
		percent = (todayDone * 100) / targetDuration
	}

	checkMark := ""
	if percent >= 100 {
		checkMark = " ✅"
	}

	todayLine := fmt.Sprintf("📊 Сегодня: %d / %d мин (%d%% выполнено%s)", todayDone, targetDuration, percent, checkMark)

	remainingLine := "📋 Остаток по плану: Все задачи на сегодня выполнены! 🎉"
	if len(remainingTasks) > 0 {
		escaped := make([]string, len(remainingTasks))
		for i, name := range remainingTasks {
			escaped[i] = html.EscapeString(name)
		}
		remainingLine = fmt.Sprintf("📋 Остаток по плану: %s", strings.Join(escaped, ", "))
	}

	msgText := fmt.Sprintf("🎯 Задача <b>%s</b> (%d мин) завершена!\n%s\n%s",
		html.EscapeString(taskName),
		timeDone,
		todayLine,
		remainingLine,
	)

	var rows [][]tgbotapi.InlineKeyboardButton
	if nextTask != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("▶️ Запустить %s", nextTask), callbackData("t_start", nextTask, "")),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("☕️ Перерыв 10м", "t_rest:10"),
		tgbotapi.NewInlineKeyboardButtonData("⏳ Продлить +15м", callbackData("t_extend", taskName, ":15")),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🌙 Вечерний топ-3", "t_evening_top3"),
	))

	msg := tgbotapi.NewMessage(t.RoomID, msgText)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	if _, err := t.Bot.Send(msg); err != nil {
		slog.Error("SendMessageCompletion: failed to send message", "error", err)
		return err
	}

	return nil
}

func (t *Telegram) SendCustomMessage(message string) error {
	if t.Bot == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}

	if message == "" {
		return fmt.Errorf("message is empty")
	}

	msg := tgbotapi.NewMessage(t.RoomID, message)
	msg.ParseMode = "HTML"

	if _, err := t.Bot.Send(msg); err != nil {
		return err
	}

	return nil
}
