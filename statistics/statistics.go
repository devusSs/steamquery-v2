package statistics

import (
	"time"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/statistics/database"
	"github.com/devusSs/steamquery-v2/statistics/database/postgres"
)

var (
	service  database.Service
	ovTicker *time.Ticker
)

func SetupStatistics(cfg *config.Postgres, logsDir string) error {
	svc, err := postgres.NewPostgresConnection(cfg, logsDir)
	if err != nil {
		return err
	}

	if err := svc.TestConnection(); err != nil {
		return err
	}

	if err := svc.VerifyVersion(); err != nil {
		return err
	}

	if err := svc.Migrate(); err != nil {
		return err
	}

	if err := svc.DeleteOldValues(); err != nil {
		return err
	}

	ovTicker = time.NewTicker(12 * time.Hour)

	go func() {
		for range ovTicker.C {
			if err := svc.DeleteOldValues(); err != nil {
				logging.LogError(err.Error())
			}
			logging.LogInfo("Cleared old statistics")
		}
	}()

	logging.LogDebug("Setup goroutine for deleting old db values every 12 hours")

	service = svc

	return nil
}

func AddStatistics(model *database.SteamQueryV2Values) error {
	return service.AddValues(model)
}

// TODO: create function to generate images from stats

func CloseStatistics() error {
	ovTicker.Stop()
	return service.Close()
}
