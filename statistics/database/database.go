package database

import (
	"sort"
	"time"

	"gorm.io/gorm"

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
	ID uuid.UUID `gorm:"type:uuid;primary_key;"`

	ItemName string
	Price    float64
	Created  time.Time
}

func (s *SteamQueryV2Values) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return
}

func SortByDate(data []*SteamQueryV2Values) {
	sort.Slice(data, func(i, j int) bool {
		return data[i].Created.Before(data[j].Created)
	})
}
