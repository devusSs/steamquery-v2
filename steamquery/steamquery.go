package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/query"
	"github.com/devusSs/steamquery-v2/statistics"
	"github.com/devusSs/steamquery-v2/system"
	"github.com/devusSs/steamquery-v2/tables"
	"github.com/devusSs/steamquery-v2/updater"
	"github.com/devusSs/steamquery-v2/utils"
)

// The maximum price items are allowed to drop (in total) before the app sends a warning mail.
//
// This will only work in watchdog mode.
var maxPriceDifference float64

func main() {
	startTime := time.Now().Local()

	cfgPathFlag := flag.String("c", "./files/config.json", "sets the config path")
	gCloudPathFlag := flag.String("g", "./files/gcloud.json", "sets the gcloud config path")
	debugFlag := flag.Bool("d", false, "sets the app to debugging mode")
	versionFlag := flag.Bool("v", false, "prints version and build info and exits")
	logDirFlag := flag.String("l", "./logs", "sets the logging directory")
	disableUpdatesFlag := flag.Bool("du", false, "disables update check on startup")
	analysisModeFlag := flag.Bool("a", false, "runs the app in analysis mode and exits")
	skipChecks := flag.Bool("sc", false, "skips last updated and error cell checks on sheets")
	betaFeatures := flag.Bool("b", false, "enables beta features, not recommended")
	watchDog := flag.Bool("w", false, "enables watchdog mode with specified interval")
	analysisFlag := flag.Bool("z", false, "performs data analysis for prices and exits")
	envFlag := flag.Bool("e", false, "uses env instead of config file, useful for docker")
	envFile := flag.String(
		"efile",
		"",
		"path for env file if desired, leave blank for raw env variables",
	)
	flag.Parse()

	alreadyRunning, err := system.CheckAlreadyRunning(*watchDog, *analysisFlag)
	if err != nil {
		log.Fatal(err)
	}

	if alreadyRunning {
		log.Println("Program already running, exiting")
		return
	}

	if *analysisModeFlag {
		if err := system.RunAnalysisMode(*logDirFlag, *cfgPathFlag, *gCloudPathFlag); err != nil {
			log.Fatalf("Error running analysis mode: %s\n", err.Error())
		}
		return
	}

	if !*disableUpdatesFlag {
		if err := updater.CheckForUpdatesAndApply(); err != nil {
			log.Fatalf("Error checking for updates: %s\n", err.Error())
		}
	}

	if err := updater.CheckMinVersion(); err != nil {
		log.Fatal(err)
	}

	system.InitClearFunc()

	printAsciiArt()

	if err := logging.CreateLogsDirectory(*logDirFlag); err != nil {
		log.Fatalf("Error creating logs directory: %s\n", err.Error())
	}

	if *debugFlag {
		if err := logging.InitLoggers("dev"); err != nil {
			log.Fatalf("Error initiating loggers: %s\n", err.Error())
		}
	} else {
		if err := logging.InitLoggers("release"); err != nil {
			log.Fatalf("Error initiating loggers: %s\n", err.Error())
		}
	}

	if *analysisFlag {
		cfg, err := config.LoadConfig(*cfgPathFlag)
		if err != nil {
			logging.LogFatal(err.Error())
		}

		if err := cfg.CheckConfig(true); err != nil {
			logging.LogFatal(err.Error())
		}

		if *watchDog {
			statistics.StartStatsAnalysis(
				&cfg.WatchDog.Postgres,
				*logDirFlag,
				statistics.DBPostgres,
			)
		} else {
			statistics.StartStatsAnalysis(&cfg.WatchDog.Postgres, *logDirFlag, statistics.DBSQLite)
		}

		return
	}

	if *versionFlag {
		updater.PrintBuildInfo()
		return
	}

	if *debugFlag {
		logging.LogWarning("Running debug mode, app might be unstable")

		updater.PrintBuildInfo()
	}

	if *betaFeatures {
		logging.LogWarning("Using beta features, please expects bugs and crucial errors")
	}

	if *disableUpdatesFlag {
		logging.LogWarning("Skipped update check because of -du flag")
	}

	var cfg *config.Config

	if *envFlag {
		cfg, err = config.LoadConfigFromEnv(*envFile)
		if err != nil {
			logging.LogFatal(err.Error())
		}
	} else {
		cfg, err = config.LoadConfig(*cfgPathFlag)
		if err != nil {
			logging.LogFatal(err.Error())
		}
	}

	if err := cfg.CheckConfig(*watchDog); err != nil {
		logging.LogFatal(err.Error())
	}

	if err := system.CheckForGCloudConfigFile(*gCloudPathFlag); err != nil {
		logging.LogFatal(err.Error())
	}

	svc, err := tables.NewSpreadsheetService(*gCloudPathFlag, cfg.SpreadSheetID)
	if err != nil {
		logging.LogFatal(err.Error())
	}

	if err := svc.TestConnection(); err != nil {
		logging.LogFatal(err.Error())
	}

	query.InitQuery(
		svc,
		cfg.ItemList,
		cfg.PriceColumn,
		cfg.PriceTotalColumn,
		cfg.AmountColumn,
		cfg.OrgCells,
		cfg.SteamAPIKey,
		cfg.SteamUserID64,
		*skipChecks,
		*betaFeatures,
	)

	logging.LogInfo("Running statistics setup, please wait")

	if *watchDog {
		if err := statistics.SetupStatistics(&cfg.WatchDog.Postgres, *logDirFlag, statistics.DBPostgres); err != nil {
			logging.LogFatal(err.Error())
		}
	} else {
		if err := statistics.SetupStatistics(&cfg.WatchDog.Postgres, *logDirFlag, statistics.DBSQLite); err != nil {
			logging.LogFatal(err.Error())
		}
	}

	logging.LogSuccess("Done with statistics setup")

	logging.LogDebug(fmt.Sprintf("init setup took %.2f second(s)", time.Since(startTime).Seconds()))

	if *watchDog {
		maxPriceDifference = cfg.WatchDog.MaxPriceDrop

		logging.LogWarning("Running app in watchdog mode")

		logging.LogInfo("Comparing potential daily requests with limit, please wait")

		if err := query.CompareRequestsDayWithLimit(cfg.WatchDog.RetryInterval); err != nil {
			logging.LogFatal(err.Error())
		}

		logging.LogSuccess("No potential request limit exceeds found")

		utils.InitMail(
			cfg.WatchDog.SMTPHost,
			cfg.WatchDog.SMTPPort,
			cfg.WatchDog.SMTPUser,
			cfg.WatchDog.SMTPPassword,
			cfg.WatchDog.SMTPFrom,
			cfg.WatchDog.SMTPTo,
		)

		stopUpdatesCheck := make(chan bool)

		go updater.PeriodicUpdateCheck(stopUpdatesCheck)

		rerunticker := time.NewTicker(time.Duration(cfg.WatchDog.RetryInterval) * time.Hour)
		stopRerun := make(chan bool)

		// Run the app once and the on every tick.
		priceDifference, err := query.RunQuery(cfg.WatchDog.SteamRetryInterval)
		if err != nil {
			if strings.Contains(err.Error(), "last run has been less than 3 minutes ago") {
				logging.LogFatal(err.Error())
			}

			if strings.Contains(err.Error(), "last error has been less than 3 minutes ago") {
				logging.LogFatal(err.Error())
			}

			if err := query.WriteErrorCell(fmt.Errorf("%s (TS: %s)", err.Error(), time.Now().Local().Format("2006-01-02 15:04:05 CEST"))); err != nil {
				logging.LogFatal(err.Error())
			}

			if *betaFeatures {
				logging.LogWarning(fmt.Sprintf("BETA ERROR: %s", err.Error()))

				if err := query.WriteNoErrorCell(); err != nil {
					logging.LogFatal(err.Error())
				}
			}

			mailData := utils.EmailData{}
			mailData.Subject = "steamquery-v2 run failed"
			mailData.Data = utils.GenerateFailRunSummary(err)
			if err := utils.SendMail(&mailData); err != nil {
				logging.LogFatal(err.Error())
			}
		} else {
			if err := query.WriteNoErrorCell(); err != nil {
				logging.LogFatal(err.Error())
			}
		}

		logging.LogDebug(fmt.Sprintf("MAX PRICE DIFF: %.2f", maxPriceDifference*-1))
		logging.LogDebug(fmt.Sprintf("OUR PRICE DIFF: %.2f", priceDifference))

		if priceDifference < (maxPriceDifference * -1) {
			mailData := utils.EmailData{}
			mailData.Subject = "steamquery-v2 price drop alert"
			mailData.Data = utils.GeneratePriceDropWarning(priceDifference)
			if err := utils.SendMail(&mailData); err != nil {
				logging.LogFatal(err.Error())
			}
		} else {
			mailData := utils.EmailData{}
			mailData.Subject = "steamquery-v2 run summary"
			mailData.Data = utils.GenerateRunSummary(priceDifference)
			if err := utils.SendMail(&mailData); err != nil {
				logging.LogFatal(err.Error())
			}
		}

		logging.LogSuccess("Initial run completed")

		logging.LogInfo(
			fmt.Sprintf("Setup routine to rerun app every %d hour(s)", cfg.WatchDog.RetryInterval),
		)

		logging.LogInfo("Press CTRL+C to cancel anytime")

		go func() {
			for {
				select {
				case <-stopRerun:
					return
				case <-rerunticker.C:
					if !query.QueryRunning {
						priceDifference, err := query.RunQuery(
							cfg.WatchDog.SteamRetryInterval,
						)
						if err != nil {
							if err := query.WriteErrorCell(fmt.Errorf("%s (TS: %s)", err.Error(), time.Now().Local().Format("2006-01-02 15:04:05 CEST"))); err != nil {
								logging.LogFatal(err.Error())
							}

							mailData := utils.EmailData{}
							mailData.Subject = "steamquery-v2 run failed"
							mailData.Data = utils.GenerateFailRunSummary(err)
							if err := utils.SendMail(&mailData); err != nil {
								logging.LogFatal(err.Error())
							}
						}

						if priceDifference < (maxPriceDifference * -1) {
							mailData := utils.EmailData{}
							mailData.Subject = "steamquery-v2 price drop alert"
							mailData.Data = utils.GeneratePriceDropWarning(priceDifference)
							if err := utils.SendMail(&mailData); err != nil {
								logging.LogFatal(err.Error())
							}
						} else {
							mailData := utils.EmailData{}
							mailData.Subject = "steamquery-v2 run summary"
							mailData.Data = utils.GenerateRunSummary(priceDifference)
							if err := utils.SendMail(&mailData); err != nil {
								logging.LogFatal(err.Error())
							}
						}

						system.PrintBytesUsed()
						system.BytesUsed = 0

						system.Clear[runtime.GOOS]()
						logging.LogSuccess("Query run completed")
						logging.LogInfo(
							fmt.Sprintf(
								"Running query again in %d hour(s)",
								cfg.WatchDog.RetryInterval,
							),
						)
						logging.LogInfo("Press CTRL+C to exit")
					}
				}
			}
		}()

		system.ListenForCTRLC()
		rerunticker.Stop()
		stopRerun <- true
		stopUpdatesCheck <- true

		if err := statistics.CloseStatistics(); err != nil {
			logging.LogFatal(err.Error())
		}
	} else {
		if _, err := query.RunQuery(cfg.WatchDog.SteamRetryInterval); err != nil {
			if strings.Contains(err.Error(), "last run has been less than 3 minutes ago") {
				logging.LogFatal(err.Error())
			}

			if strings.Contains(err.Error(), "last error has been less than 3 minutes ago") {
				logging.LogFatal(err.Error())
			}

			if err := query.WriteErrorCell(fmt.Errorf("%s (TS: %s)", err.Error(), time.Now().Local().Format("2006-01-02 15:04:05 CEST"))); err != nil {
				logging.LogFatal(err.Error())
			}

			if *betaFeatures {
				logging.LogWarning(fmt.Sprintf("BETA ERROR: %s", err.Error()))

				if err := query.WriteNoErrorCell(); err != nil {
					logging.LogFatal(err.Error())
				}
			}
		} else {
			if err := query.WriteNoErrorCell(); err != nil {
				logging.LogFatal(err.Error())
			}
		}
	}

	system.PrintBytesUsed()

	logging.LogSuccess("Done, exiting app now")

	logging.LogDebug(
		fmt.Sprintf("Program ran for %.2f second(s)", time.Since(startTime).Seconds()),
	)

	// App exit
	if err := logging.CloseLogFiles(); err != nil {
		log.Fatalf("Error closing log files: %s\n", err.Error())
	}
}

func printAsciiArt() {
	asciiArt := figure.NewColorFigure("steamquery v2", "small", "green", true)
	asciiArt.Print()
}
