package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"reposter/config"
	"reposter/database"
)

func HandleUpdate(conf *config.Config, db *database.Database, client *http.Client, tgbot *tgbotapi.BotAPI, dcbot *discordgo.Session, u tgbotapi.Update) {
	if u.ChannelPost != nil {
		var m *discordgo.Message

		var fileID *string
		var fileName string
		var contentType string

		// Send repost to Discord text channel
		if u.ChannelPost.Text != "" {
			var err error
			m, err = dcbot.ChannelMessageSend(conf.Discord.ChannelID, u.ChannelPost.Text)
			if err != nil {
				log.Printf("Cannot repost your post! See error: %s", err.Error())
				return
			}
		} else if u.ChannelPost.Photo != nil {
			if len(*u.ChannelPost.Photo) > 0 {
				p := *u.ChannelPost.Photo
				url, err := tgbot.GetFileDirectURL(p[len(*u.ChannelPost.Photo)-1].FileID)
				if err != nil {
					log.Printf("Cannot get direct file URL! See error: %s", err.Error())
					return
				}

				resp, err := client.Get(url)
				if err != nil {
					log.Printf("Cannot do GET request! See error: %s", err.Error())
					return
				}
				defer resp.Body.Close()

				if u.ChannelPost.Caption != "" {
					m, err = dcbot.ChannelMessageSendComplex(
						conf.Discord.ChannelID,
						&discordgo.MessageSend{
							Content: u.ChannelPost.Caption,
							Files: []*discordgo.File{
								{
									Name:        "photo.jpg",
									ContentType: "image/jpeg",
									Reader:      resp.Body,
								},
							},
						},
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
			contentType = "application/octet-stream"
		} else if u.ChannelPost.Video != nil {
			fileID = &u.ChannelPost.Video.FileID
			fileName = "video.mp4"
			contentType = "video/mp4"
		} else if u.ChannelPost.VideoNote != nil {
			fileID = &u.ChannelPost.VideoNote.FileID
			fileName = "videonote.mp4"
			contentType = "video/mp4"
		} else if u.ChannelPost.Audio != nil {
			fileID = &u.ChannelPost.Audio.FileID
			fileName = u.ChannelPost.Audio.Performer + " - " + u.ChannelPost.Audio.Title + ".mp3"
			contentType = "audio/mpeg"
		} else if u.ChannelPost.Voice != nil {
			fileID = &u.ChannelPost.Voice.FileID
			fileName = "voice.ogg"
			contentType = "audio/ogg"
		}

		if fileID != nil {
			url, err := tgbot.GetFileDirectURL(*fileID)
			if err != nil {
				log.Printf("Cannot get direct file URL! See error: %s", err.Error())
				return
			}

			resp, err := client.Get(url)
			if err != nil {
				log.Printf("Cannot do GET request! See error: %s", err.Error())
				return
			}
			defer resp.Body.Close()

			if u.ChannelPost.Caption != "" {
				m, err = dcbot.ChannelMessageSendComplex(
					conf.Discord.ChannelID,
					&discordgo.MessageSend{
						Content: u.ChannelPost.Caption,
						Files: []*discordgo.File{
							{
								Name:        fileName,
								ContentType: contentType,
								Reader:      resp.Body,
							},
						},
					},
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
			return
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
