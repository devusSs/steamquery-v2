package statistics

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/statistics/database"
	"github.com/devusSs/steamquery-v2/statistics/database/postgres"
	"github.com/devusSs/steamquery-v2/statistics/database/sqlite"
)

const (
	analysisDir = "stats"
	DBPostgres  = "postgres"
	DBSQLite    = "sqlite"
)

var (
	service  database.Service
	ovTicker *time.Ticker

	timeNow      = time.Now().Local()
	currentDay   = timeNow.Day()
	currentMonth = timeNow.Month()
	currentYear  = timeNow.Year()
	currentHour  = timeNow.Hour()
	currentMin   = timeNow.Minute()
	dateFormat   = fmt.Sprintf(
		"%d-%d-%d_%d-%d",
		currentYear,
		currentMonth,
		currentDay,
		currentHour,
		currentMin,
	)
)

func SetupStatistics(cfg *config.Postgres, logsDir, dbType string) error {
	var svc database.Service
	var err error

	switch dbType {
	case DBPostgres:
		logging.LogDebug("Setting up Postgres database")
		svc, err = postgres.NewPostgresConnection(cfg, logsDir)
		if err != nil {
			return err
		}
		logging.LogSuccess("Done setting up Postgres database")
	case DBSQLite:
		logging.LogDebug("Setting up SQLite database")
		svc, err = sqlite.NewSqliteConnection(cfg, logsDir)
		if err != nil {
			return err
		}
		logging.LogSuccess("Done setting up SQLite database")
	default:
		return fmt.Errorf("unsupported db type: %s, want %s or %s", dbType, DBPostgres, DBSQLite)
	}

	if err = svc.TestConnection(); err != nil {
		return err
	}

	if err = svc.VerifyVersion(); err != nil {
		return err
	}

	if err = svc.Migrate(); err != nil {
		return err
	}

	if err = svc.DeleteOldValues(); err != nil {
		return err
	}

	ovTicker = time.NewTicker(12 * time.Hour)

	go func() {
		for range ovTicker.C {
			if err := svc.DeleteOldValues(); err != nil {
				logging.LogError(err.Error())
			}
			logging.LogInfo("Cleared old statistics")
		}
	}()

	logging.LogDebug("Setup goroutine for deleting old db values every 12 hours")

	service = svc

	return nil
}

func AddStatistics(model *database.SteamQueryV2Values) error {
	return service.AddValues(model)
}

// TODO: add warning for user if data is inconsistent (like program only being run 3 times a week)
func StartStatsAnalysis(cfg *config.Postgres, logsDir, dbType string) {
	switch dbType {
	case DBPostgres:
		if err := SetupStatistics(cfg, logsDir, DBPostgres); err != nil {
			logging.LogError(err.Error())
			if err := exitStats(); err != nil {
				logging.LogFatal(err.Error())
			}
			return
		}
	case DBSQLite:
		if err := SetupStatistics(cfg, logsDir, DBSQLite); err != nil {
			logging.LogError(err.Error())
			if err := exitStats(); err != nil {
				logging.LogFatal(err.Error())
			}
			return
		}
	default:
		fmt.Printf("Unsupported db type: %s, want %s or %s", dbType, DBPostgres, DBSQLite)
		return
	}

	if err := checkDatabase(); err != nil {
		logging.LogError(err.Error())
		if err := exitStats(); err != nil {
			logging.LogFatal(err.Error())
		}
		return
	}

	if err := logging.CreateLogsDirectory(fmt.Sprintf("%s/%s", logsDir, analysisDir)); err != nil {
		logging.LogError(err.Error())
		if err := exitStats(); err != nil {
			logging.LogFatal(err.Error())
		}
		return
	}

	fmt.Println("Running app in analysis mode")

	fmt.Println("The program will now ask for your statistics specifications")
	fmt.Println("Please read the prompts carefully and answer accordingly")
	fmt.Println("")
	fmt.Println("WARNING: please use exact item names (at best copy them from your sheet)")

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("")
	fmt.Println(
		"Enter the item names seperated by commas you'd like to analyse, leave blank to analyse all (might take some time)",
	)
	fmt.Print("-> ")
	text, err := reader.ReadString('\n')
	if err != nil {
		logging.LogError(err.Error())
		if err := exitStats(); err != nil {
			logging.LogFatal(err.Error())
		}
		return
	}
	text = strings.ReplaceAll(text, "\n", "")

	itemNames := strings.Split(text, ",")

	fmt.Println("")
	fmt.Println("Enter the date range you'd like to analyse, leave blank for past 30d (max)")
	fmt.Println("Example: 7d = past 7 days, 1h = past 1 hour")
	// TODO: add possible startDates
	fmt.Println("Different date ranges (specific start) are not supported yet, sorry")
	fmt.Print("-> ")
	text, err = reader.ReadString('\n')
	if err != nil {
		logging.LogError(err.Error())
		if err := exitStats(); err != nil {
			logging.LogFatal(err.Error())
		}
		return
	}
	text = strings.ReplaceAll(text, "\n", "")

	dateRange := text

	var itemNamesFinal []string

	itemNamesFinal = itemNames

	if len(itemNames) == 0 || len(itemNames) == 1 {
		itemNamesFinal = append(itemNamesFinal, "all")
	}

	if dateRange == "" {
		dateRange = "all time"
	}

	fmt.Println("")
	fmt.Println("Following inputs will be analysed:")
	fmt.Printf("Item names: %s\n", strings.Join(itemNamesFinal, ", "))
	fmt.Printf("Date range: %s\n", dateRange)

	logging.LogDebug(fmt.Sprintf("item names: %v", itemNamesFinal))
	logging.LogDebug(fmt.Sprintf("date range: %v", dateRange))

	fmt.Println("")
	fmt.Println("Are these inputs correct (y/n)?")
	fmt.Print("-> ")
	text, err = reader.ReadString('\n')
	if err != nil {
		logging.LogError(err.Error())
		if err := exitStats(); err != nil {
			logging.LogFatal(err.Error())
		}
		return
	}
	text = strings.ReplaceAll(text, "\n", "")

	switch text {
	case "y":
		fmt.Println("")
		if err := performAnalysis(itemNamesFinal, dateRange, logsDir); err != nil {
			logging.LogError(err.Error())
			if err := exitStats(); err != nil {
				logging.LogFatal(err.Error())
			}
			return
		}
		logging.LogSuccess("Done with analysis")
	case "n":
		fmt.Println("")
		fmt.Println("Please restart the program and re-enter your values")
	default:
		fmt.Println("")
		fmt.Println("invalid input, exiting")
	}
	if err := exitStats(); err != nil {
		log.Fatal(err)
	}
}

func CloseStatistics() error {
	ovTicker.Stop()
	return service.Close()
}

func checkDatabase() error {
	if service == nil {
		return errors.New("service = nil")
	}

	testResults, err := service.GetValues()
	if err != nil {
		return err
	}

	if len(testResults) == 0 {
		return errors.New("no results on database yet")
	}

	return nil
}

func performAnalysis(itemNames []string, dateRange, logsDir string) error {
	var results []*database.SteamQueryV2Values
	var err error
	writeDir := fmt.Sprintf("%s/%s", logsDir, analysisDir)

	if sliceItemExists(itemNames, "all") {
		logging.LogWarning("Fetching all items, this will result in a cluttered chart")
		switch dateRange {
		case "all time":
			results, err = service.GetValues()
			if err != nil {
				return err
			}

			for _, result := range results {
				logging.LogDebug(fmt.Sprintf("%v", result.Created))
			}
		default:
			var startTime time.Time
			var endTime time.Time

			endTime = time.Now()

			unit, amount, err := convertDateRange(dateRange)
			if err != nil {
				return err
			}

			logging.LogDebug(fmt.Sprintf("unit: %s ; amount: %d", unit, amount))

			if unit == "h" {
				startTime = endTime.Add(-time.Duration(amount) * time.Hour)
			}

			if unit == "d" {
				startTime = endTime.AddDate(0, 0, -amount)
			}

			results, err = service.GetValuesByDate(startTime, endTime)
			if err != nil {
				return err
			}
		}
	} else {
		switch dateRange {
		case "all time":
			for _, item := range itemNames {
				resultsInst, err := service.GetValuesByItemName(item)
				if err != nil {
					return err
				}

				results = append(results, resultsInst...)
			}
		default:
			var startTime time.Time
			var endTime time.Time

			endTime = time.Now()

			unit, amount, err := convertDateRange(dateRange)
			if err != nil {
				return err
			}

			logging.LogDebug(fmt.Sprintf("unit: %s ; amount: %d", unit, amount))

			if unit == "h" {
				startTime = endTime.Add(-time.Duration(amount) * time.Hour)
			}

			if unit == "d" {
				startTime = endTime.AddDate(0, 0, -amount)
			}

			for _, item := range itemNames {
				resultsInst, err := service.GetValuesByItemNameAndDate(item, startTime, endTime)
				if err != nil {
					return err
				}

				results = append(results, resultsInst...)
			}
		}
	}

	if len(results) == 0 {
		return errors.New("no results found on database")
	}

	database.SortByDate(results)

	startDate := results[0].Created
	endDate := results[len(results)-1].Created
	totalDays := int(endDate.Sub(startDate).Hours() / 24)

	if totalDays != len(results)-1 {
		logging.LogWarning(
			"Data is inconsistent. There are missing entries, stats may be inaccurate",
		)
	}

	logging.LogInfo("Generating and writing chart, please wait")

	chart, err := generateChart(results)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s/%s", writeDir, fmt.Sprintf("%s_chart.html", dateFormat))

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := chart.Render(f); err != nil {
		return err
	}

	logging.LogSuccess(fmt.Sprintf("Wrote chart to file: %s", fileName))

	return nil
}

func generateChart(results []*database.SteamQueryV2Values) (*charts.Line, error) {
	var dateRange []string
	prices := make(map[string][]float64)

	for _, result := range results {
		if !sliceItemExists(dateRange, result.Created.Format("2006-01-02 15:04:05")) {
			dateRange = append(dateRange, result.Created.Format("2006-01-02 15:04:05"))
		}
		prices[result.ItemName] = append(prices[result.ItemName], result.Price)
	}

	if len(dateRange) < 10 {
		logging.LogWarning("Small date range, charts might not say much")
	}

	line := charts.NewLine()

	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			PageTitle: "Steamquery-v2 Statistics",
			Theme:     types.ThemeInfographic,
			Width:     "1000px",
			Height:    "800px",
		}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Datetime"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Price in €"}),
		charts.WithLegendOpts(opts.Legend{
			Show:    true,
			Type:    "scroll",
			Padding: [4]int{5, 5, 20, 5},
		}),
		charts.WithTooltipOpts(
			opts.Tooltip{Show: true},
		),
	)

	line.SetXAxis(dateRange).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true, ShowSymbol: true}),
			charts.WithMarkPointNameTypeItemOpts(
				opts.MarkPointNameTypeItem{Name: "Maximum", Type: "max"},
				opts.MarkPointNameTypeItem{Name: "Average", Type: "average"},
				opts.MarkPointNameTypeItem{Name: "Minimum", Type: "min"},
			),
			charts.WithMarkPointStyleOpts(
				opts.MarkPointStyle{Label: &opts.Label{Show: true}}),
		)

	for item, price := range prices {
		line.AddSeries(item, generateLineItems(price))
	}

	return line, nil
}

func convertDateRange(input string) (string, int, error) {
	if strings.Contains(input, "h") {
		convertInt, err := strconv.ParseInt(strings.Split(input, "h")[0], 10, 64)
		if err != nil {
			return "", 0, err
		}
		return "h", int(convertInt), nil
	}

	if strings.Contains(input, "d") {
		convertInt, err := strconv.ParseInt(strings.Split(input, "d")[0], 10, 64)
		if err != nil {
			return "", 0, err
		}
		return "d", int(convertInt), nil
	}

	return "", 0, fmt.Errorf("unsupported specifier, supported: %s and %s", "h", "d")
}

func sliceItemExists(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func generateLineItems(prices []float64) []opts.LineData {
	items := make([]opts.LineData, 0)
	for _, price := range prices {
		items = append(items, opts.LineData{Value: price})
	}
	return items
}

func exitStats() error {
	return CloseStatistics()
}
