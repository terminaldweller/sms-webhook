package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lrstanley/girc"
	"github.com/pelletier/go-toml/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	reconnectTime = 30
)

type handlerWrapper struct {
	irc    *girc.Client
	config TomlConfig
	app    *pocketbase.PocketBase
}

type Alert struct {
	Sender      string `json:"sender"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type TomlConfig struct {
	IrcServer      string
	IrcPort        int
	IrcNick        string
	IrcSaslUser    string
	IrcSaslPass    string
	IrcChannel     string
	IrcChannelPass string
}

// curl -X 'POST' 'http://127.0.0.1:8090/sms' -H 'content-type: application/json; charset=utf-8' -d $'{"from":"1234567890","text":"Test"}'
func (hw handlerWrapper) postHandler(e *core.RequestEvent) error {
	user, pass, ok := e.Request.BasicAuth()
	if !ok {
		return e.JSON(http.StatusUnauthorized, "unauthorized")
	}

	userRecord, err := hw.app.FindRecordById("users", user)
	if err != nil {
		return e.JSON(http.StatusUnauthorized, "unauthorized")
	}

	if !userRecord.ValidatePassword(pass) {
		return e.JSON(http.StatusUnauthorized, "unauthorized")
	}

	sms := new(Alert)
	if err := e.BindBody(sms); err != nil {
		return e.String(http.StatusBadRequest, "bad request")
	}

	smsInfoReal := Alert{
		Sender:      sms.Sender,
		Title:       sms.Title,
		Description: sms.Description,
	}

	for {
		if hw.irc != nil {
			if hw.irc.IsConnected() {
				break
			}
		}
	}

	hw.irc.Cmd.Message(hw.config.IrcChannel, fmt.Sprintf("Sender: %s, Title: %s, Description: %s", sms.Sender, sms.Title, sms.Description))

	log.Println(smsInfoReal)

	return e.JSON(http.StatusOK, "OK")
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
		if appConfig.IrcChannelPass != "" {
			c.Cmd.JoinKey(appConfig.IrcChannel, appConfig.IrcChannelPass)
		} else {
			c.Cmd.Join(appConfig.IrcChannel)
		}
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

// func defaultPublicDir() string {
// 	if strings.HasPrefix(os.Args[0], os.TempDir()) {
// 		return "./pb_public"
// 	}

// 	return filepath.Join(os.Args[0], "../pb_public")
// }

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
	hw := handlerWrapper{irc: nil, config: appConfig, app: app}

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		go runIRC(appConfig, ircChan)
		hw.irc = <-ircChan

		e.Router.POST("/sms", hw.postHandler)

		e.Next()

		return nil
	})

	err = app.RootCmd.ParseFlags(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
