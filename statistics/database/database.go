package database

import (
	"time"

	"github.com/google/uuid"
)

type Service interface {
	TestConnection() error
	VerifyVersion() error
	Migrate() error
	DeleteOldValues() error
	Close() error
	AddValues(*SteamQueryV2Values) error
	GetValues() ([]*SteamQueryV2Values, error)
	GetValuesByDate(time.Time, time.Time) ([]*SteamQueryV2Values, error)
	GetValuesByItemName(string) ([]*SteamQueryV2Values, error)
	GetValuesByItemNameAndDate(string, time.Time, time.Time) ([]*SteamQueryV2Values, error)
}

type SteamQueryV2Values struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`

	ItemName string
	Price    float64
	Created  time.Time
}
