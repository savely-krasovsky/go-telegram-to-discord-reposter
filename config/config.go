package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Database string `yaml:"database"`

	*Telegram `yaml:"telegram"`
	*Discord  `yaml:"discord"`

	*Proxy `yaml:"proxy"`
}

type Telegram struct {
	Token string `yaml:"token"`
}

type Discord struct {
	Token     string `yaml:"token"`
	ChannelID string `yaml:"channel_id"`
}

type Proxy struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func NewConfig(p string) (*Config, error) {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
