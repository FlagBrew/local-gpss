package utils

import (
	"context"
	"encoding/json"
	"os"

	"github.com/FlagBrew/local-gpss/internal/gui"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/apex/log"
)

func Setup(ctx context.Context, mode string) *models.Config {
	logger := log.FromContext(ctx)
	cfg := loadConfig()

	if cfg != nil {
		return cfg
	}

	if mode == "docker" {
		logger.Fatal("You're running in docker mode and did not volume mount the config.json, interactive set-up is not available for docker. Check the wiki for more information.")
	}

	// Create a new blank config
	cfg = &models.Config{}

	app := gui.New(cfg)
	err := app.Start()
	if err != nil {
		logger.WithError(err).Fatal("Failed to start interactive wizard")
	}

	// Save the config once done.
	SetConfig(ctx, cfg)

	return cfg
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
