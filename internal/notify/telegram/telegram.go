package telegram

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Telegram struct {
	Bot       *tgbotapi.BotAPI
	MessageID int
	RoomID    int64
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

	msgTheme := "\xE2\x9A\xA0 Start Timer \xE2\x9A\xA0"
	msgBody := fmt.Sprintf("\xF0\x9F\x94\x83 Work: %s", taskName)
	msgTime := fmt.Sprintf("\xF0\x9F\x95\x9C TimeBegin: %s", time.Now().Format("2 January 2006 15:04"))
	msg := tgbotapi.NewMessage(t.RoomID, fmt.Sprintf("%s\n%s\n%s", msgTheme, msgBody, msgTime))

	msg.ParseMode = "HTML"

	msg_ack, _ := t.Bot.Send(msg)
	t.MessageID = msg_ack.MessageID

	return t.MessageID, nil
}

func (t *Telegram) SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error {

	msgTheme := "\xE2\x9C\x85 End Timer \xE2\x9C\x85"
	msgBody := fmt.Sprintf("\xF0\x9F\x94\x83 Work: %s", taskName)
	msgTime := fmt.Sprintf("\xF0\x9F\x95\x9C TimeEnd: %s", timeEnd)
	msgDone := fmt.Sprintf("Done: %d", timeDone)

	msgD := tgbotapi.NewDeleteMessage(t.RoomID, msgID)
	msg := tgbotapi.NewMessage(t.RoomID, fmt.Sprintf("%s\n%s\n%s\n%s", msgTheme, msgBody, msgTime, msgDone))

	t.Bot.Send(msgD)
	t.Bot.Send(msg)

	return nil
}
