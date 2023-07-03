package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/devusSs/steamquery-v2/utils"
)

type ItemList struct {
	ColumnLetter string `json:"column_letter"`
	StartNumber  int    `json:"start_number"`
	EndNumber    int    `json:"end_number"`
}

type OrgCells struct {
	LastUpdatedCell string `json:"last_updated_cell"`
	ErrorCell       string `json:"error_cell"`
	TotalValueCell  string `json:"total_value_cell"`
	DifferenceCell  string `json:"difference_cell"`
}

type Postgres struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type WatchDog struct {
	RetryInterval      int      `json:"retry_interval"`
	SteamRetryInterval int      `json:"steam_retry_interval"`
	MaxPriceDrop       float64  `json:"max_price_drop"`
	SMTPHost           string   `json:"smtp_host"`
	SMTPPort           int      `json:"smtp_port"`
	SMTPUser           string   `json:"smtp_user"`
	SMTPPassword       string   `json:"smtp_password"`
	SMTPFrom           string   `json:"smtp_from"`
	SMTPTo             string   `json:"smtp_to"`
	Postgres           Postgres `json:"postgres"`
}

type Config struct {
	ItemList         ItemList `json:"item_list"`
	PriceColumn      string   `json:"price_column"`
	PriceTotalColumn string   `json:"price_total_column"`
	AmountColumn     string   `json:"amount_column"`
	OrgCells         OrgCells `json:"org_cells"`
	SpreadSheetID    string   `json:"spread_sheet_id"`
	SteamAPIKey      string   `json:"steam_api_key"`
	SteamUserID64    uint64   `json:"steam_user_id_64"`
	WatchDog         WatchDog `json:"watch_dog"`
}

func LoadConfig(configPath string) (*Config, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfgBody, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var cfg Config

	if err := json.Unmarshal(cfgBody, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) CheckConfig(watchDog bool) error {
	if c.ItemList.ColumnLetter == "" {
		return errors.New("missing item list column letter in config")
	}

	if c.ItemList.StartNumber == 0 {
		return errors.New("missing item list start number in config")
	}

	if c.ItemList.EndNumber == 0 {
		return errors.New("missing item list end number in config")
	}

	if c.PriceColumn == "" {
		return errors.New("missing price column in config")
	}

	if c.PriceTotalColumn == "" {
		return errors.New("missing price total column in config")
	}

	if c.AmountColumn == "" {
		return errors.New("missing amount column in config")
	}

	if c.OrgCells.DifferenceCell == "" {
		return errors.New("missing difference cell in config")
	}

	if c.OrgCells.TotalValueCell == "" {
		return errors.New("missing total value cell in config")
	}

	if c.OrgCells.ErrorCell == "" {
		return errors.New("missing error cell in config")
	}

	if c.OrgCells.LastUpdatedCell == "" {
		return errors.New("missing last updated cell in config")
	}

	if c.SpreadSheetID == "" {
		return errors.New("missing spreadsheet id in config")
	}

	if c.SteamAPIKey == "" {
		return errors.New("missing steam api key in config")
	}

	if c.SteamUserID64 == 0 {
		return errors.New("missing steam user id 64 in config")
	}

	if watchDog {
		if c.WatchDog.RetryInterval == 0 {
			return errors.New("missing retry interval in config")
		}

		if c.WatchDog.SteamRetryInterval == 0 {
			return errors.New("missing steam retry interval in config")
		}

		if c.WatchDog.SteamRetryInterval < 5 {
			return errors.New("steam retry interval needs to be at least 5 minutes")
		}

		if c.WatchDog.SMTPHost == "" {
			return errors.New("missing smpt host in config")
		}

		if c.WatchDog.SMTPPort == 0 {
			return errors.New("missing smtp port in config")
		}

		if c.WatchDog.SMTPUser == "" {
			return errors.New("missing smtp user in config")
		}

		if c.WatchDog.SMTPPassword == "" {
			return errors.New("missing smtp password in config")
		}

		if c.WatchDog.SMTPFrom == "" {
			return errors.New("missing smtp from in config")
		}

		if c.WatchDog.SMTPTo == "" {
			return errors.New("missing smtp to in config")
		}

		if c.WatchDog.Postgres.Host == "" {
			return errors.New("missing postgres host in config")
		}

		if c.WatchDog.Postgres.Port == 0 {
			return errors.New("missing postgres port in config")
		}

		if c.WatchDog.Postgres.User == "" {
			return errors.New("missing postgres user in config")
		}

		if c.WatchDog.Postgres.Password == "" {
			return errors.New("missing postgres password in config")
		}

		if c.WatchDog.Postgres.Database == "" {
			return errors.New("missing postgres database in config")
		}

		if err := utils.ValidateMail(c.WatchDog.SMTPFrom); err != nil {
			return err
		}

		if err := utils.ValidateMail(c.WatchDog.SMTPTo); err != nil {
			return err
		}
	}

	return nil
}
