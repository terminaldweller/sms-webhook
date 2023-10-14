package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v5"
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

func main() {
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
		e.Router.POST("/sms/*", postHandler)

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
