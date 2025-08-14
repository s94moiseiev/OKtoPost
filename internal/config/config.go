package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	BotToken    string `json:"BotToken"`
	AdminChatID int64  `json:"AdminChatID"`
	GroupChatID int64  `json:"GroupChatID"`
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
