package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/lrstanley/girc"
	"github.com/pelletier/go-toml/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/ghupdate"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

const (
	reconnectTime = 30
)

type handlerWrapper struct {
	irc    *girc.Client
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

func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
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

	var hooksDir string
	app.RootCmd.PersistentFlags().StringVar(
		&hooksDir,
		"hooksDir",
		"",
		"the directory with the JS app hooks",
	)

	var hooksWatch bool
	app.RootCmd.PersistentFlags().BoolVar(
		&hooksWatch,
		"hooksWatch",
		true,
		"auto restart the app on pb_hooks file change",
	)

	var hooksPool int
	app.RootCmd.PersistentFlags().IntVar(
		&hooksPool,
		"hooksPool",
		25,
		"the total prewarm goja.Runtime instances for the JS app hooks execution",
	)

	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"migrationsDir",
		"",
		"the directory with the user defined migrations",
	)

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(
		&publicDir,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(
		&indexFallback,
		"indexFallback",
		true,
		"fallback the request to index.html on missing static path (eg. when pretty urls are used with SPA)",
	)

	var queryTimeout int
	app.RootCmd.PersistentFlags().IntVar(
		&queryTimeout,
		"queryTimeout",
		30,
		"the default SELECT queries timeout in seconds",
	)

	app.RootCmd.ParseFlags(os.Args[1:])

	jsvm.MustRegister(app, jsvm.Config{
		MigrationsDir: migrationsDir,
		HooksDir:      hooksDir,
		HooksWatch:    hooksWatch,
		HooksPoolSize: hooksPool,
	})

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  automigrate,
		Dir:          migrationsDir,
	})

	ghupdate.MustRegister(app, app.RootCmd, ghupdate.Config{})

	app.OnAfterBootstrap().PreAdd(func(e *core.BootstrapEvent) error {
		app.Dao().ModelQueryTimeout = time.Duration(queryTimeout) * time.Second

		return nil
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS(publicDir), indexFallback))

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
