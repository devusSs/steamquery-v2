package postgres

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/statistics/database"
)

var logFile *os.File

type psql struct {
	db *gorm.DB
}

func NewPostgresConnection(cfg *config.Postgres, logsDir string) (database.Service, error) {
	logFile, err := createPostgresLogFile(logsDir)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Europe/Berlin",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.Port,
	)

	pLogger := logger.New(
		log.New(logFile, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: false,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)

	gDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: pLogger,
	})
	if err != nil {
		return nil, err
	}

	return &psql{gDB}, nil
}

func (p *psql) TestConnection() error {
	db, err := p.db.DB()
	if err != nil {
		return err
	}
	return db.Ping()
}

func (p *psql) VerifyVersion() error {
	db, err := p.db.DB()
	if err != nil {
		return err
	}

	var version string
	err = db.QueryRow("select version()").Scan(&version)
	if err != nil {
		return err
	}

	versionRaw := strings.Split(version, " ")[1]
	versionConv, err := strconv.ParseFloat(versionRaw, 64)
	if err != nil {
		return err
	}

	if versionConv < 13 {
		return fmt.Errorf(
			"unsupported min postgres version; want at least %d ; got %.2f",
			13,
			versionConv,
		)
	}

	return nil
}

func (p *psql) Migrate() error {
	return p.db.AutoMigrate(&database.SteamQueryV2Values{})
}

func (p *psql) DeleteOldValues() error {
	oldValuesTreshhold := time.Now().AddDate(0, 0, -30)
	logging.LogDebug(fmt.Sprintf("Deleting database values older than %v", oldValuesTreshhold))
	tx := p.db.Where("created < ?", oldValuesTreshhold).Delete(&database.SteamQueryV2Values{})
	logging.LogDebug(fmt.Sprintf("OLD VALUES AFFECTED: %d", tx.RowsAffected))
	return tx.Error
}

func (p *psql) Close() error {
	if err := closePostgresLogFile(); err != nil {
		return err
	}
	db, err := p.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (p *psql) AddValues(values *database.SteamQueryV2Values) error {
	tx := p.db.Create(values)
	return tx.Error
}

func (p *psql) GetValues() ([]*database.SteamQueryV2Values, error) {
	var returns []*database.SteamQueryV2Values
	tx := p.db.Find(&returns)
	return returns, tx.Error
}

func (p *psql) GetValuesByDate(
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

func (p *psql) GetValuesByItemName(
	name string,
) ([]*database.SteamQueryV2Values, error) {
	var returns []*database.SteamQueryV2Values
	tx := p.db.Where("item_name = ?", name).Find(&returns)
	return returns, tx.Error
}

func (p *psql) GetValuesByItemNameAndDate(
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

func createPostgresLogFile(dir string) (*os.File, error) {
	f, err := os.Create(fmt.Sprintf("%s/postgres.log", dir))
	if err != nil {
		return nil, err
	}
	logFile = f
	return f, err
}

func closePostgresLogFile() error {
	return logFile.Close()
}
