package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
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

type Config struct {
	ItemList         ItemList `json:"item_list"`
	PriceColumn      string   `json:"price_column"`
	PriceTotalColumn string   `json:"price_total_column"`
	AmountColumn     string   `json:"amount_column"`
	OrgCells         OrgCells `json:"org_cells"`
	SpreadSheetID    string   `json:"spread_sheet_id"`
	SteamAPIKey      string   `json:"steam_api_key"`
	SteamUserID64    uint64   `json:"steam_user_id_64"`
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

func (c *Config) CheckConfig() error {
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

	return nil
}
