package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/steam"
	"github.com/devusSs/steamquery-v2/tables"
	"github.com/devusSs/steamquery-v2/types"
)

const (
	baseURL = "https://steamcommunity.com/market/priceoverview/?appid=730&currency=3&market_hash_name="
)

var (
	spreadsheets *tables.SpreadsheetService

	itemColumnLetter string
	itemStartNumber  int
	itemEndNumber    int

	priceColumnLetter      string
	priceTotalColumnLetter string
	amountColumnLetter     string

	lastUpdatedCell string
	errorCell       string
	totalValueCell  string
	differenceCell  string

	steamAPIKey string
)

func InitQuery(
	service *tables.SpreadsheetService,
	itemList config.ItemList,
	priceColumn string,
	priceTotalColumn string,
	amountColumn string,
	orgCells config.OrgCells,
	steamAPIKeyConfig string,
) {
	spreadsheets = service

	itemColumnLetter = itemList.ColumnLetter
	itemStartNumber = itemList.StartNumber
	itemEndNumber = itemList.EndNumber

	priceColumnLetter = priceColumn
	priceTotalColumnLetter = priceTotalColumn
	amountColumnLetter = amountColumn

	lastUpdatedCell = orgCells.LastUpdatedCell
	errorCell = orgCells.ErrorCell
	totalValueCell = orgCells.TotalValueCell
	differenceCell = orgCells.DifferenceCell

	steamAPIKey = steamAPIKeyConfig
}

func RunQuery() error {
	steamUp, err := steam.IsSteamCSGOAPIUp(steamAPIKey)
	if err != nil {
		return err
	}

	if !steamUp {
		return errors.New("steam down, retry later")
	}

	logging.LogSuccess("Steam is up, proceeding")

	lastUpdatedString, err := getLastUpdatedCellValue()
	if err != nil {
		return err
	}

	if lastUpdatedString != "" {
		if err := compareLastUpdatedCell(lastUpdatedString); err != nil {
			return err
		}
	}

	lastErrorTimestamp, err := getLastErrorTimestamp()
	if err != nil {
		return err
	}

	if !lastErrorTimestamp.IsZero() {
		if err := compareLastErrorTimestamp(lastErrorTimestamp); err != nil {
			return err
		}
	}

	itemList, err := getItemNamesFromSheets()
	if err != nil {
		return err
	}

	amountList, err := getItemAmountCells()
	if err != nil {
		return err
	}

	priceMap, err := getItemMarketValues(itemList)
	if err != nil {
		return err
	}

	if err := writePricesForItemMap(itemList, priceMap); err != nil {
		return err
	}

	totalPricesItemsMap, err := calculateValueItemAmount(amountList)
	if err != nil {
		return err
	}

	if err := writeTotalPrices(totalPricesItemsMap); err != nil {
		return err
	}

	overallValuePreRun, err := getOverallValue()
	if err != nil {
		return err
	}

	if err := updateTotalValue(totalPricesItemsMap); err != nil {
		return err
	}

	if err := updateDifferenceCell(overallValuePreRun); err != nil {
		return err
	}

	if err := writeLastUpdatedCell(); err != nil {
		return err
	}

	return nil
}

func WriteErrorCell(err error) error {
	logging.LogError("An error occured, writing error cell, please wait")

	var values []interface{}
	values = append(values, err.Error())

	if err := spreadsheets.WriteSingleEntryToTable(errorCell, values); err != nil {
		return err
	}

	logging.LogSuccess("Successfully wrote error cell")

	return nil
}

func WriteNoErrorCell() error {
	logging.LogWarning("Writing error cell, please wait")

	var values []interface{}
	values = append(values, "No error occured.")

	if err := spreadsheets.WriteSingleEntryToTable(errorCell, values); err != nil {
		return err
	}

	logging.LogSuccess("Successfully wrote error cell")

	return nil
}

// Function to get the value of the last updated cell.
func getLastUpdatedCellValue() (string, error) {
	values, err := spreadsheets.GetValuesForCells(lastUpdatedCell, lastUpdatedCell)
	if err != nil {
		return "", err
	}

	// This might error on some IDEs depending on their module / import management.
	//
	// The error can be safely ignored, this will work anyway.
	for i := 0; i < len(values.Values); i++ {
		if fmt.Sprintf("%v", values.Values[i]) != "" {
			return strings.Replace(
				strings.Replace(fmt.Sprintf("%v", values.Values[i]), "[", "", 1),
				"]",
				"",
				1,
			), nil
		}
	}

	return "", err
}

// Function to compare the last updated cell to current time and exit if less than 5 minutes ago.
func compareLastUpdatedCell(lastUpdated string) error {
	timeObject, err := time.Parse("2006-01-02 15:04:05 MST", lastUpdated)
	if err != nil {
		return err
	}

	systemTimeCESTLoc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return err
	}

	systemTimeCEST := time.Now().Local().In(systemTimeCESTLoc)

	if systemTimeCEST.Sub(timeObject) < 3*time.Minute {
		leftOverTime := time.Until(timeObject.Add(3 * time.Minute)).Seconds()

		return fmt.Errorf(
			"last run has been less than 3 minutes ago, please wait %.2f second(s)",
			leftOverTime,
		)
	}

	return nil
}

// Function maps item names to their cell number (only number no letter).
func getItemNamesFromSheets() (map[string]int, error) {
	logging.LogWarning("Fetching item names, please wait")

	values, err := spreadsheets.GetValuesForCells(
		fmt.Sprintf("%s%d", itemColumnLetter, itemStartNumber),
		fmt.Sprintf("%s%d", itemColumnLetter, itemEndNumber),
	)
	if err != nil {
		return nil, err
	}

	returnMap := make(map[string]int)
	cellNumber := itemStartNumber

	// Might throw an error on some IDEs depending on their type handling / module management.
	//
	// The error may be ignored since the code works fine.
	for i := 0; i < len(values.Values); i++ {
		value := strings.Replace(fmt.Sprintf("%v", values.Values[i]), "[", "", 1)
		value = strings.Replace(value, "]", "", 1)
		if value != "" {
			returnMap[value] = cellNumber
		}
		cellNumber++
	}

	return returnMap, nil
}

// Function gets the market price for each item in item map.
func getItemMarketValues(items map[string]int) (map[string]string, error) {
	logging.LogWarning(
		"Fetching prices now, this might take a moment. The program WILL print something once it is done",
	)

	httpClient := http.Client{Timeout: 3 * time.Second}
	priceMap := make(map[string]string)
	getCount := 0

	for item := range items {
		if getCount == 20 {
			getCount = 0
			logging.LogWarning("Sleeping 1 minute to avoid Steam timeout")
			time.Sleep(1 * time.Minute)
		}

		req, err := http.NewRequest(
			http.MethodGet,
			fmt.Sprintf("%s%s", baseURL, url.QueryEscape(item)),
			nil,
		)
		if err != nil {
			return nil, err
		}

		// Pretend we are a legitimate browser accessing the Steam market to prevent potential blocks.
		req.Header.Add(
			"User-Agent",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36",
		)

		res, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			if res.StatusCode == http.StatusTooManyRequests {
				logging.LogError("Got timeouted by Steam, wait at least 1 minute or change IP")
			}

			return nil, fmt.Errorf(
				"unwanted Steam response: %s (code: %d)",
				res.Status,
				res.StatusCode,
			)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		var itemMarketResponse types.SteamItemResponse

		if err := json.Unmarshal(body, &itemMarketResponse); err != nil {
			return nil, err
		}

		priceMap[item] = strings.ReplaceAll(itemMarketResponse.LowestPrice, "-", "")

		getCount++

		logging.LogInfo(fmt.Sprintf("[FETCH] Done with: \t%s", item))
	}

	logging.LogSuccess(fmt.Sprintf("Successfully fetched %d item(s)", len(items)))

	return priceMap, nil
}

// Function writes the lowest market price to the corresponding price cell.
func writePricesForItemMap(itemList map[string]int, priceList map[string]string) error {
	logging.LogWarning("Writing prices to sheets now, please wait")

	for item, cellNumber := range itemList {
		if item == "" {
			continue
		}

		price, ok := priceList[item]
		if !ok {
			return fmt.Errorf("missing item %s in pricelist", item)
		}

		var values []interface{}
		values = append(values, price)

		if err := spreadsheets.WriteSingleEntryToTable(
			fmt.Sprintf("%s%d", priceColumnLetter, cellNumber),
			values,
		); err != nil {
			return err
		}

		logging.LogInfo(fmt.Sprintf("[WRITE] Done writing price for: \t%s", item))
	}

	logging.LogSuccess(fmt.Sprintf("Successfully wrote %d price(s) to sheets", len(priceList)))

	return nil
}

func getItemAmountCells() (map[int]int, error) {
	logging.LogWarning("Getting item amounts, please wait")

	startCell := itemStartNumber
	endCell := itemEndNumber
	returnMap := make(map[int]int)

	values, err := spreadsheets.GetValuesForCells(
		fmt.Sprintf("%s%d", amountColumnLetter, startCell),
		fmt.Sprintf("%s%d", amountColumnLetter, endCell),
	)
	if err != nil {
		return nil, err
	}

	currentCell := startCell

	// If user leaves amount fields empty return an error.
	if len(values.Values) == 0 {
		return nil, errors.New("did not specify any amounts in sheets")
	}

	// Might throw an error on some IDEs depending on their type handling / module management.
	//
	// The error may be ignored since the code works fine.
	for i := 0; i < len(values.Values); i++ {
		value := strings.Replace(fmt.Sprintf("%v", values.Values[i]), "[", "", 1)
		value = strings.Replace(value, "]", "", 1)

		if value == "" {
			value = "0"
		}

		convertValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}

		returnMap[currentCell] = int(convertValue)

		currentCell++
	}

	return returnMap, nil
}

// Function calculates item price * amount and returns a map of it.
func calculateValueItemAmount(
	amountList map[int]int,
) (map[int]string, error) {
	logging.LogWarning("Calculating item prices * amount, please wait")

	pricePerItemMap, err := fetchPricePerItem()
	if err != nil {
		return nil, err
	}

	logging.LogInfo(fmt.Sprintf("AMOUNT LIST: %v\n", amountList))
	logging.LogInfo(fmt.Sprintf("PRICE LIST: %v\n", pricePerItemMap))

	returnMap := make(map[int]string)

	// Map the cell number in amount list to cell number in priceperitemmap.
	for cell, amount := range amountList {
		if amount == 0 {
			continue
		}

		price, ok := pricePerItemMap[cell]
		if !ok {
			logging.LogInfo(fmt.Sprintf("missing key price map: CELL %d", cell))
			return nil, errors.New("missing key in price map")
		}

		if price == "" {
			logging.LogInfo(fmt.Sprintf("price is empty for cell %d", cell))
			return nil, errors.New("missing key in price map")
		}

		price = checkAndReplaceDotInPrice(price)
		price = strings.Replace(price, "€", "", 1)
		price = strings.Replace(price, ",", ".", 1)

		priceConvert, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return nil, err
		}

		priceTotal := float64(amount) * priceConvert

		priceTotalStr := strings.Replace(fmt.Sprintf("%.2f€", priceTotal), ".", ",", 1)

		returnMap[cell] = priceTotalStr
	}

	return returnMap, nil
}

// Function fetches the just updated price per item from sheets.
func fetchPricePerItem() (map[int]string, error) {
	returnMap := make(map[int]string)

	values, err := spreadsheets.GetValuesForCells(
		fmt.Sprintf("%s%d", priceColumnLetter, itemStartNumber),
		fmt.Sprintf("%s%d", priceColumnLetter, itemEndNumber),
	)
	if err != nil {
		return nil, err
	}

	currentCell := itemStartNumber

	for i := 0; i < len(values.Values); i++ {
		value := strings.Replace(fmt.Sprintf("%v", values.Values[i]), "[", "", 1)
		value = strings.Replace(value, "]", "", 1)

		returnMap[currentCell] = value

		currentCell++
	}

	return returnMap, nil
}

// Function writes the total prices to each cell.
func writeTotalPrices(totalPrices map[int]string) error {
	logging.LogWarning("Writing total prices (amounts), please wait")

	for cell, price := range totalPrices {
		var values []interface{}

		values = append(values, price)

		if err := spreadsheets.WriteSingleEntryToTable(fmt.Sprintf("%s%d", priceTotalColumnLetter, cell), values); err != nil {
			return err
		}

		logging.LogInfo(
			fmt.Sprintf("Done writing price to cell %s%d", priceTotalColumnLetter, cell),
		)
	}

	logging.LogSuccess("Successfully wrote total prices")

	return nil
}

// Function gets total (overall) value from sheets.
func getOverallValue() (string, error) {
	logging.LogWarning("Getting overall value pre run, please wait")

	values, err := spreadsheets.GetValuesForCells(totalValueCell, totalValueCell)
	if err != nil {
		return "", err
	}

	value := ""

	if len(values.Values) == 0 {
		value = "0,00€"
		logging.LogSuccess("Successfully fetched initial over value pre run")
		return value, nil
	}

	for i := 0; i < len(values.Values); i++ {
		value = strings.Replace(fmt.Sprintf("%v", values.Values[i]), "[", "", 1)
		value = strings.Replace(value, "]", "", 1)
	}

	logging.LogSuccess("Succesfully fetched overall value pre run")

	return value, nil
}

// Function calculates total value for items and adds it to cell.
func updateTotalValue(totalPrices map[int]string) error {
	logging.LogWarning("Updating total value cell, please wait")

	totalValue := 0.00

	for _, price := range totalPrices {
		price = strings.Replace(price, ",", ".", 1)
		price = strings.Replace(price, "€", "", 1)

		priceFloat, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}

		totalValue += priceFloat
	}

	finalPrice := strings.Replace(fmt.Sprintf("%.2f€", totalValue), ".", ",", 1)

	var values []interface{}

	values = append(values, finalPrice)

	if err := spreadsheets.WriteSingleEntryToTable(totalValueCell, values); err != nil {
		return err
	}

	logging.LogSuccess("Successfully updated total value cell")

	return nil
}

// Function calculates and updates the difference compared to last run.
func updateDifferenceCell(preRunTotal string) error {
	logging.LogWarning("Updating difference cell, please wait")

	var difference float64

	preRunTotal = checkAndReplaceDotInPrice(preRunTotal)
	preRunTotal = strings.Replace(preRunTotal, "€", "", 1)
	preRunTotal = strings.Replace(preRunTotal, ",", ".", 1)

	preRunFloat, err := strconv.ParseFloat(preRunTotal, 64)
	if err != nil {
		return err
	}

	totalValue, err := getTotalValueCell()
	if err != nil {
		return err
	}

	difference = totalValue - preRunFloat

	differenceStr := fmt.Sprintf("%.2f€", difference)
	differenceStr = strings.Replace(differenceStr, ".", ",", 1)

	var values []interface{}
	values = append(values, differenceStr)

	if err := spreadsheets.WriteSingleEntryToTable(differenceCell, values); err != nil {
		return err
	}

	logging.LogSuccess("Successfully updated difference cell")

	return nil
}

// Function gets the total value (which we updated).
func getTotalValueCell() (float64, error) {
	values, err := spreadsheets.GetValuesForCells(totalValueCell, totalValueCell)
	if err != nil {
		return 0, err
	}

	var totalValueStr string
	var totalValue float64

	if len(values.Values) == 0 {
		totalValueStr = "0,00€"
	}

	for i := 0; i < len(values.Values); i++ {
		values := fmt.Sprintf("%v", values.Values[i])
		values = checkAndReplaceDotInPrice(values)
		totalValueStr = strings.Replace(fmt.Sprintf("%v", values), "[", "", 1)
		totalValueStr = strings.Replace(totalValueStr, "]", "", 1)
	}

	totalValueStr = checkAndReplaceDotInPrice(totalValueStr)
	totalValueStr = strings.Replace(totalValueStr, "€", "", 1)
	totalValueStr = strings.Replace(totalValueStr, ",", ".", 1)

	totalValueFloat, err := strconv.ParseFloat(totalValueStr, 64)
	if err != nil {
		return 0, err
	}

	totalValue = totalValueFloat

	return totalValue, nil
}

// Function which updates last updated cell on sheet.
func writeLastUpdatedCell() error {
	logging.LogWarning("Writing last updated cell, please wait")

	var values []interface{}

	values = append(values, time.Now().Local().Format("2006-01-02 15:04:05 CEST"))

	if err := spreadsheets.WriteSingleEntryToTable(lastUpdatedCell, values); err != nil {
		return err
	}

	logging.LogSuccess("Successfully wrote last updated cell")

	return nil
}

// Helper function that checks for an already existing "." in a price and replaces it.
func checkAndReplaceDotInPrice(price string) string {
	price = strings.Replace(price, ".", "", 1)

	return price
}

// Helper function which queries the last error timestamp and returns it for analysis.
func getLastErrorTimestamp() (time.Time, error) {
	logging.LogWarning("Getting last error timestamp cell, please wait")

	values, err := spreadsheets.GetValuesForCells(errorCell, errorCell)
	if err != nil {
		return time.Time{}, err
	}

	// Error cell will be empty on first run, handle this event.
	if len(values.Values) == 0 {
		return time.Time{}, nil
	}

	var errorTimestamp time.Time

	// Might error depending on IDE's module / import management.
	//
	// Error may be ignored safely since everything works.
	for i := 0; i < len(values.Values); i++ {
		values := strings.Replace(fmt.Sprintf("%v", values.Values[i]), "[", "", 1)
		values = strings.Replace(values, "]", "", 1)

		if values == "No error occured." {
			return time.Time{}, nil
		}

		tsSplit := strings.Split(values, "TS:")
		ts := strings.Replace(tsSplit[1], ")", "", 1)

		// Convert the ts object to an actual time.Time object.
		timeObj, err := time.Parse("2006-01-02 15:04:05 MST", strings.TrimSpace(ts))
		if err != nil {
			return time.Time{}, err
		}

		errorTimestamp = timeObj
	}

	logging.LogWarning("Successfully got last error timestamp")

	return errorTimestamp, nil
}

// Helper function which compares last error timestamp to current time.
func compareLastErrorTimestamp(errorTS time.Time) error {
	systemTimeCESTLoc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return err
	}

	systemTimeCEST := time.Now().Local().In(systemTimeCESTLoc)

	if systemTimeCEST.Sub(errorTS) < 3*time.Minute {
		leftOverTime := time.Until(errorTS.Add(3 * time.Minute)).Seconds()

		return fmt.Errorf(
			"last error has been less than 3 minutes ago, please wait %.2f second(s)",
			leftOverTime,
		)
	}

	return nil
}
