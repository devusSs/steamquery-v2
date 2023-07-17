package tables

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/devusSs/steamquery-v2/logging"
	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
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
	var vr sheets.ValueRange
	vr.Values = append(vr.Values, values)

	_, err := s.service.Spreadsheets.Values.Update(s.spreadsheetID, cell, &vr).
		ValueInputOption("USER_ENTERED").
		Do()

	return err
}

func (s *SpreadsheetService) WriteMultipleEntriesToTable(
	inputMap map[int]string,
	column string,
) error {
	startTime := time.Now()

	var keys []int
	for key := range inputMap {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	var values [][]interface{}

	for _, key := range keys {
		values = append(values, []interface{}{inputMap[key]})
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	cellRange := fmt.Sprintf(
		"%s:%s",
		fmt.Sprintf("%s%d", column, keys[0]),
		fmt.Sprintf("%s%d", column, keys[len(keys)-1]),
	)

	_, err := s.service.Spreadsheets.Values.Update(s.spreadsheetID, cellRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Do()

	logging.LogDebug(fmt.Sprintf("took %.2f second(s)", time.Since(startTime).Seconds()))

	return err
}
