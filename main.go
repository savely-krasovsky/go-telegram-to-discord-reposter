package main

import (
	"flag"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"os/signal"
	"reposter/config"
	"reposter/database"
	"reposter/dcapi"
	"reposter/tgapi"
	"github.com/bwmarrin/discordgo"
	"net/http"
	"reposter/proxy"
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
		if u.ChannelPost != nil {
			var m *discordgo.Message

			var fileID   *string
			var fileName string

			// Send repost to Discord text channel
			if u.ChannelPost.Text != "" {
				m, err = dcbot.ChannelMessageSend(conf.Discord.ChannelID, u.ChannelPost.Text)
				if err != nil {
					log.Printf("Cannot repost your post! See error: %s", err.Error())
					continue
				}
			} else if u.ChannelPost.Photo != nil {
				if len(*u.ChannelPost.Photo) > 0 {
					p := *u.ChannelPost.Photo
					url, err := tgbot.GetFileDirectURL(p[len(*u.ChannelPost.Photo)-1].FileID)
					if err != nil {
						log.Printf("Cannot get direct file URL! See error: %s", err.Error())
						continue
					}

					resp, err := client.Get(url)
					if err != nil {
						log.Printf("Cannot do GET request! See error: %s", err.Error())
						continue
					}
					defer resp.Body.Close()

					if u.ChannelPost.Caption != "" {
						m, err = dcbot.ChannelFileSendWithMessage(
							conf.Discord.ChannelID,
							u.ChannelPost.Caption,
							"photo.jpg",
							resp.Body,
						)
					} else {
						m, err = dcbot.ChannelFileSend(
							conf.Discord.ChannelID,
							"photo.jpg",
							resp.Body,
						)
					}
					if err != nil {
						log.Printf("Cannot send file! See error: %s", err.Error())
					}
				}
			} else if u.ChannelPost.Document != nil {
				fileID = &u.ChannelPost.Document.FileID
				fileName = u.ChannelPost.Document.FileName
			} else if u.ChannelPost.Video != nil {
				fileID = &u.ChannelPost.Video.FileID
				fileName = "video.mp4"
			} else if u.ChannelPost.VideoNote != nil {
				fileID = &u.ChannelPost.VideoNote.FileID
				fileName = "videonote.mp4"
			} else if u.ChannelPost.Audio != nil {
				fileID = &u.ChannelPost.Audio.FileID
				fileName = u.ChannelPost.Audio.Performer + " - " + u.ChannelPost.Audio.Title + ".mp3"
			} else if u.ChannelPost.Voice != nil {
				fileID = &u.ChannelPost.Voice.FileID
				fileName = "voice.ogg"
			}

			if fileID != nil {
				url, err := tgbot.GetFileDirectURL(*fileID)
				if err != nil {
					log.Printf("Cannot get direct file URL! See error: %s", err.Error())
					continue
				}

				resp, err := client.Get(url)
				if err != nil {
					log.Printf("Cannot do GET request! See error: %s", err.Error())
					continue
				}
				defer resp.Body.Close()

				if u.ChannelPost.Caption != "" {
					m, err = dcbot.ChannelFileSendWithMessage(
						conf.Discord.ChannelID,
						u.ChannelPost.Caption,
						fileName,
						resp.Body,
					)
				} else {
					m, err = dcbot.ChannelFileSend(
						conf.Discord.ChannelID,
						fileName,
						resp.Body,
					)
				}
				if err != nil {
					log.Printf("Cannot send file! See error: %s", err.Error())
				}
			}

			if m != nil {
				// Save new record with ids from Telegram and Discord
				pm := database.PostManager{
					DB: db.Conn,
					Data: &database.Post{
						Telegram: u.ChannelPost.MessageID,
						Discord:  m.ID,
					},
				}
				if err := pm.Create(); err != nil {
					log.Printf("Cannot create new record in database! See error: %s", err.Error())
				}
			}
		} else if u.EditedChannelPost != nil {
			// Find Discord post id by Telegram post id
			pm := database.PostManager{
				DB: db.Conn,
				Data: &database.Post{
					Telegram: u.EditedChannelPost.MessageID,
				},
			}
			err := pm.FindByTelegramPost()
			if err != nil {
				log.Printf("Cannot read record in database! See error: %s", err.Error())
				continue
			}

			// Edit it with id that we got
			if u.EditedChannelPost.Text != "" {
				_, err = dcbot.ChannelMessageEdit(conf.Discord.ChannelID, pm.Data.Discord, u.EditedChannelPost.Text)
				if err != nil {
					log.Printf("Cannot edit repost! See error: %s", err.Error())
				}
			} else if u.EditedChannelPost.Caption != "" {
				_, err = dcbot.ChannelMessageEdit(conf.Discord.ChannelID, pm.Data.Discord, u.EditedChannelPost.Caption)
				if err != nil {
					log.Printf("Cannot edit repost! See error: %s", err.Error())
				}
			}
		} else if u.Message != nil {
			msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Я просто бот. Какая тебе разница, чем я занят?")
			tgbot.Send(msg)
		}
	}
}
