package tables

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"

	"github.com/devusSs/steamquery-v2/logging"
)

type SpreadsheetService struct {
	spreadsheetID string
	service       *sheets.Service
}

func NewSpreadsheetService(gCloudConfPath, spreadsheetID string) (*SpreadsheetService, error) {
	ctx := context.Background()
	srv, err := sheets.NewService(
		ctx,
		option.WithCredentialsFile(gCloudConfPath),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
	if err != nil {
		return nil, err
	}
	defer ctx.Done()

	c := &SpreadsheetService{
		spreadsheetID: spreadsheetID,
		service:       srv,
	}

	return c, nil
}

func (s *SpreadsheetService) TestConnection() error {
	_, err := s.service.Spreadsheets.Values.Get(s.spreadsheetID, "A1:Z1").Do()
	return err
}

func (s *SpreadsheetService) GetValuesForCells(
	startCell, endCell string,
) (*sheets.ValueRange, error) {
	values, err := s.service.Spreadsheets.Values.Get(s.spreadsheetID, fmt.Sprintf("%s:%s", startCell, endCell)).
		Do()
	return values, err
}

func (s *SpreadsheetService) WriteSingleEntryToTable(cell string, values []interface{}) error {
	logging.LogWarning("Sleeping 2 seconds to avoid Google timeout")
	time.Sleep(2 * time.Second)

	var vr sheets.ValueRange
	vr.Values = append(vr.Values, values)

	_, err := s.service.Spreadsheets.Values.Update(s.spreadsheetID, cell, &vr).
		ValueInputOption("USER_ENTERED").
		Do()

	return err
}
