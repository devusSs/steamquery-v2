package postgres

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/statistics/database"
)

var (
	logFile *os.File
	// Treshhold for old values, delete them when older than 30 days.
	oldValuesTreshhold = time.Now().AddDate(0, 0, 30)
)

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

	// TODO: use regex or strings functions to query raw version string
	// If version < 14 throw error.
	log.Println("PG VERSION HERE:", version)

	return nil
}

func (p *psql) Migrate() error {
	return p.db.AutoMigrate(&database.SteamQueryV2Values{})
}

func (p *psql) DeleteOldValues() error {
	tx := p.db.Where("created_at < ?", oldValuesTreshhold).Delete(&database.SteamQueryV2Values{})
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
	tx := p.db.Order("created_at desc").Find(&returns)
	return returns, tx.Error
}

func (p *psql) GetValuesByDate(
	startTime time.Time,
	endTime time.Time,
) ([]*database.SteamQueryV2Values, error) {
	var returns []*database.SteamQueryV2Values
	tx := p.db.Order("created_at desc").
		Where("created_at between ? and ?", startTime, endTime).
		Find(&returns)
	return returns, tx.Error
}

func (p *psql) GetValuesByItemName(
	name string,
) ([]*database.SteamQueryV2Values, error) {
	var returns []*database.SteamQueryV2Values
	tx := p.db.Order("created_at desc").Where("item_name = ?", name).Find(&returns)
	return returns, tx.Error
}

func (p *psql) GetValuesByItemNameAndDate(
	name string,
	startTime time.Time,
	endTime time.Time,
) ([]*database.SteamQueryV2Values, error) {
	var returns []*database.SteamQueryV2Values
	tx := p.db.Order("created_at desc").
		Where("item_name = ?", name).
		Where("created_at between ? and ?", startTime, endTime).
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
