package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/lrstanley/girc"
	"github.com/pelletier/go-toml/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	reconnectTime = 30
)

type handlerWrapper struct {
	irc *girc.Client
	// irc    chan *girc.Client
	config TomlConfig
}

type SMS struct {
	From          string `json:"from"`
	Text          string `json:"text"`
	SentStamp     int64  `json:"sentStamp"`
	ReceivedStamp int64  `json:"receivedStamp"`
	Sim           string `json:"sim"`
}

type TomlConfig struct {
	IrcServer   string
	IrcPort     int
	IrcNick     string
	IrcSaslUser string
	IrcSaslPass string
	IrcChannel  string
}

// curl -X 'POST' 'http://127.0.0.1:8090/sms' -H 'content-type: application/json; charset=utf-8' -d $'{"from":"1234567890","text":"Test"}'
func (hw handlerWrapper) postHandler(context echo.Context) error {
	sms := new(SMS)
	if err := context.Bind(sms); err != nil {
		return context.String(http.StatusBadRequest, "bad request")
	}

	smsInfoReal := SMS{
		From:          sms.From,
		Text:          sms.Text,
		SentStamp:     sms.SentStamp,
		ReceivedStamp: sms.ReceivedStamp,
		Sim:           sms.Sim,
	}

	for {
		if hw.irc != nil {
			if hw.irc.IsConnected() {
				break
			}
		}
	}

	hw.irc.Cmd.Message(hw.config.IrcChannel, fmt.Sprintf("From: %s, Text: %s", sms.From, sms.Text))

	log.Println(smsInfoReal)

	return context.JSON(http.StatusOK, "OK")
}

func runIRC(appConfig TomlConfig, ircChan chan *girc.Client) {
	irc := girc.New(girc.Config{
		Server:    appConfig.IrcServer,
		Port:      appConfig.IrcPort,
		Nick:      appConfig.IrcNick,
		User:      appConfig.IrcNick,
		Name:      appConfig.IrcNick,
		SSL:       true,
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})

	saslUser := appConfig.IrcSaslUser
	saslPass := appConfig.IrcSaslPass

	if saslUser != "" && saslPass != "" {
		irc.Config.SASL = &girc.SASLPlain{
			User: appConfig.IrcSaslUser,
			Pass: appConfig.IrcSaslPass,
		}
	}

	irc.Handlers.AddBg(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		c.Cmd.Join(appConfig.IrcChannel)
	})

	// irc.Handlers.AddBg(girc.PRIVMSG, func(c *girc.Client, e girc.Event) {})
	ircChan <- irc

	for {
		if err := irc.Connect(); err != nil {
			log.Println(err)
			log.Println("reconnecting in 30 seconds")
			time.Sleep(reconnectTime * time.Second)
		} else {
			return
		}
	}
}

func main() {
	var appConfig TomlConfig

	data, err := os.ReadFile("/opt/smswebhook/config.toml")
	if err != nil {
		log.Fatal(err)
	}

	err = toml.Unmarshal(data, &appConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(appConfig)

	app := pocketbase.New()

	ircChan := make(chan *girc.Client, 1)
	hw := handlerWrapper{irc: nil, config: appConfig}

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		go runIRC(appConfig, ircChan)
		hw.irc = <-ircChan

		e.Router.POST("/sms", hw.postHandler)

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
