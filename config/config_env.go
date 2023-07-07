package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Constants used for env variables conversion.
const (
	itemColumnLetter = "item_column_letter"
	itemStartNumber  = "item_start_number"
	itemEndNumber    = "item_end_number"
	orgLastUpdated   = "org_last_updated"
	orgErrorCell     = "org_error_cell"
	orgTotalCell     = "org_total_cell"
	orgDiffCell      = "org_diff_cell"
	pHost            = "postgres_host"
	pPort            = "postgres_port"
	pUser            = "postgres_user"
	pPass            = "postgres_password"
	pDatabase        = "postgres_db"
	watchRetry       = "retry_interval"
	watchSteam       = "steam_retry_interval"
	watchMaxDrop     = "max_price_drop"
	smtpHost         = "smtp_host"
	smtpPort         = "smtp_port"
	smtpUser         = "smtp_user"
	smtpPassword     = "smtp_password"
	smtpFrom         = "smtp_from"
	smtpTo           = "smtp_to"
	priceColumn      = "price_column"
	priceTotalColumn = "price_total_column"
	amountColumn     = "amount_column"
	spreadID         = "spreadsheet_id"
	steamAPI         = "steam_api_key"
	steamUID         = "steam_user_id_64"
)

func LoadConfigFromEnv(file string) (*Config, error) {
	if file != "" {
		if err := godotenv.Load(file); err != nil {
			return nil, err
		}
	}

	itemStartNumberInt, err := getEnvInt(itemStartNumber)
	if err != nil {
		return nil, checkError(err, itemStartNumber)
	}

	itemEndNumberInt, err := getEnvInt(itemEndNumber)
	if err != nil {
		return nil, checkError(err, itemEndNumber)
	}

	postgresPort, err := getEnvInt(pPort)
	if err != nil {
		return nil, checkError(err, pPort)
	}

	retryInterval, err := getEnvInt(watchRetry)
	if err != nil {
		return nil, checkError(err, watchRetry)
	}

	steamRetryInterval, err := getEnvInt(watchSteam)
	if err != nil {
		return nil, checkError(err, watchSteam)
	}

	maxPriceDrop, err := getEnvFloat(watchMaxDrop)
	if err != nil {
		return nil, checkError(err, watchMaxDrop)
	}

	smtpPortInt, err := getEnvInt(smtpPort)
	if err != nil {
		return nil, checkError(err, smtpPort)
	}

	steamUserID64, err := getEnvUint(steamUID)
	if err != nil {
		return nil, checkError(err, steamUID)
	}

	return &Config{ItemList: ItemList{
			ColumnLetter: getEnvString(itemColumnLetter),
			StartNumber:  itemStartNumberInt,
			EndNumber:    itemEndNumberInt,
		},
			PriceColumn:      getEnvString(priceColumn),
			PriceTotalColumn: getEnvString(priceTotalColumn),
			AmountColumn:     getEnvString(amountColumn),
			OrgCells: OrgCells{
				LastUpdatedCell: getEnvString(orgLastUpdated),
				ErrorCell:       getEnvString(orgErrorCell),
				TotalValueCell:  getEnvString(orgTotalCell),
				DifferenceCell:  getEnvString(orgDiffCell),
			},
			SpreadSheetID: getEnvString(spreadID),
			SteamAPIKey:   getEnvString(steamAPI),
			SteamUserID64: steamUserID64,
			WatchDog: WatchDog{
				RetryInterval:      retryInterval,
				SteamRetryInterval: steamRetryInterval,
				MaxPriceDrop:       maxPriceDrop,
				SMTPHost:           getEnvString(smtpHost),
				SMTPPort:           smtpPortInt,
				SMTPUser:           getEnvString(smtpUser),
				SMTPPassword:       getEnvString(smtpPassword),
				SMTPFrom:           getEnvString(smtpFrom),
				SMTPTo:             getEnvString(smtpTo),
				Postgres: Postgres{
					Host:     getEnvString(pHost),
					Port:     postgresPort,
					User:     getEnvString(pUser),
					Password: getEnvString(pPass),
					Database: getEnvString(pDatabase),
				},
			},
		},
		nil
}

func getEnvString(name string) string {
	return os.Getenv(strings.ToUpper(name))
}

func getEnvInt(name string) (int, error) {
	return strconv.Atoi(os.Getenv(strings.ToUpper(name)))
}

func getEnvUint(name string) (uint64, error) {
	return strconv.ParseUint(os.Getenv(strings.ToUpper(name)), 10, 64)
}

func getEnvFloat(name string) (float64, error) {
	return strconv.ParseFloat(os.Getenv(strings.ToUpper(name)), 64)
}

// Helper function which checks the error for keyboards.
func checkError(err error, name string) error {
	if strings.Contains(err.Error(), `parsing "": invalid syntax`) {
		return fmt.Errorf("conversion error, perhaps missing env key: %s", strings.ToUpper(name))
	}
	return err
}
