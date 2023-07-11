package sqlite

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/statistics/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var logFile *os.File

type sql struct {
	db *gorm.DB
}

func NewSqliteConnection(cfg *config.Postgres, logsDir string) (database.Service, error) {
	logFile, err := createLogFile(logsDir)
	if err != nil {
		return nil, err
	}

	sLogger := logger.New(
		log.New(logFile, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: false,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)

	gDB, err := gorm.Open(sqlite.Open("./.stats.db"), &gorm.Config{
		Logger: sLogger,
	})
	if err != nil {
		return nil, err
	}

	return &sql{gDB}, nil
}

func (p *sql) TestConnection() error {
	db, err := p.db.DB()
	if err != nil {
		return err
	}
	return db.Ping()
}

func (p *sql) VerifyVersion() error {
	db, err := p.db.DB()
	if err != nil {
		return err
	}

	var version string
	err = db.QueryRow("select sqlite_version()").Scan(&version)
	if err != nil {
		return err
	}

	versionRaw, err := strconv.ParseFloat(strings.Join(strings.Split(version, ".")[0:2], "."), 64)
	if err != nil {
		return err
	}

	if versionRaw < 3.4 {
		return fmt.Errorf(
			"unsupported sqlite version, want at least: %.2f, got: %.2f",
			3.4,
			versionRaw,
		)
	}

	logging.LogDebug(fmt.Sprintf("SQLite version: %s", version))

	return nil
}

func (s *sql) Migrate() error {
	return s.db.AutoMigrate(&database.SteamQueryV2Values{})
}

func (s *sql) DeleteOldValues() error {
	oldValuesTreshhold := time.Now().AddDate(0, 0, -30)
	logging.LogDebug(fmt.Sprintf("Deleting database values older than %v", oldValuesTreshhold))
	tx := s.db.Where("created < ?", oldValuesTreshhold).Delete(&database.SteamQueryV2Values{})
	logging.LogDebug(fmt.Sprintf("OLD VALUES AFFECTED: %d", tx.RowsAffected))
	return tx.Error
}

func (p *sql) Close() error {
	if err := closeLogFile(); err != nil {
		return err
	}
	db, err := p.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (p *sql) AddValues(values *database.SteamQueryV2Values) error {
	tx := p.db.Create(values)
	return tx.Error
}

func (p *sql) GetValues() ([]*database.SteamQueryV2Values, error) {
	var returns []*database.SteamQueryV2Values
	tx := p.db.Find(&returns)
	return returns, tx.Error
}

func (p *sql) GetValuesByDate(
	startTime time.Time,
	endTime time.Time,
) ([]*database.SteamQueryV2Values, error) {
	logging.LogDebug(
		fmt.Sprintf("start: %v ; end: %v", startTime.In(time.UTC), endTime.In(time.UTC)),
	)
	var returns []*database.SteamQueryV2Values
	tx := p.db.
		Where("created >= ? AND created <= ?", startTime.In(time.UTC), endTime.In(time.UTC)).
		Find(&returns)
	return returns, tx.Error
}

func (p *sql) GetValuesByItemName(
	name string,
) ([]*database.SteamQueryV2Values, error) {
	var returns []*database.SteamQueryV2Values
	tx := p.db.Where("item_name = ?", name).Find(&returns)
	return returns, tx.Error
}

func (p *sql) GetValuesByItemNameAndDate(
	name string,
	startTime time.Time,
	endTime time.Time,
) ([]*database.SteamQueryV2Values, error) {
	logging.LogDebug(
		fmt.Sprintf("start: %v ; end: %v", startTime.In(time.UTC), endTime.In(time.UTC)),
	)
	var returns []*database.SteamQueryV2Values
	tx := p.db.
		Where("item_name = ?", name).
		Where("created >= ? AND created <= ?", startTime.In(time.UTC), endTime.In(time.UTC)).
		Find(&returns)
	return returns, tx.Error
}

func createLogFile(dir string) (*os.File, error) {
	f, err := os.Create(fmt.Sprintf("%s/sqlite.log", dir))
	if err != nil {
		return nil, err
	}
	logFile = f
	return f, err
}

func closeLogFile() error {
	return logFile.Close()
}
