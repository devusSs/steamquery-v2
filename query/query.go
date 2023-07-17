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
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/statistics"
	"github.com/devusSs/steamquery-v2/statistics/database"
	"github.com/devusSs/steamquery-v2/steam"
	"github.com/devusSs/steamquery-v2/system"
	"github.com/devusSs/steamquery-v2/tables"
	"github.com/devusSs/steamquery-v2/types"
)

var (
	usingBeta      bool
	skipCellChecks bool

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
	steamUser64 uint64

	QueryRunning bool
)

func InitQuery(
	service *tables.SpreadsheetService,
	itemList config.ItemList,
	priceColumn string,
	priceTotalColumn string,
	amountColumn string,
	orgCells config.OrgCells,
	steamAPIKeyConfig string,
	steamUserID64 uint64,
	skipChecks bool,
	betaFeatures bool,
) {
	usingBeta = betaFeatures

	if skipChecks {
		logging.LogWarning(
			"Skip checks flag specified, skipping last updated and error cell check on sheets",
		)
	}

	skipCellChecks = skipChecks

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
	steamUser64 = steamUserID64
}

func RunQuery(steamRetryInterval int) (float64, error) {
	QueryRunning = true

	steamUp, err := steam.IsSteamCSGOAPIUp(steamAPIKey)
	if err != nil {
		return 0, err
	}

	if !steamUp {
		if steamRetryInterval != 0 {
			logging.LogInfo(fmt.Sprintf("Rerunning steamquery in %d minutes", steamRetryInterval))
			time.Sleep(time.Duration(steamRetryInterval) * time.Minute)
			return RunQuery(steamRetryInterval)
		}

		return 0, errors.New("steam down, retry later")
	}

	logging.LogSuccess("Steam is up, proceeding")

	if !skipCellChecks {
		lastUpdatedString, err := getLastUpdatedCellValue()
		if err != nil {
			return 0, err
		}

		if lastUpdatedString != "" {
			if err := compareLastUpdatedCell(lastUpdatedString); err != nil {
				return 0, err
			}
		}

		lastErrorTimestamp, err := getLastErrorTimestamp()
		if err != nil {
			return 0, err
		}

		if !lastErrorTimestamp.IsZero() {
			if err := compareLastErrorTimestamp(lastErrorTimestamp); err != nil {
				return 0, err
			}
		}
	}

	itemList, err := getItemNamesFromSheets()
	if err != nil {
		return 0, err
	}

	amountList, err := getItemAmountCells()
	if err != nil {
		return 0, err
	}

	if usingBeta {
		totalSheetsMap, err := steam.GetAndCompareSteamInventory(
			steamAPIKey,
			steamUser64,
			itemList,
			amountList,
		)
		if err != nil {
			return 0, err
		}

		logging.LogDebug(fmt.Sprintf("TOTAL SHEETS MAP: %v", totalSheetsMap))

		return 0, nil
	}

	priceMap, marketAmountMap, err := getItemMarketValues(itemList)
	if err != nil {
		return 0, err
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go statistics.AnalyseVolumes(wg, time.Now(), marketAmountMap)

	go func() {
		for item, price := range priceMap {
			wg.Add(1)
			convertedPrice, err := strconv.ParseFloat(
				strings.ReplaceAll(strings.ReplaceAll(price, "€", ""), ",", "."),
				64,
			)
			if err != nil {
				logging.LogError(fmt.Sprintf("STATS ERROR: %s", err.Error()))
			}

			if err := statistics.AddStatistics(&database.SteamQueryV2Values{ItemName: item, Price: convertedPrice, Volume: marketAmountMap[item], Created: time.Now()}); err != nil {
				logging.LogError(fmt.Sprintf("STATS ERROR: %s", err.Error()))
			}
			wg.Done()
			logging.LogDebug(fmt.Sprintf("Added statistics for %s", item))
		}
	}()

	if err := writePricesForItemMap(itemList, priceMap); err != nil {
		return 0, err
	}

	totalPricesItemsMap, err := calculateValueItemAmount(amountList)
	if err != nil {
		return 0, err
	}

	if err := writeTotalPrices(totalPricesItemsMap); err != nil {
		return 0, err
	}

	overallValuePreRun, err := getOverallValue()
	if err != nil {
		return 0, err
	}

	if err := updateTotalValue(totalPricesItemsMap); err != nil {
		return 0, err
	}

	priceDifference, err := updateDifferenceCell(overallValuePreRun)
	if err != nil {
		return 0, err
	}

	if err := writeLastUpdatedCell(); err != nil {
		return 0, err
	}

	wg.Wait()

	QueryRunning = false

	return priceDifference, nil
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
	logging.LogInfo("Writing error cell, please wait")

	var values []interface{}
	values = append(values, "No error occured.")

	if err := spreadsheets.WriteSingleEntryToTable(errorCell, values); err != nil {
		return err
	}

	logging.LogSuccess("Successfully wrote error cell")

	return nil
}

// Helper function to calculate estimated runs / requests per day on watchdog mode.
//
// Will return an error when potential requests exceed limit (20 / minute).
func CompareRequestsDayWithLimit(retryInterval int) error {
	// 24 hours = 1440min, 20 requests per minute is the limit for priceoverview
	maxRequestsDayEstimate := 1440 * 20

	items, err := getItemNamesFromSheets()
	if err != nil {
		return err
	}

	requestsPerRun := 0

	for item := range items {
		if !strings.Contains(item, "empty_cell_") {
			requestsPerRun++
		}
	}

	runsPerDay := 24 / retryInterval

	requestsPerDay := requestsPerRun * runsPerDay

	if requestsPerDay > maxRequestsDayEstimate {
		return errors.New("potential requests limit exceeded, please increase your retry interval")
	}

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
	lastUpdated = strings.Replace(lastUpdated, " CEST", "", 1)

	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return err
	}

	timeObject, err := time.ParseInLocation("2006-01-02 15:04:05", lastUpdated, location)
	if err != nil {
		return err
	}

	systemTimeCESTLoc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return err
	}

	systemTimeCEST := time.Now().Local().In(systemTimeCESTLoc)

	logging.LogDebug(fmt.Sprintf("LAST UPDATED: %s", lastUpdated))
	logging.LogDebug(fmt.Sprintf("SYSTEM TIME CEST: %s", systemTimeCEST))

	if systemTimeCEST.Sub(timeObject) < 3*time.Minute {
		logging.LogDebug(fmt.Sprintf("TIME DIFF: %v", systemTimeCEST.Sub(timeObject)))

		leftOverTime := time.Until(timeObject.Add(3 * time.Minute)).Seconds()

		logging.LogDebug(fmt.Sprintf("LEFTOVER TIME: %.2f second(s)", leftOverTime))

		return fmt.Errorf(
			"last run has been less than 3 minutes ago, please wait %.2f second(s)",
			leftOverTime,
		)
	}

	return nil
}

// Function maps item names to their cell number (only number no letter).
func getItemNamesFromSheets() (map[string]int, error) {
	logging.LogInfo("Fetching item names, please wait")

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
		if value == "" {
			returnMap[fmt.Sprintf("empty_cell_%d", cellNumber)] = cellNumber
		}
		cellNumber++
	}

	logging.LogDebug(fmt.Sprintf("Item names from sheets map: %v", returnMap))

	logging.LogSuccess("Successfully fetched item names")

	return returnMap, nil
}

// Function gets the market price for each item in item map.
func getItemMarketValues(items map[string]int) (map[string]string, map[string]int, error) {
	startTime := time.Now()

	logging.LogInfo(
		"Fetching prices now, this might take a moment. The program WILL print something once it is done",
	)

	logging.LogWarning("Please DO NOT use Steam anywhere on your network for that time")

	httpClient := http.Client{Timeout: 3 * time.Second}
	priceMap := make(map[string]string)
	volumeMap := make(map[string]int)
	getCount := 0
	itemsFetched := 0
	sleepCount := 0
	sleepCountMax := 0
	actualItemLen := 0

	for item := range items {
		if strings.Contains(item, "empty_cell") {
			continue
		}
		actualItemLen++
	}

	sleepCountMax = int(actualItemLen / 20)

	for item := range items {
		if strings.Contains(item, "empty_cell") {
			continue
		}

		item = strings.TrimSpace(item)

		if getCount == 20 {
			getCount = 0
			sleepCount++
			logging.LogInfo(
				fmt.Sprintf(
					"Sleeping 1 minute to avoid Steam timeout (%d/%d sleep(s))",
					sleepCount,
					sleepCountMax,
				),
			)
			time.Sleep(1 * time.Minute)
		}

		logging.LogDebug(fmt.Sprintf("Fetching price for \t\t%s", item))

		u := "https://steamcommunity.com/market/priceoverview/?" + url.Values{
			"appid":            {strconv.FormatUint(730, 10)},
			"country":          {"EN"},
			"currency":         {"3"},
			"market_hash_name": {item},
		}.Encode()

		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, nil, err
		}

		req.Header.Set("User-Agent", system.GetUserAgentHeaderFromOS())

		res, err := httpClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			if res.StatusCode == http.StatusTooManyRequests {
				logging.LogError("Got timeouted by Steam, wait at least 1 minute or change IP")
			}

			if res.StatusCode == http.StatusInternalServerError {
				logging.LogError(
					fmt.Sprintf("Could not find item on Steam community market: %s", item),
				)

				logging.LogWarning(
					fmt.Sprintf("Proceeding with list, setting item price for %s to 0,00€", item),
				)

				priceMap[item] = "0,00€"

				getCount++

				continue
			}

			return nil, nil, fmt.Errorf(
				"unwanted Steam response: %s (code: %d)",
				res.Status,
				res.StatusCode,
			)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, nil, err
		}

		system.BytesUsed += len(body)

		var itemMarketResponse types.SteamItemResponse

		if err := json.Unmarshal(body, &itemMarketResponse); err != nil {
			return nil, nil, err
		}

		// Replace price with 0 when item has no active listing on Steam market.
		if !itemMarketResponse.Success {
			logging.LogWarning(fmt.Sprintf("No Steam market listing for item %s", item))
			priceMap[item] = "0,00€"
			getCount++
			itemsFetched++
			logging.LogDebug(fmt.Sprintf("Done fetching price for: \t%s", item))
			continue
		}

		priceMap[item] = strings.ReplaceAll(itemMarketResponse.LowestPrice, "-", "")

		volumeConv, err := strconv.Atoi(strings.Replace(itemMarketResponse.Volume, ",", "", 1))
		if err != nil {
			return nil, nil, err
		}

		volumeMap[item] = volumeConv

		getCount++

		itemsFetched++

		logging.LogDebug(
			fmt.Sprintf(
				"Done fetching price for: \t%s (price: %s)",
				item,
				itemMarketResponse.LowestPrice,
			),
		)
	}

	logging.LogDebug(fmt.Sprintf("Item prices post fetch: %v", priceMap))

	logging.LogSuccess(fmt.Sprintf("Successfully fetched %d item price(s)", itemsFetched))

	logging.LogDebug(fmt.Sprintf("took %.2f second(s)", time.Since(startTime).Seconds()))

	return priceMap, volumeMap, nil
}

// Function writes the lowest market price to the corresponding price cell.
func writePricesForItemMap(itemList map[string]int, priceList map[string]string) error {
	logging.LogInfo("Writing prices to sheets now, please wait")

	priceMap := make(map[int]string)

	for item, cellNumber := range itemList {
		if strings.Contains(item, "empty_cell") {
			priceMap[cellNumber] = ""
			continue
		}

		price, ok := priceList[item]
		if !ok {
			return fmt.Errorf("missing item %s in priceList", item)
		}

		priceMap[cellNumber] = price
	}

	logging.LogDebug(fmt.Sprintf("Price map pre write: %v", priceMap))

	if err := spreadsheets.WriteMultipleEntriesToTable(priceMap, priceColumnLetter); err != nil {
		return err
	}

	logging.LogSuccess(fmt.Sprintf("Successfully wrote %d price(s) to sheets", len(priceList)))

	return nil
}

func getItemAmountCells() (map[int]int, error) {
	logging.LogInfo("Getting item amounts, please wait")

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

	logging.LogSuccess("Successfully got item amounts")

	return returnMap, nil
}

// Function calculates item price * amount and returns a map of it.
func calculateValueItemAmount(
	amountList map[int]int,
) (map[int]string, error) {
	logging.LogInfo("Calculating item prices * amount, please wait")

	pricePerItemMap, err := fetchPricePerItem()
	if err != nil {
		return nil, err
	}

	returnMap := make(map[int]string)

	// Map the cell number in amount list to cell number in priceperitemmap.
	for cell, amount := range amountList {
		if amount == 0 {
			returnMap[cell] = ""
			continue
		}

		price, ok := pricePerItemMap[cell]
		if !ok {
			logging.LogWarning(fmt.Sprintf("missing key price map: CELL %d", cell))
			return nil, errors.New("missing key in price map")
		}

		if price == "" {
			logging.LogWarning(fmt.Sprintf("price is empty for cell %d", cell))
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

	logging.LogDebug(fmt.Sprintf("TOTAL VALUE MAP: %v", returnMap))

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

	logging.LogDebug(fmt.Sprintf("PRICE PER ITEM MAP: %v", returnMap))

	return returnMap, nil
}

// Function writes the total prices to each cell.
func writeTotalPrices(totalPrices map[int]string) error {
	logging.LogInfo("Writing total prices (amounts), please wait")

	if err := spreadsheets.WriteMultipleEntriesToTable(totalPrices, priceTotalColumnLetter); err != nil {
		return err
	}

	logging.LogSuccess("Successfully wrote total prices")

	return nil
}

// Function gets total (overall) value from sheets.
func getOverallValue() (string, error) {
	logging.LogInfo("Getting overall value pre run, please wait")

	values, err := spreadsheets.GetValuesForCells(totalValueCell, totalValueCell)
	if err != nil {
		return "", err
	}

	value := ""

	if len(values.Values) == 0 {
		value = "0,00€"
		logging.LogSuccess("Successfully fetched initial overall value pre run")
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
	logging.LogInfo("Updating total value cell, please wait")

	totalValue := 0.00

	for _, price := range totalPrices {
		if price == "" {
			continue
		}

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
func updateDifferenceCell(preRunTotal string) (float64, error) {
	logging.LogInfo("Updating difference cell, please wait")

	var difference float64

	preRunTotal = checkAndReplaceDotInPrice(preRunTotal)
	preRunTotal = strings.Replace(preRunTotal, "€", "", 1)
	preRunTotal = strings.Replace(preRunTotal, ",", ".", 1)

	preRunFloat, err := strconv.ParseFloat(preRunTotal, 64)
	if err != nil {
		return 0, err
	}

	totalValue, err := getTotalValueCell()
	if err != nil {
		return 0, err
	}

	difference = totalValue - preRunFloat

	differenceStr := fmt.Sprintf("%.2f€", difference)
	differenceStr = strings.Replace(differenceStr, ".", ",", 1)

	var values []interface{}
	values = append(values, differenceStr)

	if err := spreadsheets.WriteSingleEntryToTable(differenceCell, values); err != nil {
		return 0, err
	}

	logging.LogSuccess("Successfully updated difference cell")

	return difference, nil
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
	logging.LogInfo("Writing last updated cell, please wait")

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
	logging.LogInfo("Getting last error timestamp cell, please wait")

	values, err := spreadsheets.GetValuesForCells(errorCell, errorCell)
	if err != nil {
		return time.Time{}, err
	}

	// Error cell will be empty on first run, handle this event.
	if len(values.Values) == 0 {
		logging.LogSuccess("First run, no error timestamp, proceeding")
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
			logging.LogSuccess("No error occured on last run, proceeding")
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

	logging.LogSuccess("Successfully got last error timestamp")

	return errorTimestamp, nil
}

// Helper function which compares last error timestamp to current time.
func compareLastErrorTimestamp(errorTS time.Time) error {
	logging.LogDebug(fmt.Sprintf("ERROR TS: %v", errorTS))

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
