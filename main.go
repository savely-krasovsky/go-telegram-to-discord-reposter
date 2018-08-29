package main // import "reposter"

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"reposter/config"
	"reposter/database"
	"reposter/dcapi"
	"reposter/handler"
	"reposter/proxy"
	"reposter/tgapi"
)

var (
	path = flag.String(
		"config",
		"",
		"enter path to config file",
	)
)

func main() {
	// Parse at first startup
	flag.Parse()

	// Read config
	conf, err := config.NewConfig(*path)
	if err != nil {
		fmt.Println("Incorrect path or config itself! See help.")
		os.Exit(2)
	}

	// Init discord api
	dcbot, err := dcapi.NewSession(conf)
	if err != nil {
		fmt.Println("Discord bot cannot be initialized! See, error:")
		panic(err)
	}

	// Init http proxy transport
	tr := proxy.NewProxyTransport(conf)

	// http client with proxy
	var client *http.Client
	if conf.Proxy != nil {
		client = &http.Client{
			Transport: tr,
		}
	}

	// Init telegram api
	tgbot, err := tgapi.NewBot(conf, tr)
	if err != nil {
		fmt.Println("Telegram bot cannot be initialized! See, error:")
		panic(err)
	}

	fmt.Printf("Authorized on account @%s\n", tgbot.Self.UserName)

	// Init database
	db, err := database.NewDatabase(conf)
	if err != nil {
		fmt.Println("Database cannot be initialized! See, error:")
		panic(err)
	}

	// Try auto migration for first start
	err = db.AutoMigrate()
	if err != nil {
		fmt.Println("Cannot auto migrate! See, error:")
		panic(err)
	}

	uc := tgbotapi.NewUpdate(0)
	uc.Timeout = 60

	updates, err := tgbot.GetUpdatesChan(uc)

	// Graceful shutdown
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill)

	go func() {
		<-s
		updates.Clear()
		dcbot.Close()
		os.Exit(1)
	}()

	// Main loop, check all changes in Telegram Channel
	for u := range updates {
		handler.HandleUpdate(conf, db, client, tgbot, dcbot, u)
	}
}
