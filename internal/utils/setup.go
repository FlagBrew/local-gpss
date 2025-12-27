package utils

import (
	"context"
	"encoding/json"
	"os"

	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/apex/log"
)

func Setup(mode string) *models.Config {
	cfg := loadConfig()

	if cfg != nil {
		return cfg
	}

	// TODO if cli then run through interactive setup using bubble
	return nil
}

func SetConfig(ctx context.Context, cfg *models.Config) {
	logger := log.FromContext(ctx)
	f, err := os.OpenFile("config.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.WithError(err).Error("Error opening config.json")
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(cfg)
	if err != nil {
		logger.WithError(err).Error("Error encoding config.json")
	}
}

func loadConfig() *models.Config {
	data, err := os.ReadFile("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return nil
	}

	var config models.Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil
	}

	return &config
}
