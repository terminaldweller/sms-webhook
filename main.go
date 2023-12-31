package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v5"
	"github.com/lrstanley/girc"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var rdb *redis.Client

const (
	redisDialTimeout   = 5
	redisReadTimeout   = 10
	redisWriteTimetout = 10
)

type SMSInfo struct {
	From          string `json:"from"`
	Text          string `json:"text"`
	SentStamp     int64  `json:"sentStamp"`
	ReceivedStamp int64  `json:"receivedStamp"`
	Sim           string `json:"sim"`
}

type IRCInfo struct {
	ircServer   string
	ircPort     int
	ircNick     string
	ircSaslUser string
	ircSaslPass string
	ircChannel  string
}

func postHandler(context echo.Context) error {
	smsInfo := new(SMSInfo)
	if err := context.Bind(smsInfo); err != nil {
		return context.String(http.StatusBadRequest, "bad request")
	}

	smsInfoReal := SMSInfo{
		From:          smsInfo.From,
		Text:          smsInfo.Text,
		SentStamp:     smsInfo.SentStamp,
		ReceivedStamp: smsInfo.ReceivedStamp,
		Sim:           smsInfo.Sim,
	}

	fmt.Println(smsInfoReal)

	return context.JSON(http.StatusOK, smsInfo)
}

func runIRC(ircInfo IRCInfo) *girc.Client {
	irc := girc.New(girc.Config{
		Server:    ircInfo.ircServer,
		Port:      ircInfo.ircPort,
		Nick:      ircInfo.ircNick,
		User:      "soulshack",
		Name:      "soulshack",
		SSL:       true,
		TLSConfig: &tls.Config{InsecureSkipVerify: false},
	})

	saslUser := ircInfo.ircSaslUser
	saslPass := ircInfo.ircSaslPass
	if saslUser != "" && saslPass != "" {
		irc.Config.SASL = &girc.SASLPlain{
			User: ircInfo.ircSaslUser,
			Pass: ircInfo.ircSaslPass,
		}
	}

	irc.Handlers.AddBg(girc.PRIVMSG, func(c *girc.Client, e girc.Event) {
	})

	if err := irc.Connect(); err != nil {
		log.Fatal(err)
		return nil
	}
	return irc
}

func main() {
	ircServer := flag.String("ircserver", "irc.terminaldweller.com", "the address of the irc server to connect to")
	ircPort := flag.Int("ircport", 6697, "the port of the irc server to connect to")
	ircNick := flag.String("ircnick", "soulhack", "the nick to use on the irc server")
	ircSaslUser := flag.String("ircsasluser", "soulhack", "the sasl user to use on the irc server")
	ircSaslPass := flag.String("ircsaslpass", "", "the sasl password to use on the irc server")
	ircChannel := flag.String("ircchannel", "#soulhack", "the channel to join on the irc server")

	ircInfo := IRCInfo{
		ircServer:   *ircServer,
		ircPort:     *ircPort,
		ircNick:     *ircNick,
		ircSaslUser: *ircSaslUser,
		ircSaslPass: *ircSaslPass,
		ircChannel:  *ircChannel,
	}

	ircClient := runIRC(ircInfo)

	redisAddress := flag.String("redisaddress", "redis:6379", "determines the address of the redis instance")
	redisPassword := flag.String("redispassword", "", "determines the password of the redis db")
	redisDB := flag.Int64("redisdb", 0, "determines the db number")
	flag.Parse()

	rdb = redis.NewClient(&redis.Options{
		Addr:         *redisAddress,
		Password:     *redisPassword,
		DB:           int(*redisDB),
		DialTimeout:  redisDialTimeout,
		ReadTimeout:  redisReadTimeout,
		WriteTimeout: redisWriteTimetout,
	})
	defer rdb.Close()

	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.POST("/sms", postHandler)

		ircClient.Handlers.AddBg(girc.PRIVMSG, func(c *girc.Client, e girc.Event) {})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
