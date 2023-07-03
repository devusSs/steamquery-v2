package statistics

import (
	"sync"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/statistics/database"
	"github.com/devusSs/steamquery-v2/statistics/database/postgres"
)

var service database.Service

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

	service = svc

	return nil
}

// Designed to be run as Goroutine, using waitgroup for that.
func AddStatistics(model *database.SteamQueryV2Values, wg *sync.WaitGroup) error {
	if err := service.AddValues(model); err != nil {
		return err
	}
	wg.Done()
	return nil
}

// TODO: create function to generate images from stats

func CloseStatistics() error {
	return service.Close()
}
