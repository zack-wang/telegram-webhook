package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"time"
)

const (
	LOGO     = `Flatbed - load and Go`
	TENANT   = `tenant0`
	CLIENT   = `c000001`
	BROKER   = `tcp://localhost:1883`
	BOTTOKEN = `this is your api token`
)

type MqMsg struct {
	Topic   string
	Payload []byte
}

var (
	bot      *tgbotapi.BotAPI
	SUBTOPIC = `+/+/e2c`
	PUBTOPIC = `SuperUser/c2e`
	MqChan   = make(chan MqMsg, 10)
	ChatId   = int64(0)
)

func main() {
	// tg bot api
	bot, err := tgbotapi.NewBotAPI(BOTTOKEN)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Setup your BOT's WebHook to https://myhost.mydomain/tg
	updates := bot.ListenForWebhook("/tg")
	mqc := RunMqtt(BROKER, `SuperUser`, `SuperPassword`)
	go http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/myhost.mydomain/fullchain.pem", "/etc/letsencrypt/live/myhost.mydomain/privkey.pem", nil)

	go func() {
		for update := range updates {
			log.Printf("%+v %s\n", update.Message, update.Message.Text)
			if len(update.Message.Text) > 2 && update.Message.Text[:2] == `X ` {
				MqttPublish(mqc, `tenant0/c000001/c2e`, false, update.Message.Text[2:])
			} else {
				ChatId = update.Message.Chat.ID
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
			}
		}
	}()

	go func() {
		for mqmsg := range MqChan {
			log.Println(mqmsg.Topic, string(mqmsg.Payload))
			msg := tgbotapi.NewMessage(ChatId, "topic:"+mqmsg.Topic+"\n"+string(mqmsg.Payload))
			bot.Send(msg)
		}
	}()

	t1 := time.NewTicker(600 * time.Second)
	for {
		select {
		case <-t1.C:
			log.Println(`.`)
		}
	}

}
