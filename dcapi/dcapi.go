package dcapi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reposter/config"
)

func NewSession(conf *config.Config) (*discordgo.Session, error) {
	s, err := discordgo.New(fmt.Sprintf("Bot %s", conf.Discord.Token))
	if err != nil {
		return nil, err
	}

	// Open websocket connetction
	err = s.Open()
	if err != nil {
		return nil, err
	}

	return s, nil
}
